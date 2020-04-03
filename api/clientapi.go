package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
	"github.com/juju/errors"
	"gxclient-adapter/types"
	gxcTypes "gxclient-go/types"
	"gxclient-go/util"
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

func EncryptMemo(memoPriHex, memo string, fromPub, toPub *gxcTypes.PublicKey) (*gxcTypes.Memo, error) {
	wif, _ := PriKeyHexToWif(memoPriHex)
	memoPriKey, err := gxcTypes.NewPrivateKeyFromWif(wif)
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

func DeserializeMemo(memoPriHex, from, to, message string, nonce gxcTypes.UInt64) (string, error) {
	wif, _ := PriKeyHexToWif(memoPriHex)
	priKey, err := gxcTypes.NewPrivateKeyFromWif(wif)
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

func Sign(activePriHex, chainId, raw_tx_hex string) (string, error) {
	var stx *gxcTypes.SignedTransaction
	json.Unmarshal([]byte(raw_tx_hex), &stx)

	wif, err := PriKeyHexToWif(activePriHex)
	if err != nil {
		return "", err
	}

	if err := stx.Sign([]string{wif}, chainId); err != nil {
		return "", errors.Annotate(err, "failed to sign the transaction")
	}

	//return signature only
	return stx.Signatures[0], nil
	//result, _ := json.Marshal(stx)
	//return string(result), nil
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
		extra["feeAmount"] = strconv.FormatUint(transferOp.Fee.Amount, 10)
		extra["feeTokenIdentifier"] = transferOp.Fee.AssetID.String()

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

func PriKeyHexToWif(priHex string) (string, error) {
	h, err := hex.DecodeString(priHex)
	if err != nil {
		return "", errors.Annotate(err, "DecodeHEX")
	}
	pri, _ := btcec.PrivKeyFromBytes(btcec.S256(), h)
	raw := append([]byte{128}, pri.D.Bytes()...)
	raw = append(raw, checksum(raw)...)
	return base58.Encode(raw), nil
}

func checksum(data []byte) []byte {
	c1 := sha256.Sum256(data)
	c2 := sha256.Sum256(c1[:])
	return c2[0:4]
}

func PriKeyWifToHex(priWif string) (string, error) {
	w, err := btcutil.DecodeWIF(priWif)
	if err != nil {
		return "", errors.Annotate(err, "DecodeWIF")
	}
	return hex.EncodeToString(w.PrivKey.Serialize()), nil
}

func PubKeyHexToBase58(pubHex string) (string, error) {
	h, err := hex.DecodeString(pubHex)
	if err != nil {
		return "", errors.Annotate(err, "DecodeHEX")
	}
	pubKey, _ := btcec.ParsePubKey(h, btcec.S256())

	buf := pubKey.SerializeCompressed()
	chk, err := util.Ripemd160Checksum(buf)
	if err != nil {
		return "", errors.Annotate(err, "Ripemd160Checksum")
	}

	b := append(pubKey.SerializeCompressed(), chk...)
	prefixChain := "GXC"
	return fmt.Sprintf("%s%s", prefixChain, base58.Encode(b)), err
}

func PubKeyBase58ToHex(pubBase58 string) (string, error) {
	prefixChain := "GXC"

	prefix := pubBase58[:len(prefixChain)]

	if prefix != prefixChain {
		return "", gxcTypes.ErrPublicKeyChainPrefixMismatch
	}

	b58 := base58.Decode(pubBase58[len(prefixChain):])
	if len(b58) < 5 {
		return "", gxcTypes.ErrInvalidPublicKey
	}
	chk1 := b58[len(b58)-4:]

	keyBytes := b58[:len(b58)-4]
	chk2, err := util.Ripemd160Checksum(keyBytes)
	if err != nil {
		return "", errors.Annotate(err, "Ripemd160Checksum")
	}
	if !bytes.Equal(chk1, chk2) {
		return "", gxcTypes.ErrInvalidPublicKey
	}

	pub, err := btcec.ParsePubKey(keyBytes, btcec.S256())
	if err != nil {
		return "", errors.Annotate(err, "ParsePubKey")
	}

	return hex.EncodeToString(pub.SerializeCompressed()), nil
}
