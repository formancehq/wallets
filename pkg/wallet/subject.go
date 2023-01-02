package wallet

import (
	"fmt"

	"github.com/formancehq/wallets/pkg/core"
)

const (
	SubjectTypeLedgerAccount string = "ACCOUNT"
	SubjectTypeWallet        string = "WALLET"
)

type Subject struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

func (s Subject) resolveAccount(chart *core.Chart) string {
	switch s.Type {
	case SubjectTypeLedgerAccount:
		return s.Identifier
	case SubjectTypeWallet:
		return chart.GetMainAccount(s.Identifier)
	}
	panic("unknown type")
}

func (s Subject) Validate() error {
	if s.Type != SubjectTypeWallet && s.Type != SubjectTypeLedgerAccount {
		return fmt.Errorf("unknown source type: %s", s.Type)
	}
	return nil
}

type Subjects []Subject

func (subjects Subjects) resolveAccounts(chart *core.Chart) []string {
	if len(subjects) == 0 {
		subjects = []Subject{DefaultCreditSource}
	}
	resolvedSources := make([]string, 0)
	for _, source := range subjects {
		resolvedSources = append(resolvedSources, source.resolveAccount(chart))
	}
	return resolvedSources
}

func (subjects Subjects) Validate() error {
	for _, source := range subjects {
		if err := source.Validate(); err != nil {
			return err
		}
	}
	return nil
}
