package component

import (
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

func TestDescriptorContainsOnlyTheWalletsCommandFacet(t *testing.T) {
	descriptor := Descriptor()
	if descriptor.Metadata.Name != "wallets" || len(descriptor.Commands) != 14 {
		t.Fatalf("descriptor identity/commands = %q/%d", descriptor.Metadata.Name, len(descriptor.Commands))
	}
	if len(descriptor.AuthProviders) != 0 || len(descriptor.TargetProviders) != 0 || len(descriptor.SignerProviders) != 0 {
		t.Fatalf("descriptor unexpectedly owns privileged facets: %#v", descriptor)
	}
	if err := sdk.ValidateCatalogue(descriptor.Commands, descriptor.DocumentationResources); err != nil {
		t.Fatalf("descriptor catalogue invalid: %v", err)
	}
}
