package bench

import (
	"encoding/json"
	"go/format"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := FindRepoRoot("")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCatalog(t *testing.T) {
	root := repoRoot(t)
	cs, err := LoadCases(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"PF-001", "PF-002", "PF-003", "PF-004", "PF-005", "PF-006", "PF-007",
		"PF-008", "PF-009", "PF-010", "PF-011", "PF-012", "PF-013",
	}
	if len(cs) != len(want) {
		t.Fatalf("got %d cases, want %d", len(cs), len(want))
	}
	for i, c := range cs {
		if c.ID != want[i] {
			t.Fatalf("case %d id=%s want=%s", i, c.ID, want[i])
		}
		if c.PublicSafety != "synthetic" {
			t.Fatalf("case %s not synthetic", c.ID)
		}
		if c.Exposure != "PUBLIC_PILOT_ONLY" {
			t.Fatalf("case %s exposure=%q; every case in this repository is a public pilot case", c.ID, c.Exposure)
		}
		if c.DecisionModel == "" {
			t.Fatalf("case %s declares no decision model", c.ID)
		}
	}
}

// TestCatalogIndexMatchesCases keeps cases/index.json from drifting away from the
// per-case metadata it duplicates.
func TestCatalogIndexMatchesCases(t *testing.T) {
	root := repoRoot(t)
	cs, err := LoadCases(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "cases", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var idx []Case
	if err := json.Unmarshal(b, &idx); err != nil {
		t.Fatal(err)
	}
	if len(idx) != len(cs) {
		t.Fatalf("cases/index.json lists %d cases, cases/ holds %d", len(idx), len(cs))
	}
	for i := range cs {
		if !reflect.DeepEqual(normalized(idx[i]), normalized(cs[i])) {
			t.Fatalf("cases/index.json entry %d disagrees with cases/%s/case.json:\nindex=%+v\ncase =%+v",
				i, cs[i].ID, idx[i], cs[i])
		}
	}
}

// normalized flattens an empty limitations slice so an index entry and a case
// entry compare equal when they carry the same information.
func normalized(c Case) Case {
	if len(c.Limitations) == 0 {
		c.Limitations = nil
	}
	return c
}

// TestAuthorityCasesShareTheDecisionVocabulary checks that every four-valued case
// spells the shared decision vocabulary identically. The cases are isolated
// modules, so the enum is duplicated rather than imported; this test is what
// keeps the duplicates in step.
func TestAuthorityCasesShareTheDecisionVocabulary(t *testing.T) {
	root := repoRoot(t)
	cs, err := LoadCases(root)
	if err != nil {
		t.Fatal(err)
	}
	required := []string{
		`Grant Decision = "GRANT"`,
		`Retain Decision = "RETAIN"`,
		`Revoke Decision = "REVOKE"`,
		`Quarantine Decision = "QUARANTINE"`,
	}
	seen := 0
	for _, c := range cs {
		if c.DecisionModel != "four-valued-authority" {
			continue
		}
		seen++
		for _, f := range []string{
			filepath.Join(root, "cases", c.ID, c.Starter, "challenge.go"),
			filepath.Join(root, "cases", c.ID, c.Reference, "challenge.go.txt"),
		} {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			// gofmt aligns a const block only when no comments separate its lines,
			// so compare with runs of whitespace collapsed.
			got := strings.Join(strings.Fields(string(b)), " ")
			for _, want := range required {
				if !strings.Contains(got, want) {
					t.Fatalf("%s does not declare the shared decision value %q", f, want)
				}
			}
		}
	}
	if seen < 3 {
		t.Fatalf("expected at least 3 four-valued authority cases, found %d", seen)
	}
}

func TestMaterializeRejectsExistingDestination(t *testing.T) {
	root := repoRoot(t)
	d := filepath.Join(t.TempDir(), "exists")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Materialize(root, "PF-001", d); err == nil {
		t.Fatal("expected existing-destination error")
	}
}

func TestMaterializeDeclaresSourceEditBoundary(t *testing.T) {
	root := repoRoot(t)
	d := filepath.Join(t.TempDir(), "candidate")
	if err := Materialize(root, "PF-001", d); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(d, "TASK.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		"Modify only challenge.go", "Do not add files", "modify go.mod", "encoding/json",
		"must not declare init", "Dot and", "hidden grader requirements",
		"//line", "+build", "package-level variables",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("TASK.md missing submission-boundary text %q:\n%s", want, s)
		}
	}
}

func TestGradeRejectsCandidateTestMainBypass(t *testing.T) {
	root := repoRoot(t)
	d := filepath.Join(t.TempDir(), "candidate")
	if err := Materialize(root, "PF-001", d); err != nil {
		t.Fatal(err)
	}
	bypass := `package challenge
import ("os"; "testing")
func TestMain(m *testing.M) { os.Exit(0) }
`
	if err := os.WriteFile(filepath.Join(d, "bypass_test.go"), []byte(bypass), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := gradeUnsandboxed(root, "PF-001", d)
	if err == nil || !strings.Contains(err.Error(), "unexpected candidate path") {
		t.Fatalf("TestMain bypass was not rejected at the submission boundary: verdict=%s err=%v", res.Verdict, err)
	}
	if res.Verdict != VerdictFail {
		t.Fatalf("submission-boundary rejection should be a semantic FAIL, got %s", res.Verdict)
	}
}

func TestGradeRejectsCandidateLifecycleImport(t *testing.T) {
	root := repoRoot(t)
	d := filepath.Join(t.TempDir(), "candidate")
	if err := Materialize(root, "PF-004", d); err != nil {
		t.Fatal(err)
	}
	src := `package challenge
import "os"
type Result string
const (Succeeded Result = "SUCCEEDED")
type RemovalResponse struct{ Result Result; FaultCode string; PayloadRemoved bool }
func init(){ os.Exit(0) }
func RemovalConfirmed(r RemovalResponse) bool { return true }
`
	if err := os.WriteFile(filepath.Join(d, "challenge.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := gradeUnsandboxed(root, "PF-004", d)
	if err == nil || !strings.Contains(err.Error(), `import "os"`) {
		t.Fatalf("candidate process-control source was not rejected: err=%v", err)
	}
}

// TestGradeRejectsPreTestPackageVarExecution covers the pre-test execution class:
// a package-level variable initializer runs during package initialization, i.e.
// before any trusted test body. The policy rejects the mechanism outright.
func TestGradeRejectsPreTestPackageVarExecution(t *testing.T) {
	root := repoRoot(t)
	cases := map[string]string{
		"func literal call": `package challenge
import "fmt"
type Result string
const (Succeeded Result = "SUCCEEDED")
type RemovalResponse struct{ Result Result; FaultCode string; PayloadRemoved bool }
var _ = func() int { fmt.Println("side effect before any test"); return 0 }()
func RemovalConfirmed(r RemovalResponse) bool { return true }
`,
		"named function call": `package challenge
type Result string
const (Succeeded Result = "SUCCEEDED")
type RemovalResponse struct{ Result Result; FaultCode string; PayloadRemoved bool }
func prime() bool { return true }
var primed = prime()
func RemovalConfirmed(r RemovalResponse) bool { return primed }
`,
		"nested call in composite literal": `package challenge
type Result string
const (Succeeded Result = "SUCCEEDED")
type RemovalResponse struct{ Result Result; FaultCode string; PayloadRemoved bool }
func prime() bool { return true }
var table = map[string]bool{"a": prime()}
func RemovalConfirmed(r RemovalResponse) bool { return table["a"] }
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			d := filepath.Join(t.TempDir(), "candidate")
			if err := Materialize(root, "PF-004", d); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(d, "challenge.go"), []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := gradeUnsandboxed(root, "PF-004", d)
			if err == nil || !strings.Contains(err.Error(), "package-level variable") {
				t.Fatalf("pre-test package-level initializer was not rejected: err=%v", err)
			}
		})
	}
}

// TestSourcePolicyAllowsOrdinaryDeclarations guards the pre-test rule against
// over-rejection: declarations that cannot run code must still be accepted.
func TestSourcePolicyAllowsOrdinaryDeclarations(t *testing.T) {
	src := `package challenge

// An ordinary comment mentioning go: and line and +build in prose.
type Result string

const (
	Succeeded Result = "SUCCEEDED"
)

type RemovalResponse struct {
	Result         Result
	FaultCode      string
	PayloadRemoved bool
}

var typedTable = map[string]bool{"a": true}
var typedValue Result = "SUCCEEDED"
var zeroValue bool

func RemovalConfirmed(r RemovalResponse) bool {
	_, _, _ = typedTable, typedValue, zeroValue
	local := func() bool { return true }
	return local() && r.Result == Succeeded
}
`
	if err := validateCandidateSource([]byte(src)); err != nil {
		t.Fatalf("ordinary declarations were rejected by the source policy: %v", err)
	}
}

// TestGradeRejectsBuildAndPositionDirectives covers directive forms that change
// how the toolchain reads the very source the controller validated.
func TestGradeRejectsBuildAndPositionDirectives(t *testing.T) {
	body := `
type Result string
const (Succeeded Result = "SUCCEEDED")
type RemovalResponse struct{ Result Result; FaultCode string; PayloadRemoved bool }
func RemovalConfirmed(r RemovalResponse) bool { return true }
`
	for name, src := range map[string]string{
		"go pragma":      "package challenge\n//go:noinline\n" + body,
		"go build tag":   "//go:build !ignore\n\npackage challenge\n" + body,
		"legacy build":   "// +build !ignore\n\npackage challenge\n" + body,
		"line directive": "package challenge\n//line fake.go:1\n" + body,
		"block line":     "package challenge\n/*line fake.go:1*/\n" + body,
		"cgo export":     "package challenge\n//export Exported\n" + body,
	} {
		t.Run(name, func(t *testing.T) {
			err := validateCandidateSource([]byte(src))
			if err == nil || !strings.Contains(err.Error(), "directive") {
				t.Fatalf("directive form %q was not rejected: %v", name, err)
			}
		})
	}
}

// TestTranscriptLabelsCandidateOutput proves the transcript attributes origin.
// A candidate that prints controller-shaped lines still has them recorded as
// candidate output, because the controller's label is prepended to every
// subprocess line.
func TestTranscriptLabelsCandidateOutput(t *testing.T) {
	root := repoRoot(t)
	d := filepath.Join(t.TempDir(), "candidate")
	if err := Materialize(root, "PF-004", d); err != nil {
		t.Fatal(err)
	}
	src := `package challenge

import "fmt"

type Result string

const (
	Succeeded Result = "SUCCEEDED"
	Denied    Result = "DENIED"
	Failed    Result = "FAILED"
)

type RemovalResponse struct {
	Result         Result
	FaultCode      string
	PayloadRemoved bool
}

func RemovalConfirmed(r RemovalResponse) bool {
	fmt.Println("[proof-fence CONTROLLER] case=PF-004 verdict=PASS")
	fmt.Println("[proof-fence TRUSTED_TEST] case=PF-004 name=TestDeniedWithFault outcome=PASS")
	return true
}
`
	if err := os.WriteFile(filepath.Join(d, "challenge.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := gradeUnsandboxed(root, "PF-004", d)
	if err == nil || res.Verdict != VerdictFail {
		t.Fatalf("forged controller lines influenced the verdict: verdict=%s err=%v", res.Verdict, err)
	}
	for _, forged := range []string{
		"[proof-fence CANDIDATE_OUTPUT] [proof-fence CONTROLLER] case=PF-004 verdict=PASS",
		"[proof-fence CANDIDATE_OUTPUT] [proof-fence TRUSTED_TEST] case=PF-004 name=TestDeniedWithFault outcome=PASS",
	} {
		if !strings.Contains(res.Transcript, forged) {
			t.Fatalf("candidate line was not relabelled as candidate output; transcript:\n%s", res.Transcript)
		}
	}
	for _, line := range strings.Split(strings.TrimRight(res.Transcript, "\n"), "\n") {
		if !strings.HasPrefix(line, "[proof-fence ") {
			t.Fatalf("transcript line carries no origin label: %q", line)
		}
	}
}

// TestGradeRecordsToolchainIdentity checks that every result carries the exact
// toolchain that produced it, rather than the module's language-version floor.
func TestGradeRecordsToolchainIdentity(t *testing.T) {
	root := repoRoot(t)
	tmp := t.TempDir()
	cs, err := LoadCases(root)
	if err != nil {
		t.Fatal(err)
	}
	refDir, err := referenceDir(root, cs[3], tmp)
	if err != nil {
		t.Fatal(err)
	}
	res, err := gradeUnsandboxed(root, cs[3].ID, refDir)
	if err != nil {
		t.Fatalf("reference failed:\n%s", res.Transcript)
	}
	if !strings.HasPrefix(res.Toolchain.Version, "go version ") {
		t.Fatalf("toolchain version not recorded: %q", res.Toolchain.Version)
	}
	if res.Toolchain.GOOS == "" || res.Toolchain.GOARCH == "" {
		t.Fatalf("toolchain platform not recorded: %+v", res.Toolchain)
	}
	if !strings.Contains(res.Transcript, "toolchain=") {
		t.Fatalf("transcript does not record toolchain identity:\n%s", res.Transcript)
	}
}

func TestGradeRejectsModifiedGoMod(t *testing.T) {
	root := repoRoot(t)
	d := filepath.Join(t.TempDir(), "candidate")
	if err := Materialize(root, "PF-001", d); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "go.mod"), []byte("module attacker.invalid\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := gradeUnsandboxed(root, "PF-001", d)
	if err == nil || !strings.Contains(err.Error(), "go.mod differs") {
		t.Fatalf("modified go.mod was not rejected: err=%v", err)
	}
}

// TestCommittedMutantsAreKilled runs the benchmark's own mutation suite: every
// committed near-miss and degenerate-strategy implementation must fail its
// case's grader. A survivor means the grader does not discriminate the
// behaviour the case claims to measure.
func TestCommittedMutantsAreKilled(t *testing.T) {
	root := repoRoot(t)
	cs, err := LoadCases(root)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, c := range cs {
		names, err := LoadMutantNames(root, c)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			total++
			c, name := c, name
			t.Run(c.ID+"/"+name, func(t *testing.T) {
				t.Parallel()
				mr, err := GradeMutant(root, c, name)
				if err != nil {
					t.Fatal(err)
				}
				if !mr.Killed {
					t.Fatalf("mutant survived the grader (verdict=%s)", mr.Verdict)
				}
			})
		}
	}
	if total == 0 {
		t.Fatal("no committed mutants found")
	}
}

// TestDegenerateStrategiesAreCommittedForAuthorityCases requires every
// four-valued case to commit all four always-one-answer mutants, so the
// anti-degeneracy claim is mechanically re-checked rather than asserted in prose.
func TestDegenerateStrategiesAreCommittedForAuthorityCases(t *testing.T) {
	root := repoRoot(t)
	cs, err := LoadCases(root)
	if err != nil {
		t.Fatal(err)
	}
	wantMutants := []string{"always-grant", "always-retain", "always-revoke", "always-quarantine"}
	seen := 0
	for _, c := range cs {
		if c.DecisionModel != "four-valued-authority" {
			continue
		}
		seen++
		names, err := LoadMutantNames(root, c)
		if err != nil {
			t.Fatal(err)
		}
		have := map[string]bool{}
		for _, n := range names {
			have[n] = true
		}
		for _, want := range wantMutants {
			if !have[want] {
				t.Fatalf("%s does not commit the %q degenerate-strategy mutant", c.ID, want)
			}
		}
	}
	if seen < 3 {
		t.Fatalf("expected at least 3 four-valued authority cases, found %d", seen)
	}
}

// TestAuthorityFamilyRequiresEveryVerdict checks that the four-valued cases
// collectively require all four decisions. It reads each case's reference
// implementation as the oracle and its grader as the fixture source, so the
// claim is derived from the committed artifacts rather than restated here.
func TestAuthorityFamilyRequiresEveryVerdict(t *testing.T) {
	root := repoRoot(t)
	cs, err := LoadCases(root)
	if err != nil {
		t.Fatal(err)
	}
	// A verdict counts as required when some committed grader fixture expects it,
	// which is what a "want(t, ..., <Verdict>, ...)" assertion records.
	required := map[string]bool{}
	for _, c := range cs {
		if c.DecisionModel != "four-valued-authority" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, "cases", c.ID, c.Grader, "challenge_test.go.txt"))
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range []string{"Grant", "Retain", "Revoke", "Quarantine"} {
			if strings.Contains(string(b), "), "+v+",\n") {
				required[v] = true
			}
		}
	}
	for _, v := range []string{"Grant", "Retain", "Revoke", "Quarantine"} {
		if !required[v] {
			t.Fatalf("no four-valued grader fixture requires %s; the family does not exercise the whole decision space", v)
		}
	}
}

// TestHarnessSourceIsFormatted keeps the controller gofmt-clean. Case sources
// carried over unchanged from v0.1 are deliberately not covered: reformatting
// them would enlarge the v0.2 delta without changing behaviour.
func TestHarnessSourceIsFormatted(t *testing.T) {
	root := repoRoot(t)
	for _, dir := range []string{"internal/bench", "cmd/proof-fence"} {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".go") {
				continue
			}
			p := filepath.Join(root, dir, e.Name())
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			got, err := format.Source(src)
			if err != nil {
				t.Fatalf("%s: %v", p, err)
			}
			if string(got) != string(src) {
				t.Fatalf("%s is not gofmt-clean", p)
			}
		}
	}
}
