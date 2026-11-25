package cmd

import (
	"context"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/audit"
	"github.com/formancehq/go-libs/v5/pkg/authn/jwt"
	"github.com/formancehq/go-libs/v5/pkg/authn/licence"
	"github.com/formancehq/go-libs/v5/pkg/fx/authnfx"
	"github.com/formancehq/go-libs/v5/pkg/fx/messagingfx"
	"github.com/formancehq/go-libs/v5/pkg/fx/observefx"
	"github.com/formancehq/go-libs/v5/pkg/messaging/publish"
	"github.com/formancehq/go-libs/v5/pkg/observe"
	"github.com/formancehq/go-libs/v5/pkg/observe/traces"
	"github.com/formancehq/go-libs/v5/pkg/service"
	sharedapi "github.com/formancehq/go-libs/v5/pkg/transport/api"
	wallet "github.com/formancehq/wallets/pkg"
	"github.com/formancehq/wallets/pkg/api"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	StackClientIDFlag     = "stack-client-id"
	StackClientSecretFlag = "stack-client-secret"
	StackURLFlag          = "stack-url"
	LedgerNameFlag        = "ledger"
	LedgerAPIVersionFlag  = "ledger-api-version"
	LedgerV3CursorKeyFlag = "ledger-v3-cursor-key"
	LedgerURLFlag         = "ledger-url"
	AccountPrefixFlag     = "account-prefix"
	ListenFlag            = "listen"
)

func newServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "serve",
		Aliases:      []string{"server"},
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			stackClientID, _ := cmd.Flags().GetString(StackClientIDFlag)
			stackClientSecret, _ := cmd.Flags().GetString(StackClientSecretFlag)
			stackURL, _ := cmd.Flags().GetString(StackURLFlag)
			ledgerName, _ := cmd.Flags().GetString(LedgerNameFlag)
			ledgerVersion, _ := cmd.Flags().GetString(LedgerAPIVersionFlag)
			cursorKey, _ := cmd.Flags().GetString(LedgerV3CursorKeyFlag)
			ledgerURL, _ := cmd.Flags().GetString(LedgerURLFlag)
			accountPrefix, _ := cmd.Flags().GetString(AccountPrefixFlag)
			listen, _ := cmd.Flags().GetString(ListenFlag)
			auditEnabled, _ := cmd.Flags().GetBool(audit.AuditEnabledFlag)

			options := []fx.Option{
				fx.Supply(audit.Config{Enabled: auditEnabled}),
				fx.Provide(func() (*http.Client, error) {
					return GetHTTPClient(
						cmd.Context(),
						stackClientID,
						stackClientSecret,
						stackURL,
						service.IsDebug(cmd),
					)
				}),
				wallet.ModuleWithLedgerOptions(cmd.Context(), stackURL, ledgerName, accountPrefix, wallet.LedgerOptions{
					APIVersion:  ledgerVersion,
					V3CursorKey: cursorKey,
					URL:         ledgerURL,
				}),
				api.Module(sharedapi.ServiceInfo{
					Version: Version,
					Debug:   service.IsDebug(cmd),
				}, listen),
				messagingfx.PublishModuleFromFlags(cmd, service.IsDebug(cmd)),
				observefx.ResourceModuleFromFlags(cmd),
				observefx.TracesModuleFromFlags(cmd),
				authnfx.JWTModuleFromFlags(cmd),
				authnfx.LicenceModuleFromFlags(cmd, ServiceName),
			}

			return service.New(cmd.OutOrStdout(), options...).Run(cmd)
		},
	}
	cmd.Flags().String(StackClientIDFlag, "", "Client ID")
	cmd.Flags().String(StackClientSecretFlag, "", "Client Secret")
	cmd.Flags().String(StackURLFlag, "", "Token URL")
	cmd.Flags().String(LedgerNameFlag, "wallets-002", "Target ledger")
	cmd.Flags().String(LedgerAPIVersionFlag, "v2", "Ledger API version (v2, v3, or auto)")
	cmd.Flags().String(LedgerV3CursorKeyFlag, "", "Shared Ledger v3 cursor signing secret (minimum 32 bytes)")
	cmd.Flags().String(LedgerURLFlag, "", "Ledger service URL, including its gateway mount if applicable")
	cmd.Flags().String(AccountPrefixFlag, "", "Account prefix flag")
	cmd.Flags().String(ListenFlag, ":8080", "Listen address")
	cmd.Flags().Bool(audit.AuditEnabledFlag, false, "Enable HTTP audit")

	service.AddFlags(cmd.Flags())
	licence.AddFlags(cmd.Flags())
	jwt.AddFlags(cmd.Flags())
	traces.AddFlags(cmd.Flags())
	publish.AddFlags(ServiceName, cmd.Flags())

	return cmd
}

func GetHTTPClient(ctx context.Context, clientID, clientSecret, stackURL string, debug bool) (*http.Client, error) {
	return getHTTPClient(ctx, clientID, clientSecret, stackURL, debug, 5*time.Second)
}

func getHTTPClient(ctx context.Context, clientID, clientSecret, stackURL string, debug bool, tokenTimeout time.Duration) (*http.Client, error) {
	httpClient := &http.Client{
		Transport: observe.NewRoundTripper(http.DefaultTransport, debug),
	}

	if clientID == "" {
		return httpClient, nil
	}

	clientCredentialsConfig := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     stackURL + "/api/auth/oauth/token",
		Scopes:       []string{"openid ledger:read ledger:write"},
	}

	// The OAuth token source inherits the service context, so bound token HTTP
	// calls independently of the Ledger request awaiting the token.
	tokenClient := *httpClient
	tokenClient.Timeout = tokenTimeout
	authenticated := clientCredentialsConfig.Client(context.WithValue(ctx, oauth2.HTTPClient, &tokenClient))
	authenticated.Timeout = httpClient.Timeout
	return authenticated, nil
}
