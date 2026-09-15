package core

import (
	"context"
	"errors"
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

func TestInputHelpersRejectAmbiguousValues(t *testing.T) {
	if _, err := parseMetadata([]string{"missing-separator"}); err == nil {
		t.Fatal("parseMetadata() accepted missing separator")
	}
	if _, err := parseMetadata([]string{"a=1", "a=2"}); err == nil {
		t.Fatal("parseMetadata() accepted duplicate key")
	}
	if _, err := parseNonNegativeAmount("-1"); err == nil {
		t.Fatal("parseNonNegativeAmount() accepted negative amount")
	}
	if _, err := parseNonNegativeAmount("nope"); err == nil {
		t.Fatal("parseNonNegativeAmount() accepted malformed amount")
	}
	if value, err := optionalNonNegativeAmount(""); err != nil || value != nil {
		t.Fatalf("optionalNonNegativeAmount(empty) = %v, %v", value, err)
	}
	if value, err := optionalSignedBigInt("-2147483649"); err != nil || value == nil || value.String() != "-2147483649" {
		t.Fatalf("optionalSignedBigInt(negative) = %v, %v", value, err)
	}
	if value, err := optionalBool(""); err != nil || value != nil {
		t.Fatalf("optionalBool(empty) = %v, %v", value, err)
	}
	if value, err := optionalBool("true"); err != nil || value == nil || !*value {
		t.Fatalf("optionalBool(true) = %v, %v", value, err)
	}
	if _, err := optionalBool("sometimes"); err == nil {
		t.Fatal("optionalBool() accepted malformed boolean")
	}
	if value, err := optionalTime(""); err != nil || value != nil {
		t.Fatalf("optionalTime(empty) = %v, %v", value, err)
	}
	if value, err := optionalTime("2026-09-11T10:00:00Z"); err != nil || value == nil {
		t.Fatalf("optionalTime(valid) = %v, %v", value, err)
	}
	if _, err := optionalTime("tomorrow"); err == nil {
		t.Fatal("optionalTime() accepted malformed date")
	}
}

func TestSubjectParserSupportsOnlyUnambiguousCurrentForms(t *testing.T) {
	if subject, err := parseOptionalSubject(""); err != nil || subject != nil {
		t.Fatalf("empty subject = %#v, %v", subject, err)
	}
	account, err := parseOptionalSubject("account=users:42")
	if err != nil || account == nil || account.LedgerAccountSubject == nil || account.LedgerAccountSubject.Identifier != "users:42" {
		t.Fatalf("account subject = %#v, %v", account, err)
	}
	wallet, err := parseOptionalSubject("wallet=id:w2/savings")
	if err != nil || wallet == nil || wallet.WalletSubject == nil || wallet.WalletSubject.Identifier != "w2" || wallet.WalletSubject.Balance == nil || *wallet.WalletSubject.Balance != "savings" {
		t.Fatalf("wallet subject = %#v, %v", wallet, err)
	}
	for _, invalid := range []string{"missing", "wallet=name:alice", "wallet=id:", "other=value"} {
		if _, err := parseOptionalSubject(invalid); err == nil {
			t.Errorf("parseOptionalSubject(%q) error = nil", invalid)
		}
	}
	if values, err := parseSubjects(nil); err != nil || values != nil {
		t.Fatalf("parseSubjects(nil) = %#v, %v", values, err)
	}
	if values, err := parseSubjects([]string{"account=world", "wallet=id:w2"}); err != nil || len(values) != 2 {
		t.Fatalf("parseSubjects(valid) = %#v, %v", values, err)
	}
	if _, err := parseSubjects([]string{"other=value"}); err == nil {
		t.Fatal("parseSubjects() accepted invalid item")
	}
}

func TestFlagCollectionAndCommandLookupFailClosed(t *testing.T) {
	command, ok := commandByID("wallets.v2.show")
	if !ok {
		t.Fatal("show command missing")
	}
	if _, ok := commandByID("wallets.v2.missing"); ok {
		t.Fatal("unknown command found")
	}
	if _, err := collectFlags(command, []sdk.FlagOccurrence{{Name: "other", Value: "x"}}); err == nil {
		t.Fatal("collectFlags() accepted unknown flag")
	}
	if _, err := collectFlags(command, []sdk.FlagOccurrence{{Name: "id", Value: "w1"}, {Name: "id", Value: "w2"}}); err == nil {
		t.Fatal("collectFlags() accepted repeated scalar")
	}
	if _, err := collectFlags(command, nil); err == nil {
		t.Fatal("collectFlags() accepted missing required flag")
	}
}

func TestPaginationGuardsCyclesAndBudgets(t *testing.T) {
	if err := validateCollectionBudget([]int{1, 2}, sdk.ContinuationControl{MaxItems: 1, MaxBytes: 100}); err == nil {
		t.Fatal("item budget accepted overflow")
	}
	if err := validateCollectionBudget([]string{"long"}, sdk.ContinuationControl{MaxItems: 2, MaxBytes: 2}); err == nil {
		t.Fatal("byte budget accepted overflow")
	}
	if _, _, err := collectPages(sdk.ContinuationControl{Mode: sdk.ContinuationAllPages, MaxPages: 1, MaxItems: 10, MaxBytes: 100}, func(*string) ([]int, *string, *bool, error) {
		next, more := "next", true
		return []int{1}, &next, &more, nil
	}); err == nil {
		t.Fatal("page budget accepted overflow")
	}
	call := 0
	if _, _, err := collectPages(sdk.ContinuationControl{Mode: sdk.ContinuationAllPages, MaxPages: 3, MaxItems: 10, MaxBytes: 100}, func(*string) ([]int, *string, *bool, error) {
		call++
		next, more := "same", true
		return []int{call}, &next, &more, nil
	}); err == nil {
		t.Fatal("cursor cycle was accepted")
	}
	more := true
	if _, _, err := collectPages(sdk.ContinuationControl{Mode: sdk.ContinuationAllPages, MaxPages: 2, MaxItems: 10, MaxBytes: 100}, func(*string) ([]int, *string, *bool, error) {
		return nil, nil, &more, nil
	}); err == nil {
		t.Fatal("hasMore without cursor was accepted")
	}
	sentinel := errors.New("fetch")
	if _, _, err := collectPages(sdk.SinglePageContinuationControl(), func(*string) ([]int, *string, *bool, error) {
		return nil, nil, nil, sentinel
	}); !errors.Is(err, sentinel) {
		t.Fatalf("fetch error = %v", err)
	}
}

func TestResultEncodingAndStructuredFailures(t *testing.T) {
	host := sdk.NewMemoryHost(nil)
	if err := emitJSON(host, "x", sdk.ResultObject, make(chan int), nil); err == nil {
		t.Fatal("emitJSON() accepted unencodable result")
	}
	for _, err := range []error{invalidArgument("bad"), descriptorInvalid("bad"), budgetExhausted("bad")} {
		var failure sdk.Failure
		if !errors.As(err, &failure) || failure.Code == "" {
			t.Fatalf("structured failure = %#v", err)
		}
	}
	if err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{CommandID: "missing"}, host); err == nil {
		t.Fatal("Execute() accepted unknown command")
	}
}
