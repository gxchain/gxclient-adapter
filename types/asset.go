package types

type Assets []Asset

type Asset struct {
	TokenCode       string `json:"token_code"`
	TokenIdentifier string `json:"token_identifier"`
	TokenDecimal    uint8  `json:"token_decimal"`
	Balance         uint64 `json:"balance"`
}
