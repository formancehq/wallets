// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/wallets/pkg/client/models/components"
)

type DebitWalletRequest struct {
	ID string `pathParam:"style=simple,explode=false,name=id"`
	// Use an idempotency key
	IdempotencyKey     *string                        `header:"style=simple,explode=false,name=Idempotency-Key"`
	DebitWalletRequest *components.DebitWalletRequest `request:"mediaType=application/json"`
}

func (o *DebitWalletRequest) GetID() string {
	if o == nil {
		return ""
	}
	return o.ID
}

func (o *DebitWalletRequest) GetIdempotencyKey() *string {
	if o == nil {
		return nil
	}
	return o.IdempotencyKey
}

func (o *DebitWalletRequest) GetDebitWalletRequest() *components.DebitWalletRequest {
	if o == nil {
		return nil
	}
	return o.DebitWalletRequest
}

type DebitWalletResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Wallet successfully debited as a pending hold
	DebitWalletResponse *components.DebitWalletResponse
}

func (o *DebitWalletResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *DebitWalletResponse) GetDebitWalletResponse() *components.DebitWalletResponse {
	if o == nil {
		return nil
	}
	return o.DebitWalletResponse
}
