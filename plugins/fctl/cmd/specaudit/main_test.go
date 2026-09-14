package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// specPath is the authoritative contract, read from the command directory.
func specPath() string { return filepath.Join("..", "..", "..", "..", "openapi.yaml") }

// moduleRoot is the committed plugin module the check mode reads from.
func moduleRoot() string { return filepath.Join("..", "..") }

// TestCheckModePassesAgainstTheCommittedArtefacts is the same gate
// `just fctl-audit-check` runs.
func TestCheckModePassesAgainstTheCommittedArtefacts(t *testing.T) {
	if err := run(specPath(), moduleRoot(), true); err != nil {
		t.Errorf("check mode failed against the committed artefacts: %v", err)
	}
}

// TestWriteModeIsIdempotent proves regeneration is a no-op on a clean tree: it
// writes into a scratch directory, then checks the same content back.
func TestWriteModeIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{filepath.Join("audit", "testdata"), "docs"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("prepare scratch tree: %v", err)
		}
	}

	if err := run(specPath(), dir, false); err != nil {
		t.Fatalf("write mode: %v", err)
	}
	if err := run(specPath(), dir, true); err != nil {
		t.Errorf("check mode failed immediately after write mode: %v", err)
	}
}

// TestWriteModeReproducesTheCommittedArtefacts proves the committed files are
// exactly what a fresh run produces, byte for byte.
func TestWriteModeReproducesTheCommittedArtefacts(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{filepath.Join("audit", "testdata"), "docs"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("prepare scratch tree: %v", err)
		}
	}
	if err := run(specPath(), dir, false); err != nil {
		t.Fatalf("write mode: %v", err)
	}

	for _, rel := range []string{
		filepath.Join("audit", "testdata", "report.json"),
		filepath.Join("docs", "operations.generated.md"),
	} {
		fresh, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatalf("read fresh %s: %v", rel, err)
		}
		committed, err := os.ReadFile(filepath.Join(moduleRoot(), rel))
		if err != nil {
			t.Fatalf("read committed %s: %v", rel, err)
		}
		if string(fresh) != string(committed) {
			t.Errorf("%s differs from a fresh run; run `just fctl-audit`", rel)
		}
	}
}

// TestCheckModeFailsOnAStaleArtefact proves the gate actually gates.
func TestCheckModeFailsOnAStaleArtefact(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{filepath.Join("audit", "testdata"), "docs"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("prepare scratch tree: %v", err)
		}
	}
	if err := run(specPath(), dir, false); err != nil {
		t.Fatalf("write mode: %v", err)
	}

	stale := filepath.Join(dir, "docs", "operations.generated.md")
	if err := os.WriteFile(stale, []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("corrupt artefact: %v", err)
	}
	if err := run(specPath(), dir, true); err == nil {
		t.Error("check mode accepted a stale artefact")
	}
}

// TestRunRejectsAMissingDocument keeps the failure path honest.
func TestRunRejectsAMissingDocument(t *testing.T) {
	if err := run(filepath.Join("testdata", "does-not-exist.yaml"), t.TempDir(), false); err == nil {
		t.Error("run accepted a missing document")
	}
}

func TestExecuteCheckModeReturnsSuccessWithoutDiagnostics(t *testing.T) {
	var stderr bytes.Buffer
	code := execute([]string{"-spec", specPath(), "-out", moduleRoot(), "-check"}, &stderr)
	if code != 0 {
		t.Fatalf("execute check exit = %d, stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("execute check stderr = %q, want empty", stderr.String())
	}
}

func TestExecuteWriteModeReportsWrittenArtefactsAndTotals(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{filepath.Join("audit", "testdata"), "docs"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("prepare scratch tree: %v", err)
		}
	}

	var stderr bytes.Buffer
	code := execute([]string{"-spec", specPath(), "-out", dir}, &stderr)
	if code != 0 {
		t.Fatalf("execute write exit = %d, stderr = %q", code, stderr.String())
	}
	for _, want := range []string{
		"wrote " + filepath.Join(dir, "audit", "testdata", "report.json"),
		"wrote " + filepath.Join(dir, "docs", "operations.generated.md"),
		"operations=16 leaves=14 mapped=14 excluded=0 targets=14 blocked=0 moduleBlockers=0 divergences=7",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("execute write stderr = %q, want substring %q", stderr.String(), want)
		}
	}
}

func TestExecuteReturnsFailureAndDiagnosticForMissingSpec(t *testing.T) {
	var stderr bytes.Buffer
	code := execute([]string{"-spec", filepath.Join(t.TempDir(), "missing.yaml"), "-out", t.TempDir()}, &stderr)
	if code != 1 {
		t.Fatalf("execute missing spec exit = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "specaudit:") || !strings.Contains(stderr.String(), "missing.yaml") {
		t.Fatalf("execute missing spec stderr = %q", stderr.String())
	}
}

func TestExecuteReturnsUsageFailureForInvalidFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := execute([]string{"-unknown"}, &stderr)
	if code != 2 {
		t.Fatalf("execute invalid flag exit = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("execute invalid flag stderr = %q", stderr.String())
	}
}

func TestRunReportsReadAndWriteFailures(t *testing.T) {
	t.Run("check missing artefact", func(t *testing.T) {
		dir := t.TempDir()
		err := run(specPath(), dir, true)
		if err == nil || !strings.Contains(err.Error(), "read ") {
			t.Fatalf("run check error = %v, want read failure", err)
		}
	})

	t.Run("write missing parent", func(t *testing.T) {
		err := run(specPath(), filepath.Join(t.TempDir(), "missing-parent"), false)
		if err == nil || !strings.Contains(err.Error(), "write ") {
			t.Fatalf("run write error = %v, want write failure", err)
		}
	})
}
