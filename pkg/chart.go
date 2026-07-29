package wallet

import (
	"strings"
)

const MainBalance = "main"

type Address []string

// String renders the address as a ledger account path by joining the
// segments with ':'.
//
// WARNING — known aliasing hazard: every '-' is stripped from the joined
// result. This means any two segments that differ only by dashes collapse
// to the same ledger account: "foo-bar" and "foobar" both render to
// "foobar", as do wallet IDs / balance names / hold IDs differing only in
// dash placement. Because wallet and hold IDs are UUIDs (which contain
// dashes), distinct inputs can therefore resolve to the SAME underlying
// account, causing silent collisions on create/get/debit/credit — two
// wallets, balances, or holds sharing one ledger account.
//
// Scope: this only affects segments routed through the Chart — wallet IDs,
// balance names and hold IDs. SubjectTypeLedgerAccount identifiers are used
// verbatim (see Subject.getAccount) and bypass this strip, so their dashes
// are preserved.
//
// Note segment validation deliberately ALLOWS '-' (so dashed/UUID names keep
// working), which means this collision is live, not theoretical: a name like
// "foo-bar" passes validation yet still aliases to "foobar" here.
//
// The strip cannot simply be removed: all existing accounts were created
// with dashes already stripped, so dropping it would change the address of
// every wallet/balance/hold already in the ledger and requires a data
// migration. Fixing this properly (stop stripping + migrate) should be a
// dedicated ticket.
func (addr Address) String() string {
	s := strings.Join(addr, ":")
	s = strings.ReplaceAll(s, "-", "")

	return s
}

type Chart struct {
	Prefix string
}

func NewChart(prefix string) *Chart {
	return &Chart{Prefix: prefix}
}

func (c *Chart) BasePath() Address {
	addr := Address{}

	if c.Prefix != "" {
		addr = append(addr, c.Prefix)
	}

	addr = append(addr, "wallets")

	return addr
}

func (c *Chart) GetMainBalanceAccount(walletID string) string {
	return c.GetBalanceAccount(walletID, MainBalance)
}

func (c *Chart) GetHoldAccount(holdID string) string {
	addr := c.BasePath()
	addr = append(addr, "holds")
	addr = append(addr, holdID)

	return addr.String()
}

func (c *Chart) GetBalanceAccount(walletID, balanceName string) string {
	addr := c.BasePath()
	addr = append(addr, walletID, balanceName)

	return addr.String()
}
