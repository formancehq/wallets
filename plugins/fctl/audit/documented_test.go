package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This file pins the numbers and claims that docs/command-inventory.md states
// in prose. Prose cannot be regenerated, so every quantity it quotes is
// asserted here against the live document. A contract change therefore fails a
// test with the exact number that moved, instead of leaving a stale sentence.
//
// When one of these fails, fix the document and this file together. Do not
// relax the assertion.

func TestDocumentedTotals(t *testing.T) {
	got := build(t).Totals

	want := Totals{
		Operations:             16,
		UniqueOperationIDs:     16,
		DeprecatedOperations:   0,
		OperationsWithSecurity: 16,
		ReadScoped:             9,
		WriteScoped:            7,

		BaselineCommands:  18,
		BaselineGroupings: 4,
		BaselineLeaves:    14,
		BaselineMapped:    14,
		BaselineExcluded:  0,
		BaselineTargets:   14,

		WithBaseline:    14,
		WithoutBaseline: 2,
		Blocked:         3,
		Admissible:      13,
		ModuleBlockers:  1,
		Divergences:     8,

		Mutating:     7,
		Idempotent:   7,
		Paginated:    4,
		FundMoving:   4,
		MultiSuccess: 1,

		SecretBearing: 0,
		DisplayOnce:   0,
		Destructive:   0,
		Streaming:     0,
	}

	if got != want {
		t.Errorf("totals drifted from the inventory prose:\n got %+v\nwant %+v", got, want)
	}
}

// TestDocumentedOperationsWithoutLegacyPrecedent pins §3 of the inventory: the
// two operations the legacy CLI never exposed.
func TestDocumentedOperationsWithoutLegacyPrecedent(t *testing.T) {
	report := build(t)

	var got []string
	for _, rec := range report.Operations {
		if len(rec.BaselineCommands) == 0 {
			got = append(got, rec.OperationID)
		}
	}
	want := []string{"getServerInfo", "getWalletSummary"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("operations without legacy precedent = %v, want %v", got, want)
	}
}

// TestDocumentedFamilySizes pins the functional groupings §2 quotes.
func TestDocumentedFamilySizes(t *testing.T) {
	byFamily := build(t).ByFamily()

	want := map[Family]int{
		FamilyWallets:      7,
		FamilyBalances:     3,
		FamilyHolds:        4,
		FamilyTransactions: 1,
		FamilyService:      1,
	}
	for family, size := range want {
		if got := len(byFamily[family]); got != size {
			t.Errorf("family %s has %d operations, want %d", family, got, size)
		}
	}
	if len(byFamily) != len(want) {
		t.Errorf("report has %d families, want %d", len(byFamily), len(want))
	}
}

// TestDocumentedBlockerIdentity pins §5: the blocker set and which operations
// each one applies to.
func TestDocumentedBlockerIdentity(t *testing.T) {
	want := map[string][]string{
		"B1": {"confirmHold", "voidHold"},
		"B2": {"debitWallet"},
		"B3": {},
	}

	if len(Blockers) != len(want) {
		t.Fatalf("%d blockers recorded, want %d", len(Blockers), len(want))
	}
	for _, b := range Blockers {
		ops, ok := want[b.ID]
		if !ok {
			t.Errorf("unexpected blocker %s", b.ID)
			continue
		}
		if strings.Join(b.OperationIDs, ",") != strings.Join(ops, ",") {
			t.Errorf("blocker %s applies to %v, want %v", b.ID, b.OperationIDs, ops)
		}
	}
}

// TestDocumentedDivergenceIdentity pins §6.
func TestDocumentedDivergenceIdentity(t *testing.T) {
	want := []string{"D1", "D2", "D3", "D4", "D5", "D6", "D7", "D8"}

	if len(Divergences) != len(want) {
		t.Fatalf("%d divergences recorded, want %d", len(Divergences), len(want))
	}
	for i, d := range Divergences {
		if d.ID != want[i] {
			t.Errorf("divergence %d is %s, want %s", i, d.ID, want[i])
		}
	}
}

// TestDocumentedSpecFacts pins the document-level facts behind D3 and D7.
func TestDocumentedSpecFacts(t *testing.T) {
	report := build(t)

	if report.SpecVersion != "0.1.0" {
		t.Errorf("info.version = %q, want 0.1.0; divergence D3 quotes this value", report.SpecVersion)
	}
	if len(report.Servers) != 1 || report.Servers[0] != "http://localhost:8080/" {
		t.Errorf("servers = %v, want exactly [http://localhost:8080/]; divergence D7 quotes this", report.Servers)
	}
	if report.SpecDocument != "openapi.yaml" {
		t.Errorf("spec document = %q, want openapi.yaml", report.SpecDocument)
	}
}

// TestDocumentedBaselineAliases pins the alias set the inventory promises to
// preserve. Parity is not just the 14 canonical paths: every accepted alias at
// the pinned fctl revision has to keep working.
func TestDocumentedBaselineAliases(t *testing.T) {
	want := map[string][]string{
		"wallets":                   {"wal", "wa", "wallet"},
		"wallets create":            {"cr"},
		"wallets list":              {"ls", "l"},
		"wallets show":              {"sh"},
		"wallets update":            {"up"},
		"wallets credit":            {"cr"},
		"wallets debit":             {"deb"},
		"wallets balances":          {"balance", "bls", "bal"},
		"wallets balances create":   {"c", "cr"},
		"wallets balances list":     {"ls", "l"},
		"wallets balances show":     {"sh"},
		"wallets holds":             {"h", "hold"},
		"wallets holds list":        {"ls", "l"},
		"wallets holds show":        {"sh"},
		"wallets holds confirm":     {"c", "conf"},
		"wallets holds void":        {"v"},
		"wallets transactions":      {"transaction", "tx", "txs"},
		"wallets transactions list": {"ls", "l"},
	}

	if len(Baseline) != len(want) {
		t.Fatalf("baseline has %d commands, want %d", len(Baseline), len(want))
	}
	for _, c := range Baseline {
		aliases, ok := want[c.Path]
		if !ok {
			t.Errorf("unexpected baseline command %q", c.Path)
			continue
		}
		if strings.Join(c.Aliases, ",") != strings.Join(aliases, ",") {
			t.Errorf("command %q aliases = %v, want %v", c.Path, c.Aliases, aliases)
		}
	}
}

// TestCommittedReportMatchesTheDocument gates the golden artefact the inventory
// links to. It is the same comparison `just fctl-audit-check` runs, kept here so
// `go test ./...` alone catches a stale commit.
func TestCommittedReportMatchesTheDocument(t *testing.T) {
	report := build(t)

	want, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("encode report: %v", err)
	}
	want = append(want, '\n')

	got, err := os.ReadFile(filepath.Join("testdata", "report.json"))
	if err != nil {
		t.Fatalf("read committed report: %v", err)
	}
	if string(got) != string(want) {
		t.Error("audit/testdata/report.json is out of date; run `just fctl-audit`")
	}
}

// TestCommittedMarkdownMatchesTheDocument gates the generated tables.
func TestCommittedMarkdownMatchesTheDocument(t *testing.T) {
	want := build(t).Markdown()

	got, err := os.ReadFile(filepath.Join("..", "docs", "operations.generated.md"))
	if err != nil {
		t.Fatalf("read committed markdown: %v", err)
	}
	if string(got) != want {
		t.Error("docs/operations.generated.md is out of date; run `just fctl-audit`")
	}
}

// TestInventoryDocumentReferencesEveryBlockerAndDivergence keeps the prose and
// the recorded records from drifting apart: a blocker nobody documents is a
// blocker nobody acts on.
func TestInventoryDocumentReferencesEveryBlockerAndDivergence(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "docs", "command-inventory.md"))
	if err != nil {
		t.Fatalf("read inventory document: %v", err)
	}
	text := string(raw)

	for _, b := range Blockers {
		if !strings.Contains(text, b.ID) {
			t.Errorf("command-inventory.md does not mention blocker %s", b.ID)
		}
	}
	for _, d := range Divergences {
		if !strings.Contains(text, d.ID) {
			t.Errorf("command-inventory.md does not mention divergence %s", d.ID)
		}
	}
}
