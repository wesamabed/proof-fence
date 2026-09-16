package bench

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Case is the on-disk description of one ProofFence challenge.
type Case struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Category      string   `json:"category"`
	DecisionModel string   `json:"decision_model"`
	Exposure      string   `json:"exposure"`
	Invariant     string   `json:"invariant"`
	Starter       string   `json:"starter"`
	Grader        string   `json:"grader"`
	Reference     string   `json:"reference"`
	Mutants       string   `json:"mutants"`
	PublicSafety  string   `json:"public_safety"`
	Limitations   []string `json:"limitations"`
}

// Verdict is the controller-level outcome of one grading run.
//
// Semantic outcomes (PASS/FAIL) describe the candidate submission. INFRASTRUCTURE
// describes the measurement apparatus instead: a build or execution timeout, or a
// toolchain failure. Keeping the two apart stops a slow cold compile from being
// recorded as a model failure.
type Verdict string

const (
	VerdictPass           Verdict = "PASS"
	VerdictFail           Verdict = "FAIL"
	VerdictInfrastructure Verdict = "INFRASTRUCTURE"
)

// Toolchain records the exact Go toolchain that produced a result. The module's
// `go` directive states a language-version floor, not the host toolchain, so the
// host identity is captured separately for every reported run.
type Toolchain struct {
	Version string `json:"version"`
	GOOS    string `json:"goos"`
	GOARCH  string `json:"goarch"`
}

func (t Toolchain) String() string {
	return fmt.Sprintf("%s (GOOS=%s GOARCH=%s)", t.Version, t.GOOS, t.GOARCH)
}

// DetectToolchain reports the exact toolchain identity used for grading.
func DetectToolchain() (Toolchain, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "go", "version").Output()
	if err != nil {
		return Toolchain{}, fmt.Errorf("determine go toolchain identity: %w", err)
	}
	return Toolchain{
		Version: strings.TrimSpace(string(out)),
		GOOS:    runtime.GOOS,
		GOARCH:  runtime.GOARCH,
	}, nil
}

// TestRecord is the controller's own record of one trusted test execution. It is
// derived from process exit status, never from subprocess text.
type TestRecord struct {
	Name    string  `json:"name"`
	Verdict Verdict `json:"verdict"`
	Detail  string  `json:"detail,omitempty"`
}

// GradeResult is the structured result of grading one submission.
type GradeResult struct {
	CaseID     string       `json:"case_id"`
	Verdict    Verdict      `json:"verdict"`
	Toolchain  Toolchain    `json:"toolchain"`
	Tests      []TestRecord `json:"tests"`
	Transcript string       `json:"-"`
}

// Passed reports whether the submission satisfied every trusted test.
func (r GradeResult) Passed() bool { return r.Verdict == VerdictPass }

func FindRepoRoot(start string) (string, error) {
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	cur, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err1 := os.Stat(filepath.Join(cur, "go.mod")); err1 == nil {
			if st, err2 := os.Stat(filepath.Join(cur, "cases")); err2 == nil && st.IsDir() {
				return cur, nil
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", errors.New("ProofFence repository root not found")
		}
		cur = parent
	}
}

func LoadCases(root string) ([]Case, error) {
	entries, err := os.ReadDir(filepath.Join(root, "cases"))
	if err != nil {
		return nil, err
	}
	var out []Case
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "PF-") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, "cases", e.Name(), "case.json"))
		if err != nil {
			return nil, err
		}
		var c Case
		if err := json.Unmarshal(b, &c); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if c.ID != e.Name() {
			return nil, fmt.Errorf("case directory %s has id %s", e.Name(), c.ID)
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func findCase(root, id string) (Case, string, error) {
	cases, err := LoadCases(root)
	if err != nil {
		return Case{}, "", err
	}
	for _, c := range cases {
		if c.ID == id {
			return c, filepath.Join(root, "cases", id), nil
		}
	}
	return Case{}, "", fmt.Errorf("unknown case %q", id)
}

func copyTree(src, dst string, renameGoTxt bool) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if renameGoTxt && strings.HasSuffix(target, ".go.txt") {
			target = strings.TrimSuffix(target, ".txt")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

const submissionBoundary = `## Submission boundary

ProofFence is a source-edit benchmark. Modify only challenge.go.
Do not add files or modify go.mod. The grader reconstructs the module from
the trusted starter and accepts only challenge.go from the candidate.

Candidate challenge.go must remain in package challenge; must not contain
compiler or build directives (//go:, //line, /*line, // +build, //export,
//extern); must not declare init, Test*, Benchmark*, or Fuzz* functions; must
not initialize package-level variables with function calls, conversions, or
function literals (declare the type instead, e.g. var x Kind = "A"); and may
import only: bytes, encoding/json, errors, fmt, io, maps, regexp, slices, sort,
strconv, strings, unicode, and unicode/utf8. Dot and blank imports are not
allowed. These restrictions are part of the public benchmark contract, not
hidden grader requirements.

`

func Materialize(root, id, dest string) error {
	c, dir, err := findCase(root, id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("destination already exists: %s", dest)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := copyTree(filepath.Join(dir, c.Starter), dest, false); err != nil {
		return err
	}
	task, err := os.ReadFile(filepath.Join(dir, "task.md"))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dest, "TASK.md"), append([]byte(submissionBoundary), task...), 0o644)
}

func trustedTestNames(graderDir string) ([]string, error) {
	var names []string
	err := filepath.WalkDir(graderDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go.txt") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, src, 0)
		if err != nil {
			return fmt.Errorf("parse trusted grader %s: %w", path, err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name == "TestMain" || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			if fn.Type.Results != nil && len(fn.Type.Results.List) != 0 {
				continue
			}
			if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
				continue
			}
			names = append(names, fn.Name.Name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, errors.New("trusted grader declares no top-level tests")
	}
	for i := 1; i < len(names); i++ {
		if names[i] == names[i-1] {
			return nil, fmt.Errorf("duplicate trusted grader test name %q", names[i])
		}
	}
	return names, nil
}

var allowedCandidateImports = map[string]bool{
	"bytes":         true,
	"encoding/json": true,
	"errors":        true,
	"fmt":           true,
	"io":            true,
	"maps":          true,
	"regexp":        true,
	"slices":        true,
	"sort":          true,
	"strconv":       true,
	"strings":       true,
	"unicode":       true,
	"unicode/utf8":  true,
}

// rejectedDirective matches comment forms the Go toolchain interprets as
// compiler, build, or position directives. Position (`//line`) and build-tag
// directives can change how the compiler reads the very source the controller
// validated, so they are rejected alongside `//go:` pragmas. Ordinary prose
// comments do not match: the forms below require the directive spelling with no
// space after the slashes.
var rejectedDirective = regexp.MustCompile(`^(//go:|//line[ \t]|/\*line[ \t]|// ?\+build|//export[ \t]|//extern[ \t])`)

// checkNoExecutablePackageVars rejects package-level variable initializers that
// can run code before any trusted test does.
//
// The Go runtime evaluates package-level variable initializers during package
// initialization, i.e. before the trusted test bodies execute. Rather than try to
// decide which pre-test execution is harmless, the source policy removes the
// mechanism: a package-level var initializer may not contain a call or a function
// literal. This is deliberately conservative — a type conversion parses as a call
// and is rejected too — and the equivalent typed declaration is always available.
// It is a mechanical, documented rule, not a malicious-Go analyzer.
func checkNoExecutablePackageVars(f *ast.File) error {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, v := range vs.Values {
				name := "_"
				if i < len(vs.Names) {
					name = vs.Names[i].Name
				}
				var found string
				ast.Inspect(v, func(n ast.Node) bool {
					if found != "" {
						return false
					}
					switch n.(type) {
					case *ast.CallExpr:
						found = "a function call or conversion"
					case *ast.FuncLit:
						found = "a function literal"
					}
					return found == ""
				})
				if found != "" {
					return fmt.Errorf(
						"candidate package-level variable %q is initialized with %s; package-level initializers run before the trusted tests and are not allowed",
						name, found)
				}
			}
		}
	}
	return nil
}

func validateCandidateSource(src []byte) error {
	if len(src) > 256*1024 {
		return errors.New("candidate challenge.go exceeds 256 KiB")
	}
	f, err := parser.ParseFile(token.NewFileSet(), "challenge.go", src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse candidate challenge.go: %w", err)
	}
	if f.Name == nil || f.Name.Name != "challenge" {
		return errors.New(`candidate challenge.go must declare package "challenge"`)
	}
	for _, group := range f.Comments {
		for _, c := range group.List {
			text := strings.TrimSpace(c.Text)
			if rejectedDirective.MatchString(text) {
				first := text
				if i := strings.IndexByte(first, '\n'); i >= 0 {
					first = first[:i]
				}
				return fmt.Errorf("candidate compiler/build directive is not allowed: %s", first)
			}
		}
	}
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return fmt.Errorf("invalid candidate import: %w", err)
		}
		if imp.Name != nil && (imp.Name.Name == "_" || imp.Name.Name == ".") {
			return fmt.Errorf("candidate import mode %q is not allowed for %s", imp.Name.Name, path)
		}
		if !allowedCandidateImports[path] {
			return fmt.Errorf("candidate import %q is outside the safe-import policy", path)
		}
	}
	if err := checkNoExecutablePackageVars(f); err != nil {
		return err
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		if fn.Name.Name == "init" {
			return errors.New("candidate init function is not allowed")
		}
		if fn.Name.Name == "TestMain" || strings.HasPrefix(fn.Name.Name, "Test") ||
			strings.HasPrefix(fn.Name.Name, "Benchmark") || strings.HasPrefix(fn.Name.Name, "Fuzz") {
			return fmt.Errorf("candidate test/lifecycle function %q is not allowed", fn.Name.Name)
		}
	}
	return nil
}

func copyValidatedCandidateSource(solution, trustedStarter, dst string) error {
	entries, err := os.ReadDir(solution)
	if err != nil {
		return err
	}
	allowedPaths := map[string]bool{"challenge.go": true, "go.mod": true, "TASK.md": true}
	for _, e := range entries {
		if !allowedPaths[e.Name()] {
			return fmt.Errorf("unexpected candidate path %q; ProofFence accepts only challenge.go with the trusted go.mod/TASK.md", e.Name())
		}
		if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("candidate path %q must be a regular file", e.Name())
		}
	}
	challengePath := filepath.Join(solution, "challenge.go")
	st, err := os.Lstat(challengePath)
	if err != nil {
		return fmt.Errorf("candidate challenge.go: %w", err)
	}
	if !st.Mode().IsRegular() {
		return errors.New("candidate challenge.go must be a regular file")
	}
	src, err := os.ReadFile(challengePath)
	if err != nil {
		return err
	}
	if err := validateCandidateSource(src); err != nil {
		return err
	}

	trustedMod, err := os.ReadFile(filepath.Join(trustedStarter, "go.mod"))
	if err != nil {
		return err
	}
	candidateMod, err := os.ReadFile(filepath.Join(solution, "go.mod"))
	if err != nil {
		return fmt.Errorf("candidate go.mod: %w", err)
	}
	if !bytes.Equal(candidateMod, trustedMod) {
		return errors.New("candidate go.mod differs from the trusted starter")
	}

	if err := os.WriteFile(filepath.Join(dst, "challenge.go"), src, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dst, "go.mod"), trustedMod, 0o644)
}

func controlledGoEnv() []string {
	env := os.Environ()
	out := make([]string, 0, len(env)+2)
	for _, kv := range env {
		if strings.HasPrefix(kv, "GOWORK=") || strings.HasPrefix(kv, "GOFLAGS=") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, "GOWORK=off", "GOFLAGS=")
}

// Timeouts. Compilation and trusted-test execution are budgeted separately so a
// cold build does not consume the execution budget and turn a correct submission
// into a FAIL. Both remain finite and both fail closed: exceeding either budget
// never yields PASS.
const (
	defaultBuildTimeout = 180 * time.Second
	defaultTestTimeout  = 60 * time.Second
)

func envTimeout(name string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func BuildTimeout() time.Duration {
	return envTimeout("PROOF_FENCE_BUILD_TIMEOUT", defaultBuildTimeout)
}
func TestTimeout() time.Duration { return envTimeout("PROOF_FENCE_TEST_TIMEOUT", defaultTestTimeout) }

// transcript accumulates an origin-labelled record of a grading run.
//
// Every line is attributed. CONTROLLER lines are the benchmark's own statements,
// TRUSTED_TEST lines are controller-derived verdicts for a named trusted test,
// and CANDIDATE_OUTPUT lines are raw subprocess bytes. Because the controller
// prepends its label to every subprocess line, candidate text that imitates a
// controller line still arrives labelled as candidate output.
type transcript struct {
	b strings.Builder
}

const (
	originController  = "CONTROLLER"
	originTrustedTest = "TRUSTED_TEST"
	originCandidate   = "CANDIDATE_OUTPUT"
)

func (t *transcript) line(origin, format string, args ...any) {
	fmt.Fprintf(&t.b, "[proof-fence %s] %s\n", origin, fmt.Sprintf(format, args...))
}

// subprocess records raw bytes produced by the candidate's process. The bytes
// are a mixture of trusted testing-framework output and arbitrary candidate
// stdout/stderr that cannot be separated inside a single stream, so the whole
// block is labelled as candidate output and carries no result authority.
func (t *transcript) subprocess(b []byte) {
	s := strings.TrimRight(string(b), "\n")
	if s == "" {
		return
	}
	for _, ln := range strings.Split(s, "\n") {
		fmt.Fprintf(&t.b, "[proof-fence %s] %s\n", originCandidate, ln)
	}
}

func (t *transcript) String() string { return t.b.String() }

type timeoutError struct{ msg string }

func (e *timeoutError) Error() string { return e.msg }

func isTimeout(err error) bool {
	var te *timeoutError
	return errors.As(err, &te)
}

// buildTestBinary compiles the reconstructed module (trusted go.mod + validated
// candidate challenge.go + trusted grader) into a test binary.
func buildTestBinary(tmp string) (string, []byte, error) {
	bin := filepath.Join(tmp, "proof-fence-case.test")
	ctx, cancel := context.WithTimeout(context.Background(), BuildTimeout())
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-c", "-o", bin, ".")
	cmd.Dir = tmp
	cmd.Env = controlledGoEnv()
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", out, &timeoutError{fmt.Sprintf("compilation exceeded the %s build budget", BuildTimeout())}
	}
	if err != nil {
		return "", out, fmt.Errorf("compile case module: %w", err)
	}
	return bin, out, nil
}

// runTrustedTest executes exactly one controller-selected trusted test in the
// precompiled binary. The verdict comes from the process exit status.
func runTrustedTest(tmp, bin, name string) ([]byte, error) {
	budget := TestTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()

	pattern := "^" + regexp.QuoteMeta(name) + "$"
	// The in-binary timeout is kept strictly below the controller budget so a hung
	// test normally self-reports as a semantic failure before the outer deadline.
	inner := budget - (budget / 4)
	cmd := exec.CommandContext(ctx, bin,
		"-test.run", pattern,
		"-test.count=1",
		"-test.timeout="+inner.String(),
	)
	cmd.Dir = tmp
	cmd.Env = controlledGoEnv()
	b, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return b, &timeoutError{fmt.Sprintf("trusted test %s exceeded the %s execution budget", name, budget)}
	}
	if err != nil {
		return b, fmt.Errorf("trusted test %s failed: %w", name, err)
	}
	return b, nil
}

func gradeUnsandboxed(root, id, solution string) (GradeResult, error) {
	res := GradeResult{CaseID: id, Verdict: VerdictFail}
	var t transcript

	c, dir, err := findCase(root, id)
	if err != nil {
		return res, err
	}
	res.CaseID = c.ID

	tc, err := DetectToolchain()
	if err != nil {
		res.Verdict = VerdictInfrastructure
		t.line(originController, "case=%s verdict=INFRASTRUCTURE detail=%q", c.ID, err.Error())
		res.Transcript = t.String()
		return res, err
	}
	res.Toolchain = tc
	t.line(originController, "case=%s toolchain=%q", c.ID, tc.String())
	t.line(originController, "case=%s build_timeout=%s test_timeout=%s", c.ID, BuildTimeout(), TestTimeout())

	expectedTests, err := trustedTestNames(filepath.Join(dir, c.Grader))
	if err != nil {
		return res, err
	}
	t.line(originController, "case=%s trusted_tests=%d selected=%s", c.ID, len(expectedTests), strings.Join(expectedTests, ","))

	tmp, err := os.MkdirTemp("", "proof-fence-grade-*")
	if err != nil {
		return res, err
	}
	defer os.RemoveAll(tmp)

	if err := copyValidatedCandidateSource(solution, filepath.Join(dir, c.Starter), tmp); err != nil {
		t.line(originController, "case=%s verdict=FAIL stage=source-policy detail=%q", c.ID, err.Error())
		res.Verdict = VerdictFail
		res.Transcript = t.String()
		return res, err
	}
	if err := copyTree(filepath.Join(dir, c.Grader), tmp, true); err != nil {
		return res, err
	}

	bin, buildOut, err := buildTestBinary(tmp)
	t.subprocess(buildOut)
	if err != nil {
		if isTimeout(err) {
			res.Verdict = VerdictInfrastructure
			t.line(originController, "case=%s verdict=INFRASTRUCTURE stage=build detail=%q", c.ID, err.Error())
		} else {
			res.Verdict = VerdictFail
			t.line(originController, "case=%s verdict=FAIL stage=build detail=%q", c.ID, err.Error())
		}
		res.Transcript = t.String()
		return res, err
	}
	t.line(originController, "case=%s stage=build outcome=ok", c.ID)

	for _, name := range expectedTests {
		b, err := runTrustedTest(tmp, bin, name)
		t.subprocess(b)
		if err != nil {
			rec := TestRecord{Name: name, Verdict: VerdictFail, Detail: err.Error()}
			if isTimeout(err) {
				rec.Verdict = VerdictInfrastructure
				res.Verdict = VerdictInfrastructure
			} else {
				res.Verdict = VerdictFail
			}
			res.Tests = append(res.Tests, rec)
			t.line(originTrustedTest, "case=%s name=%s outcome=%s detail=%q", c.ID, name, rec.Verdict, err.Error())
			t.line(originController, "case=%s verdict=%s", c.ID, res.Verdict)
			res.Transcript = t.String()
			return res, err
		}
		res.Tests = append(res.Tests, TestRecord{Name: name, Verdict: VerdictPass})
		t.line(originTrustedTest, "case=%s name=%s outcome=PASS", c.ID, name)
	}

	res.Verdict = VerdictPass
	t.line(originController, "case=%s verdict=PASS trusted_tests_passed=%d", c.ID, len(res.Tests))
	res.Transcript = t.String()
	return res, nil
}

func Grade(root, id, solution string) (GradeResult, error) {
	if os.Getenv("PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE") != "1" {
		return GradeResult{CaseID: id, Verdict: VerdictInfrastructure},
			errors.New("refusing to execute candidate code without explicit opt-in; run in a disposable sandbox and set PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE=1")
	}
	return gradeUnsandboxed(root, id, solution)
}

// referenceDir materializes the trusted starter overlaid with the case reference
// implementation, which is what a correct submission is expected to behave like.
func referenceDir(root string, c Case, base string) (string, error) {
	_, dir, err := findCase(root, c.ID)
	if err != nil {
		return "", err
	}
	refDir := filepath.Join(base, "reference")
	if err := copyTree(filepath.Join(dir, c.Starter), refDir, false); err != nil {
		return "", err
	}
	if err := copyTree(filepath.Join(dir, c.Reference), refDir, true); err != nil {
		return "", err
	}
	return refDir, nil
}

// SelfTest proves, for every case, that the starter fails and the reference
// passes. A starter that fails for infrastructure reasons is itself an error:
// the self-test must demonstrate a semantic gap, not a flaky apparatus.
func SelfTest(root string, w io.Writer) error {
	cases, err := LoadCases(root)
	if err != nil {
		return err
	}
	tc, err := DetectToolchain()
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "[proof-fence CONTROLLER] selftest toolchain=%q\n", tc.String())
	fmt.Fprintf(w, "[proof-fence CONTROLLER] selftest build_timeout=%s test_timeout=%s cases=%d\n",
		BuildTimeout(), TestTimeout(), len(cases))
	for _, c := range cases {
		tmp, err := os.MkdirTemp("", "proof-fence-selftest-*")
		if err != nil {
			return err
		}
		if err := Materialize(root, c.ID, filepath.Join(tmp, "starter")); err != nil {
			os.RemoveAll(tmp)
			return err
		}
		res, err := gradeUnsandboxed(root, c.ID, filepath.Join(tmp, "starter"))
		if err == nil {
			os.RemoveAll(tmp)
			return fmt.Errorf("%s starter unexpectedly passed:\n%s", c.ID, res.Transcript)
		}
		if res.Verdict != VerdictFail {
			os.RemoveAll(tmp)
			return fmt.Errorf("%s starter did not fail semantically (verdict=%s):\n%s", c.ID, res.Verdict, res.Transcript)
		}
		refDir, err := referenceDir(root, c, tmp)
		if err != nil {
			os.RemoveAll(tmp)
			return err
		}
		res, err = gradeUnsandboxed(root, c.ID, refDir)
		if err != nil {
			os.RemoveAll(tmp)
			return fmt.Errorf("%s reference failed:\n%s", c.ID, res.Transcript)
		}
		fmt.Fprintf(w, "[proof-fence CONTROLLER] %s starter=FAIL reference=PASS trusted_tests=%d\n", c.ID, len(res.Tests))
		os.RemoveAll(tmp)
	}
	return nil
}

// MutantResult records one committed near-miss / degenerate-strategy mutant.
type MutantResult struct {
	CaseID  string  `json:"case_id"`
	Name    string  `json:"name"`
	Verdict Verdict `json:"verdict"`
	Killed  bool    `json:"killed"`
}

// LoadMutantNames lists the committed mutants for a case, in stable order.
func LoadMutantNames(root string, c Case) ([]string, error) {
	if c.Mutants == "" {
		return nil, nil
	}
	dir := filepath.Join(root, "cases", c.ID, c.Mutants)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go.txt") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".go.txt"))
	}
	sort.Strings(names)
	return names, nil
}

// GradeMutant grades one committed mutant against the case grader. Every
// committed mutant is expected to be killed, i.e. not to reach PASS.
func GradeMutant(root string, c Case, name string) (MutantResult, error) {
	tmp, err := os.MkdirTemp("", "proof-fence-mutant-*")
	if err != nil {
		return MutantResult{}, err
	}
	defer os.RemoveAll(tmp)

	d := filepath.Join(tmp, "candidate")
	if err := Materialize(root, c.ID, d); err != nil {
		return MutantResult{}, err
	}
	src, err := os.ReadFile(filepath.Join(root, "cases", c.ID, c.Mutants, name+".go.txt"))
	if err != nil {
		return MutantResult{}, err
	}
	if err := os.WriteFile(filepath.Join(d, "challenge.go"), src, 0o644); err != nil {
		return MutantResult{}, err
	}
	res, _ := gradeUnsandboxed(root, c.ID, d)
	return MutantResult{
		CaseID:  c.ID,
		Name:    name,
		Verdict: res.Verdict,
		Killed:  res.Verdict != VerdictPass,
	}, nil
}

// MutationSuite runs every committed mutant for every case and reports any
// survivor. A surviving mutant means a grader does not actually discriminate the
// behaviour the case claims to measure.
func MutationSuite(root string, w io.Writer) error {
	cases, err := LoadCases(root)
	if err != nil {
		return err
	}
	tc, err := DetectToolchain()
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "[proof-fence CONTROLLER] mutation toolchain=%q\n", tc.String())
	var survivors []string
	total := 0
	for _, c := range cases {
		names, err := LoadMutantNames(root, c)
		if err != nil {
			return err
		}
		for _, n := range names {
			total++
			mr, err := GradeMutant(root, c, n)
			if err != nil {
				return err
			}
			status := "killed"
			if !mr.Killed {
				status = "SURVIVED"
				survivors = append(survivors, c.ID+"/"+n)
			}
			fmt.Fprintf(w, "[proof-fence CONTROLLER] %s mutant=%s verdict=%s %s\n", c.ID, n, mr.Verdict, status)
		}
	}
	fmt.Fprintf(w, "[proof-fence CONTROLLER] mutation total=%d survivors=%d\n", total, len(survivors))
	if len(survivors) > 0 {
		return fmt.Errorf("mutants survived: %s", strings.Join(survivors, ", "))
	}
	if total == 0 {
		return errors.New("no committed mutants found")
	}
	return nil
}
