//go:build it

package suite_test

import (
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	. "github.com/formancehq/go-libs/v3/testing/deferred/ginkgo"
	"github.com/formancehq/go-libs/v3/testing/testservice"
	"github.com/formancehq/wallets/pkg/client/models/components"
	"github.com/formancehq/wallets/pkg/client/models/operations"
	"github.com/formancehq/wallets/pkg/client/models/sdkerrors"
	. "github.com/formancehq/wallets/pkg/testserver"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"math/big"
	"time"
)

var _ = Context("Wallets - credit", func() {

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
			response *operations.CreateWalletResponse
			err      error
		)
		BeforeEach(func(specContext SpecContext) {
			response, err = Client(Wait(specContext, srv)).Wallets.V1.CreateWallet(
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
		When("crediting it", func() {
			BeforeEach(func(specContext SpecContext) {
				_, err := Client(Wait(specContext, srv)).Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
					CreditWalletRequest: &components.CreditWalletRequest{
						Amount: components.Monetary{
							Amount: big.NewInt(1000),
							Asset:  "USD/2",
						},
						Sources:  []components.Subject{},
						Metadata: map[string]string{},
					},
					ID:             response.CreateWalletResponse.Data.ID,
					IdempotencyKey: pointer.For("foo"),
				})
				Expect(err).To(Succeed())
			})
			It("should be ok", func() {})
			When("crediting again with the same ik", func() {
				BeforeEach(func(specContext SpecContext) {
					_, err := Client(Wait(specContext, srv)).Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
						CreditWalletRequest: &components.CreditWalletRequest{
							Amount: components.Monetary{
								Amount: big.NewInt(1000),
								Asset:  "USD/2",
							},
							Sources:  []components.Subject{},
							Metadata: map[string]string{},
						},
						ID:             response.CreateWalletResponse.Data.ID,
						IdempotencyKey: pointer.For("foo"),
					})
					Expect(err).To(Succeed())
				})
				It("Should not trigger any movements", func(specContext SpecContext) {
					balance, err := Client(Wait(specContext, srv)).Wallets.V1.GetBalance(ctx, operations.GetBalanceRequest{
						BalanceName: "main",
						ID:          response.CreateWalletResponse.Data.ID,
					})
					Expect(err).To(Succeed())
					Expect(balance.GetBalanceResponse.Data.Assets["USD/2"]).To(Equal(big.NewInt(1000)))
				})
			})
		})
		When("crediting it with specified timestamp", func() {
			now := time.Now().Round(time.Microsecond).UTC()
			BeforeEach(func(specContext SpecContext) {
				_, err := Client(Wait(specContext, srv)).Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
					CreditWalletRequest: &components.CreditWalletRequest{
						Amount: components.Monetary{
							Amount: big.NewInt(1000),
							Asset:  "USD/2",
						},
						Sources:   []components.Subject{},
						Metadata:  map[string]string{},
						Timestamp: &now,
					},
					ID: response.CreateWalletResponse.Data.ID,
				})
				Expect(err).To(Succeed())
			})
		})
		When("crediting it with invalid source", func() {
			It("should fail", func(specContext SpecContext) {
				_, err := Client(Wait(specContext, srv)).Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
					CreditWalletRequest: &components.CreditWalletRequest{
						Amount: components.Monetary{
							Amount: big.NewInt(1000),
							Asset:  "USD/2",
						},
						Sources: []components.Subject{components.CreateSubjectAccount(components.LedgerAccountSubject{
							Identifier: "@xxx",
						})},
						Metadata: map[string]string{},
					},
					ID: response.CreateWalletResponse.Data.ID,
				})
				Expect(err).NotTo(Succeed())
				sdkError := &sdkerrors.ErrorResponse{}
				Expect(errors.As(err, &sdkError)).To(BeTrue())
				Expect(sdkError.ErrorCode).To(Equal(sdkerrors.ErrorCodeValidation))
			})
		})
		When("crediting it with negative amount", func() {
			It("should fail", func(specContext SpecContext) {
				_, err := Client(Wait(specContext, srv)).Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
					CreditWalletRequest: &components.CreditWalletRequest{
						Amount: components.Monetary{
							Amount: big.NewInt(-1000),
							Asset:  "USD/2",
						},
						Sources:  []components.Subject{},
						Metadata: map[string]string{},
					},
					ID: response.CreateWalletResponse.Data.ID,
				})
				Expect(err).NotTo(Succeed())

				sdkError := &sdkerrors.ErrorResponse{}
				Expect(errors.As(err, &sdkError)).To(BeTrue())
				Expect(sdkError.ErrorCode).To(Equal(sdkerrors.ErrorCodeValidation))
			})
		})
		When("crediting it with invalid asset name", func() {
			It("should fail", func(specContext SpecContext) {
				_, err := Client(Wait(specContext, srv)).Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
					CreditWalletRequest: &components.CreditWalletRequest{
						Amount: components.Monetary{
							Amount: big.NewInt(1000),
							Asset:  "test",
						},
						Sources:  []components.Subject{},
						Metadata: map[string]string{},
					},
					ID: response.CreateWalletResponse.Data.ID,
				})
				Expect(err).NotTo(Succeed())
				sdkError := &sdkerrors.ErrorResponse{}
				Expect(errors.As(err, &sdkError)).To(BeTrue())
				Expect(sdkError.ErrorCode).To(Equal(sdkerrors.ErrorCodeValidation))
			})
		})
	})
})
