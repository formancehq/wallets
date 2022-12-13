package core

type Wallet struct {
	ID       string              `json:"id"`
	Balances map[string]Monetary `json:"balances"`
	Metadata Metadata            `json:"metadata"`
}
