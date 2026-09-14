package audit

import (
	"path/filepath"
	"sort"
	"testing"
)

// specPath is the authoritative contract, read from the audit package
// directory. Everything in this file is checked against the live document, not
// against the committed golden report, so a change to openapi.yaml fails here
// before it can be rationalised in prose.
func specPath() string { return filepath.Join("..", "..", "..", "openapi.yaml") }

func load(t *testing.T) *Document {
	t.Helper()
	doc, err := Load(specPath())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return doc
}

func build(t *testing.T) *Report {
	t.Helper()
	report, err := Build(specPath())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return report
}

func TestEveryOperationHasAUniqueIDAndTheSingleKnownTag(t *testing.T) {
	doc := load(t)

	if len(doc.Operations) == 0 {
		t.Fatal("document declares no operations")
	}
	seen := map[string]Operation{}
	for _, op := range doc.Operations {
		if prev, dup := seen[op.OperationID]; dup {
			t.Errorf("operationId %q declared twice: %s %s and %s %s",
				op.OperationID, prev.Method, prev.Path, op.Method, op.Path)
		}
		seen[op.OperationID] = op
		if op.Tag != TagV1 {
			t.Errorf("operation %s carries tag %q, want %q", op.OperationID, op.Tag, TagV1)
		}
		if op.OperationID == "" || op.Method == "" || op.Path == "" {
			t.Errorf("operation at %s %s is missing an identity field", op.Method, op.Path)
		}
	}
}

func TestOperationsAreSortedByOperationID(t *testing.T) {
	doc := load(t)

	ids := make([]string, 0, len(doc.Operations))
	for _, op := range doc.Operations {
		ids = append(ids, op.OperationID)
	}
	if !sort.StringsAreSorted(ids) {
		t.Errorf("operations are not sorted by operationId: %v", ids)
	}
}

func TestEveryOperationIsClassifiedIntoAKnownFamily(t *testing.T) {
	report := build(t)

	known := map[Family]struct{}{}
	for _, f := range Families {
		known[f] = struct{}{}
	}
	total := 0
	for _, f := range Families {
		total += len(report.ByFamily()[f])
	}
	if total != len(report.Operations) {
		t.Errorf("families cover %d operations, want %d", total, len(report.Operations))
	}
	for _, rec := range report.Operations {
		if _, ok := known[rec.Family]; !ok {
			t.Errorf("operation %s is in unknown family %q", rec.OperationID, rec.Family)
		}
	}
}

func TestFamilyMapHasNoEntryTheDocumentDoesNotDeclare(t *testing.T) {
	doc := load(t)

	present := Index(doc.Operations)
	for id := range familyOf {
		if _, ok := present[id]; !ok {
			t.Errorf("family map classifies %q, which the document does not declare", id)
		}
	}
}

func TestEveryOperationDeclaresExactlyOneKnownScope(t *testing.T) {
	doc := load(t)

	for _, op := range doc.Operations {
		if !op.HasSecurity {
			t.Errorf("operation %s declares no security block", op.OperationID)
			continue
		}
		if len(op.Scopes) != 1 {
			t.Errorf("operation %s declares scopes %v, want exactly one", op.OperationID, op.Scopes)
			continue
		}
		switch op.Scopes[0] {
		case "wallets:read", "wallets:write":
		default:
			t.Errorf("operation %s declares unknown scope %q", op.OperationID, op.Scopes[0])
		}
	}
}

// TestScopeDeclarationIsInternallyInconsistent pins divergence D2. It asserts
// the defect on purpose: when the document is fixed this test fails, which is
// the signal to retire D2 rather than to relax the test.
func TestScopeDeclarationIsInternallyInconsistent(t *testing.T) {
	report := build(t)

	if len(report.SchemeScopes) != 0 {
		t.Fatalf("securitySchemes now declare scopes %v; divergence D2 is fixed and must be retired",
			report.SchemeScopes)
	}
	undeclared := report.UndeclaredScopes()
	want := []string{"wallets:read", "wallets:write"}
	if len(undeclared) != len(want) {
		t.Fatalf("undeclared scopes = %v, want %v", undeclared, want)
	}
	for i := range want {
		if undeclared[i] != want[i] {
			t.Errorf("undeclared scopes = %v, want %v", undeclared, want)
			break
		}
	}
}

func TestNoOperationIsDeprecated(t *testing.T) {
	doc := load(t)

	for _, op := range doc.Operations {
		if op.Deprecated {
			t.Errorf("operation %s is deprecated; record the deprecation in the inventory", op.OperationID)
		}
	}
}

// TestDebitWalletIsTheOnlyMultiSuccessOperation pins the status union the
// command result schema must preserve: 201 carries a Hold and 204 is empty.
func TestDebitWalletIsTheOnlyMultiSuccessOperation(t *testing.T) {
	doc := load(t)

	for _, op := range doc.Operations {
		switch op.OperationID {
		case "debitWallet":
			codes := op.SuccessCodes()
			if len(codes) != 2 || codes[0] != "201" || codes[1] != "204" {
				t.Errorf("debitWallet success codes = %v, want [201 204]", codes)
			}
			for _, s := range op.Successes {
				if s.Code == "201" && s.Body != "DebitWalletResponse" {
					t.Errorf("debitWallet 201 body = %q, want DebitWalletResponse", s.Body)
				}
				if s.Code == "204" && s.Body != "" {
					t.Errorf("debitWallet 204 body = %q, want no body", s.Body)
				}
			}
		default:
			if len(op.Successes) != 1 {
				t.Errorf("operation %s declares %d success responses, want 1", op.OperationID, len(op.Successes))
			}
		}
	}
}

func TestEveryMutatingOperationAcceptsAnIdempotencyKeyAndNoReadDoes(t *testing.T) {
	doc := load(t)

	for _, op := range doc.Operations {
		if op.Mutating() && !op.Idempotent() {
			t.Errorf("mutating operation %s does not declare Idempotency-Key", op.OperationID)
		}
		if !op.Mutating() && op.Idempotent() {
			t.Errorf("read operation %s declares Idempotency-Key", op.OperationID)
		}
	}
}

func TestEveryListingDeclaresPagination(t *testing.T) {
	doc := load(t)

	paginated := map[string]bool{}
	for _, op := range doc.Operations {
		paginated[op.OperationID] = op.Paginated()
	}
	for _, id := range []string{"getHolds", "getTransactions", "listBalances", "listWallets"} {
		if !paginated[id] {
			t.Errorf("%s no longer declares cursor pagination", id)
		}
	}
}

// TestDuplicateIdempotencyKeyParameters pins divergence D4.
func TestDuplicateIdempotencyKeyParameters(t *testing.T) {
	doc := load(t)

	got := map[string][]string{}
	for _, op := range doc.Operations {
		if len(op.DuplicateParameters) > 0 {
			got[op.OperationID] = op.DuplicateParameters
		}
	}
	want := []string{"creditWallet", "debitWallet", "voidHold"}
	if len(got) != len(want) {
		t.Fatalf("operations with duplicate parameters = %v, want exactly %v", got, want)
	}
	for _, id := range want {
		dups, ok := got[id]
		if !ok {
			t.Errorf("%s no longer declares a duplicate parameter; divergence D4 may be partly fixed", id)
			continue
		}
		if len(dups) != 1 || dups[0] != "Idempotency-Key/header" {
			t.Errorf("%s duplicate parameters = %v, want [Idempotency-Key/header]", id, dups)
		}
	}
}

func TestBaselineIsWellFormed(t *testing.T) {
	paths := map[string]struct{}{}
	sources := map[string]struct{}{}
	for _, c := range Baseline {
		if _, dup := paths[c.Path]; dup {
			t.Errorf("baseline declares command path %q twice", c.Path)
		}
		paths[c.Path] = struct{}{}

		if c.Source == "" {
			t.Errorf("baseline command %q records no source file", c.Path)
		}
		if _, dup := sources[c.Source]; dup {
			t.Errorf("baseline reuses source file %q", c.Source)
		}
		sources[c.Source] = struct{}{}

		switch {
		case c.Grouping:
			if len(c.Ops) != 0 {
				t.Errorf("grouping command %q maps to operations %v", c.Path, c.Ops)
			}
			if c.Exclusion != "" {
				t.Errorf("grouping command %q records an exclusion reason", c.Path)
			}
		case len(c.Ops) == 0:
			if c.Exclusion == "" {
				t.Errorf("excluded leaf %q records no exclusion evidence", c.Path)
			}
		default:
			if c.Exclusion != "" {
				t.Errorf("mapped leaf %q records an exclusion reason", c.Path)
			}
		}
	}
}

func TestEveryBaselineLeafMapsToADistinctExistingOperation(t *testing.T) {
	report := build(t)

	if missing := report.UnknownBaselineTargets(); len(missing) > 0 {
		t.Errorf("baseline maps to operations the document does not declare: %v", missing)
	}

	owner := map[string]string{}
	for _, c := range MappedLeaves() {
		if len(c.Ops) != 1 {
			t.Errorf("leaf %q maps to %d operations, want 1", c.Path, len(c.Ops))
			continue
		}
		op := c.Ops[0]
		if prev, dup := owner[op]; dup {
			t.Errorf("operation %s is claimed by both %q and %q", op, prev, c.Path)
		}
		owner[op] = c.Path
	}
	if len(owner) != len(MappedLeaves()) {
		t.Errorf("%d leaves map to %d distinct operations", len(MappedLeaves()), len(owner))
	}
}

func TestGroupingsAndLeavesPartitionTheBaseline(t *testing.T) {
	if got := len(Groupings()) + len(Leaves()); got != len(Baseline) {
		t.Errorf("groupings + leaves = %d, want %d", got, len(Baseline))
	}
	if got := len(MappedLeaves()) + len(ExcludedLeaves()); got != len(Leaves()) {
		t.Errorf("mapped + excluded = %d, want %d leaves", got, len(Leaves()))
	}
}

func TestBlockersAndDivergencesAreWellFormedAndTargetRealOperations(t *testing.T) {
	report := build(t)

	if missing := report.UnknownBlockerTargets(); len(missing) > 0 {
		t.Errorf("blockers or divergences name operations the document does not declare: %v", missing)
	}

	ids := map[string]struct{}{}
	for _, b := range Blockers {
		if _, dup := ids[b.ID]; dup {
			t.Errorf("blocker id %q declared twice", b.ID)
		}
		ids[b.ID] = struct{}{}
		if b.Title == "" || b.Evidence == "" || b.Consequence == "" || b.FixIn == "" {
			t.Errorf("blocker %s is missing a required field", b.ID)
		}
		if !sort.StringsAreSorted(b.OperationIDs) {
			t.Errorf("blocker %s operationIds are not sorted: %v", b.ID, b.OperationIDs)
		}
	}
	for _, d := range Divergences {
		if _, dup := ids[d.ID]; dup {
			t.Errorf("divergence id %q collides with another record", d.ID)
		}
		ids[d.ID] = struct{}{}
		if d.Title == "" || d.Spec == "" || d.Server == "" {
			t.Errorf("divergence %s is missing a required field", d.ID)
		}
		if !sort.StringsAreSorted(d.OperationIDs) {
			t.Errorf("divergence %s operationIds are not sorted: %v", d.ID, d.OperationIDs)
		}
	}
}

func TestBlockedOperationsAreExactlyTheOnesBlockersName(t *testing.T) {
	report := build(t)

	named := map[string]struct{}{}
	for _, b := range Blockers {
		for _, op := range b.OperationIDs {
			named[op] = struct{}{}
		}
	}
	for _, rec := range report.Operations {
		_, want := named[rec.OperationID]
		got := len(rec.Blockers) > 0
		if got != want {
			t.Errorf("operation %s blocked=%v but blockers name it=%v", rec.OperationID, got, want)
		}
	}
	if report.Totals.Blocked != len(named) {
		t.Errorf("blocked total = %d, want %d", report.Totals.Blocked, len(named))
	}
}

// TestNoOperationCarriesASecretOrDestructiveRisk states the four hazard classes
// this surface does not have. It is an assertion, not a comment, so an
// operation that introduces one cannot land unrecorded.
func TestNoOperationCarriesASecretOrDestructiveRisk(t *testing.T) {
	report := build(t)

	for _, rec := range report.Operations {
		if rec.Risk.Secret {
			t.Errorf("operation %s is marked secret-bearing; record the handling rule", rec.OperationID)
		}
		if rec.Risk.DisplayOnce {
			t.Errorf("operation %s is marked display-once; record the handling rule", rec.OperationID)
		}
		if rec.Risk.Destructive {
			t.Errorf("operation %s is marked destructive; record the confirmation rule", rec.OperationID)
		}
		if rec.Risk.Streaming {
			t.Errorf("operation %s is marked streaming; record the transport rule", rec.OperationID)
		}
	}
}

func TestFundMovingOperationsAreMutatingAndIdempotent(t *testing.T) {
	report := build(t)

	for _, rec := range report.Operations {
		if !rec.Risk.Moves {
			continue
		}
		if !rec.Mutating() {
			t.Errorf("fund-moving operation %s is not mutating", rec.OperationID)
		}
		if !rec.Risk.Idempotent {
			t.Errorf("fund-moving operation %s does not accept an idempotency key", rec.OperationID)
		}
	}
}

func TestFundMovingSetOnlyNamesRealOperations(t *testing.T) {
	doc := load(t)

	present := Index(doc.Operations)
	for id := range fundMoving {
		if _, ok := present[id]; !ok {
			t.Errorf("fundMoving names %q, which the document does not declare", id)
		}
	}
}

func TestTotalsArePartitionsNotEstimates(t *testing.T) {
	report := build(t)
	t2 := report.Totals

	if t2.Operations != len(report.Operations) {
		t.Errorf("Operations total = %d, want %d", t2.Operations, len(report.Operations))
	}
	if t2.UniqueOperationIDs != t2.Operations {
		t.Errorf("unique operationIds = %d, want %d", t2.UniqueOperationIDs, t2.Operations)
	}
	if got := t2.WithBaseline + t2.WithoutBaseline; got != t2.Operations {
		t.Errorf("withBaseline + withoutBaseline = %d, want %d", got, t2.Operations)
	}
	if got := t2.Blocked + t2.Admissible; got != t2.Operations {
		t.Errorf("blocked + admissible = %d, want %d", got, t2.Operations)
	}
	if got := t2.ReadScoped + t2.WriteScoped; got != t2.Operations {
		t.Errorf("read + write scoped = %d, want %d", got, t2.Operations)
	}
	if got := t2.BaselineGroupings + t2.BaselineLeaves; got != t2.BaselineCommands {
		t.Errorf("groupings + leaves = %d, want %d", got, t2.BaselineCommands)
	}
	if got := t2.BaselineMapped + t2.BaselineExcluded; got != t2.BaselineLeaves {
		t.Errorf("mapped + excluded = %d, want %d", got, t2.BaselineLeaves)
	}
	if t2.OperationsWithSecurity != t2.Operations {
		t.Errorf("operations with security = %d, want all %d", t2.OperationsWithSecurity, t2.Operations)
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	first := build(t)
	second := build(t)

	if first.Markdown() != second.Markdown() {
		t.Error("two Build calls on the same document produced different markdown")
	}
}

func TestBuildRejectsAMissingDocument(t *testing.T) {
	if _, err := Build(filepath.Join("testdata", "does-not-exist.yaml")); err == nil {
		t.Error("Build accepted a missing document")
	}
}

func TestPinnedProvenanceIdentifiers(t *testing.T) {
	for name, rev := range map[string]string{
		"BaseRevision":     BaseRevision,
		"BaselineRevision": BaselineRevision,
	} {
		if len(rev) != 40 {
			t.Errorf("%s = %q, want a 40-character commit SHA", name, rev)
			continue
		}
		for _, c := range rev {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				t.Errorf("%s = %q contains a non-hex character %q", name, rev, c)
				break
			}
		}
	}
	if len(ClientSpecSHA256) != 64 {
		t.Fatalf("ClientSpecSHA256 = %q, want a 64-character SHA-256", ClientSpecSHA256)
	}
	for _, c := range ClientSpecSHA256 {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Fatalf("ClientSpecSHA256 = %q contains a non-hex character %q", ClientSpecSHA256, c)
		}
	}
}
