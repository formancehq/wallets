package audit

import "sort"

// Family is a functional grouping of Wallets operations. The groupings follow
// the resource the operation acts on, which is also how the legacy fctl command
// tree was shaped, so a future command catalogue can reuse them directly.
type Family string

const (
	// FamilyWallets covers the wallet resource itself and the two fund
	// movements that target it.
	FamilyWallets Family = "wallets"
	// FamilyBalances covers the named sub-balances of a wallet.
	FamilyBalances Family = "balances"
	// FamilyHolds covers pending debits and their resolution.
	FamilyHolds Family = "holds"
	// FamilyTransactions covers read-only transaction listing.
	FamilyTransactions Family = "transactions"
	// FamilyService covers service-level metadata, not wallet data.
	FamilyService Family = "service"
)

// Families is every family in report order.
var Families = []Family{
	FamilyWallets,
	FamilyBalances,
	FamilyHolds,
	FamilyTransactions,
	FamilyService,
}

// familyOf is the explicit operationId-to-family assignment. It is a total map
// on purpose: a new operation in the document fails the audit until it is
// classified here, rather than being silently grouped by a prefix heuristic.
var familyOf = map[string]Family{
	"createWallet":     FamilyWallets,
	"listWallets":      FamilyWallets,
	"getWallet":        FamilyWallets,
	"updateWallet":     FamilyWallets,
	"getWalletSummary": FamilyWallets,
	"creditWallet":     FamilyWallets,
	"debitWallet":      FamilyWallets,

	"createBalance": FamilyBalances,
	"listBalances":  FamilyBalances,
	"getBalance":    FamilyBalances,

	"getHolds":    FamilyHolds,
	"getHold":     FamilyHolds,
	"confirmHold": FamilyHolds,
	"voidHold":    FamilyHolds,

	"getTransactions": FamilyTransactions,

	"getServerInfo": FamilyService,
}

// FamilyOf returns the family assigned to operationID.
func FamilyOf(operationID string) (Family, bool) {
	f, ok := familyOf[operationID]
	return f, ok
}

// Risk records the operator-facing hazards of one operation. Every field is
// derived from the pinned document or from the pinned product source; none is
// a judgement call left to the reader.
type Risk struct {
	// Secret is true when a success response carries a credential or other
	// secret material. No Wallets operation does: the only non-wallet response
	// body is ServerInfo, whose single field is a version string.
	Secret bool `json:"secret"`
	// DisplayOnce is true when a value is returned exactly once and cannot be
	// re-read. No Wallets operation does.
	DisplayOnce bool `json:"displayOnce"`
	// Moves is true when the operation moves funds. These are the operations a
	// host must never replay speculatively.
	Moves bool `json:"moves"`
	// Destructive is true when the operation can remove or overwrite state a
	// caller did not supply. No Wallets operation does: the only update is
	// updateWallet, and pkg.Manager.UpdateWallet merges the supplied metadata
	// into the existing metadata rather than replacing it.
	Destructive bool `json:"destructive"`
	// Idempotent mirrors Operation.Idempotent: the operation accepts an
	// Idempotency-Key header and the handler passes it through to the manager.
	Idempotent bool `json:"idempotent"`
	// Paginated mirrors Operation.Paginated: cursor pagination is declared.
	Paginated bool `json:"paginated"`
	// Streaming is true when the operation returns an unbounded stream. No
	// Wallets operation does; every listing is a bounded cursor page.
	Streaming bool `json:"streaming"`
	// MultiSuccess is true when the operation declares more than one 2xx
	// response, so its result shape depends on the request.
	MultiSuccess bool `json:"multiSuccess"`
}

// fundMoving is the set of operations that move funds, read from pkg/api and
// pkg/manager.go at the pinned product revision:
//
//	creditWallet  -> Manager.Credit, posts to the ledger
//	debitWallet   -> Manager.Debit, posts to the ledger or creates a hold
//	confirmHold   -> Manager.ConfirmHold, settles a pending debit
//	voidHold      -> Manager.VoidHold, cancels a pending debit
//
// createBalance and updateWallet write state but move no funds.
var fundMoving = map[string]struct{}{
	"creditWallet": {},
	"debitWallet":  {},
	"confirmHold":  {},
	"voidHold":     {},
}

// RiskOf derives the risk record of op.
func RiskOf(op Operation) Risk {
	_, moves := fundMoving[op.OperationID]
	return Risk{
		Moves:        moves,
		Idempotent:   op.Idempotent(),
		Paginated:    op.Paginated(),
		MultiSuccess: len(op.Successes) > 1,
	}
}

// ScopeKinds returns the distinct scope names the document declares across all
// operations, sorted.
func ScopeKinds(ops []Operation) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, op := range ops {
		for _, s := range op.Scopes {
			if _, dup := seen[s]; dup {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
