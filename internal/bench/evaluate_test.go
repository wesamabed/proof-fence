package bench

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMaterializedTaskCarriesTheDecisionVocabulary is the regression guard for
// the defect a blind audit of v0.2 rated its most widespread: the four-valued
// vocabulary reached a solver only through four const doc comments, so the
// distinction between RETAIN as a conclusion and QUARANTINE as the absence of
// one never arrived. The check is on the materialized artifact, because what a
// docs/ file says is irrelevant to a candidate that never receives it.
func TestMaterializedTaskCarriesTheDecisionVocabulary(t *testing.T) {
	root := repoRoot(t)
	cs, err := LoadCases(root)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, c := range cs {
		if c.DecisionModel != "four-valued-authority" {
			continue
		}
		seen++
		d := filepath.Join(t.TempDir(), c.ID)
		if err := Materialize(root, c.ID, d); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(d, "TASK.md"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		// The appendix is hard-wrapped, so a phrase can straddle a newline.
		// Compare with runs of whitespace collapsed.
		flat := strings.Join(strings.Fields(s), " ")
		for _, want := range []string{
			"authority decision vocabulary",
			"`QUARANTINE` is the absence of one",
			"`RETAIN` is a conclusion",
			"not a default, not a deferral",
			"is an error in its own right",
			"what *this pass's evidence* establishes",
		} {
			if !strings.Contains(flat, want) {
				t.Fatalf("%s TASK.md does not reach the solver with %q", c.ID, want)
			}
		}
		if strings.Contains(s, "## The authority decision vocabulary\n## The authority decision vocabulary") {
			t.Fatalf("%s TASK.md repeats the vocabulary section", c.ID)
		}
	}
	if seen < 3 {
		t.Fatalf("expected at least 3 four-valued authority cases, found %d", seen)
	}
}

// TestNonAuthorityCasesDoNotCarryTheVocabulary keeps the appendix scoped to the
// family whose audit found it missing. Adding generic text to a case that did
// not need it would change an instrument no finding asked to change.
func TestNonAuthorityCasesDoNotCarryTheVocabulary(t *testing.T) {
	root := repoRoot(t)
	d := filepath.Join(t.TempDir(), "PF-001")
	if err := Materialize(root, "PF-001", d); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(d, "TASK.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "authority decision vocabulary") {
		t.Fatal("PF-001 is not a four-valued case and must not receive the authority vocabulary")
	}
}

// writeCandidate materializes a case and overwrites challenge.go, which is the
// only file a submission may change.
func writeCandidate(t *testing.T, root, id, src string) string {
	t.Helper()
	d := filepath.Join(t.TempDir(), "candidate")
	if err := Materialize(root, id, d); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "challenge.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return d
}

// TestEvaluateRecordsTheWholeFixtureVector is the point of the evaluation mode.
// The starter fails its case, and grading stops at the first failure, so a
// grade transcript can never say how the rest of the suite would have gone. An
// evaluation must record every fixture, including the ones after the failure.
func TestEvaluateRecordsTheWholeFixtureVector(t *testing.T) {
	t.Setenv("PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE", "1")
	root := repoRoot(t)

	d := filepath.Join(t.TempDir(), "starter")
	if err := Materialize(root, "PF-013", d); err != nil {
		t.Fatal(err)
	}
	e, err := EvaluateCase(root, "PF-013", d)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := trustedTestNames(filepath.Join(root, "cases", "PF-013", "grader"))
	if err != nil {
		t.Fatal(err)
	}
	if e.Counts.Total != len(expected) || len(e.Fixtures) != len(expected) {
		t.Fatalf("recorded %d fixtures, case declares %d", len(e.Fixtures), len(expected))
	}
	if e.Verdict != VerdictFail {
		t.Fatalf("starter verdict=%s, want FAIL", e.Verdict)
	}
	if e.Counts.Pass == 0 || e.Counts.Fail == 0 {
		t.Fatalf("a truncated vector: pass=%d fail=%d; the starter both passes and fails fixtures",
			e.Counts.Pass, e.Counts.Fail)
	}
	if !e.Complete || e.Counts.Infrastructure != 0 {
		t.Fatalf("clean run reported infrastructure trouble: %+v", e.Counts)
	}
	if e.Stage != StageFixtures {
		t.Fatalf("stage=%s, want %s", e.Stage, StageFixtures)
	}

	// The same artifact under the fail-fast path stops early, which is exactly
	// why formal evaluation may not use it.
	g, _ := gradeUnsandboxed(root, "PF-013", d)
	if len(g.Tests) >= len(e.Fixtures) {
		t.Fatalf("grade recorded %d of %d fixtures; it is supposed to stop at the first failure",
			len(g.Tests), len(e.Fixtures))
	}
}

// TestEvaluateReferenceIsAllPass anchors the other end of the range.
func TestEvaluateReferenceIsAllPass(t *testing.T) {
	t.Setenv("PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE", "1")
	root := repoRoot(t)
	cs, err := LoadCases(root)
	if err != nil {
		t.Fatal(err)
	}
	c := cs[0]
	base := t.TempDir()
	refDir, err := referenceDir(root, c, base)
	if err != nil {
		t.Fatal(err)
	}
	e, err := EvaluateCase(root, c.ID, refDir)
	if err != nil {
		t.Fatal(err)
	}
	if e.Verdict != VerdictPass || e.Counts.Fail != 0 || !e.Complete {
		t.Fatalf("%s reference: %+v", c.ID, e)
	}
}

// TestEvaluateSeparatesInfrastructureFromSemantics checks that exhausting the
// execution budget is recorded as apparatus trouble, not as a wrong answer. A
// study that cannot tell these apart scores its own flakiness as model failure.
func TestEvaluateSeparatesInfrastructureFromSemantics(t *testing.T) {
	t.Setenv("PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE", "1")
	t.Setenv("PROOF_FENCE_TEST_TIMEOUT", "1ns")
	root := repoRoot(t)
	d := filepath.Join(t.TempDir(), "starter")
	if err := Materialize(root, "PF-001", d); err != nil {
		t.Fatal(err)
	}
	e, err := EvaluateCase(root, "PF-001", d)
	if err != nil {
		t.Fatal(err)
	}
	if e.Counts.Infrastructure == 0 {
		t.Fatalf("a 1ns execution budget produced no INFRASTRUCTURE record: %+v", e.Counts)
	}
	if e.Complete {
		t.Fatal("a vector with infrastructure records is not complete")
	}
	if e.Counts.Fail > 0 {
		t.Fatalf("timeouts were scored as semantic failures: %+v", e.Counts)
	}
	if e.Verdict != VerdictInfrastructure {
		t.Fatalf("verdict=%s, want INFRASTRUCTURE", e.Verdict)
	}
}

// TestEvaluateRecordsUncompilableCandidateAsSemantic keeps the other half of the
// distinction honest: code that does not compile is the candidate's problem.
func TestEvaluateRecordsUncompilableCandidateAsSemantic(t *testing.T) {
	t.Setenv("PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE", "1")
	root := repoRoot(t)
	d := writeCandidate(t, root, "PF-001", "package challenge\n\nfunc Broken() { this is not go }\n")
	e, err := EvaluateCase(root, "PF-001", d)
	if err != nil {
		t.Fatal(err)
	}
	if e.Verdict != VerdictFail || e.Counts.Infrastructure != 0 {
		t.Fatalf("uncompilable candidate: verdict=%s counts=%+v", e.Verdict, e.Counts)
	}
	if e.Counts.Fail != e.Counts.Total || e.Counts.Total == 0 {
		t.Fatalf("expected a full-length FAIL vector, got %+v", e.Counts)
	}
}

// TestEvaluationRecordIsMachineReadable checks the published shape survives a
// round trip, because an external runner reads it, not a human.
func TestEvaluationRecordIsMachineReadable(t *testing.T) {
	t.Setenv("PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE", "1")
	root := repoRoot(t)
	d := filepath.Join(t.TempDir(), "starter")
	if err := Materialize(root, "PF-009", d); err != nil {
		t.Fatal(err)
	}
	e, err := EvaluateCase(root, "PF-009", d)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var back CaseEvaluation
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.SchemaVersion != EvaluationSchemaVersion {
		t.Fatalf("schema_version=%q", back.SchemaVersion)
	}
	if len(back.Fixtures) != len(e.Fixtures) {
		t.Fatalf("round trip lost fixtures: %d vs %d", len(back.Fixtures), len(e.Fixtures))
	}
	for _, f := range back.Fixtures {
		if f.Name == "" || f.Verdict == "" {
			t.Fatalf("fixture record without a name or verdict: %+v", f)
		}
	}
}

// TestManifestRejectsCredentials checks the one thing a provenance record must
// never do. The check is deliberately shallow; it catches an obvious mistake,
// and the README says so rather than claiming a secret scanner.
func TestManifestRejectsCredentials(t *testing.T) {
	for _, m := range []RunManifest{
		{Notes: "api_key for the run"},
		{APIRequestID: "sk-abcdefghijklmnopqrstuvwx"},
		{ModelID: "AKIAIOSFODNN7EXAMPLE"},
		{Notes: "-----BEGIN RSA PRIVATE KEY-----"},
	} {
		if err := m.Validate(); err == nil {
			t.Fatalf("manifest %+v was accepted; it looks like it carries a credential", m)
		}
	}
}

// TestManifestValidatesShapes keeps a runner from recording a digest or a
// timestamp that is not one.
func TestManifestValidatesShapes(t *testing.T) {
	if err := (&RunManifest{PromptSHA256: "not-a-digest"}).Validate(); err == nil {
		t.Fatal("accepted a prompt_sha256 that is not 64 hex characters")
	}
	if err := (&RunManifest{StartedAtUTC: "yesterday"}).Validate(); err == nil {
		t.Fatal("accepted a started_at_utc that is not RFC 3339")
	}
	ok := &RunManifest{
		Provider:     "example",
		ModelID:      "example-model-2026-01-01",
		PromptSHA256: strings.Repeat("a", 64),
		StartedAtUTC: "2026-09-17T00:00:00Z",
	}
	if err := ok.Validate(); err != nil {
		t.Fatalf("rejected a valid manifest: %v", err)
	}
}

// TestManifestTemplateIsValidAndCarriesNoModelClaim checks that the template
// ProofFence prints records only what ProofFence can actually observe.
func TestManifestTemplateIsValidAndCarriesNoModelClaim(t *testing.T) {
	root := repoRoot(t)
	m := ManifestTemplate(root)
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	if m.Provider != "" || m.ModelID != "" || m.ReasoningEffort != "" {
		t.Fatalf("template asserts model provenance ProofFence cannot observe: %+v", m)
	}
	if m.GoVersion == "" || m.OS == "" || m.Arch == "" {
		t.Fatalf("template omits the apparatus identity it can observe: %+v", m)
	}
}

// TestLoadManifestRejectsUnknownFields stops a runner's private field from being
// silently dropped and then reported as if ProofFence had recorded it.
func TestLoadManifestRejectsUnknownFields(t *testing.T) {
	p := filepath.Join(t.TempDir(), "m.json")
	if err := os.WriteFile(p, []byte(`{"provider":"x","temperature":0.7}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(p); err == nil {
		t.Fatal("accepted a manifest carrying a field the schema does not model")
	}
}

// TestEveryCommittedMutantIsClassified keeps the mutation report honest. A
// mutant with no recorded kind cannot be reasoned about, and an entry naming no
// mutant is a stale claim.
func TestEveryCommittedMutantIsClassified(t *testing.T) {
	root := repoRoot(t)
	cs, err := LoadCases(root)
	if err != nil {
		t.Fatal(err)
	}
	valid := map[string]bool{
		MutantDegenerate:         true,
		MutantMutation:           true,
		MutantAlternativeReading: true,
	}
	total := 0
	for _, c := range cs {
		names, err := LoadMutantNames(root, c)
		if err != nil {
			t.Fatal(err)
		}
		mc, err := LoadMutantClassification(root, c)
		if err != nil {
			t.Fatal(err)
		}
		if len(names) == 0 {
			if len(mc.Mutants) != 0 {
				t.Fatalf("%s commits no mutants but classifies %d", c.ID, len(mc.Mutants))
			}
			continue
		}
		if !mc.Adjudicated {
			t.Fatalf("%s classification does not record that it is an adjudicated judgment", c.ID)
		}
		have := map[string]bool{}
		for _, n := range names {
			have[n] = true
			total++
			note, ok := mc.Mutants[n]
			if !ok {
				t.Fatalf("%s mutant %q has no recorded classification", c.ID, n)
			}
			if !valid[note.Kind] {
				t.Fatalf("%s mutant %q has kind %q, which is not one of the recorded kinds", c.ID, n, note.Kind)
			}
		}
		for n := range mc.Mutants {
			if !have[n] {
				t.Fatalf("%s classifies %q, which is not a committed mutant", c.ID, n)
			}
		}
	}
	if total == 0 {
		t.Fatal("no committed mutants found")
	}
}

// TestMutationReportStatesWhatItDoesNotProve checks the caveat travels with the
// number. The v0.2 finding that motivates it is that this suite reported 78
// killed / 0 survivors on an artifact an independent audit then found to have 20
// underspecified fixtures.
func TestMutationReportStatesWhatItDoesNotProve(t *testing.T) {
	for _, want := range []string{
		"does not show",
		"the grader is correct",
		"defensible alternative reading",
		"independent specification audit",
	} {
		if !strings.Contains(mutationCaveat, want) {
			t.Fatalf("mutation caveat does not say %q:\n%s", want, mutationCaveat)
		}
	}
}

// TestFixtureDistributionIsDerivedFromGraders checks that the published table is
// computed. v0.2 documented PF-013 as 1/2/1/15 against a stated total of 24, a
// row that sums to 19; a hand-maintained table is a table that drifts.
func TestFixtureDistributionIsDerivedFromGraders(t *testing.T) {
	root := repoRoot(t)
	cs, err := LoadCases(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		if c.DecisionModel != "four-valued-authority" {
			continue
		}
		d, err := FixtureDistribution(root, c)
		if err != nil {
			t.Fatal(err)
		}
		if got := d.Grant + d.Retain + d.Revoke + d.Quarantine + d.Meta; got != d.Total {
			t.Fatalf("%s distribution sums to %d against a total of %d: %+v", c.ID, got, d.Total, d)
		}
		names, err := trustedTestNames(filepath.Join(root, "cases", c.ID, c.Grader))
		if err != nil {
			t.Fatal(err)
		}
		if d.Total != len(names) {
			t.Fatalf("%s distribution total %d, grader declares %d fixtures", c.ID, d.Total, len(names))
		}
	}
}

// TestPublishedDistributionTableMatchesTheGraders is the drift guard: the table
// in docs/CONSTRUCT_VALIDITY.md must be the table the graders produce.
func TestPublishedDistributionTableMatchesTheGraders(t *testing.T) {
	root := repoRoot(t)
	want, err := DistributionTable(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "docs", "CONSTRUCT_VALIDITY.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(want)) {
		t.Fatalf("docs/CONSTRUCT_VALIDITY.md does not contain the derived distribution table.\n"+
			"Regenerate it with `go run ./cmd/proof-fence distribution`:\n%s", want)
	}
}
