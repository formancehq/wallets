package core

import (
	"reflect"
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

func TestPluginDeclaresOnlyTheGeneratedClientCommandFacet(t *testing.T) {
	plugin := Plugin{}
	metadata := plugin.Metadata()
	if metadata.Name != Name || metadata.Version != Version {
		t.Fatalf("metadata identity = %q@%q", metadata.Name, metadata.Version)
	}
	wantFacets := []sdk.Facet{{
		Kind:                     sdk.FacetCommandProvider,
		ProtocolVersion:          sdk.CurrentCommandProviderFacetProtocolVersion,
		RequiredHostCapabilities: []string{sdk.HostCapabilityGeneratedClientV1},
	}}
	if !reflect.DeepEqual(metadata.Facets, wantFacets) {
		t.Fatalf("metadata facets = %#v, want %#v", metadata.Facets, wantFacets)
	}
	if err := sdk.ValidateCommandFacetHostRequirements(metadata.Facets, plugin.Commands()); err != nil {
		t.Fatalf("facet requirements do not match catalogue: %v", err)
	}
	if resources := plugin.DocumentationResources(); resources != nil {
		t.Fatalf("documentation resources = %#v, want nil", resources)
	}
}
