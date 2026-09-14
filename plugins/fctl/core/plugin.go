package core

import (
	"context"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

const (
	Name    = "wallets"
	Version = "0.1.0"
)

type Plugin struct{}

var _ sdk.Plugin = Plugin{}

func (Plugin) Metadata() sdk.Metadata {
	return sdk.Metadata{
		Name:    Name,
		Version: Version,
		Facets: []sdk.Facet{{
			Kind:                     sdk.FacetCommandProvider,
			ProtocolVersion:          sdk.CurrentCommandProviderFacetProtocolVersion,
			RequiredHostCapabilities: []string{sdk.HostCapabilityGeneratedClientV1},
		}},
	}
}

func (Plugin) Commands() []sdk.Command { return Catalogue() }

func (Plugin) DocumentationResources() []sdk.DocumentationResource { return nil }

func (Plugin) Execute(ctx context.Context, request sdk.ExecuteRequest, host sdk.Host) error {
	return executeV2(ctx, request, host)
}
