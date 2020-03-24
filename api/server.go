package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"gxclient-adapter/types"
	"gxclient-go/api/broadcast"
	"gxclient-go/api/database"
	"gxclient-go/api/history"
	"gxclient-go/api/login"
	"gxclient-go/rpc"
	"gxclient-go/rpc/http"
	"gxclient-go/rpc/websocket"
	"gxclient-go/sign"
	"gxclient-go/transaction"
	gxcTypes "gxclient-go/types"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

var restClient *RestClient
var once *sync.Once = &sync.Once{}

type RestClient struct {
	cc rpc.CallCloser

	// Database represents database_api
	Database *database.API

	// NetworkBroadcast represents network_broadcast_api
	Broadcast *broadcast.API
	//
	// History represents history_api
	History *history.API

	// Login represents login_api
	Login *login.API

	chainID string
}

func NewRestClient(url string) (*RestClient, error) {
	// transport
	var cc rpc.CallCloser
	var err error
	if strings.HasPrefix(url, "http") || strings.HasPrefix(url, "https") {
		cc, err = http.NewHttpTransport(url)
	} else {
		cc, err = websocket.NewTransport(url)
	}
	if err != nil {
		return nil, err
	}

	client := &RestClient{cc: cc}

	if strings.HasPrefix(url, "http") || strings.HasPrefix(url, "https") {
		client.Database = database.NewAPI(0, cc)
		chainID, err := client.Database.GetChainId()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get database ID")
		}
		client.chainID = chainID
		client.History = history.NewAPI(1, cc)
		client.Broadcast = broadcast.NewAPI(1, cc)
		return client, nil
	}

	// login
	loginAPI := login.NewAPI(cc)
	client.Login = loginAPI

	// database
	databaseAPIID, err := loginAPI.Database()
	if err != nil {
		return nil, err
	}
	client.Database = database.NewAPI(databaseAPIID, client.cc)

	// database ID
	chainID, err := client.Database.GetChainId()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database ID")
	}
	client.chainID = chainID

	// history
	historyAPIID, err := loginAPI.History()
	if err != nil {
		return nil, err
	}
	client.History = history.NewAPI(historyAPIID, client.cc)

	// network broadcast
	networkBroadcastAPIID, err := loginAPI.NetworkBroadcast()
	if err != nil {
		return nil, err
	}
	client.Broadcast = broadcast.NewAPI(networkBroadcastAPIID, client.cc)

	return client, nil
}

func GetInstance(url string) (*RestClient, error) {
	var err error
	once.Do(func() {
		restClient, err = NewRestClient(url)
	})
	if err != nil {
		return nil, err
	}
	return restClient, err
}

//pubkey to accountId
func (restClient *RestClient) Pubkey2accountId(pubkey string) ([]string, error) {
	accounts, err := restClient.Database.GetAccountsByPublicKey(pubkey)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

//accountId to address
func (restClient *RestClient) AccountId2address(accountId string) (string, error) {
	accounts, err := restClient.Database.GetAccountsByIds(accountId)
	if err != nil {
		return "", err
	}
	if len(accounts) == 0 {
		return "", errors.Errorf("account %s not exist", accountId)
	}
	return accounts[0].Name, nil
}

//accountId to address
func (restClient *RestClient) Pubkey2address(pubkey string) ([]string, error) {
	var adds []string
	ids, err := restClient.Pubkey2accountId(pubkey)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, errors.Errorf("no linked account")
	}
	for _, id := range ids {
		if add, err := restClient.AccountId2address(id); err == nil {
			adds = append(adds, add)
		}
	}
	return adds, nil
}

//max block number
func (restClient *RestClient) GetBlockCount() (uint32, error) {
	properties, err := restClient.Database.GetDynamicGlobalProperties()
	if err != nil {
		return 0, err
	}
	return properties.LastIrreversibleBlockNum, nil
}

//blocks containing txs
func (restClient *RestClient) GetBlockTxs(block_no uint32) ([]*types.Tx, error) {
	block, err := restClient.Database.GetBlock(block_no)
	if err != nil {
		return nil, err
	}

	var result []*types.Tx
	transactions := block.Transactions

	for i, transaction := range transactions {
		txs, err := restClient.TransactionToTx(&transaction, block.TransactionIds[i])
		if err != nil {
			return nil, err
		}
		result = append(result, txs...)
	}
	return result, nil
}

//address balance
func (restClient *RestClient) BalanceForAddress(address string, symbol string) ([]*types.Asset, error) {
	//未指定则返回主资产GXC
	if len(symbol) == 0 {
		symbol = "GXC"
	}

	asset, err := restClient.Database.GetAsset(symbol)
	if err != nil {
		return nil, err
	}

	var assets []*types.Asset
	assetsAmount, err := restClient.Database.GetNamedAccountBalances(address, asset.ID.String())
	if err != nil {
		return nil, err
	}

	assets = append(assets, &types.Asset{
		TokenCode:       symbol,
		TokenIdentifier: asset.ID.String(),
		TokenDecimal:    asset.Precision,
		Balance:         assetsAmount[0].Amount,
	})
	return assets, nil
}

//address balances
func (restClient *RestClient) BalancesForAddress(address string) ([]*types.Asset, error) {
	var assets []*types.Asset
	assetsAmounts, err := restClient.Database.GetNamedAccountBalances(address)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, a := range assetsAmounts {
		ids = append(ids, a.AssetID.String())
	}
	gxcAssets, err := restClient.Database.GetAssets(ids...)
	if err != nil {
		return nil, err
	}
	gxcAssetsMap := map[string]database.Asset{}
	for _, gxcAsset := range gxcAssets {
		gxcAssetsMap[gxcAsset.ID.String()] = *gxcAsset
	}
	for _, a := range assetsAmounts {
		ids = append(ids, a.AssetID.String())
		assets = append(assets, &types.Asset{
			TokenCode:       gxcAssetsMap[a.AssetID.String()].Symbol,
			TokenIdentifier: a.AssetID.String(),
			TokenDecimal:    gxcAssetsMap[a.AssetID.String()].Precision,
			Balance:         a.Amount,
		})
	}
	return assets, nil
}

//address tx list
func (restClient *RestClient) TxsForAddress(address, since_tx_id string, limit int) ([]*types.Tx, error) {
	var txs []*types.Tx
	acc, err := restClient.Database.GetAccount(address)
	if err != nil {
		return nil, err
	}
	//sice为空，取最近的limit条
	if len(since_tx_id) == 0 {
		since_tx_id = "1.11.0"
	}
	ophs, err := restClient.History.GetAccountHistory(acc.ID.String(), "1.11.0", limit, since_tx_id)
	if err != nil {
		return nil, err
	}

	for _, oph := range ophs {
		if byte_s, err := json.Marshal(oph); err == nil {
			tx := gjson.ParseBytes(byte_s)
			operation := tx.Get("op")
			tx_op_code := operation.Get("0").Uint()
			if gxcTypes.OpType(tx_op_code) != gxcTypes.TransferOpType {
				continue
			}

			in := &types.UTXO{
				Value:     operation.Get("1.amount.amount").Uint(),
				Address:   operation.Get("1.from").String(),
				TokenCode: operation.Get("1.amount.asset_id").String(),
			}
			out := &types.UTXO{
				Value:     operation.Get("1.amount.amount").Uint(),
				Address:   operation.Get("1.to").String(),
				TokenCode: operation.Get("1.amount.asset_id").String(),
			}

			extra := map[string]string{}
			if operation.Get("1.memo").Exists() {
				extra["from"] = operation.Get("1.memo.from").String()
				extra["to"] = operation.Get("1.memo.to").String()
				extra["message"] = operation.Get("1.memo.from").String()
				extra["nonce"] = strconv.FormatUint(operation.Get("1.memo.nonce").Uint(), 10)
			}
			extra["block_num"] = strconv.FormatInt(tx.Get("block_num").Int(), 10)
			extra["trx_in_block"] = strconv.FormatInt(tx.Get("trx_in_block").Int(), 10)
			extra["id"] = tx.Get("id").String()

			txOb := &types.Tx{
				TxHash:      "",
				Inputs:      []types.UTXO{*in},
				Outputs:     []types.UTXO{*out},
				TxAt:        "",
				BlockNumber: 0,
				ConfirmedAt: "",
				Extra:       extra,
			}
			txs = append(txs, txOb)
		}
	}
	return txs, nil
}

//tx detail by id
func (restClient *RestClient) GetTransactionByBlockNumAndId(block_num uint32, trx_in_block int) ([]*types.Tx, error) {
	block, err := restClient.Database.GetBlock(block_num)
	if err != nil {
		return nil, err
	}
	txs, err := restClient.TransactionToTx(&block.Transactions[trx_in_block], block.TransactionIds[trx_in_block])
	if err != nil {
		return nil, err
	}
	return txs, nil
}

//tx detail by id
func (restClient *RestClient) GetTransaction(tx_hash string) ([]*types.Tx, error) {
	transaction, err := restClient.Database.GetTransactionByTxid(tx_hash)
	if err != nil {
		return nil, err
	}
	txs, err := restClient.TransactionToTx(transaction, tx_hash)
	if err != nil {
		return nil, err
	}
	return txs, nil
}

func (restClient *RestClient) BuildTransaction(from_address, to_address, symbol string, amount float64, memoOb *gxcTypes.Memo) (string, error) {
	fromAccount, err := restClient.Database.GetAccount(from_address)
	if err != nil {
		return "", err
	}

	toAccount, err := restClient.Database.GetAccount(to_address)
	if err != nil {
		return "", err
	}

	//token_identifier(empty for the main coin)
	if symbol == "" {
		symbol = "GXC"
	}
	amountSymbol, err := restClient.Database.GetAsset(symbol)
	if err != nil {
		return "", err
	}
	amountAssets := gxcTypes.AssetAmount{
		AssetID: amountSymbol.ID,
		Amount:  uint64(amount * math.Pow10(int(amountSymbol.Precision))),
	}

	fee, err := restClient.Database.GetAsset("GXC")
	if err != nil {
		return "", err
	}
	feeAssets := gxcTypes.AssetAmount{
		AssetID: fee.ID,
		Amount:  0,
	}

	op := gxcTypes.NewTransferOperation(gxcTypes.MustParseObjectID(fromAccount.ID.String()), gxcTypes.MustParseObjectID(toAccount.ID.String()), amountAssets, feeAssets, memoOb)

	fees, err := restClient.Database.GetRequiredFee([]gxcTypes.Operation{op}, feeAssets.AssetID.String())
	if err != nil {
		return "", err
	}
	op.Fee.Amount = fees[0].Amount

	props, err := restClient.Database.GetDynamicGlobalProperties()
	if err != nil {
		return "", errors.Wrap(err, "failed to get dynamic global properties")
	}

	block, err := restClient.Database.GetBlock(props.LastIrreversibleBlockNum)
	if err != nil {
		return "", errors.Wrap(err, "failed to get block")
	}

	refBlockPrefix, err := sign.RefBlockPrefix(block.Previous)
	if err != nil {
		return "", errors.Wrap(err, "failed to sign block prefix")
	}

	expiration := props.Time.Add(10 * time.Minute)
	stx := gxcTypes.NewSignedTransaction(&gxcTypes.Transaction{
		RefBlockNum:    sign.RefBlockNum(props.LastIrreversibleBlockNum - 1&0xffff),
		RefBlockPrefix: refBlockPrefix,
		Expiration:     gxcTypes.Time{Time: &expiration},
	})

	stx.PushOperation(op)

	var b bytes.Buffer
	x := transaction.NewEncoder(&b)

	if err := x.Encode(stx.Transaction); err != nil {
		return "", nil
	}
	s := hex.EncodeToString(b.Bytes())
	fmt.Println(s)

	str, _ := json.Marshal(stx)
	return string(str), nil
}

func (restClient *RestClient) TransactionFee(raw_unsigned_tx_hex string) (string, error) {
	var stx *gxcTypes.SignedTransaction
	json.Unmarshal([]byte(raw_unsigned_tx_hex), &stx)
	var transferOp *gxcTypes.TransferOperation
	byte, err := json.Marshal(stx.Transaction.Operations[0])
	if err != nil {
		return "", err
	}
	json.Unmarshal(byte, &transferOp)

	fees, err := restClient.Database.GetRequiredFee([]gxcTypes.Operation{stx.Operations[0]}, transferOp.Fee.AssetID.String())
	if err != nil {
		return "", err
	}

	transferOp.Fee.Amount = fees[0].Amount
	stx.Operations = nil
	stx.PushOperation(transferOp)

	str, _ := json.Marshal(stx)
	return string(str), nil
}

//sign unsigned tx with given signature and broadcast
func (restClient *RestClient) SignTransaction(raw_unsigned_tx_hex string) (*types.Tx, error) {
	var stx *gxcTypes.SignedTransaction
	json.Unmarshal([]byte(raw_unsigned_tx_hex), &stx)
	resp, err := restClient.Broadcast.BroadcastTransactionSynchronous(stx.Transaction)
	if err != nil {
		return nil, err
	}

	txs, err := restClient.TransactionToTx(stx.Transaction, resp.ID)
	if err != nil {
		return nil, err
	}
	tx := txs[0]
	tx.BlockNumber = int64(resp.BlockNum)
	return tx, nil
}

//token_code or token_identifier to token detail
func (restClient *RestClient) TokenDetail(token string) (*types.Asset, error) {
	gxcAsset, err := restClient.Database.GetAsset(token)
	if err != nil {
		return nil, err
	}
	return &types.Asset{
		TokenCode:       gxcAsset.Symbol,
		TokenIdentifier: gxcAsset.ID.String(),
		TokenDecimal:    gxcAsset.Precision,
		Balance:         0,
	}, nil

}

func (restClient *RestClient) TransactionToTx(transaction *gxcTypes.Transaction, transactionId string) ([]*types.Tx, error) {
	var txs []*types.Tx
	for _, op := range transaction.Operations {
		if op.Type() != gxcTypes.TransferOpType {
			continue
		}
		var transferOp gxcTypes.TransferOperation
		byte, err := json.Marshal(op)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(byte, &transferOp)

		asset, err := restClient.Database.GetAsset(transferOp.Amount.AssetID.String())
		if err != nil {
			return nil, err
		}
		accounts, err := restClient.Database.GetAccountsByIds(transferOp.From.String(), transferOp.To.String())
		if err != nil {
			return nil, err
		}

		in := &types.UTXO{
			Value:           transferOp.Amount.Amount,
			Address:         accounts[0].Name,
			TokenCode:       asset.Symbol,
			TokenIdentifier: asset.ID.String(),
			TokenDecimal:    asset.Precision,
		}

		out := &types.UTXO{
			Value:           transferOp.Amount.Amount,
			Address:         accounts[1].Name,
			TokenCode:       asset.Symbol,
			TokenIdentifier: asset.ID.String(),
			TokenDecimal:    asset.Precision,
		}

		extra := map[string]string{}
		if transferOp.Memo != nil {
			extra["from"] = transferOp.Memo.From.String()
			extra["to"] = transferOp.Memo.To.String()
			extra["message"] = transferOp.Memo.Message.String()
			extra["nonce"] = strconv.FormatUint(uint64(transferOp.Memo.Nonce), 10)
		}
		var txHash string
		if len(transactionId) > 0 {
			txHash = transactionId
		}

		tx := &types.Tx{
			TxHash:      txHash,
			Inputs:      []types.UTXO{*in},
			Outputs:     []types.UTXO{*out},
			TxAt:        "",
			BlockNumber: 0,
			ConfirmedAt: "",
			Extra:       extra,
		}
		txs = append(txs, tx)
	}
	return txs, nil
}
