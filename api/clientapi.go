package api

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gxclient-adapter/types"
	gxcTypes "gxclient-go/types"
	"strconv"
)

func Deserialize(raw_tx_hex string) ([]*types.Tx, error) {
	var stx gxcTypes.SignedTransaction
	json.Unmarshal([]byte(raw_tx_hex), &stx)

	txs, err := transactionToTx(stx.Transaction)
	if err != nil {
		return nil, err
	}
	return txs, err
}

func EncryptMemo(memoPri, memo string, fromPub, toPub *gxcTypes.PublicKey) (*gxcTypes.Memo, error) {
	memoPriKey, err := gxcTypes.NewPrivateKeyFromWif(memoPri)
	if err != nil {
		return nil, err
	}

	var memoOb = &gxcTypes.Memo{}
	memoOb.From = *fromPub
	memoOb.To = *toPub
	memoOb.Nonce = gxcTypes.GetNonce()

	err = memoOb.Encrypt(memoPriKey, memo)

	return memoOb, err
}

func DeserializeMemo(memoPri, from, to, message string, nonce gxcTypes.UInt64) (string, error) {
	priKey, err := gxcTypes.NewPrivateKeyFromWif(memoPri)
	if err != nil {
		return "", err
	}
	toPubKey, err := gxcTypes.NewPublicKeyFromString(to)
	if err != nil {
		return "", err
	}
	fromPubKey, err := gxcTypes.NewPublicKeyFromString(from)
	if err != nil {
		return "", err
	}
	var buffer gxcTypes.Buffer
	err = buffer.FromString(message)
	if err != nil {
		return "", err
	}
	memo := &gxcTypes.Memo{
		From:    *fromPubKey,
		To:      *toPubKey,
		Nonce:   nonce,
		Message: buffer,
	}

	result, err := memo.Decrypt(priKey)
	if err != nil {
		return "", err
	}
	return result, nil
}

func Sign(activePri, chainId, raw_tx_hex string) (string, error) {
	var stx *gxcTypes.SignedTransaction
	json.Unmarshal([]byte(raw_tx_hex), &stx)

	if err := stx.Sign([]string{activePri}, chainId); err != nil {
		return "", errors.Wrap(err, "failed to sign the transaction")
	}

	result, _ := json.Marshal(stx)
	return string(result), nil
}

func transactionToTx(transaction *gxcTypes.Transaction) ([]*types.Tx, error) {
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

		in := &types.UTXO{
			Value:     transferOp.Amount.Amount,
			Address:   transferOp.From.String(),
			TokenCode: transferOp.Amount.AssetID.String(),
		}

		out := &types.UTXO{
			Value:     transferOp.Amount.Amount,
			Address:   transferOp.To.String(),
			TokenCode: transferOp.Amount.AssetID.String(),
		}

		extra := map[string]string{}
		if transferOp.Memo != nil {
			extra["from"] = transferOp.Memo.From.String()
			extra["to"] = transferOp.Memo.To.String()
			extra["message"] = transferOp.Memo.Message.String()
			extra["nonce"] = strconv.FormatUint(uint64(transferOp.Memo.Nonce), 10)
		}

		tx := &types.Tx{
			TxHash:      "",
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
