package core

type Wallet struct {
	ID             string `json:"id"`
	Balances       map[string]Monetary
	CounterpartyID string
	Metadata       Metadata
}
