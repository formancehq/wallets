package wallet

import (
	"encoding/json"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
)

type Transaction struct {
	shared.V2Transaction
	Ledger string `json:"ledger"`
}

func (t Transaction) MarshalJSON() ([]byte, error) {
	asJSON, err := json.Marshal(t.V2Transaction)
	if err != nil {
		return nil, err
	}
	asMap := make(map[string]any)
	if err := json.Unmarshal(asJSON, &asMap); err != nil {
		return nil, err
	}
	asMap["ledger"] = t.Ledger
	return json.Marshal(asMap)
}
