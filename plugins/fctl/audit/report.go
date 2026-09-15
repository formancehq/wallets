package audit

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Record is one operation with everything this audit can prove about it.
type Record struct {
	Operation
	Family Family `json:"family"`
	Risk   Risk   `json:"risk"`
	// BaselineCommands are the legacy fctl command paths that map onto this
	// operation, sorted. Empty means the operation has no legacy precedent.
	BaselineCommands []string `json:"baselineCommands"`
	// Blockers are the operation-scoped blocker IDs recorded against this
	// operation, sorted. Module-level blockers are not listed here.
	Blockers []string `json:"blockers"`
}

// Totals are the counts the inventory document quotes. Every one of them is
// derived, never transcribed.
type Totals struct {
	// Operations is every operation the document declares.
	Operations int `json:"operations"`
	// UniqueOperationIDs must equal Operations.
	UniqueOperationIDs int `json:"uniqueOperationIds"`
	// DeprecatedOperations is the count marked deprecated.
	DeprecatedOperations int `json:"deprecatedOperations"`
	// OperationsWithSecurity is the count declaring any security block.
	OperationsWithSecurity int `json:"operationsWithSecurity"`
	// ReadScoped and WriteScoped partition the operations by declared scope.
	ReadScoped  int `json:"readScoped"`
	WriteScoped int `json:"writeScoped"`

	// BaselineCommands is every legacy fctl constructor under cmd/wallets/.
	BaselineCommands int `json:"baselineCommands"`
	// BaselineGroupings and BaselineLeaves partition BaselineCommands.
	BaselineGroupings int `json:"baselineGroupings"`
	BaselineLeaves    int `json:"baselineLeaves"`
	// BaselineMapped and BaselineExcluded partition BaselineLeaves.
	BaselineMapped   int `json:"baselineMapped"`
	BaselineExcluded int `json:"baselineExcluded"`
	// BaselineTargets is the number of distinct operations the baseline reaches.
	BaselineTargets int `json:"baselineTargets"`

	// WithBaseline and WithoutBaseline partition Operations by whether any
	// legacy command reaches them.
	WithBaseline    int `json:"withBaseline"`
	WithoutBaseline int `json:"withoutBaseline"`
	// Blocked is the number of operations carrying an operation-scoped blocker.
	Blocked int `json:"blocked"`
	// Admissible is Operations minus Blocked: proven facts, no operation-scoped
	// blocker. It is not an acceptance claim; the runtime gates are separate,
	// and the module-level blockers in ModuleBlockers still apply to all of it.
	Admissible int `json:"admissible"`
	// ModuleBlockers is the number of blockers that apply to the whole surface.
	ModuleBlockers int `json:"moduleBlockers"`
	// Divergences is the number of recorded spec-versus-server or
	// spec-internal disagreements.
	Divergences int `json:"divergences"`

	// Mutating, Idempotent, Paginated, FundMoving and MultiSuccess are risk
	// counts over Operations.
	Mutating     int `json:"mutating"`
	Idempotent   int `json:"idempotent"`
	Paginated    int `json:"paginated"`
	FundMoving   int `json:"fundMoving"`
	MultiSuccess int `json:"multiSuccess"`
	// SecretBearing, DisplayOnce, Destructive and Streaming are the hazard
	// classes this surface does not have. They are counted rather than
	// asserted absent, so a future operation that introduces one fails the
	// documented-invariant test.
	SecretBearing int `json:"secretBearing"`
	DisplayOnce   int `json:"displayOnce"`
	Destructive   int `json:"destructive"`
	Streaming     int `json:"streaming"`
}

// Report is the whole deterministic inventory.
type Report struct {
	// SpecDocument is the base name of the document the report was built from.
	// Only the base name is recorded so the report is identical whether it is
	// produced from the module root or from a package directory.
	SpecDocument string `json:"specDocument"`
	// BaseRevision pins the upstream Wallets tree this work started from.
	BaseRevision string `json:"baseRevision"`
	// SpecSHA256 identifies the exact OpenAPI bytes read for this report.
	SpecSHA256 string `json:"specSHA256"`
	// ClientSpecSHA256 identifies the exact OpenAPI bytes used to regenerate
	// pkg/client. Equality with SpecSHA256 is enforced by Build.
	ClientSpecSHA256 string `json:"clientSpecSHA256"`
	// BaselineRevision pins the legacy fctl tree the baseline came from.
	BaselineRevision string `json:"baselineRevision"`
	// SpecVersion, Servers and SchemeScopes are document-level facts kept
	// because each one is the evidence behind a recorded divergence.
	SpecVersion  string       `json:"specVersion"`
	Servers      []string     `json:"servers"`
	SchemeScopes []string     `json:"schemeScopes"`
	ScopeKinds   []string     `json:"scopeKinds"`
	Totals       Totals       `json:"totals"`
	Operations   []Record     `json:"operations"`
	Baseline     []Command    `json:"baseline"`
	Blockers     []Blocker    `json:"blockers"`
	Divergences  []Divergence `json:"divergences"`
}

// BaseRevision pins the upstream Wallets tree this inventory was based on:
// github.com/formancehq/wallets origin/main, which is v2.2.0-6-ga48a7d0.
const BaseRevision = "a48a7d0590b8b1cfdbfbd22cba0e475713550a75"

// ClientSpecSHA256 pins the OpenAPI bytes used by `just generate-client`.
const ClientSpecSHA256 = "715d87d7ea85344183afd1a4e16bdb67b161c1fff5665aaba74200521682a8fd"

// Build reads the document at specPath and produces the full report.
func Build(specPath string) (*Report, error) {
	specBytes, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}
	specSHA256 := fmt.Sprintf("%x", sha256.Sum256(specBytes))
	if specSHA256 != ClientSpecSHA256 {
		return nil, fmt.Errorf("OpenAPI SHA-256 %s differs from generated-client source %s; regenerate pkg/client and update ClientSpecSHA256", specSHA256, ClientSpecSHA256)
	}

	doc, err := Load(specPath)
	if err != nil {
		return nil, err
	}

	byID := Index(doc.Operations)
	if len(byID) != len(doc.Operations) {
		return nil, fmt.Errorf("document declares %d operations but only %d unique operationIds", len(doc.Operations), len(byID))
	}

	commandsByOp := map[string][]string{}
	for _, c := range Baseline {
		for _, op := range c.Ops {
			commandsByOp[op] = append(commandsByOp[op], c.Path)
		}
	}
	blockersByOp := blockersByOperation()

	report := &Report{
		SpecDocument:     filepath.Base(specPath),
		BaseRevision:     BaseRevision,
		SpecSHA256:       specSHA256,
		ClientSpecSHA256: ClientSpecSHA256,
		BaselineRevision: BaselineRevision,
		SpecVersion:      doc.SpecVersion,
		Servers:          doc.Servers,
		SchemeScopes:     doc.SchemeScopes,
		ScopeKinds:       ScopeKinds(doc.Operations),
		Baseline:         Baseline,
		Blockers:         Blockers,
		Divergences:      Divergences,
	}
	if report.SchemeScopes == nil {
		report.SchemeScopes = []string{}
	}

	for _, op := range doc.Operations {
		if op.Tag != TagV1 {
			return nil, fmt.Errorf("operation %s carries unknown tag %q", op.OperationID, op.Tag)
		}
		family, ok := FamilyOf(op.OperationID)
		if !ok {
			return nil, fmt.Errorf("operation %s is not classified into a family", op.OperationID)
		}
		commands := append([]string(nil), commandsByOp[op.OperationID]...)
		sort.Strings(commands)
		blocks := append([]string(nil), blockersByOp[op.OperationID]...)
		sort.Strings(blocks)
		report.Operations = append(report.Operations, Record{
			Operation:        op,
			Family:           family,
			Risk:             RiskOf(op),
			BaselineCommands: commands,
			Blockers:         blocks,
		})
	}

	t := Totals{
		Operations:         len(doc.Operations),
		UniqueOperationIDs: len(byID),
		BaselineCommands:   len(Baseline),
		BaselineGroupings:  len(Groupings()),
		BaselineLeaves:     len(Leaves()),
		BaselineMapped:     len(MappedLeaves()),
		BaselineExcluded:   len(ExcludedLeaves()),
		BaselineTargets:    len(BaselineTargets()),
		ModuleBlockers:     len(ModuleBlockers()),
		Divergences:        len(Divergences),
	}
	for _, r := range report.Operations {
		if r.Deprecated {
			t.DeprecatedOperations++
		}
		if r.HasSecurity {
			t.OperationsWithSecurity++
		}
		for _, s := range r.Scopes {
			switch s {
			case "wallets:read":
				t.ReadScoped++
			case "wallets:write":
				t.WriteScoped++
			}
		}
		if len(r.BaselineCommands) > 0 {
			t.WithBaseline++
		} else {
			t.WithoutBaseline++
		}
		if len(r.Blockers) > 0 {
			t.Blocked++
		}
		if r.Mutating() {
			t.Mutating++
		}
		if r.Risk.Idempotent {
			t.Idempotent++
		}
		if r.Risk.Paginated {
			t.Paginated++
		}
		if r.Risk.Moves {
			t.FundMoving++
		}
		if r.Risk.MultiSuccess {
			t.MultiSuccess++
		}
		if r.Risk.Secret {
			t.SecretBearing++
		}
		if r.Risk.DisplayOnce {
			t.DisplayOnce++
		}
		if r.Risk.Destructive {
			t.Destructive++
		}
		if r.Risk.Streaming {
			t.Streaming++
		}
	}
	t.Admissible = t.Operations - t.Blocked
	report.Totals = t

	return report, nil
}

// UnknownBaselineTargets returns baseline Ops entries that do not exist in the
// document. A non-empty result means the mapping references an operation the
// current source does not have.
func (r *Report) UnknownBaselineTargets() []string {
	present := map[string]struct{}{}
	for _, rec := range r.Operations {
		present[rec.OperationID] = struct{}{}
	}
	var missing []string
	for _, op := range BaselineTargets() {
		if _, ok := present[op]; !ok {
			missing = append(missing, op)
		}
	}
	sort.Strings(missing)
	return missing
}

// UnknownBlockerTargets returns blocker OperationIDs entries that do not exist
// in the document.
func (r *Report) UnknownBlockerTargets() []string {
	present := map[string]struct{}{}
	for _, rec := range r.Operations {
		present[rec.OperationID] = struct{}{}
	}
	seen := map[string]struct{}{}
	var missing []string
	for _, list := range [][]string{blockerOperations(), divergenceOperations()} {
		for _, op := range list {
			if _, ok := present[op]; ok {
				continue
			}
			if _, dup := seen[op]; dup {
				continue
			}
			seen[op] = struct{}{}
			missing = append(missing, op)
		}
	}
	sort.Strings(missing)
	return missing
}

func blockerOperations() []string {
	var out []string
	for _, b := range Blockers {
		out = append(out, b.OperationIDs...)
	}
	return out
}

func divergenceOperations() []string {
	var out []string
	for _, d := range Divergences {
		out = append(out, d.OperationIDs...)
	}
	return out
}

// ByFamily groups the records by family, preserving operationId order.
func (r *Report) ByFamily() map[Family][]Record {
	out := map[Family][]Record{}
	for _, rec := range r.Operations {
		out[rec.Family] = append(out[rec.Family], rec)
	}
	return out
}

// UndeclaredScopes returns the scope names operations reference that the
// security schemes do not declare, sorted. It is the derived evidence behind
// divergence D2.
func (r *Report) UndeclaredScopes() []string {
	declared := map[string]struct{}{}
	for _, s := range r.SchemeScopes {
		declared[s] = struct{}{}
	}
	var out []string
	for _, s := range r.ScopeKinds {
		if _, ok := declared[s]; !ok {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
