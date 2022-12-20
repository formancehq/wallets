package cmd

import (
	"fmt"

	"github.com/formancehq/go-libs/sharedotlp/pkg/sharedotlptraces"
	"github.com/formancehq/wallets/pkg"
	"github.com/formancehq/wallets/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

const (
	clientIDFlag     = "client-id"
	clientSecretFlag = "client-secret"
	tokenURLFlag     = "token-url"
)

var serveCmd = &cobra.Command{
	Use: "serve",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return bindFlagsToViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		options := []fx.Option{
			fx.NopLogger,
			// TODO: Make configurable
			pkg.Module("wallets-002", ""),
			client.NewModule(viper.GetString(clientIDFlag), viper.GetString(clientSecretFlag), viper.GetString(tokenURLFlag)),
			sharedotlptraces.CLITracesModule(viper.GetViper()),
		}

		app := fx.New(options...)
		if err := app.Start(cmd.Context()); err != nil {
			return fmt.Errorf("fx.App.Start: %w", err)
		}

		<-app.Done()

		if err := app.Stop(cmd.Context()); err != nil {
			return fmt.Errorf("fx.App.Stop: %w", err)
		}

		return nil
	},
}

func init() {
	serveCmd.Flags().String(clientIDFlag, "", "Client ID")
	serveCmd.Flags().String(clientSecretFlag, "", "Client Secret")
	serveCmd.Flags().String(tokenURLFlag, "", "Token URL")
	rootCmd.AddCommand(serveCmd)
}
