package wallet

import (
	"testing"

	"github.com/formancehq/go-libs/v5/pkg/types/metadata"
)

func TestWalletCreateRequestFingerprint(t *testing.T) {
	t.Parallel()

	// Distinct requests must not collide via separator bytes embedded in a
	// metadata value: {"a":"b\x00c\x00d"} once joined to the same byte stream as
	// {"a":"b","c":"d"} under a NUL-separated encoding.
	collidingValue := walletCreateRequestFingerprint("w", metadata.Metadata{"a": "b\x00c\x00d"})
	twoFields := walletCreateRequestFingerprint("w", metadata.Metadata{"a": "b", "c": "d"})
	if collidingValue == twoFields {
		t.Fatalf("distinct requests produced the same fingerprint: %s", collidingValue)
	}

	// An identical request is stable regardless of metadata insertion order.
	if walletCreateRequestFingerprint("w", metadata.Metadata{"a": "1", "b": "2"}) !=
		walletCreateRequestFingerprint("w", metadata.Metadata{"b": "2", "a": "1"}) {
		t.Fatal("fingerprint is not stable for an identical request")
	}

	// The name is part of the fingerprint.
	if walletCreateRequestFingerprint("w", nil) == walletCreateRequestFingerprint("x", nil) {
		t.Fatal("different names produced the same fingerprint")
	}
}
