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
	"time"
)

var _ = Context("Wallets - summary", func() {

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

	When(`creating a wallet with 1000 USD, two available balances with 1000 USD, one expired with 1000 USD and three holds`, func() {
		var (
			createWalletResponse  *operations.CreateWalletResponse
			createBalanceResponse *operations.CreateBalanceResponse
			now                   = time.Now().Round(time.Second).UTC()
			err                   error
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

			_, err = Client(Wait(specContext, srv)).Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
				ID: createWalletResponse.CreateWalletResponse.Data.ID,
				CreditWalletRequest: &components.CreditWalletRequest{
					Amount: components.Monetary{
						Asset:  "USD",
						Amount: big.NewInt(1000),
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			createBalanceResponse, err = Client(Wait(specContext, srv)).Wallets.V1.CreateBalance(ctx, operations.CreateBalanceRequest{
				CreateBalanceRequest: &components.CreateBalanceRequest{
					Name:     "balance1",
					Priority: big.NewInt(10),
				},
				ID: createWalletResponse.CreateWalletResponse.Data.ID,
			})
			Expect(err).To(Succeed())

			_, err = Client(Wait(specContext, srv)).Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
				ID: createWalletResponse.CreateWalletResponse.Data.ID,
				CreditWalletRequest: &components.CreditWalletRequest{
					Amount: components.Monetary{
						Asset:  "USD",
						Amount: big.NewInt(1000),
					},
					Balance: pointer.For(createBalanceResponse.CreateBalanceResponse.Data.Name),
				},
			})
			Expect(err).ToNot(HaveOccurred())

			createBalanceResponse, err = Client(Wait(specContext, srv)).Wallets.V1.CreateBalance(ctx, operations.CreateBalanceRequest{
				CreateBalanceRequest: &components.CreateBalanceRequest{
					Name:      "balance2",
					ExpiresAt: pointer.For(now.Add(time.Minute)),
				},
				ID: createWalletResponse.CreateWalletResponse.Data.ID,
			})
			Expect(err).To(Succeed())

			_, err = Client(Wait(specContext, srv)).Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
				ID: createWalletResponse.CreateWalletResponse.Data.ID,
				CreditWalletRequest: &components.CreditWalletRequest{
					Amount: components.Monetary{
						Asset:  "USD",
						Amount: big.NewInt(1000),
					},
					Balance: pointer.For(createBalanceResponse.CreateBalanceResponse.Data.Name),
				},
			})
			Expect(err).ToNot(HaveOccurred())

			createBalanceResponse, err = Client(Wait(specContext, srv)).Wallets.V1.CreateBalance(ctx, operations.CreateBalanceRequest{
				CreateBalanceRequest: &components.CreateBalanceRequest{
					Name:      "balance3",
					ExpiresAt: pointer.For(now.Add(-time.Minute)),
				},
				ID: createWalletResponse.CreateWalletResponse.Data.ID,
			})
			Expect(err).To(Succeed())

			_, err = Client(Wait(specContext, srv)).Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
				ID: createWalletResponse.CreateWalletResponse.Data.ID,
				CreditWalletRequest: &components.CreditWalletRequest{
					Amount: components.Monetary{
						Asset:  "USD",
						Amount: big.NewInt(1000),
					},
					Balance: pointer.For(createBalanceResponse.CreateBalanceResponse.Data.Name),
				},
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = Client(Wait(specContext, srv)).Wallets.V1.CreateBalance(ctx, operations.CreateBalanceRequest{
				CreateBalanceRequest: &components.CreateBalanceRequest{
					Name: "balance4",
				},
				ID: createWalletResponse.CreateWalletResponse.Data.ID,
			})
			Expect(err).To(Succeed())

			_, err = Client(Wait(specContext, srv)).Wallets.V1.DebitWallet(ctx, operations.DebitWalletRequest{
				ID: createWalletResponse.CreateWalletResponse.Data.ID,
				DebitWalletRequest: &components.DebitWalletRequest{
					Amount: components.Monetary{
						Asset:  "USD",
						Amount: big.NewInt(1500),
					},
					Balances: []string{"main", "balance1"},
					Pending:  pointer.For(true),
				},
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = Client(Wait(specContext, srv)).Wallets.V1.DebitWallet(ctx, operations.DebitWalletRequest{
				ID: createWalletResponse.CreateWalletResponse.Data.ID,
				DebitWalletRequest: &components.DebitWalletRequest{
					Amount: components.Monetary{
						Asset:  "USD",
						Amount: big.NewInt(500),
					},
					Balances: []string{"main", "balance2"},
					Pending:  pointer.For(true),
				},
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = Client(Wait(specContext, srv)).Wallets.V1.DebitWallet(ctx, operations.DebitWalletRequest{
				ID: createWalletResponse.CreateWalletResponse.Data.ID,
				DebitWalletRequest: &components.DebitWalletRequest{
					Amount: components.Monetary{
						Asset:  "USD",
						Amount: big.NewInt(700),
					},
					Balances: []string{"balance1", "balance2"},
					Pending:  pointer.For(true),
				},
			})
			Expect(err).ToNot(HaveOccurred())
		})
		It("the summary should be correct", func(specContext SpecContext) {
			summary, err := Client(Wait(specContext, srv)).Wallets.V1.GetWalletSummary(ctx, operations.GetWalletSummaryRequest{
				ID: createWalletResponse.CreateWalletResponse.Data.ID,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(summary.GetWalletSummaryResponse.Data).To(Equal(components.WalletSummary{
				Balances: []components.BalanceWithAssets{
					{
						Name:     "balance1",
						Priority: big.NewInt(10),
						Assets: map[string]*big.Int{
							"USD": big.NewInt(0),
						},
					},
					{
						Name:      "balance2",
						Priority:  new(big.Int),
						ExpiresAt: pointer.For(now.Add(time.Minute)),
						Assets: map[string]*big.Int{
							"USD": big.NewInt(300),
						},
					},
					{
						Name:      "balance3",
						Priority:  new(big.Int),
						ExpiresAt: pointer.For(now.Add(-time.Minute)),
						Assets: map[string]*big.Int{
							"USD": big.NewInt(1000),
						},
					},
					{
						Name:     "balance4",
						Priority: new(big.Int),
						Assets:   map[string]*big.Int{},
					},
					{
						Name:     "main",
						Priority: new(big.Int),
						Assets: map[string]*big.Int{
							"USD": big.NewInt(0),
						},
					},
				},
				AvailableFunds: map[string]*big.Int{
					"USD": big.NewInt(300),
				},
				ExpiredFunds: map[string]*big.Int{
					"USD": big.NewInt(1000),
				},
				ExpirableFunds: map[string]*big.Int{
					"USD": big.NewInt(300),
				},
				HoldFunds: map[string]*big.Int{
					"USD": big.NewInt(2700),
				},
			}))
		})
	})
})
