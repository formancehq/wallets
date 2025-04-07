//go:build it

package suite_test

import (
	"fmt"
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

var _ = Context("Wallets - transactions", func() {

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

	const nbWallets = 3
	When(fmt.Sprintf("creating %d wallet", nbWallets), func() {
		var (
			wallets []components.Wallet
		)
		BeforeEach(func() {
			for range nbWallets {
				response, err := srv.Client().Wallets.V1.CreateWallet(
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
			BeforeEach(func() {
				for i := range nbWallets {
					_, err := srv.Client().Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
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

					_, err = srv.Client().Wallets.V1.DebitWallet(ctx, operations.DebitWalletRequest{
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
				It("should returns two results", func() {
					txs, err := srv.Client().Wallets.V1.GetTransactions(ctx, operations.GetTransactionsRequest{
						WalletID: pointer.For(wallets[0].ID),
					})
					Expect(err).To(Succeed())
					Expect(txs.GetTransactionsResponse.Cursor.Data).To(HaveLen(2))
				})
			})
		})
	})
})
