package tests

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"gxclient-adapter/api"
	"gxclient-go/keypair"
	gxcTypes "gxclient-go/types"
	"math"
	"testing"
)

const (
	testNetHttp = "https://testnet.gxchain.org"
	testNetWss  = "wss://testnet.gxchain.org"
	testFaucet  = "https://testnet.faucet.gxchain.org/account/register"

	testAccountName = "cli-wallet-test"
	testAccountId   = "1.2.4015"
	testPri         = "5JsvYffKR8n4yNfCk36KkKFCzg6vo5fdBqqDJLavSifXSV9NABo"
	testMemoPri     = "5JsvYffKR8n4yNfCk36KkKFCzg6vo5fdBqqDJLavSifXSV9NABo"
	testPriHex      = "8bf481abeecbb3654e5f8581af0c8bd8d83df31fb2df8cac0440c100d84a3141"
	testMemoPriHex  = "8bf481abeecbb3654e5f8581af0c8bd8d83df31fb2df8cac0440c100d84a3141"
	testPub         = "GXC58owosbFrudGVp8VCuMvDWpenx7AZSLwxEtAVqjWeqZ4YVLLWb"
	testPubHexCom   = "0220843df25002cef45f3a5896806d4b11fcd3f554693107c24622c4bdd1199ae3"
)

func Test_Simple(t *testing.T) {
	client, err := api.GetInstance(testNetWss)
	require.Nil(t, err)
	client.Database.GetAccount("nathan")
	require.Nil(t, err)
}

func Test_Deserialize(t *testing.T) {
	raw_tx_hex := "{\"ref_block_num\":14710,\"ref_block_prefix\":3383196508,\"expiration\":\"2020-03-19T04:18:42\",\"operations\":[[0,{\"from\":\"1.2.4015\",\"to\":\"1.2.17\",\"amount\":{\"amount\":318000,\"asset_id\":\"1.3.1\"},\"fee\":{\"amount\":1210,\"asset_id\":\"1.3.1\"},\"memo\":{\"from\":\"GXC58owosbFrudGVp8VCuMvDWpenx7AZSLwxEtAVqjWeqZ4YVLLWb\",\"to\":\"GXC8AoHzhXhMRV9AFTihMAcQPNXKFEZCeYNYomdcc7vh8Gzp7b7xP\",\"nonce\":13402076872543869991,\"message\":\"2a127ecb4ed849f5806ea2bdabbdc1ae24c7ae268f5759169c032f68628b6e3e\"},\"extensions\":[]}]],\"signatures\":null}"
	tx, err := api.Deserialize(raw_tx_hex)
	require.Nil(t, err)
	str, _ := json.Marshal(tx)
	fmt.Println(string(str))
}

func Test_DeserializeMemo(t *testing.T) {
	from := testPub
	to := "GXC8AoHzhXhMRV9AFTihMAcQPNXKFEZCeYNYomdcc7vh8Gzp7b7xP"
	message := "78ac2144776911f195c934c000f3036c374015f991d3d4b928c418f98ab2926e"
	nonce := gxcTypes.UInt64(3768974234669558428)

	str, err := api.DeserializeMemo(testMemoPriHex, from, to, message, nonce)
	require.Nil(t, err)
	fmt.Println(str)
}

func Test_Transfer(t *testing.T) {
	restClient, err := api.GetInstance(testNetHttp)
	require.Nil(t, err)

	to := "init0"
	memo := "transfer memo"
	var memoOb *gxcTypes.Memo

	//step0:	client do param preparation
	if len(memo) > 0 {
		fromAccount, err := restClient.Database.GetAccount(testAccountName)
		require.Nil(t, err)
		toAccount, err := restClient.Database.GetAccount(to)
		require.Nil(t, err)
		memoOb, err = api.EncryptMemo(testMemoPriHex, memo, &fromAccount.Options.MemoKey, &toAccount.Options.MemoKey)
		require.Nil(t, err)
	}

	//step1:	server build transaction
	realAmount := 3.26
	symbol := "GXC"
	tokenDetail, err := restClient.TokenDetail(symbol)
	amount := uint64(realAmount * math.Pow10(int(tokenDetail.TokenDecimal)))
	unSignedTxStr, err := restClient.BuildTransaction(testAccountName, to, symbol, amount, memoOb)
	require.Nil(t, err)
	fmt.Printf("Build Transaction %s \n", unSignedTxStr)

	//step2:	client sign transaction
	chainId, err := restClient.Database.GetChainId()
	require.Nil(t, err)
	signature, err := api.Sign(testPriHex, chainId, unSignedTxStr)
	fmt.Printf("signature %s \n", signature)
	require.Nil(t, err)

	//step3:	server broadcast signed transaction
	tx, err := restClient.SignTransaction(unSignedTxStr, signature)
	txResultStr, _ := json.Marshal(tx)
	fmt.Printf("tx result %s \n", txResultStr)
	require.Nil(t, err)

	result, _ := json.Marshal(tx)
	fmt.Println(string(result))
}

func Test_GenerateKeyPair(t *testing.T) {
	keyPair, err := keypair.GenerateKeyPair("")
	require.Nil(t, err)
	fmt.Println(keyPair.BrainKey)
	fmt.Println(keyPair.PrivateKey.ToWIF())
	fmt.Println(keyPair.PrivateKey.PublicKey().String())
}

func Test_PrivateToPublic(t *testing.T) {
	pub, err := keypair.PrivateToPublic(testPri)
	require.Nil(t, err)
	fmt.Println(pub)
}

func Test_IsValidPrivate(t *testing.T) {
	bool := keypair.IsValidPrivate(testPri)
	fmt.Println(bool)
}

func Test_IsValidPublic(t *testing.T) {
	bool := keypair.IsValidPublic(testPub)
	fmt.Println(bool)
}

func Test_Convert(t *testing.T) {
	priHex, err := api.PriKeyWifToHex(testPri)
	require.Nil(t, err)
	fmt.Println(priHex)
	require.Equal(t, priHex, testPriHex)

	priWif, err := api.PriKeyHexToWif(testPriHex)
	require.Nil(t, err)
	fmt.Println(priWif)
	require.Equal(t, priWif, testPri)

	pubHex, err := api.PubKeyBase58ToHex(testPub)
	require.Nil(t, err)
	fmt.Println(pubHex)
	require.Equal(t, pubHex, testPubHexCom)

	pubBase58, err := api.PubKeyHexToBase58(testPubHexCom)
	require.Nil(t, err)
	fmt.Println(pubBase58)
	require.Equal(t, pubBase58, testPub)
}
