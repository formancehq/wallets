//go:build it

package suite_test

import (
	"fmt"
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

var _ = Context("Wallets - transactions", func() {

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

	const nbWallets = 3
	When(fmt.Sprintf("creating %d wallet", nbWallets), func() {
		var (
			wallets []components.Wallet
		)
		BeforeEach(func(specContext SpecContext) {
			for range nbWallets {
				response, err := Client(Wait(specContext, srv)).Wallets.V1.CreateWallet(
					ctx,
					operations.CreateWalletRequest{
						CreateWalletRequest: &components.CreateWalletRequest{
							Name:     uuid.NewString(),
							Metadata: map[string]string{},
						},
					},
				)
				Expect(err).ToNot(HaveOccurred())
				wallets = append(wallets, response.CreateWalletResponse.Data)
			}
		})
		When("crediting them then debiting them", func() {
			BeforeEach(func(specContext SpecContext) {
				for i := range nbWallets {
					_, err := Client(Wait(specContext, srv)).Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
						CreditWalletRequest: &components.CreditWalletRequest{
							Amount: components.Monetary{
								Amount: big.NewInt(1000),
								Asset:  "USD/2",
							},
							Sources:  []components.Subject{},
							Metadata: map[string]string{},
						},
						ID: wallets[i].ID,
					})
					Expect(err).To(Succeed())

					_, err = Client(Wait(specContext, srv)).Wallets.V1.DebitWallet(ctx, operations.DebitWalletRequest{
						DebitWalletRequest: &components.DebitWalletRequest{
							Amount: components.Monetary{
								Amount: big.NewInt(100),
								Asset:  "USD/2",
							},
							Metadata: map[string]string{},
						},
						ID: wallets[i].ID,
					})
					Expect(err).To(Succeed())
				}
			})
			When("listing transactions of the first wallet", func() {
				It("should returns two results", func(specContext SpecContext) {
					txs, err := Client(Wait(specContext, srv)).Wallets.V1.GetTransactions(ctx, operations.GetTransactionsRequest{
						WalletID: pointer.For(wallets[0].ID),
					})
					Expect(err).To(Succeed())
					Expect(txs.GetTransactionsResponse.Cursor.Data).To(HaveLen(2))
				})
			})
		})
	})
})
