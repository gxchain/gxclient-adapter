package types

type UTXO struct {
	Value     uint64 `json:"value"`
	Address   string `json:"address"`
	TokenCode string `json:"token_code"`
}

type Tx struct {
	TxHash      string            `json:"tx_hash,omitempty"`
	Inputs      []UTXO            `json:"inputs"`
	Outputs     []UTXO            `json:"outputs"`
	TxAt        string            `json:"tx_at"`
	BlockNumber int64             `json:"block_no,omitempty"`
	ConfirmedAt string            `json:"confirmed_at,omitempty"`
	Extra       map[string]string `json:"extra"`
}
