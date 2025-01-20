// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"openapi/models/components"
)

type VoidHoldRequest struct {
	HoldID string `pathParam:"style=simple,explode=false,name=hold_id"`
	// Use an idempotency key
	IdempotencyKey *string `header:"style=simple,explode=false,name=Idempotency-Key"`
}

func (o *VoidHoldRequest) GetHoldID() string {
	if o == nil {
		return ""
	}
	return o.HoldID
}

func (o *VoidHoldRequest) GetIdempotencyKey() *string {
	if o == nil {
		return nil
	}
	return o.IdempotencyKey
}

type VoidHoldResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Error
	ErrorResponse *components.ErrorResponse
}

func (o *VoidHoldResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *VoidHoldResponse) GetErrorResponse() *components.ErrorResponse {
	if o == nil {
		return nil
	}
	return o.ErrorResponse
}
