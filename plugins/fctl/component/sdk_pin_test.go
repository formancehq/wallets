package component

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repositoryRoot is the Wallets checkout root as seen from this package.
const repositoryRoot = "../../.."

// sdkLockPath is the single source of truth for the fctl SDK pin.
const sdkLockPath = "../fctl-sdk.lock.json"

type sdkLock struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	SDKNarHash string `json:"sdkNarHash"`
	WITSHA256  string `json:"witSha256"`
}

func readSDKLock(t *testing.T) sdkLock {
	t.Helper()
	encoded, err := os.ReadFile(sdkLockPath)
	if err != nil {
		t.Fatal(err)
	}
	var lock sdkLock
	if err := json.Unmarshal(encoded, &lock); err != nil {
		t.Fatalf("decode fctl SDK lock: %v", err)
	}
	if lock.Commit == "" || lock.Repository == "" {
		t.Fatalf("fctl SDK lock is incomplete: %#v", lock)
	}
	return lock
}

// commitToken matches a maximal hexadecimal run. recognizedSDKRevision applies
// the stricter length and prose-context rules that distinguish abbreviated Git
// revisions from payload values and SHA-256 digests.
var commitToken = regexp.MustCompile(`[0-9a-fA-F]+`)

// sdkPinMarker recognises the contexts that make a commit an fctl SDK pin
// restatement rather than some other hexadecimal value: a mention of the lock
// file, of the `fctlSDKRevision` identifier, or of the canonical repository.
// Comparison runs on the context with every non-alphanumeric byte removed, so
// `fctl-sdk.lock.json`, `fctlSDKRevision` and `fctl-v2-poc` all match without
// depending on punctuation or line wrapping.
var sdkPinMarkers = []string{"fctlsdk", "fctlv2poc"}

func normalizeContext(value string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(value) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func isSDKPinContext(context string) bool {
	normalized := normalizeContext(context)
	for _, marker := range sdkPinMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func recognizedSDKRevision(context, token string) bool {
	if len(token) == 40 {
		return isSDKPinContext(context)
	}
	if len(token) < 7 || len(token) > 12 {
		return false
	}
	normalized := normalizeContext(context)
	token = strings.ToLower(token)
	return strings.Contains(normalized, "fctl"+token) ||
		strings.Contains(normalized, "fctlsdkcommit"+token)
}

func sdkRevisionMatchesLock(candidate, locked string) bool {
	candidate = strings.ToLower(candidate)
	locked = strings.ToLower(locked)
	if len(candidate) < len(locked) {
		return len(candidate) >= 7 && strings.HasPrefix(locked, candidate)
	}
	return candidate == locked
}

func TestRecognizedSDKRevisionAcceptsPinnedAbbreviationsWithoutDigestFalsePositives(t *testing.T) {
	tests := []struct {
		name, context, token string
		want                 bool
	}{
		{name: "direct fctl abbreviation", context: "fctl `e9b1395f` maps product errors", token: "e9b1395f", want: true},
		{name: "named SDK commit abbreviation", context: "land the pinned fctl SDK commit `e9b1395f` on main", token: "e9b1395f", want: true},
		{name: "full lock restatement", context: "fctl-sdk.lock.json is sealed at e9b1395f46f3100b381dbe00f5213de28e6df0e1", token: "e9b1395f46f3100b381dbe00f5213de28e6df0e1", want: true},
		{name: "unrelated short hex", context: "fctl output fixture uses `deadbeef` as payload data", token: "deadbeef"},
		{name: "sha256 prefix", context: "the fctl SDK NAR digest begins DnTiEFya3R9K", token: "d71e9c0a"},
		{name: "too short", context: "fctl `e9b139`", token: "e9b139"},
		{name: "too long abbreviation", context: "fctl `e9b1395f46f31`", token: "e9b1395f46f31"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := recognizedSDKRevision(test.context, test.token); got != test.want {
				t.Fatalf("recognizedSDKRevision(%q, %q) = %v, want %v", test.context, test.token, got, test.want)
			}
		})
	}
}

func TestSDKRevisionMatchRejectsTheRetiredAbbreviatedPin(t *testing.T) {
	const locked = "e9b1395f46f3100b381dbe00f5213de28e6df0e1"
	if !sdkRevisionMatchesLock("e9b1395f", locked) {
		t.Fatal("current abbreviated revision does not match the lock")
	}
	if sdkRevisionMatchesLock("545521bf", locked) {
		t.Fatal("retired abbreviated revision matches the lock")
	}
}

// restatementContexts returns the text a reader would treat as each recognized
// full or abbreviated SDK revision's statement. Markdown uses the enclosing
// block — a table row on its own, otherwise the paragraph — so a revision stated
// in wrapped prose is still bound to the sentence that names the lock. Every
// other format uses the declaration line itself, which keeps a negative fixture
// in a contract script from being read as a statement of the pin.
func restatementContexts(path, content string) map[string][]string {
	lines := strings.Split(content, "\n")
	markdown := strings.EqualFold(filepath.Ext(path), ".md")
	found := make(map[string][]string)
	for index, line := range lines {
		for _, token := range commitToken.FindAllString(line, -1) {
			context := line
			if markdown {
				context = markdownBlock(lines, index)
			}
			if recognizedSDKRevision(context, token) {
				found[token] = append(found[token], context)
			}
		}
	}
	return found
}

func markdownBlock(lines []string, index int) string {
	isRow := func(line string) bool { return strings.HasPrefix(strings.TrimSpace(line), "|") }
	if isRow(lines[index]) {
		return lines[index]
	}
	start := index
	for start > 0 && strings.TrimSpace(lines[start-1]) != "" && !isRow(lines[start-1]) {
		start--
	}
	end := index
	for end+1 < len(lines) && strings.TrimSpace(lines[end+1]) != "" && !isRow(lines[end+1]) {
		end++
	}
	return strings.Join(lines[start:end+1], "\n")
}

func trackedRepositoryFiles(t *testing.T, root string) []string {
	t.Helper()
	// Only versioned sources can authoritatively restate the pin. This excludes
	// SDK materializations, build products, and agent scratch files by design.
	output, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		t.Fatalf("list tracked repository files: %v", err)
	}
	encoded := strings.TrimSuffix(string(output), "\x00")
	if encoded == "" {
		return nil
	}
	return strings.Split(encoded, "\x00")
}

func sweepRestatements(t *testing.T) map[string][]string {
	t.Helper()
	root, err := filepath.Abs(repositoryRoot)
	if err != nil {
		t.Fatal(err)
	}
	restatements := make(map[string][]string)
	for _, relative := range trackedRepositoryFiles(t, root) {
		path := filepath.Join(root, filepath.FromSlash(relative))
		info, err := os.Stat(path)
		if err != nil || info.Size() > 4<<20 {
			continue
		}
		encoded, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read tracked file %s: %v", relative, err)
		}
		if strings.IndexByte(string(encoded), 0) >= 0 {
			continue
		}
		for token, contexts := range restatementContexts(relative, string(encoded)) {
			for range contexts {
				restatements[filepath.ToSlash(relative)] = append(restatements[filepath.ToSlash(relative)], token)
			}
		}
	}
	return restatements
}

// TestEveryFctlSDKRevisionRestatementMatchesTheLock binds every place in the
// repository that restates the pinned fctl SDK revision to fctl-sdk.lock.json.
// The sweep is content-driven rather than a list of line numbers, so a repin
// that updates the gated files and leaves a document advertising the previous
// revision fails here, and a new copy added anywhere is bound the moment it
// appears.
//
// The lock itself is the statement everything else is compared against, and
// scripts/test-fctl-sdk-contract.sh deliberately states wrong revisions as
// negative fixtures; both are excluded from the sweep and bound exactly by
// TestSDKPinIsIdenticalAcrossTheLockToolchainAndContractScript instead.
func TestEveryFctlSDKRevisionRestatementMatchesTheLock(t *testing.T) {
	lock := readSDKLock(t)
	restatements := sweepRestatements(t)

	for _, required := range []string{
		"plugins/fctl/README.md",
		"plugins/fctl/docs/command-inventory.md",
		"nix/fctl-component-tools.nix",
		"plugins/fctl/component/authoring_contract_test.go",
	} {
		if len(restatements[required]) == 0 {
			t.Errorf("%s states no fctl SDK revision, so the sweep does not bind it", required)
		}
	}
	abbreviations := 0
	for _, revision := range restatements["plugins/fctl/docs/command-inventory.md"] {
		if len(revision) >= 7 && len(revision) <= 12 {
			abbreviations++
		}
	}
	if abbreviations != 2 {
		t.Errorf("plugins/fctl/docs/command-inventory.md binds %d abbreviated fctl SDK revisions, want 2", abbreviations)
	}

	for path, commits := range restatements {
		for _, commit := range commits {
			if !sdkRevisionMatchesLock(commit, lock.Commit) {
				t.Errorf("%s states fctl SDK revision %q, want the locked %q", path, commit, lock.Commit)
			}
		}
	}
}

// goTestInvocation matches the start of a `go test` command inside a Justfile
// recipe line. A single recipe line can chain several invocations, and only the
// segment belonging to one invocation may be searched for its own flags.
var goTestInvocation = regexp.MustCompile(`\bgo test\b`)

// goTestInvocations splits a recipe line into one segment per `go test`
// command, so a `-count=1` on a later invocation cannot vouch for an earlier
// one on the same line.
func goTestInvocations(line string) []string {
	starts := goTestInvocation.FindAllStringIndex(line, -1)
	segments := make([]string, 0, len(starts))
	for index, start := range starts {
		end := len(line)
		if index+1 < len(starts) {
			end = starts[index+1][0]
		}
		segments = append(segments, line[start[0]:end])
	}
	return segments
}

// TestPluginJustGatesRunUncached requires every `go test` the plugin's real
// Just gates run to carry -count=1. The component contract tests read
// flake.nix, nix/fctl-component-tools.nix and the repository Justfiles, all of
// which live outside the plugin Go module; Go's test cache does not invalidate
// on them, so without -count=1 a Nix-only or Justfile-only drift returns a
// stale PASS on a warm developer cache.
func TestPluginJustGatesRunUncached(t *testing.T) {
	gates := map[string]string{
		"plugins/fctl/Justfile": "../Justfile",
		"Justfile":              filepath.Join(repositoryRoot, "Justfile"),
	}
	checked := 0
	for name, path := range gates {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		pluginJustfile := name == "plugins/fctl/Justfile"
		for _, line := range strings.Split(string(content), "\n") {
			if !pluginJustfile && !strings.Contains(line, "plugins/fctl") {
				continue
			}
			for _, invocation := range goTestInvocations(line) {
				checked++
				if !strings.Contains(invocation, "-count=1") {
					t.Errorf("%s runs the plugin test suite without -count=1, so an out-of-module drift can return a cached PASS:\n\t%s", name, strings.TrimSpace(invocation))
				}
			}
		}
	}
	if checked < 3 {
		t.Fatalf("found %d plugin `go test` invocations across the Just gates, want at least 3", checked)
	}
}
