package tests

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"gxclient-adapter/api"
	"gxclient-go/faucet"
	gxcTypes "gxclient-go/types"
	"strconv"
	"testing"
)

func Test_Pubkey2address(t *testing.T) {
	restClient, err := api.GetInstance(testNetHttp)
	require.Nil(t, err)
	address, err := restClient.Pubkey2address(testPubHexCom)
	require.Nil(t, err)
	fmt.Println(address)
}

func Test_AccountId2address(t *testing.T) {
	restClient, err := api.GetInstance(testNetHttp)
	require.Nil(t, err)
	address, err := restClient.AccountId2address(testAccountId)
	require.Nil(t, err)
	fmt.Println(address)
}

func Test_Address2AccountId(t *testing.T) {
	restClient, err := api.GetInstance(testNetHttp)
	require.Nil(t, err)
	id, err := restClient.Address2AccountId(testAccountName)
	require.Nil(t, err)
	fmt.Println(id)
}

func Test_GetBlockCount(t *testing.T) {
	restClient, err := api.GetInstance(testNetHttp)
	require.Nil(t, err)
	blockNum, err := restClient.GetBlockCount()
	require.Nil(t, err)
	fmt.Println(blockNum)
}

func Test_GetBlockTxs(t *testing.T) {
	restClient, err := api.GetInstance(testNetHttp)
	require.Nil(t, err)
	txs, err := restClient.GetBlockTxs(29617777)
	require.Nil(t, err)
	str, _ := json.Marshal(txs)
	fmt.Println(string(str))
}

func Test_BalanceForAddress(t *testing.T) {
	restClient, err := api.GetInstance(testNetHttp)
	require.Nil(t, err)
	balance, err := restClient.BalanceForAddress("dev", "GXC")
	require.Nil(t, err)
	str, _ := json.Marshal(balance)
	fmt.Println(string(str))
}

func Test_BalancesForAddress(t *testing.T) {
	restClient, err := api.GetInstance(testNetHttp)
	require.Nil(t, err)
	balance, err := restClient.BalancesForAddress("dev")
	require.Nil(t, err)
	str, _ := json.Marshal(balance)
	fmt.Println(string(str))
}

func Test_TxsForAddress(t *testing.T) {
	restClient, err := api.GetInstance(testNetHttp)
	require.Nil(t, err)
	//第一页
	txs1, err := restClient.TxsForAddress(testAccountName, "", 5)
	require.Nil(t, err)
	str1, _ := json.Marshal(txs1)
	fmt.Println(string(str1))
	//下一页
	id := txs1[len(txs1)-1].Extra["id"]
	obId := gxcTypes.MustParseObjectID(id)
	obId.ID = obId.ID - 1

	txs2, err := restClient.TxsForAddress(testAccountName, obId.String(), 10)
	require.Nil(t, err)
	str2, _ := json.Marshal(txs2)
	fmt.Println(string(str2))

	//tx detail
	tx := txs1[0]
	blockNum, err := strconv.ParseUint(tx.Extra["block_num"], 10, 32)
	trxInBlock, err := strconv.ParseInt(tx.Extra["trx_in_block"], 10, 32)
	txs, err := restClient.GetTransactionByBlockNumAndId(uint32(blockNum), int(trxInBlock))
	require.Nil(t, err)
	txStr, _ := json.Marshal(txs)
	fmt.Println(string(txStr))
}

func Test_TxsForAddressFull(t *testing.T) {
	restClient, err := api.GetInstance(testNetHttp)
	require.Nil(t, err)
	//第一页
	txs1, err := restClient.TxsForAddressFull(testAccountName, "", 10)
	require.Nil(t, err)
	str1, _ := json.Marshal(txs1)
	fmt.Println(string(str1))
	//下一页
	id := txs1[len(txs1)-1].Extra["id"]
	obId := gxcTypes.MustParseObjectID(id)
	obId.ID = obId.ID - 1

	txs2, err := restClient.TxsForAddressFull(testAccountName, obId.String(), 10)
	require.Nil(t, err)
	str2, _ := json.Marshal(txs2)
	fmt.Println(string(str2))
}

func Test_GetTransaction(t *testing.T) {
	restClient, err := api.GetInstance(testNetHttp)
	require.Nil(t, err)
	txs, err := restClient.GetTransaction("0101813c34fb033b7ba7a30c675bfa1b949357d8")
	require.Nil(t, err)
	str, _ := json.Marshal(txs)
	fmt.Println(string(str))
}

func Test_TokenDetail(t *testing.T) {
	restClient, err := api.GetInstance(testNetHttp)
	require.Nil(t, err)
	asset1, err := restClient.TokenDetail("GXC")
	require.Nil(t, err)
	str1, _ := json.Marshal(asset1)
	fmt.Println(string(str1))

	asset2, err := restClient.TokenDetail("1.3.1")
	require.Nil(t, err)
	str2, _ := json.Marshal(asset2)
	fmt.Println(string(str2))
}

func TestApi_GetRegister(t *testing.T) {
	transaction, err := faucet.Register(testFaucet, "cli-wallet-test-16", testPub, testPub, testPub)
	require.Nil(t, err)
	str, _ := json.Marshal(transaction)
	fmt.Println(string(str))
}
