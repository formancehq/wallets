package testserver

import (
	"context"
	"github.com/formancehq/go-libs/v3/httpserver"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/formancehq/go-libs/v3/testing/testservice"
	. "github.com/formancehq/go-libs/v3/testing/testservice/ginkgo"
	"github.com/formancehq/wallets/cmd"
	walletsclient "github.com/formancehq/wallets/pkg/client"
	. "github.com/onsi/ginkgo/v2/dsl/core"
)

func StackURLInstrumentation(stackURL *deferred.Deferred[string]) testservice.Instrumentation {
	return testservice.InstrumentationFunc(func(ctx context.Context, runConfiguration *testservice.RunConfiguration) error {
		stackURL, err := stackURL.Wait(ctx)
		if err != nil {
			return err
		}
		runConfiguration.AppendArgs(
			"--"+cmd.StackURLFlag, stackURL,
		)

		return nil
	})
}

func DeferTestServer(stackURL *deferred.Deferred[string], options ...testservice.Option) *deferred.Deferred[*testservice.Service] {
	return DeferNew(cmd.NewRootCommand,
		append([]testservice.Option{
			testservice.WithLogger(GinkgoT()),
			testservice.WithInstruments(
				testservice.AppendArgsInstrumentation("serve", "--"+cmd.ListenFlag, ":0"),
				testservice.InstrumentationFunc(func(ctx context.Context, cfg *testservice.RunConfiguration) error {
					cfg.AppendArgs("--"+cmd.LedgerNameFlag, cfg.GetID())
					return nil
				}),
				testservice.HTTPServerInstrumentation(),
				StackURLInstrumentation(stackURL),
			),
		}, options...)...,
	)
}

func Client(srv *testservice.Service) *walletsclient.Formance {
	return walletsclient.New(
		walletsclient.WithServerURL(httpserver.URL(srv.GetContext())),
	)
}
