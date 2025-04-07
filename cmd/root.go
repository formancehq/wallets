package cmd

import (
	"github.com/formancehq/go-libs/v3/service"

	"github.com/spf13/cobra"
)

var (
	ServiceName = "wallets"
	Version     = "develop"
	BuildDate   = "-"
	Commit      = "-"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{}

	cobra.EnableTraverseRunHooks = true

	cmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	serveCmd := newServeCommand()
	cmd.AddCommand(serveCmd)
	return cmd
}

func Execute() {
	service.Execute(NewRootCommand())
}
