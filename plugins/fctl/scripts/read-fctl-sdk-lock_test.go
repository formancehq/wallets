package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validLock = `{"schemaVersion":1,"modulePath":"github.com/formancehq/fctl-v2-poc/pkg/plugin","repository":"https://github.com/formancehq/fctl-v2-poc.git","commit":"fixture-commit","sdkPath":"pkg/plugin","sdkNarHash":"sha256-example","witPath":"wit/plugin.wit","witSha256":"abcdef"}`

func TestRunPrintsTheLockedFieldsAsOneTabSeparatedRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock.json")
	if err := os.WriteFile(path, []byte(validLock), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{path}, &out); err != nil {
		t.Fatal(err)
	}
	want := "github.com/formancehq/fctl-v2-poc/pkg/plugin\thttps://github.com/formancehq/fctl-v2-poc.git\tfixture-commit\tpkg/plugin\tsha256-example\twit/plugin.wit\tabcdef\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRunRejectsInvalidInvocationAndInput(t *testing.T) {
	if err := run(nil, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("missing arguments error = %v", err)
	}
	if err := run([]string{filepath.Join(t.TempDir(), "missing")}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "open fctl SDK lock:") {
		t.Fatalf("missing lock error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{path}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "decode fctl SDK lock:") {
		t.Fatalf("invalid lock error = %v", err)
	}
}

func TestDecodeLockRejectsTrailingJSON(t *testing.T) {
	input := `{"schemaVersion":1,"modulePath":"m","repository":"r","commit":"c","sdkPath":"pkg/plugin","sdkNarHash":"h","witPath":"wit/plugin.wit","witSha256":"w"} {}`
	if _, err := decodeLock(strings.NewReader(input)); err == nil {
		t.Fatal("decodeLock accepted a second JSON value")
	}
}

func TestDecodeLockRejectsUnsupportedUnsafeAndMissingFields(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{name: "unknown field", input: strings.TrimSuffix(validLock, "}") + `,"unknown":true}`, want: "unknown field"},
		{name: "schema version", input: strings.Replace(validLock, `"schemaVersion":1`, `"schemaVersion":2`, 1), want: "unsupported schema version"},
		{name: "empty field", input: strings.Replace(validLock, `"modulePath":"github.com/formancehq/fctl-v2-poc/pkg/plugin"`, `"modulePath":""`, 1), want: "empty or unsafe field"},
		{name: "unsafe field", input: strings.Replace(validLock, `"repository":"https://github.com/formancehq/fctl-v2-poc.git"`, `"repository":"bad\nrepository"`, 1), want: "empty or unsafe field"},
		{name: "bad sdk path", input: strings.Replace(validLock, `"sdkPath":"pkg/plugin"`, `"sdkPath":"../plugin"`, 1), want: "sdkPath must be"},
		{name: "bad wit path", input: strings.Replace(validLock, `"witPath":"wit/plugin.wit"`, `"witPath":"/plugin.wit"`, 1), want: "witPath must be"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeLock(strings.NewReader(test.input))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("decodeLock error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestValidateRelativePathRejectsNonPortableForms(t *testing.T) {
	for _, path := range []string{
		`../pkg/plugin`,
		`pkg/../plugin`,
		`/pkg/plugin`,
		`C:\\pkg\\plugin`,
		`C:/pkg/plugin`,
		`\\\\server\\share\\plugin`,
		`pkg\\plugin`,
	} {
		t.Run(path, func(t *testing.T) {
			if err := validateRelativePath("sdkPath", path); err == nil {
				t.Fatalf("accepted non-portable path %q", path)
			}
		})
	}
}

func TestValidateRelativePathAcceptsCanonicalSlashPath(t *testing.T) {
	if err := validateRelativePath("sdkPath", "pkg/plugin"); err != nil {
		t.Fatalf("rejected canonical relative path: %v", err)
	}
}
