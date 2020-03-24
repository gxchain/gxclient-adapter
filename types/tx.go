package types

type UTXO struct {
	Value           uint64 `json:"value,omitempty"`
	Address         string `json:"address,omitempty"`
	TokenCode       string `json:"token_code,omitempty"`
	TokenIdentifier string `json:"token_identifier,omitempty"`
	TokenDecimal    uint8  `json:"token_decimal,omitempty"`
}

type Tx struct {
	TxHash      string            `json:"tx_hash,omitempty"`
	Inputs      []UTXO            `json:"inputs"`
	Outputs     []UTXO            `json:"outputs"`
	TxAt        string            `json:"tx_at,omitempty"`
	BlockNumber int64             `json:"block_no,omitempty"`
	ConfirmedAt string            `json:"confirmed_at,omitempty"`
	Extra       map[string]string `json:"extra"`
}
