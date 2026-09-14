package component

import (
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
