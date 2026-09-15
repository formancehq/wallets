package component

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const fctlSDKRevision = "e9b1395f46f3100b381dbe00f5213de28e6df0e1"

// TestComponentToolchainIsPinnedAndIsolatedFromTheDefaultShell binds the three
// properties the component authoring toolchain must keep at once.
//
// It is pinned: flake.nix and nix/fctl-component-tools.nix still state the
// exact Rust toolchain, source revisions, and content hashes frozen by this
// fctl SDK revision.
//
// It is reachable: the flake exposes each tool as its own package output, and
// the repository-root `just fctl-component-build` recipe enters exactly those
// package outputs, so a component build needs no developer-only shell.
//
// It is isolated: the default development shell carries none of them. These
// are Rust builds from source; putting them in the shell every Go job enters
// makes an unrelated crates.io rate limit fail lint, tidy and test CI.
func TestComponentToolchainIsPinnedAndIsolatedFromTheDefaultShell(t *testing.T) {
	flake, err := os.ReadFile("../../../flake.nix")
	if err != nil {
		t.Fatal(err)
	}
	toolDefinitions, err := os.ReadFile("../../../nix/fctl-component-tools.nix")
	if err != nil {
		t.Fatal(err)
	}
	contents := string(flake) + string(toolDefinitions)
	for _, required := range []string{
		"fctlSDKRevision = \"" + fctlSDKRevision + "\"",
		"github:oxalica/rust-overlay",
		"rust-overlay.overlays.default",
		"./nix/fctl-component-tools.nix",
		"inherit (componentTools) componentize-go wasi-virt wasm-tools;",
		"wasm-opt = pkgs.binaryen;",
		"rust-bin.stable.\"1.91.1\".minimal",
		"wasmToolsVersion = \"1.239.0\"",
		"sha256-9wJSNC/clO8M7E840i1lRWJQT7AdRQX468swmZ4O1rg=",
		"sha256-xIfTYJMVP47timzquEYEb9M8BHsj83NjgD44lbzgd+Y=",
		"version = \"0.4.1\"",
		"version = \"0.2.0-448f6df8\"",
	} {
		if !strings.Contains(contents, required) {
			t.Errorf("Wallets flake does not pin or expose required component input %q", required)
		}
	}

	devShellsStart := strings.Index(string(flake), "devShells =")
	if devShellsStart < 0 {
		t.Fatal("flake.nix declares no devShells output")
	}
	devShells := string(flake)[devShellsStart:]
	for _, forbidden := range []string{
		"componentTools",
		"binaryen",
	} {
		if strings.Contains(devShells, forbidden) {
			t.Errorf("the default development shell carries %q; the Rust component toolchain must stay out of the shell every Go CI job enters", forbidden)
		}
	}

	rootJustfile, err := os.ReadFile("../../../Justfile")
	if err != nil {
		t.Fatal(err)
	}
	componentBuild := recipeBody(string(rootJustfile), "fctl-component-build")
	if componentBuild == "" {
		t.Fatal("the repository root Justfile has no fctl-component-build recipe, so the component build is unreachable from the root")
	}
	for _, required := range []string{
		"nix shell",
		".#componentize-go",
		".#wasi-virt",
		".#wasm-tools",
		".#wasm-opt",
		"just build-component",
	} {
		if !strings.Contains(componentBuild, required) {
			t.Errorf("fctl-component-build does not enter its isolated toolchain: missing %q in:\n\t%s", required, strings.TrimSpace(componentBuild))
		}
	}
}

// recipeBody returns the indented body lines of a Just recipe, or "" when the
// recipe is absent. Comments above the target are not part of the body.
func recipeBody(justfile, target string) string {
	lines := strings.Split(justfile, "\n")
	body := make([]string, 0, 4)
	inRecipe := false
	for _, line := range lines {
		if strings.HasPrefix(line, target+":") {
			inRecipe = true
			continue
		}
		if !inRecipe {
			continue
		}
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break
		}
		body = append(body, line)
	}
	if !inRecipe {
		return ""
	}
	return strings.Join(body, "\n")
}

// TestSDKPinIsIdenticalAcrossTheLockAndContractScript fails a partial repin of
// the two surfaces it can read exactly: this package's constant against the
// lock the wrapper enforces, and the contract script's four expectations
// against the same lock. The Nix authoring toolchain is asserted by
// TestComponentToolchainIsPinnedAndIsolatedFromTheDefaultShell, and every
// restatement in the repository — README, inventory — by
// TestEveryFctlSDKRevisionRestatementMatchesTheLock. A plugin pinned to two
// revisions at once is not reproducible.
func TestSDKPinIsIdenticalAcrossTheLockAndContractScript(t *testing.T) {
	encoded, err := os.ReadFile("../fctl-sdk.lock.json")
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		Repository string `json:"repository"`
		Commit     string `json:"commit"`
		SDKNarHash string `json:"sdkNarHash"`
		WITSHA256  string `json:"witSha256"`
	}
	if err := json.Unmarshal(encoded, &lock); err != nil {
		t.Fatalf("decode fctl SDK lock: %v", err)
	}
	if lock.Commit != fctlSDKRevision {
		t.Errorf("lock commit = %q, want the authoring revision %q", lock.Commit, fctlSDKRevision)
	}

	script, err := os.ReadFile("../scripts/test-fctl-sdk-contract.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"readonly expected_commit='" + lock.Commit + "'",
		"readonly expected_repository='" + lock.Repository + "'",
		"readonly expected_nar_hash='" + lock.SDKNarHash + "'",
		"readonly expected_wit_hash='" + lock.WITSHA256 + "'",
	} {
		if !strings.Contains(string(script), required) {
			t.Errorf("test-fctl-sdk-contract.sh does not expect the locked value %q", required)
		}
	}
}
