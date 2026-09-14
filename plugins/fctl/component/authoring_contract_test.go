package component

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const fctlSDKRevision = "e9b1395f46f3100b381dbe00f5213de28e6df0e1"

func TestAuthoringDevShellPinsTheCompleteComponentToolchain(t *testing.T) {
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
		"componentTools.componentize-go",
		"componentTools.wasi-virt",
		"componentTools.wasm-tools",
		"binaryen",
		"rust-bin.stable.\"1.91.1\".minimal",
		"wasmToolsVersion = \"1.239.0\"",
		"sha256-9wJSNC/clO8M7E840i1lRWJQT7AdRQX468swmZ4O1rg=",
		"sha256-xIfTYJMVP47timzquEYEb9M8BHsj83NjgD44lbzgd+Y=",
		"version = \"0.4.1\"",
		"version = \"0.2.0-448f6df8\"",
	} {
		if !strings.Contains(contents, required) {
			t.Errorf("Wallets authoring shell does not pin required component input %q", required)
		}
	}
}

// TestSDKPinIsIdenticalAcrossTheLockToolchainAndContractScript fails a partial
// repin. The SDK revision is asserted in four independent places — the lock the
// wrapper enforces, the Nix authoring toolchain, this package's constant, and
// the contract script's expectations — and a plugin pinned to two revisions at
// once is not reproducible.
func TestSDKPinIsIdenticalAcrossTheLockToolchainAndContractScript(t *testing.T) {
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
