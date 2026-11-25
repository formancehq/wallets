package wallet

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

type v3Segment struct {
	Type  string `json:"type"`
	Regex string `json:"regex"`
}

type v3AccountType struct {
	Name         string               `json:"name"`
	Pattern      string               `json:"pattern"`
	Persistence  string               `json:"persistence"`
	SegmentTypes map[string]v3Segment `json:"segmentTypes"`
}

func (t v3AccountType) matches(other v3AccountType) bool {
	return t.Name == other.Name && t.Pattern == other.Pattern && t.Persistence == other.Persistence &&
		maps.Equal(t.SegmentTypes, other.SegmentTypes)
}

type v3MetadataField struct {
	Target string `json:"targetType"`
	Key    string `json:"key"`
	Type   string `json:"type"`
}

func (f v3MetadataField) index() string {
	return "metadata:TARGET_TYPE_" + strings.ToUpper(f.Target) + ":" + f.Key
}

type v3Provisioning struct {
	verifyOnly      bool
	indexStatusURLs []*url.URL
	accountTypes    map[string]v3AccountType
	fields          []v3MetadataField
	indexedMetadata map[string]struct{}
}

func newV3Provisioning(chart *Chart, mode string, customKeys []string) (v3Provisioning, error) {
	plan := v3Provisioning{indexedMetadata: make(map[string]struct{}, len(customKeys))}
	switch mode {
	case "", "managed":
	case "verify":
		plan.verifyOnly = true
	default:
		return plan, fmt.Errorf("invalid Ledger v3 provisioning %q, expected managed or verify", mode)
	}

	prefix := chart.BasePath().String() + ":"
	id := v3Segment{Type: "regex", Regex: "[0-9a-f]{32}"}
	plan.accountTypes = map[string]v3AccountType{
		"wallets-balance": {
			Name: "wallets-balance", Pattern: prefix + "{walletId}:{balanceName}", Persistence: "NORMAL",
			SegmentTypes: map[string]v3Segment{"walletId": id, "balanceName": {Type: "regex", Regex: "[0-9A-Za-z_]+"}},
		},
		"wallets-hold": {
			Name: "wallets-hold", Pattern: prefix + "holds:{holdId}", Persistence: "NORMAL",
			SegmentTypes: map[string]v3Segment{"holdId": id},
		},
	}
	for _, name := range []string{
		MetadataKeyWalletSpecType, MetadataKeyWalletName, MetadataKeyWalletID, MetadataKeyWalletBalance, MetadataKeyHoldWalletID,
	} {
		plan.fields = append(plan.fields, v3MetadataField{Target: "account", Key: name, Type: "string"})
	}
	for _, name := range []string{MetadataKeyWalletTransaction, v3HoldReplayKey} {
		plan.fields = append(plan.fields, v3MetadataField{Target: "transaction", Key: name, Type: "string"})
	}
	for _, key := range customKeys {
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "[]") {
			return plan, fmt.Errorf("invalid Ledger v3 custom metadata index key %q", key)
		}
		plan.indexedMetadata[MetadataKeyWalletCustomDataPrefix+key] = struct{}{}
	}
	for _, key := range slices.Sorted(maps.Keys(plan.indexedMetadata)) {
		plan.fields = append(plan.fields, v3MetadataField{Target: "account", Key: key, Type: "string"})
	}
	return plan, nil
}

func (p v3Provisioning) indexes() []string {
	indexes := make([]string, 0, len(p.fields)+2)
	for _, field := range p.fields {
		indexes = append(indexes, field.index())
	}
	return append(indexes, "tx_builtin:TX_BUILTIN_INDEX_ADDRESS", "tx_builtin:TX_BUILTIN_INDEX_DESTINATION_ADDRESS")
}

func (l *ledgerV3) EnsureLedgerExists(ctx context.Context, name string) error {
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("invalid Ledger v3 ledger name")
	}
	info, err := l.getLedger(ctx, name)
	if v3HasError(err, http.StatusNotFound, "NOT_FOUND", "LEDGER_NOT_FOUND") && !l.provision.verifyOnly {
		body := struct {
			Mode                   string                   `json:"mode"`
			DefaultEnforcementMode string                   `json:"defaultEnforcementMode"`
			InitialSchema          []v3MetadataField        `json:"initialSchema"`
			AccountTypes           map[string]v3AccountType `json:"accountTypes"`
		}{
			Mode: "NORMAL", DefaultEnforcementMode: "AUDIT",
			InitialSchema: l.provision.fields, AccountTypes: l.provision.accountTypes,
		}
		err = l.request(ctx, http.MethodPost, name, nil, nil, body, "", http.StatusCreated, nil)
		failure, conflict := errors.AsType[*ledgerV3Error](err)
		if err == nil || (conflict && failure.Status == http.StatusConflict) {
			info, err = l.getLedger(ctx, name)
		}
	}
	if err != nil {
		return fmt.Errorf("initialize Ledger v3 ledger %q: %w", name, err)
	}
	if info.Mode != "NORMAL" {
		return fmt.Errorf("expected a NORMAL Ledger v3 ledger, found %q", info.Mode)
	}
	if err := l.reconcileDefinitions(ctx, name, info); err != nil {
		return err
	}

	for _, id := range l.provision.indexes() {
		if err := l.ensureIndex(ctx, name, id); err != nil {
			return fmt.Errorf("Ledger v3 index %q on %q: %w", id, name, err)
		}
	}
	_, err = v3List[v3Account](ctx, l, "accounts", v3Cursor{
		Ledger: name, Query: v3ListQuery{Size: 1, Metadata: map[string]string{MetadataKeyWalletSpecType: PrimaryWallet}},
	})
	if err != nil {
		return fmt.Errorf("check Wallets account index on Ledger v3: %w", err)
	}
	return l.ensureScriptLibrary(ctx, name)
}

func (l *ledgerV3) reconcileDefinitions(ctx context.Context, name string, info v3LedgerInfo) error {
	var missingTypes []v3AccountType
	for _, typeName := range slices.Sorted(maps.Keys(l.provision.accountTypes)) {
		expected := l.provision.accountTypes[typeName]
		actual, present := info.AccountTypes[typeName]
		if present && !expected.matches(actual) {
			return fmt.Errorf("Ledger v3 account type %q differs from the Wallets definition", typeName)
		}
		if !present {
			missingTypes = append(missingTypes, expected)
		}
	}
	var missingFields []v3MetadataField
	for _, field := range l.provision.fields {
		actual, present := info.metadataField(field)
		if present && actual.Type != field.Type {
			return fmt.Errorf("Ledger v3 metadata field %s %q has type %q, want %q", field.Target, field.Key, actual.Type, field.Type)
		}
		if !present {
			missingFields = append(missingFields, field)
		}
	}
	if len(missingTypes)+len(missingFields) == 0 {
		return nil
	}
	if l.provision.verifyOnly {
		return fmt.Errorf("Ledger v3 ledger %q is missing Wallets definitions (account types %v, metadata fields %v)", name, missingTypes, missingFields)
	}
	for _, accountType := range missingTypes {
		err := l.request(ctx, http.MethodPost, name, []string{"account-types"}, nil, accountType, "", http.StatusCreated, nil)
		if failure, ok := errors.AsType[*ledgerV3Error](err); err != nil && (!ok || failure.Status != http.StatusConflict) {
			return fmt.Errorf("provision Ledger v3 account type %q: %w", accountType.Name, err)
		}
	}
	for _, field := range missingFields {
		// RC.3 has no conditional schema write. Avoid rewriting fields another
		// initializer has already added after the original ledger read.
		current, err := l.getLedger(ctx, name)
		if err != nil {
			return fmt.Errorf("read Ledger v3 metadata field %q before declaring it: %w", field.Key, err)
		}
		if actual, present := current.metadataField(field); present {
			if actual.Type != field.Type {
				return fmt.Errorf("Ledger v3 metadata field %s %q has type %q, want %q", field.Target, field.Key, actual.Type, field.Type)
			}
			continue
		}
		err = l.request(ctx, http.MethodPut, name, []string{"metadata-schema", field.Target, field.Key},
			nil, map[string]string{"type": field.Type}, "", http.StatusNoContent, nil)
		if err != nil {
			return fmt.Errorf("declare Ledger v3 metadata field %q: %w", field.Key, err)
		}
	}
	actual, err := l.getLedger(ctx, name)
	if err != nil {
		return fmt.Errorf("verify Ledger v3 definitions: %w", err)
	}
	for _, expected := range missingTypes {
		if !expected.matches(actual.AccountTypes[expected.Name]) {
			return fmt.Errorf("verify Ledger v3 account type %q: unexpected definition", expected.Name)
		}
	}
	for _, expected := range missingFields {
		if field, ok := actual.metadataField(expected); !ok || field.Type != expected.Type {
			return fmt.Errorf("verify Ledger v3 metadata field %q: unexpected definition", expected.Key)
		}
	}
	return nil
}

func (l *ledgerV3) indexReady(ctx context.Context, ledger, id string) (bool, error) {
	var status struct {
		Data struct {
			CurrentVersion uint64 `json:"currentVersion"`
			PendingVersion uint64 `json:"pendingVersion"`
		} `json:"data"`
	}
	err := l.request(ctx, http.MethodGet, ledger, []string{"indexes", id, "status"}, nil, nil, "", http.StatusOK, &status)
	return status.Data.CurrentVersion > 0 && status.Data.PendingVersion == 0, err
}

func (l *ledgerV3) ensureIndex(ctx context.Context, ledger, id string) error {
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	ready, err := l.indexReady(waitCtx, ledger, id)
	if v3HasError(err, http.StatusNotFound, "INDEX_NOT_FOUND", "NOT_FOUND") && !l.provision.verifyOnly {
		err = l.request(waitCtx, http.MethodPost, ledger, []string{"indexes"}, nil,
			map[string]string{"id": id}, "", http.StatusCreated, nil)
		if v3HasError(err, http.StatusConflict, "INDEX_ALREADY_EXISTS") {
			err = nil
		}
	}
	if err != nil {
		return err
	}
	if !ready {
		if err := l.waitForIndex(waitCtx, ledger, id); err != nil {
			return err
		}
	}
	for _, base := range l.provision.indexStatusURLs {
		replica := *l
		replica.base = base
		if err := replica.waitForIndex(waitCtx, ledger, id); err != nil {
			return err
		}
	}
	return nil
}

func (l *ledgerV3) waitForIndex(ctx context.Context, ledger, id string) error {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		ready, err := l.indexReady(ctx, ledger, id)
		if err != nil && !v3HasError(err, http.StatusNotFound, "INDEX_NOT_FOUND", "NOT_FOUND") {
			return fmt.Errorf("check index at %s: %w", l.base, err)
		}
		if ready {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for usable index at %s: %w", l.base, ctx.Err())
		case <-tick.C:
		}
	}
}

func (l *ledgerV3) validateCustomFilters(metadata map[string]string) error {
	for key := range metadata {
		customKey, custom := strings.CutPrefix(key, MetadataKeyWalletCustomDataPrefix)
		if _, indexed := l.provision.indexedMetadata[key]; custom && !indexed {
			return fmt.Errorf("%w: custom metadata filter %q is not indexed", ErrValidation, customKey)
		}
	}
	return nil
}
