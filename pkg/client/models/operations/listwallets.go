// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/wallets/pkg/client/internal/utils"
	"github.com/formancehq/wallets/pkg/client/models/components"
)

type ListWalletsRequest struct {
	// Filter on wallet name
	Name *string `queryParam:"style=form,explode=true,name=name"`
	// Filter wallets by metadata key value pairs. Nested objects can be used as seen in the example below.
	Metadata map[string]string `queryParam:"style=deepObject,explode=true,name=metadata"`
	// The maximum number of results to return per page
	PageSize *int64 `default:"15" queryParam:"style=form,explode=true,name=pageSize"`
	// Parameter used in pagination requests.
	// Set to the value of next for the next page of results.
	// Set to the value of previous for the previous page of results.
	// No other parameters can be set when the pagination token is set.
	//
	Cursor *string `queryParam:"style=form,explode=true,name=cursor"`
	Expand *string `queryParam:"style=form,explode=true,name=expand"`
}

func (l ListWalletsRequest) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(l, "", false)
}

func (l *ListWalletsRequest) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &l, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *ListWalletsRequest) GetName() *string {
	if o == nil {
		return nil
	}
	return o.Name
}

func (o *ListWalletsRequest) GetMetadata() map[string]string {
	if o == nil {
		return nil
	}
	return o.Metadata
}

func (o *ListWalletsRequest) GetPageSize() *int64 {
	if o == nil {
		return nil
	}
	return o.PageSize
}

func (o *ListWalletsRequest) GetCursor() *string {
	if o == nil {
		return nil
	}
	return o.Cursor
}

func (o *ListWalletsRequest) GetExpand() *string {
	if o == nil {
		return nil
	}
	return o.Expand
}

type ListWalletsResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// OK
	ListWalletsResponse *components.ListWalletsResponse
	// OK
	ErrorResponse *components.ErrorResponse
}

func (o *ListWalletsResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *ListWalletsResponse) GetListWalletsResponse() *components.ListWalletsResponse {
	if o == nil {
		return nil
	}
	return o.ListWalletsResponse
}

func (o *ListWalletsResponse) GetErrorResponse() *components.ErrorResponse {
	if o == nil {
		return nil
	}
	return o.ErrorResponse
}
