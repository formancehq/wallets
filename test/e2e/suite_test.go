//go:build it

package suite_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"testing"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/go-libs/v5/pkg/testing/deferred"
	"github.com/formancehq/go-libs/v5/pkg/testing/docker"
	. "github.com/formancehq/go-libs/v5/pkg/testing/platform/pgtesting"
	"github.com/formancehq/go-libs/v5/pkg/testing/testservice"
	ledgertestserver "github.com/formancehq/ledger/pkg/testserver"
	"github.com/go-chi/chi/v5"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestExamples(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Wallets Testing Suite")
}

var (
	dockerPool = deferred.New[*docker.Pool]()
	stackURL   = deferred.New[string]()
	debug      = os.Getenv("DEBUG") == "true"
	logger     = logging.NewDefaultLogger(GinkgoWriter, debug, false, false)
)

type ParallelExecutionContext struct {
	StackURL string
}

var _ = SynchronizedBeforeSuite(func() []byte {
	deferred.RegisterRecoverHandler(GinkgoRecover)

	By("Initializing docker pool")
	dockerPool.SetValue(docker.NewPool(GinkgoT(), logger))

	ret := CreatePostgresServer(
		GinkgoT(),
		dockerPool.GetValue(),
		WithPGStatsExtension(),
		WithPGCrypto(),
	)
	By("Postgres address: " + ret.GetDSN())

	db := ret.NewDatabase(GinkgoT())

	ledgerServer := ledgertestserver.NewTestServer(
		deferred.FromValue(db.ConnectionOptions()),
		testservice.WithLogger(GinkgoT()),
		testservice.WithInstruments(
			testservice.DebugInstrumentation(debug),
			testservice.OutputInstrumentation(GinkgoWriter),
		),
	)
	Expect(ledgerServer.Start(context.Background())).To(BeNil())
	DeferCleanup(ledgerServer.Stop)

	r := chi.NewRouter()
	r.Mount("/api/ledger",
		http.StripPrefix("/api/ledger", httputil.NewSingleHostReverseProxy(testservice.GetServerURL(ledgerServer))),
	)

	srv := httptest.NewServer(r)
	DeferCleanup(func() {
		srv.Close()
	})

	data, err := json.Marshal(ParallelExecutionContext{
		StackURL: srv.URL,
	})
	Expect(err).To(BeNil())

	return data
}, func(data []byte) {
	pec := ParallelExecutionContext{}
	err := json.Unmarshal(data, &pec)
	Expect(err).To(BeNil())

	stackURL.SetValue(pec.StackURL)
})
