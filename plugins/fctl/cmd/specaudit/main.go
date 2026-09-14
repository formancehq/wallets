// Command specaudit regenerates the deterministic Wallets operation inventory
// from this repository's OpenAPI document.
//
// It writes two artefacts and nothing else:
//
//	audit/testdata/report.json        the golden report the audit tests pin
//	docs/operations.generated.md      the generated tables the inventory links
//
// Both are committed. Regenerate with `just fctl-audit` from the repository
// root after any change to openapi.yaml or to the pinned legacy baseline, and
// review the diff: a change here is a change to the plugin's source of truth.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/formancehq/wallets/plugins/fctl/audit"
)

func main() {
	os.Exit(execute(os.Args[1:], os.Stderr))
}

func execute(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("specaudit", flag.ContinueOnError)
	flags.SetOutput(stderr)
	spec := flags.String("spec", filepath.Join("..", "..", "openapi.yaml"), "path to the Wallets OpenAPI document")
	out := flags.String("out", ".", "plugin module root to write artefacts under")
	check := flags.Bool("check", false, "verify the committed artefacts are up to date instead of writing them")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	if err := runWithWriter(*spec, *out, *check, stderr); err != nil {
		fmt.Fprintln(stderr, "specaudit:", err)
		return 1
	}
	return 0
}

func run(spec, out string, check bool) error {
	return runWithWriter(spec, out, check, os.Stderr)
}

func runWithWriter(spec, out string, check bool, stderr io.Writer) error {
	report, err := audit.Build(spec)
	if err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	encoded = append(encoded, '\n')

	artefacts := map[string][]byte{
		filepath.Join(out, "audit", "testdata", "report.json"): encoded,
		filepath.Join(out, "docs", "operations.generated.md"):  []byte(report.Markdown()),
	}

	for path, want := range artefacts {
		if check {
			got, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read %s: %w", path, err)
			}
			if string(got) != string(want) {
				return fmt.Errorf("%s is out of date; run `just fctl-audit`", path)
			}
			continue
		}
		if err := os.WriteFile(path, want, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		fmt.Fprintln(stderr, "wrote", path)
	}

	if !check {
		t := report.Totals
		fmt.Fprintf(stderr,
			"operations=%d leaves=%d mapped=%d excluded=%d targets=%d blocked=%d moduleBlockers=%d divergences=%d\n",
			t.Operations, t.BaselineLeaves, t.BaselineMapped, t.BaselineExcluded,
			t.BaselineTargets, t.Blocked, t.ModuleBlockers, t.Divergences)
	}
	return nil
}
