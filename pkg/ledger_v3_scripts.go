package wallet

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type walletScript string

const (
	scriptDefaultCredit walletScript = "wallets-credit-world"
	scriptConfirmHold   walletScript = "wallets-confirm-hold"
	walletScriptVersion              = "1.0.0"
)

type v3ScriptReference struct {
	Name    walletScript      `json:"name"`
	Version string            `json:"version"`
	Vars    map[string]string `json:"vars"`
}

func (l *ledgerV3) ensureScriptLibrary(ctx context.Context, ledger string) error {
	scripts := []struct {
		name    walletScript
		content string
	}{
		{scriptDefaultCredit, BuildCreditWalletScript("world")},
		{scriptConfirmHold, BuildConfirmHoldScript(false, "")},
	}
	for _, script := range scripts {
		if err := l.ensureScript(ctx, ledger, script.name, script.content); err != nil {
			return fmt.Errorf("Ledger v3 Numscript %s@%s: %w", script.name, walletScriptVersion, err)
		}
	}
	return nil
}

func (l *ledgerV3) ensureScript(ctx context.Context, ledger string, name walletScript, content string) error {
	path := []string{"numscripts", string(name)}
	params := url.Values{"version": {walletScriptVersion}}
	var result struct {
		Data struct {
			Version string `json:"version"`
			Content string `json:"content"`
		} `json:"data"`
	}
	err := l.request(ctx, http.MethodGet, ledger, path, params, nil, "", http.StatusOK, &result)
	if v3HasError(err, http.StatusNotFound, "NOT_FOUND", "NUMSCRIPT_NOT_FOUND", "NUMSCRIPT_VERSION_NOT_FOUND") && !l.provision.verifyOnly {
		err = l.request(ctx, http.MethodPut, ledger, path, nil,
			map[string]string{"version": walletScriptVersion, "content": content}, "", http.StatusCreated, nil)
		if err == nil || v3HasError(err, http.StatusConflict, "NUMSCRIPT_VERSION_ALREADY_EXISTS") {
			err = l.request(ctx, http.MethodGet, ledger, path, params, nil, "", http.StatusOK, &result)
		}
	}
	if err != nil {
		return err
	}
	if result.Data.Version != walletScriptVersion || result.Data.Content != content {
		return fmt.Errorf("stored version differs from Wallets Numscript")
	}
	return nil
}
