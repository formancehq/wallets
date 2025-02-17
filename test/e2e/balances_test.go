//go:build it

package suite_test

import (
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/wallets/pkg/client/models/components"
	"github.com/formancehq/wallets/pkg/client/models/operations"
	"github.com/formancehq/wallets/pkg/testserver"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"math/big"
)

var _ = Context("Wallets - balances", func() {

	var (
		srv *testserver.Server
		ctx = logging.TestingContext()
	)
	BeforeEach(func() {
		srv = testserver.New(GinkgoT(), testserver.Configuration{
			Output:   GinkgoWriter,
			Debug:    debug,
			StackURL: stackURL.GetValue(),
		})
	})

	When("creating a wallet", func() {
		var (
			createWalletResponse *operations.CreateWalletResponse
			err                  error
		)
		BeforeEach(func() {
			createWalletResponse, err = srv.Client().Wallets.V1.CreateWallet(
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
			BeforeEach(func() {
				createBalanceResponse, err = srv.Client().Wallets.V1.CreateBalance(ctx, operations.CreateBalanceRequest{
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
				BeforeEach(func() {
					listBalancesResponse, err = srv.Client().Wallets.V1.ListBalances(ctx, operations.ListBalancesRequest{
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
