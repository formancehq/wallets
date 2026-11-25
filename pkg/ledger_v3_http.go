package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type ledgerV3 struct {
	cursorKey []byte
	client    *http.Client
	base      *url.URL
}

type ledgerV3Error struct {
	Status  int    `json:"-"`
	Code    string `json:"errorCode"`
	Message string `json:"errorMessage"`
}

func (e *ledgerV3Error) Error() string {
	return fmt.Sprintf("Ledger v3 HTTP %d %s: %s", e.Status, e.Code, e.Message)
}

func (l *ledgerV3) request(ctx context.Context, method, ledger string, path []string, query url.Values, body any,
	key string, status int, result any,
) error {
	parts := append([]string{"v3", ledger}, path...)
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	endpoint := l.base.JoinPath(parts...)
	endpoint.RawQuery = query.Encode()
	return ledgerJSON(ctx, l.client, method, endpoint, body, key, status, result)
}

func ledgerJSON(ctx context.Context, client *http.Client, method string, endpoint *url.URL, body any,
	key string, status int, result any,
) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode Ledger request: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != status {
		failure := &ledgerV3Error{Status: resp.StatusCode}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(failure)
		return failure
	}
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode Ledger response: %w", err)
		}
	}
	return nil
}
