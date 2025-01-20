// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type GetHoldsResponseCursor struct {
	PageSize int64   `json:"pageSize"`
	HasMore  *bool   `json:"hasMore,omitempty"`
	Previous *string `json:"previous,omitempty"`
	Next     *string `json:"next,omitempty"`
	Data     []Hold  `json:"data"`
}

func (o *GetHoldsResponseCursor) GetPageSize() int64 {
	if o == nil {
		return 0
	}
	return o.PageSize
}

func (o *GetHoldsResponseCursor) GetHasMore() *bool {
	if o == nil {
		return nil
	}
	return o.HasMore
}

func (o *GetHoldsResponseCursor) GetPrevious() *string {
	if o == nil {
		return nil
	}
	return o.Previous
}

func (o *GetHoldsResponseCursor) GetNext() *string {
	if o == nil {
		return nil
	}
	return o.Next
}

func (o *GetHoldsResponseCursor) GetData() []Hold {
	if o == nil {
		return []Hold{}
	}
	return o.Data
}

type GetHoldsResponse struct {
	Cursor GetHoldsResponseCursor `json:"cursor"`
}

func (o *GetHoldsResponse) GetCursor() GetHoldsResponseCursor {
	if o == nil {
		return GetHoldsResponseCursor{}
	}
	return o.Cursor
}
