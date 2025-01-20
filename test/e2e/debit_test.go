//go:build it

package suite_test

import (
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/wallets/pkg/client/models/components"
	"github.com/formancehq/wallets/pkg/client/models/operations"
	"github.com/formancehq/wallets/pkg/client/models/sdkerrors"
	"github.com/formancehq/wallets/pkg/testserver"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"math/big"
	"time"
)

var _ = Context("Wallets - debit", func() {

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
		When("crediting it", func() {
			BeforeEach(func() {
				_, err := srv.Client().Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
					CreditWalletRequest: &components.CreditWalletRequest{
						Amount: components.Monetary{
							Amount: big.NewInt(1000),
							Asset:  "USD/2",
						},
						Sources:  []components.Subject{},
						Metadata: map[string]string{},
					},
					ID: createWalletResponse.CreateWalletResponse.Data.ID,
				})
				Expect(err).To(Succeed())
			})
			When("debiting it", func() {
				BeforeEach(func() {
					_, err := srv.Client().Wallets.V1.DebitWallet(ctx, operations.DebitWalletRequest{
						DebitWalletRequest: &components.DebitWalletRequest{
							Amount: components.Monetary{
								Amount: big.NewInt(100),
								Asset:  "USD/2",
							},
							Metadata: map[string]string{},
						},
						ID: createWalletResponse.CreateWalletResponse.Data.ID,
					})
					Expect(err).To(Succeed())
				})
				It("should be ok", func() {})
			})
			When("debiting it using timestamp", func() {
				now := time.Now().Round(time.Microsecond).UTC()
				BeforeEach(func() {
					_, err := srv.Client().Wallets.V1.DebitWallet(ctx, operations.DebitWalletRequest{
						DebitWalletRequest: &components.DebitWalletRequest{
							Amount: components.Monetary{
								Amount: big.NewInt(100),
								Asset:  "USD/2",
							},
							Metadata:  map[string]string{},
							Timestamp: &now,
						},
						ID: createWalletResponse.CreateWalletResponse.Data.ID,
					})
					Expect(err).To(Succeed())
				})
			})
			When("debiting it using a hold", func() {
				var (
					debitWalletResponse *operations.DebitWalletResponse
					ts                  *time.Time
				)
				JustBeforeEach(func() {
					debitWalletResponse, err = srv.Client().Wallets.V1.DebitWallet(ctx, operations.DebitWalletRequest{
						DebitWalletRequest: &components.DebitWalletRequest{
							Amount: components.Monetary{
								Amount: big.NewInt(100),
								Asset:  "USD/2",
							},
							Pending:   pointer.For(true),
							Metadata:  map[string]string{},
							Timestamp: ts,
						},
						ID: createWalletResponse.CreateWalletResponse.Data.ID,
					})
					Expect(err).To(Succeed())
				})
				It("should be ok", func() {})
				When("confirm the hold", func() {
					JustBeforeEach(func() {
						_, err := srv.Client().Wallets.V1.ConfirmHold(ctx, operations.ConfirmHoldRequest{
							HoldID: debitWalletResponse.DebitWalletResponse.Data.ID,
						})
						Expect(err).To(Succeed())
					})
					It("should be ok", func() {})
				})
				When("void the hold", func() {
					JustBeforeEach(func() {
						_, err := srv.Client().Wallets.V1.VoidHold(ctx, operations.VoidHoldRequest{
							HoldID: debitWalletResponse.DebitWalletResponse.Data.ID,
						})
						Expect(err).To(Succeed())
					})
					It("should be ok", func() {})
				})
			})
			When("debiting it using invalid destination", func() {
				It("should fail", func() {
					_, err := srv.Client().Wallets.V1.DebitWallet(ctx, operations.DebitWalletRequest{
						DebitWalletRequest: &components.DebitWalletRequest{
							Amount: components.Monetary{
								Amount: big.NewInt(100),
								Asset:  "USD/2",
							},
							Metadata: map[string]string{},
							Destination: pointer.For(components.CreateSubjectAccount(components.LedgerAccountSubject{
								Identifier: "@xxx",
							})),
						},
						ID: createWalletResponse.CreateWalletResponse.Data.ID,
					})
					Expect(err).NotTo(Succeed())
					sdkError := &sdkerrors.ErrorResponse{}
					Expect(errors.As(err, &sdkError)).To(BeTrue())
					Expect(sdkError.ErrorCode).To(Equal(sdkerrors.ErrorCodeValidation))
				})
			})
			When("debiting it using negative amount", func() {
				It("should fail", func() {
					_, err := srv.Client().Wallets.V1.DebitWallet(ctx, operations.DebitWalletRequest{
						DebitWalletRequest: &components.DebitWalletRequest{
							Amount: components.Monetary{
								Amount: big.NewInt(-100),
								Asset:  "USD/2",
							},
							Metadata: map[string]string{},
						},
						ID: createWalletResponse.CreateWalletResponse.Data.ID,
					})
					Expect(err).NotTo(Succeed())
					sdkError := &sdkerrors.ErrorResponse{}
					Expect(errors.As(err, &sdkError)).To(BeTrue())
					Expect(sdkError.ErrorCode).To(Equal(sdkerrors.ErrorCodeValidation))
				})
			})
			When("debiting it using invalid asset", func() {
				It("should fail", func() {
					_, err := srv.Client().Wallets.V1.DebitWallet(ctx, operations.DebitWalletRequest{
						DebitWalletRequest: &components.DebitWalletRequest{
							Amount: components.Monetary{
								Amount: big.NewInt(100),
								Asset:  "test",
							},
							Metadata: map[string]string{},
						},
						ID: createWalletResponse.CreateWalletResponse.Data.ID,
					})
					Expect(err).NotTo(Succeed())
					sdkError := &sdkerrors.ErrorResponse{}
					Expect(errors.As(err, &sdkError)).To(BeTrue())
					Expect(sdkError.ErrorCode).To(Equal(sdkerrors.ErrorCodeValidation))
				})
			})
		})
	})
})
