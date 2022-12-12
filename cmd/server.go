package cmd

import (
	"fmt"

	"github.com/formancehq/go-libs/sharedotlp/pkg/sharedotlptraces"
	"github.com/formancehq/wallets/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var serveCmd = &cobra.Command{
	Use: "server",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return bindFlagsToViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		options := []fx.Option{
			pkg.Module(),
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
	rootCmd.AddCommand(serveCmd)
}
