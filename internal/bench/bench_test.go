package bench

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalog(t *testing.T) {
	root, err := FindRepoRoot("")
	if err != nil {
		t.Fatal(err)
	}
	cs, err := LoadCases(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 10 {
		t.Fatalf("got %d cases, want 10", len(cs))
	}
	for i, c := range cs {
		if c.PublicSafety != "synthetic" {
			t.Fatalf("case %s not synthetic", c.ID)
		}
		want := "PF-" + []string{"001", "002", "003", "004", "005", "006", "007", "008", "009", "010"}[i]
		if c.ID != want {
			t.Fatalf("case %d id=%s want=%s", i, c.ID, want)
		}
	}
}

func TestMaterializeRejectsExistingDestination(t *testing.T) {
	root, _ := FindRepoRoot("")
	d := filepath.Join(t.TempDir(), "exists")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Materialize(root, "PF-001", d); err == nil {
		t.Fatal("expected existing-destination error")
	}
}

func TestGradeRejectsCandidateTestMainBypass(t *testing.T) {
	root, _ := FindRepoRoot("")
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
	out, err := gradeUnsandboxed(root, "PF-001", d)
	if err == nil {
		t.Fatalf("TestMain bypass unexpectedly passed:\n%s", out)
	}
}

func writeMutant(t *testing.T, root, id, src string) string {
	t.Helper()
	d := filepath.Join(t.TempDir(), id)
	if err := Materialize(root, id, d); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "challenge.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return d
}

func requireMutantKilled(t *testing.T, root, id, src string) {
	t.Helper()
	d := writeMutant(t, root, id, src)
	if out, err := gradeUnsandboxed(root, id, d); err == nil {
		t.Fatalf("%s prompt-violating mutant survived:\n%s", id, out)
	}
}

func TestGradersKillPromptViolatingFamilyMutants(t *testing.T) {
	root, _ := FindRepoRoot("")
	t.Run("admission", func(t *testing.T) {
		requireMutantKilled(t, root, "PF-002", `package challenge
import("bytes";"encoding/json";"errors";"fmt";"io")
type Binding struct { RunID string `+"`json:\"run_id\"`"+`; Authorized bool `+"`json:\"authorized\"`"+` }
func ParseBinding(data []byte)(Binding,error){d:=json.NewDecoder(bytes.NewReader(data));t,e:=d.Token();if e!=nil||t!=json.Delim('{'){return Binding{},errors.New("object")};seen:=map[string]bool{};var b Binding;for d.More(){k,_:=d.Token();s:=k.(string);if s=="Run_ID"{return Binding{},errors.New("one blocked alias")};if seen[s]{return Binding{},fmt.Errorf("dup")};seen[s]=true;switch s{case"run_id":e=d.Decode(&b.RunID);case"authorized","Authorized":e=d.Decode(&b.Authorized);default:return Binding{},fmt.Errorf("unknown")};if e!=nil{return Binding{},e}};if _,e=d.Token();e!=nil{return Binding{},e};if _,e=d.Token();e!=io.EOF{return Binding{},errors.New("trailing")};return b,nil}`)
	})
	t.Run("coherence", func(t *testing.T) {
		requireMutantKilled(t, root, "PF-004", `package challenge
type Outcome string
const(Succeeded Outcome="SUCCEEDED";Denied Outcome="DENIED";Failed Outcome="FAILED")
type DeleteEvidence struct{Outcome Outcome;ErrorCode string;ResponseDeleted bool}
func DeletionComplete(e DeleteEvidence)bool{return e.Outcome!=Denied&&e.ErrorCode==""&&e.ResponseDeleted}`)
	})
	t.Run("closure", func(t *testing.T) {
		requireMutantKilled(t, root, "PF-005", `package challenge
type Fact struct{Known bool;Value bool}
type Evidence struct{ProducerPresent bool;Authenticated bool;DerivedSafe bool}
func NetworkSafety(e Evidence)Fact{if !e.Authenticated{return Fact{}};return Fact{Known:true,Value:e.DerivedSafe}}`)
	})
	t.Run("provenance", func(t *testing.T) {
		requireMutantKilled(t, root, "PF-007", `package challenge
type Reachability struct{CallerSaysIsolated bool;ObservationPresent bool;ObservationAuthenticated bool;ObservedIsolated bool}
func IsolationProven(r Reachability)bool{return r.ObservationAuthenticated&&r.ObservedIsolated}`)
	})
	t.Run("execution-proof", func(t *testing.T) {
		requireMutantKilled(t, root, "PF-010", `package challenge
type ExecutionEvidence struct{ClaimedRun []string;ClaimedPass []string;TrustedObservedRun []string;TrustedObservedPass []string}
func ExecutionProven(required []string,e ExecutionEvidence)bool{pass:=map[string]bool{};for _,x:=range e.TrustedObservedPass{pass[x]=true};for _,x:=range required{if !pass[x]{return false}};return true}`)
	})
}
