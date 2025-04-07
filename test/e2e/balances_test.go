//go:build it

package suite_test

import (
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	. "github.com/formancehq/go-libs/v3/testing/deferred/ginkgo"
	"github.com/formancehq/go-libs/v3/testing/testservice"
	"github.com/formancehq/wallets/pkg/client/models/components"
	"github.com/formancehq/wallets/pkg/client/models/operations"
	. "github.com/formancehq/wallets/pkg/testserver"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"math/big"
)

var _ = Context("Wallets - balances", func() {

	var (
		srv = DeferTestServer(stackURL,
			testservice.WithLogger(GinkgoT()),
			testservice.WithInstruments(
				testservice.DebugInstrumentation(debug),
				testservice.OutputInstrumentation(GinkgoWriter),
			),
		)
		ctx = logging.TestingContext()
	)

	When("creating a wallet", func() {
		var (
			createWalletResponse *operations.CreateWalletResponse
			err                  error
		)
		BeforeEach(func(specContext SpecContext) {
			createWalletResponse, err = Client(Wait(specContext, srv)).Wallets.V1.CreateWallet(
				ctx,
				operations.CreateWalletRequest{
					CreateWalletRequest: &components.CreateWalletRequest{
						Name:     uuid.NewString(),
						Metadata: map[string]string{},
					},
				},
			)
			Expect(err).ToNot(HaveOccurred())
		})
		When("creating a new balance", func() {
			var (
				createBalanceResponse *operations.CreateBalanceResponse
				err                   error
			)
			BeforeEach(func(specContext SpecContext) {
				createBalanceResponse, err = Client(Wait(specContext, srv)).Wallets.V1.CreateBalance(ctx, operations.CreateBalanceRequest{
					CreateBalanceRequest: &components.CreateBalanceRequest{
						Name:     "balance1",
						Priority: big.NewInt(10),
					},
					ID:             createWalletResponse.CreateWalletResponse.Data.ID,
					IdempotencyKey: pointer.For("foo"),
				})
				Expect(err).To(Succeed())
			})
			It("should be ok", func() {
				Expect(createBalanceResponse.CreateBalanceResponse.Data.Name).To(Equal("balance1"))
				Expect(createBalanceResponse.CreateBalanceResponse.Data.Priority).To(Equal(big.NewInt(10)))
			})
			When("listing balances", func() {
				var (
					listBalancesResponse *operations.ListBalancesResponse
				)
				BeforeEach(func(specContext SpecContext) {
					listBalancesResponse, err = Client(Wait(specContext, srv)).Wallets.V1.ListBalances(ctx, operations.ListBalancesRequest{
						ID: createWalletResponse.CreateWalletResponse.Data.ID,
					})
					Expect(err).ToNot(HaveOccurred())
				})
				It("should return created balance", func() {
					Expect(listBalancesResponse.ListBalancesResponse.Cursor.Data).To(HaveLen(2))
					Expect(listBalancesResponse.ListBalancesResponse.Cursor.Data).To(ContainElements(
						components.Balance{
							Name:     "balance1",
							Priority: big.NewInt(10),
						},
						components.Balance{
							Name:     "main",
							Priority: big.NewInt(0),
						},
					))
				})
			})
		})
	})
})
