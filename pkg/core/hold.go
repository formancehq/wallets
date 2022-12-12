package core

type Hold struct {
	ID       string   `json:"id"`
	WalletID string   `json:"wallet_id"`
	Metadata Metadata `json:"metadata"`
}
