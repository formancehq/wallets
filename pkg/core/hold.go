package core

type Hold struct {
	ID       string   `json:"id"`
	WalletID string   `json:"walletID"`
	Metadata Metadata `json:"metadata"`
}
