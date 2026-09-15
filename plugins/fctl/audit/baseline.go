package audit

import "sort"

// BaselineRevision pins the legacy fctl tree the command baseline below was
// read from: github.com/formancehq/fctl at commit 693c58e2, subtree
// cmd/wallets/. Re-reading the baseline against a different fctl revision is a
// deliberate change to this file, never an implicit one.
const BaselineRevision = "693c58e27865f83332e6c3199d61fed81b742f41"

// Command is one legacy fctl command constructor under cmd/wallets/.
type Command struct {
	// Path is the canonical command path as typed, without the fctl binary.
	Path string `json:"path"`
	// Aliases are the accepted alternative names for this path's last segment,
	// in declaration order. Parity means every one of them keeps working.
	Aliases []string `json:"aliases"`
	// Grouping is true for constructors that only carry child commands and
	// execute nothing. They are excluded from the executable-leaf count.
	Grouping bool `json:"grouping"`
	// Ops are the Wallets operationIds this leaf calls. Empty on a grouping
	// command, and empty on a leaf that is deliberately excluded.
	Ops []string `json:"ops"`
	// Source is the file the constructor lives in, relative to the fctl root.
	Source string `json:"source"`
	// Exclusion, when non-empty, is the evidenced reason this executable leaf
	// carries no operation. An excluded leaf must always state why.
	Exclusion string `json:"exclusion,omitempty"`
}

// Baseline is every legacy fctl command constructor under cmd/wallets/ at
// BaselineRevision: 4 grouping commands and 14 executable leaves.
//
// The SDK call in each leaf was read from its controller's Run method, which
// reaches the operation through stackClient.Wallets.V1.<Method>. The mapping is
// one leaf to exactly one operation with no exclusions, which is why this
// product has no exclusion evidence to record.
var Baseline = []Command{
	{
		Path:     "wallets",
		Aliases:  []string{"wal", "wa", "wallet"},
		Grouping: true,
		Source:   "cmd/wallets/root.go",
	},
	{
		Path:    "wallets create",
		Aliases: []string{"cr"},
		Ops:     []string{"createWallet"},
		Source:  "cmd/wallets/create.go",
	},
	{
		Path:    "wallets list",
		Aliases: []string{"ls", "l"},
		Ops:     []string{"listWallets"},
		Source:  "cmd/wallets/list.go",
	},
	{
		Path:    "wallets show",
		Aliases: []string{"sh"},
		Ops:     []string{"getWallet"},
		Source:  "cmd/wallets/show.go",
	},
	{
		Path:    "wallets update",
		Aliases: []string{"up"},
		Ops:     []string{"updateWallet"},
		Source:  "cmd/wallets/update.go",
	},
	{
		Path:    "wallets credit",
		Aliases: []string{"cr"},
		Ops:     []string{"creditWallet"},
		Source:  "cmd/wallets/credit.go",
	},
	{
		Path:    "wallets debit",
		Aliases: []string{"deb"},
		Ops:     []string{"debitWallet"},
		Source:  "cmd/wallets/debit.go",
	},
	{
		Path:     "wallets balances",
		Aliases:  []string{"balance", "bls", "bal"},
		Grouping: true,
		Source:   "cmd/wallets/balances/root.go",
	},
	{
		Path:    "wallets balances create",
		Aliases: []string{"c", "cr"},
		Ops:     []string{"createBalance"},
		Source:  "cmd/wallets/balances/create.go",
	},
	{
		Path:    "wallets balances list",
		Aliases: []string{"ls", "l"},
		Ops:     []string{"listBalances"},
		Source:  "cmd/wallets/balances/list.go",
	},
	{
		Path:    "wallets balances show",
		Aliases: []string{"sh"},
		Ops:     []string{"getBalance"},
		Source:  "cmd/wallets/balances/show.go",
	},
	{
		Path:     "wallets holds",
		Aliases:  []string{"h", "hold"},
		Grouping: true,
		Source:   "cmd/wallets/holds/root.go",
	},
	{
		Path:    "wallets holds list",
		Aliases: []string{"ls", "l"},
		Ops:     []string{"getHolds"},
		Source:  "cmd/wallets/holds/list.go",
	},
	{
		Path:    "wallets holds show",
		Aliases: []string{"sh"},
		Ops:     []string{"getHold"},
		Source:  "cmd/wallets/holds/show.go",
	},
	{
		Path:    "wallets holds confirm",
		Aliases: []string{"c", "conf"},
		Ops:     []string{"confirmHold"},
		Source:  "cmd/wallets/holds/confirm.go",
	},
	{
		Path:    "wallets holds void",
		Aliases: []string{"v"},
		Ops:     []string{"voidHold"},
		Source:  "cmd/wallets/holds/void.go",
	},
	{
		Path:     "wallets transactions",
		Aliases:  []string{"transaction", "tx", "txs"},
		Grouping: true,
		Source:   "cmd/wallets/transactions/root.go",
	},
	{
		Path:    "wallets transactions list",
		Aliases: []string{"ls", "l"},
		Ops:     []string{"getTransactions"},
		Source:  "cmd/wallets/transactions/list.go",
	},
}

// Leaves returns the executable baseline commands, preserving order.
func Leaves() []Command {
	var out []Command
	for _, c := range Baseline {
		if !c.Grouping {
			out = append(out, c)
		}
	}
	return out
}

// Groupings returns the grouping-only baseline commands, preserving order.
func Groupings() []Command {
	var out []Command
	for _, c := range Baseline {
		if c.Grouping {
			out = append(out, c)
		}
	}
	return out
}

// MappedLeaves returns the executable leaves that carry at least one operation.
func MappedLeaves() []Command {
	var out []Command
	for _, c := range Leaves() {
		if len(c.Ops) > 0 {
			out = append(out, c)
		}
	}
	return out
}

// ExcludedLeaves returns the executable leaves that deliberately carry no
// operation. Each one must record its Exclusion evidence.
func ExcludedLeaves() []Command {
	var out []Command
	for _, c := range Leaves() {
		if len(c.Ops) == 0 {
			out = append(out, c)
		}
	}
	return out
}

// BaselineTargets returns every distinct operationId the baseline reaches,
// sorted.
func BaselineTargets() []string {
	seen := map[string]struct{}{}
	var out []string
	for _, c := range Baseline {
		for _, op := range c.Ops {
			if _, dup := seen[op]; dup {
				continue
			}
			seen[op] = struct{}{}
			out = append(out, op)
		}
	}
	sort.Strings(out)
	return out
}
