package testserver

import (
	"context"
	"errors"
	"github.com/formancehq/wallets/cmd"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v2/httpclient"
	"github.com/formancehq/go-libs/v2/httpserver"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/otlp"
	"github.com/formancehq/go-libs/v2/otlp/otlpmetrics"
	"github.com/formancehq/go-libs/v2/service"
	walletsclient "github.com/formancehq/wallets/pkg/client"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type T interface {
	require.TestingT
	Cleanup(func())
	Helper()
	Logf(format string, args ...any)
}

type OTLPConfig struct {
	BaseConfig otlp.Config
	Metrics    *otlpmetrics.ModuleConfig
}

type Configuration struct {
	Output     io.Writer
	Debug      bool
	OTLPConfig *OTLPConfig
	StackURL   string
}

type Logger interface {
	Logf(fmt string, args ...any)
}

type Server struct {
	configuration Configuration
	logger        Logger
	sdkClient     *walletsclient.Formance
	cancel        func()
	ctx           context.Context
	errorChan     chan error
	id            string
	httpClient    *http.Client
	serverURL     string
}

func (s *Server) Start() error {
	rootCmd := cmd.NewRootCommand()
	args := []string{
		"serve",
		"--" + cmd.ListenFlag, ":0",
		"--" + cmd.StackURLFlag, s.configuration.StackURL,
		"--" + cmd.LedgerNameFlag, s.id,
	}
	if s.configuration.OTLPConfig != nil {
		if s.configuration.OTLPConfig.Metrics != nil {
			args = append(
				args,
				"--"+otlpmetrics.OtelMetricsExporterFlag, s.configuration.OTLPConfig.Metrics.Exporter,
			)
			if s.configuration.OTLPConfig.Metrics.KeepInMemory {
				args = append(
					args,
					"--"+otlpmetrics.OtelMetricsKeepInMemoryFlag,
				)
			}
			if s.configuration.OTLPConfig.Metrics.OTLPConfig != nil {
				args = append(
					args,
					"--"+otlpmetrics.OtelMetricsExporterOTLPEndpointFlag, s.configuration.OTLPConfig.Metrics.OTLPConfig.Endpoint,
					"--"+otlpmetrics.OtelMetricsExporterOTLPModeFlag, s.configuration.OTLPConfig.Metrics.OTLPConfig.Mode,
				)
				if s.configuration.OTLPConfig.Metrics.OTLPConfig.Insecure {
					args = append(args, "--"+otlpmetrics.OtelMetricsExporterOTLPInsecureFlag)
				}
			}
			if s.configuration.OTLPConfig.Metrics.RuntimeMetrics {
				args = append(args, "--"+otlpmetrics.OtelMetricsRuntimeFlag)
			}
			if s.configuration.OTLPConfig.Metrics.MinimumReadMemStatsInterval != 0 {
				args = append(
					args,
					"--"+otlpmetrics.OtelMetricsRuntimeMinimumReadMemStatsIntervalFlag,
					s.configuration.OTLPConfig.Metrics.MinimumReadMemStatsInterval.String(),
				)
			}
			if s.configuration.OTLPConfig.Metrics.PushInterval != 0 {
				args = append(
					args,
					"--"+otlpmetrics.OtelMetricsExporterPushIntervalFlag,
					s.configuration.OTLPConfig.Metrics.PushInterval.String(),
				)
			}
			if len(s.configuration.OTLPConfig.Metrics.ResourceAttributes) > 0 {
				args = append(
					args,
					"--"+otlp.OtelResourceAttributesFlag,
					strings.Join(s.configuration.OTLPConfig.Metrics.ResourceAttributes, ","),
				)
			}
		}
		if s.configuration.OTLPConfig.BaseConfig.ServiceName != "" {
			args = append(args, "--"+otlp.OtelServiceNameFlag, s.configuration.OTLPConfig.BaseConfig.ServiceName)
		}
	}
	if s.configuration.Debug {
		args = append(args, "--"+service.DebugFlag)
	}

	s.logger.Logf("Starting application with flags: %s", strings.Join(args, " "))
	rootCmd.SetArgs(args)
	rootCmd.SilenceErrors = true
	output := s.configuration.Output
	if output == nil {
		output = io.Discard
	}
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)

	ctx := logging.TestingContext()
	ctx = service.ContextWithLifecycle(ctx)
	ctx = httpserver.ContextWithServerInfo(ctx)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		s.errorChan <- rootCmd.ExecuteContext(ctx)
	}()

	select {
	case <-service.Ready(ctx):
	case err := <-s.errorChan:
		cancel()
		if err != nil {
			return err
		}

		return errors.New("unexpected service stop")
	}

	s.ctx, s.cancel = ctx, cancel

	var transport http.RoundTripper = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
	}
	if s.configuration.Debug {
		transport = httpclient.NewDebugHTTPTransport(transport)
	}

	s.httpClient = &http.Client{
		Transport: transport,
	}
	s.serverURL = httpserver.URL(s.ctx)

	s.sdkClient = walletsclient.New(
		walletsclient.WithServerURL(s.serverURL),
		walletsclient.WithClient(s.httpClient),
	)

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.cancel == nil {
		return nil
	}
	s.cancel()
	s.cancel = nil

	// Wait app to be marked as stopped
	select {
	case <-service.Stopped(s.ctx):
	case <-ctx.Done():
		return errors.New("service should have been stopped")
	}

	// Ensure the app has been properly shutdown
	select {
	case err := <-s.errorChan:
		return err
	case <-ctx.Done():
		return errors.New("service should have been stopped without error")
	}
}

func (s *Server) Client() *walletsclient.Formance {
	return s.sdkClient
}

func (s *Server) HTTPClient() *http.Client {
	return s.httpClient
}

func (s *Server) ServerURL() string {
	return s.serverURL
}

func (s *Server) Restart(ctx context.Context) error {
	if err := s.Stop(ctx); err != nil {
		return err
	}
	if err := s.Start(); err != nil {
		return err
	}

	return nil
}

func (s *Server) URL() string {
	return httpserver.URL(s.ctx)
}

func New(t T, configuration Configuration) *Server {
	t.Helper()

	srv := &Server{
		logger:        t,
		configuration: configuration,
		id:            uuid.NewString()[:8],
		errorChan:     make(chan error, 1),
	}
	t.Logf("Start testing server")
	require.NoError(t, srv.Start())
	t.Cleanup(func() {
		t.Logf("Stop testing server")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		require.NoError(t, srv.Stop(ctx))
	})

	return srv
}
