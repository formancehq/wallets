//go:build it

package suite_test

import (
	"context"
	"fmt"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	. "github.com/formancehq/go-libs/v3/testing/deferred/ginkgo"
	"github.com/formancehq/go-libs/v3/testing/testservice"
	"github.com/formancehq/wallets/pkg/client/models/components"
	"github.com/formancehq/wallets/pkg/client/models/operations"
	. "github.com/formancehq/wallets/pkg/testserver"
	"math/big"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Wallets - create", func() {
	const countWallets = 3
	var (
		ctx = logging.TestingContext()
	)
	When(fmt.Sprintf("creating %d wallets", countWallets), func() {
		var (
			srv = DeferTestServer(stackURL,
				testservice.WithLogger(GinkgoT()),
				testservice.WithInstruments(
					testservice.DebugInstrumentation(debug),
					testservice.OutputInstrumentation(GinkgoWriter),
				),
			)
		)
		JustBeforeEach(func(specContext SpecContext) {
			for i := 0; i < countWallets; i++ {
				name := uuid.NewString()
				response, err := Client(Wait(specContext, srv)).Wallets.V1.CreateWallet(
					context.Background(),
					operations.CreateWalletRequest{
						CreateWalletRequest: &components.CreateWalletRequest{
							Metadata: map[string]string{
								"wallets_number": fmt.Sprint(i),
							},
							Name: name,
						},
					},
				)
				Expect(err).ToNot(HaveOccurred())

				_, err = Client(Wait(specContext, srv)).Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
					CreditWalletRequest: &components.CreditWalletRequest{
						Amount: components.Monetary{
							Amount: big.NewInt(100),
							Asset:  "USD",
						},
					},
					ID: response.CreateWalletResponse.Data.ID,
				})
				Expect(err).ToNot(HaveOccurred())
			}
		})
		When("listing them", func() {
			var (
				request  operations.ListWalletsRequest
				response *operations.ListWalletsResponse
				err      error
			)
			BeforeEach(func() {
				request = operations.ListWalletsRequest{}
			})
			JustBeforeEach(func(specContext SpecContext) {
				Eventually(func(g Gomega) bool {
					response, err = Client(Wait(specContext, srv)).Wallets.V1.ListWallets(ctx, request)
					g.Expect(err).ToNot(HaveOccurred())

					return true
				}).Should(BeTrue())
			})
			It(fmt.Sprintf("should return %d items", countWallets), func() {
				Expect(response.ListWalletsResponse.Cursor.Data).To(HaveLen(countWallets))
			})
			Context("using a metadata filter", func() {
				BeforeEach(func() {
					request.Metadata = map[string]string{
						"wallets_number": "0",
					}
				})
				It("should return only one item", func() {
					Expect(response.ListWalletsResponse.Cursor.Data).To(HaveLen(1))
				})
			})
			Context("expanding balances", func() {
				BeforeEach(func() {
					request.Expand = pointer.For("balances")
				})
				It("should return all items with volumes and balances", func() {
					Expect(response.ListWalletsResponse.Cursor.Data).To(HaveLen(3))
					for _, wallet := range response.ListWalletsResponse.Cursor.Data {
						Expect(wallet.Balances.Main.Assets["USD"]).To(Equal(big.NewInt(100)))
					}
				})
			})
		})
	})
})
