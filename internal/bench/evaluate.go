package bench

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// EvaluationSchemaVersion identifies the record and manifest formats below.
// External runners should refuse a record whose version they do not understand
// rather than guess at a field's meaning.
const EvaluationSchemaVersion = "proof-fence/evaluation/v1"

// Stage names the point a case evaluation reached. It is recorded because the
// three ways a case can end short of running fixtures are not interchangeable:
// a source-policy rejection and a compile error are statements about the
// candidate, while a build timeout is a statement about the apparatus.
type Stage string

const (
	StageSourcePolicy Stage = "source-policy"
	StageBuild        Stage = "build"
	StageFixtures     Stage = "fixtures"
)

// FixtureResult is one trusted test's outcome. Every trusted test of an
// evaluated case gets exactly one of these, whatever the others did.
type FixtureResult struct {
	Name    string  `json:"name"`
	Verdict Verdict `json:"verdict"`
	Detail  string  `json:"detail,omitempty"`
}

// FixtureCounts is the per-verdict tally for one case.
type FixtureCounts struct {
	Total          int `json:"total"`
	Pass           int `json:"pass"`
	Fail           int `json:"fail"`
	Infrastructure int `json:"infrastructure"`
}

// CaseEvaluation is the full-vector result of evaluating one submission against
// one case.
//
// It differs from GradeResult in the one way that matters for a study: grading
// stops at the first failing fixture, so its Tests slice is a prefix of the
// truth and the rest of the vector is unrecoverable. An evaluation runs every
// trusted fixture and records every outcome, so the same artifact always yields
// the same vector no matter which fixture happens to fail first.
type CaseEvaluation struct {
	SchemaVersion string          `json:"schema_version"`
	CaseID        string          `json:"case_id"`
	DecisionModel string          `json:"decision_model,omitempty"`
	Verdict       Verdict         `json:"verdict"`
	Stage         Stage           `json:"stage"`
	Complete      bool            `json:"complete"`
	Counts        FixtureCounts   `json:"counts"`
	Fixtures      []FixtureResult `json:"fixtures"`
	Toolchain     Toolchain       `json:"toolchain"`
	BuildTimeout  string          `json:"build_timeout"`
	TestTimeout   string          `json:"test_timeout"`
	StartedAtUTC  time.Time       `json:"started_at_utc"`
	EndedAtUTC    time.Time       `json:"ended_at_utc"`
	DurationSec   float64         `json:"duration_seconds"`
	Detail        string          `json:"detail,omitempty"`
	Manifest      *RunManifest    `json:"manifest,omitempty"`
}

// rollUp derives the case verdict from the fixture vector.
//
// A single semantic failure is decisive: the candidate demonstrably got a
// fixture wrong, and no amount of apparatus trouble elsewhere undoes that. With
// no semantic failure but some fixture the apparatus could not run, the vector
// is incomplete and PASS would assert more than was measured, so the case is
// INFRASTRUCTURE. Complete records whether the vector is whole, so a study can
// drop incomplete trials without re-deriving this rule.
func (e *CaseEvaluation) rollUp() {
	e.Counts = FixtureCounts{Total: len(e.Fixtures)}
	for _, f := range e.Fixtures {
		switch f.Verdict {
		case VerdictPass:
			e.Counts.Pass++
		case VerdictFail:
			e.Counts.Fail++
		default:
			e.Counts.Infrastructure++
		}
	}
	e.Complete = e.Counts.Infrastructure == 0
	switch {
	case e.Counts.Fail > 0:
		e.Verdict = VerdictFail
	case e.Counts.Infrastructure > 0 || e.Counts.Total == 0:
		e.Verdict = VerdictInfrastructure
	default:
		e.Verdict = VerdictPass
	}
}

// EvaluateCase runs every trusted fixture of one case against one submission and
// returns the complete outcome vector. Unlike Grade it does not stop at the
// first semantic failure, and it returns no error for a semantic outcome: a
// failing submission is a result, not a malfunction. The error return is
// reserved for the harness being unable to produce a record at all.
func EvaluateCase(root, id, solution string) (CaseEvaluation, error) {
	if os.Getenv("PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE") != "1" {
		return CaseEvaluation{}, errors.New("refusing to execute candidate code without explicit opt-in; run in a disposable sandbox and set PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE=1")
	}
	return evaluateCaseUnsandboxed(root, id, solution)
}

func evaluateCaseUnsandboxed(root, id, solution string) (CaseEvaluation, error) {
	start := time.Now().UTC()
	e := CaseEvaluation{
		SchemaVersion: EvaluationSchemaVersion,
		CaseID:        id,
		Stage:         StageSourcePolicy,
		BuildTimeout:  BuildTimeout().String(),
		TestTimeout:   TestTimeout().String(),
		StartedAtUTC:  start,
	}
	finish := func() (CaseEvaluation, error) {
		e.rollUp()
		e.EndedAtUTC = time.Now().UTC()
		e.DurationSec = e.EndedAtUTC.Sub(e.StartedAtUTC).Seconds()
		return e, nil
	}

	c, dir, err := findCase(root, id)
	if err != nil {
		return e, err
	}
	e.CaseID = c.ID
	e.DecisionModel = c.DecisionModel

	tc, err := DetectToolchain()
	if err != nil {
		return e, err
	}
	e.Toolchain = tc

	expected, err := trustedTestNames(filepath.Join(dir, c.Grader))
	if err != nil {
		return e, err
	}

	// fill records every expected fixture with one verdict, so a case that never
	// reached its fixtures still yields a whole-length vector rather than an
	// empty one a reader could mistake for "nothing was required".
	fill := func(v Verdict, detail string) {
		e.Detail = detail
		for _, n := range expected {
			e.Fixtures = append(e.Fixtures, FixtureResult{Name: n, Verdict: v, Detail: detail})
		}
	}

	tmp, err := os.MkdirTemp("", "proof-fence-evaluate-*")
	if err != nil {
		return e, err
	}
	defer os.RemoveAll(tmp)

	if err := copyValidatedCandidateSource(solution, filepath.Join(dir, c.Starter), tmp); err != nil {
		fill(VerdictFail, err.Error())
		return finish()
	}
	if err := copyTree(filepath.Join(dir, c.Grader), tmp, true); err != nil {
		return e, err
	}

	e.Stage = StageBuild
	bin, _, err := buildTestBinary(tmp)
	if err != nil {
		// A compile error is the candidate's; exhausting the build budget is the
		// apparatus's. Only the second may not be scored against a submission.
		if isTimeout(err) {
			fill(VerdictInfrastructure, err.Error())
		} else {
			fill(VerdictFail, err.Error())
		}
		return finish()
	}

	e.Stage = StageFixtures
	for _, name := range expected {
		b, err := runTrustedTest(tmp, bin, name)
		_ = b
		switch {
		case err == nil:
			e.Fixtures = append(e.Fixtures, FixtureResult{Name: name, Verdict: VerdictPass})
		case isTimeout(err):
			e.Fixtures = append(e.Fixtures, FixtureResult{Name: name, Verdict: VerdictInfrastructure, Detail: err.Error()})
		default:
			e.Fixtures = append(e.Fixtures, FixtureResult{Name: name, Verdict: VerdictFail, Detail: err.Error()})
		}
	}
	return finish()
}

// SuiteEvaluation is the result of evaluating one submission set across cases.
type SuiteEvaluation struct {
	SchemaVersion string           `json:"schema_version"`
	Cases         []CaseEvaluation `json:"cases"`
	Counts        FixtureCounts    `json:"fixture_counts"`
	CasesPassed   int              `json:"cases_passed"`
	CasesFailed   int              `json:"cases_failed"`
	CasesInfra    int              `json:"cases_infrastructure"`
}

// EvaluateSuite evaluates every case that has a directory named after it under
// solutionRoot. Cases with no submission directory are skipped rather than
// scored, because "not attempted" and "attempted and wrong" are different facts.
func EvaluateSuite(root, solutionRoot string, manifest *RunManifest, w io.Writer) (SuiteEvaluation, error) {
	cases, err := LoadCases(root)
	if err != nil {
		return SuiteEvaluation{}, err
	}
	out := SuiteEvaluation{SchemaVersion: EvaluationSchemaVersion}
	enc := json.NewEncoder(w)
	for _, c := range cases {
		d := filepath.Join(solutionRoot, c.ID)
		if st, err := os.Stat(d); err != nil || !st.IsDir() {
			continue
		}
		e, err := EvaluateCase(root, c.ID, d)
		if err != nil {
			return out, err
		}
		if manifest != nil {
			m := *manifest
			m.CaseID = c.ID
			e.Manifest = &m
		}
		if w != nil {
			if err := enc.Encode(e); err != nil {
				return out, err
			}
		}
		out.Cases = append(out.Cases, e)
		out.Counts.Total += e.Counts.Total
		out.Counts.Pass += e.Counts.Pass
		out.Counts.Fail += e.Counts.Fail
		out.Counts.Infrastructure += e.Counts.Infrastructure
		switch e.Verdict {
		case VerdictPass:
			out.CasesPassed++
		case VerdictFail:
			out.CasesFailed++
		default:
			out.CasesInfra++
		}
	}
	if len(out.Cases) == 0 {
		return out, fmt.Errorf("no case submission directories found under %s", solutionRoot)
	}
	return out, nil
}

// RunManifest is the provenance record for one evaluation trial.
//
// ProofFence never calls a model, so it cannot observe any of this; an external
// runner supplies it and ProofFence carries it verbatim into the result. The
// fields exist because the first exploratory pass against v0.2 recorded four
// lines of metadata, which was not enough to attribute its own artifact to any
// model, and so could support no capability claim at all. Every field is
// optional at the type level and checked by Validate, so a runner that cannot
// observe something records nothing rather than a plausible-looking guess.
//
// No field is for a credential. Validate rejects a manifest that looks like it
// carries one.
type RunManifest struct {
	SchemaVersion string `json:"schema_version"`

	// Model identity, exactly as the runner received it.
	Provider        string   `json:"provider,omitempty"`
	ModelID         string   `json:"model_id,omitempty"`
	ReasoningEffort string   `json:"reasoning_effort,omitempty"`
	EndpointFamily  string   `json:"endpoint_family,omitempty"`
	Retention       string   `json:"retention,omitempty"`
	ToolPermissions []string `json:"tool_permissions,omitempty"`

	// Inputs, by digest. The prompts themselves are not stored here: a hash
	// proves which prompt was used without putting a possibly held-out case
	// statement into a result file.
	PromptSHA256       string `json:"prompt_sha256,omitempty"`
	SystemPromptSHA256 string `json:"system_prompt_sha256,omitempty"`

	// Trial identity.
	CaseID            string `json:"case_id,omitempty"`
	TrialID           string `json:"trial_id,omitempty"`
	RandomizationSeed string `json:"randomization_seed,omitempty"`
	OrderIndex        *int   `json:"order_index,omitempty"`

	// Timing, as observed by the runner around its model call.
	StartedAtUTC     string   `json:"started_at_utc,omitempty"`
	EndedAtUTC       string   `json:"ended_at_utc,omitempty"`
	WallClockSeconds *float64 `json:"wall_clock_seconds,omitempty"`

	// Usage, when the provider reports it.
	InputTokens     *int   `json:"input_tokens,omitempty"`
	OutputTokens    *int   `json:"output_tokens,omitempty"`
	ReasoningTokens *int   `json:"reasoning_tokens,omitempty"`
	APIRequestID    string `json:"api_request_id,omitempty"`

	// Apparatus identity.
	BenchmarkGitSHA      string `json:"benchmark_git_sha,omitempty"`
	ContainerImageDigest string `json:"container_image_digest,omitempty"`
	GoVersion            string `json:"go_version,omitempty"`
	OS                   string `json:"os,omitempty"`
	Arch                 string `json:"arch,omitempty"`
	BuildTimeout         string `json:"build_timeout,omitempty"`
	TestTimeout          string `json:"test_timeout,omitempty"`

	// InfrastructureErrorCategory names why a trial could not be completed, for
	// trials a study drops rather than scores.
	InfrastructureErrorCategory string `json:"infrastructure_error_category,omitempty"`

	// Notes is free text for anything the schema does not model.
	Notes string `json:"notes,omitempty"`
}

var (
	hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)
	// secretKey matches field names a credential is usually spelled with, and
	// secretValue matches the shapes of common issued tokens. Neither is a
	// general secret detector; both exist so an obvious mistake fails loudly
	// instead of being published.
	secretKey   = regexp.MustCompile(`(?i)\b(api[_-]?key|secret|token|password|passwd|credential|authorization|bearer|private[_-]?key)\b`)
	secretValue = regexp.MustCompile(`(?i)(sk-[A-Za-z0-9_-]{16,}|AKIA[0-9A-Z]{16}|ghp_[A-Za-z0-9]{20,}|-----BEGIN [A-Z ]*PRIVATE KEY-----)`)
)

// Validate checks a manifest for internal consistency and for anything that
// looks like a credential. It deliberately does not require any field: a runner
// that cannot observe the reasoning effort should leave it empty, and an empty
// field is an honest record where a default would be a fabricated one.
func (m *RunManifest) Validate() error {
	if m.SchemaVersion != "" && m.SchemaVersion != EvaluationSchemaVersion {
		return fmt.Errorf("manifest schema_version %q is not %q", m.SchemaVersion, EvaluationSchemaVersion)
	}
	for name, v := range map[string]string{
		"prompt_sha256":        m.PromptSHA256,
		"system_prompt_sha256": m.SystemPromptSHA256,
	} {
		if v != "" && !hex64.MatchString(v) {
			return fmt.Errorf("manifest %s must be 64 lower-case hex characters", name)
		}
	}
	for _, ts := range []struct{ name, v string }{
		{"started_at_utc", m.StartedAtUTC},
		{"ended_at_utc", m.EndedAtUTC},
	} {
		if ts.v == "" {
			continue
		}
		if _, err := time.Parse(time.RFC3339, ts.v); err != nil {
			return fmt.Errorf("manifest %s must be an RFC 3339 timestamp: %w", ts.name, err)
		}
	}
	return m.checkNoSecrets()
}

func (m *RunManifest) checkNoSecrets() error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	var fields map[string]any
	if err := json.Unmarshal(b, &fields); err != nil {
		return err
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if secretKey.MatchString(k) {
			return fmt.Errorf("manifest field %q looks like a credential field; ProofFence manifests never carry credentials", k)
		}
	}
	if loc := secretValue.FindIndex(b); loc != nil {
		return errors.New("manifest value matches a known credential shape; ProofFence manifests never carry credentials")
	}
	if secretKey.MatchString(m.Notes) {
		return errors.New("manifest notes name a credential field; ProofFence manifests never carry credentials")
	}
	return nil
}

// LoadManifest reads and validates a run manifest supplied by an external
// runner.
func LoadManifest(path string) (*RunManifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m RunManifest
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("parse run manifest %s: %w", path, err)
	}
	if m.SchemaVersion == "" {
		m.SchemaVersion = EvaluationSchemaVersion
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// ManifestTemplate returns an empty, valid manifest carrying the apparatus
// fields ProofFence can observe for itself. A runner fills in the rest.
func ManifestTemplate(root string) *RunManifest {
	m := &RunManifest{
		SchemaVersion: EvaluationSchemaVersion,
		BuildTimeout:  BuildTimeout().String(),
		TestTimeout:   TestTimeout().String(),
	}
	if tc, err := DetectToolchain(); err == nil {
		m.GoVersion = tc.Version
		m.OS = tc.GOOS
		m.Arch = tc.GOARCH
	}
	return m
}

// DecisionDistribution is the per-decision fixture count for one case, derived
// from the committed grader rather than restated in prose.
//
// It exists because v0.2's documented distribution for PF-013 was wrong — the
// published row summed to 19 against a stated total of 24 — and a table a human
// types is a table that drifts. Meta counts fixtures that assert a property
// across several states instead of pinning one state to one decision, so
// Grant+Retain+Revoke+Quarantine+Meta always equals Total.
type DecisionDistribution struct {
	CaseID     string `json:"case_id"`
	Grant      int    `json:"grant"`
	Retain     int    `json:"retain"`
	Revoke     int    `json:"revoke"`
	Quarantine int    `json:"quarantine"`
	Meta       int    `json:"meta"`
	Total      int    `json:"total"`
}

// FixtureDistribution derives a case's decision distribution by reading its
// grader's syntax tree: every `want(t, got, <Decision>, ...)` assertion pins one
// evidence state to one decision, and every trusted test that makes no such
// assertion is counted as a meta-fixture.
func FixtureDistribution(root string, c Case) (DecisionDistribution, error) {
	d := DecisionDistribution{CaseID: c.ID}
	graderDir := filepath.Join(root, "cases", c.ID, c.Grader)
	names, err := trustedTestNames(graderDir)
	if err != nil {
		return d, err
	}
	d.Total = len(names)

	entries, err := os.ReadDir(graderDir)
	if err != nil {
		return d, err
	}
	counted := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go.txt") {
			continue
		}
		path := filepath.Join(graderDir, entry.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			return d, err
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, src, 0)
		if err != nil {
			return d, fmt.Errorf("parse trusted grader %s: %w", path, err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") || fn.Name.Name == "TestMain" {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) < 3 {
					return true
				}
				id, ok := call.Fun.(*ast.Ident)
				if !ok || id.Name != "want" {
					return true
				}
				dec, ok := call.Args[2].(*ast.Ident)
				if !ok {
					return true
				}
				switch dec.Name {
				case "Grant":
					d.Grant++
				case "Retain":
					d.Retain++
				case "Revoke":
					d.Revoke++
				case "Quarantine":
					d.Quarantine++
				default:
					return true
				}
				counted[fn.Name.Name] = true
				return true
			})
		}
	}
	d.Meta = d.Total - len(counted)
	if sum := d.Grant + d.Retain + d.Revoke + d.Quarantine; sum != len(counted) {
		return d, fmt.Errorf("%s: %d decision assertions across %d fixtures; a fixture pins more than one state and the row cannot be reported as a distribution",
			c.ID, sum, len(counted))
	}
	return d, nil
}

// DistributionTable renders the four-valued cases' decision distributions as the
// Markdown table the documentation publishes, so the table can be regenerated
// from the graders instead of maintained by hand.
func DistributionTable(root string) (string, error) {
	cases, err := LoadCases(root)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("| Case | GRANT | RETAIN | REVOKE | QUARANTINE | meta | total |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	dash := func(n int) string {
		if n == 0 {
			return "—"
		}
		return fmt.Sprintf("%d", n)
	}
	for _, c := range cases {
		if c.DecisionModel != "four-valued-authority" {
			continue
		}
		d, err := FixtureDistribution(root, c)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %d |\n",
			d.CaseID, dash(d.Grant), dash(d.Retain), dash(d.Revoke), dash(d.Quarantine), dash(d.Meta), d.Total)
	}
	return b.String(), nil
}
