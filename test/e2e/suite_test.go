//go:build it

package suite_test

import (
	"encoding/json"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/testing/docker"
	. "github.com/formancehq/go-libs/v2/testing/platform/pgtesting"
	. "github.com/formancehq/go-libs/v2/testing/utils"
	ledgertestserver "github.com/formancehq/ledger/pkg/testserver"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestExamples(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Wallets Testing Suite")
}

var (
	dockerPool = NewDeferred[*docker.Pool]()
	stackURL   = NewDeferred[string]()
	debug      = os.Getenv("DEBUG") == "true"
	logger     = logging.NewDefaultLogger(GinkgoWriter, debug, false, false)
)

type ParallelExecutionContext struct {
	StackURL string
}

var _ = SynchronizedBeforeSuite(func() []byte {
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

	ledgerServer := ledgertestserver.New(GinkgoT(), ledgertestserver.Configuration{
		PostgresConfiguration: db.ConnectionOptions(),
		Output:                GinkgoWriter,
		Debug:                 debug,
	})

	Expect(ledgerServer.Start()).To(BeNil())
	DeferCleanup(func() {
		ctx := logging.TestingContext()
		Expect(ledgerServer.Stop(ctx)).To(BeNil())
	})

	ledgerURL, err := url.Parse(ledgerServer.URL())
	if err != nil {
		panic(err)
	}

	r := chi.NewRouter()
	r.Mount("/api/ledger",
		http.StripPrefix("/api/ledger", httputil.NewSingleHostReverseProxy(ledgerURL)),
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
