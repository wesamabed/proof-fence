package bench

import (
	"bufio"
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
	"sort"
	"strings"
)

type Case struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Category     string   `json:"category"`
	Invariant    string   `json:"invariant"`
	Starter      string   `json:"starter"`
	Grader       string   `json:"grader"`
	Reference    string   `json:"reference"`
	PublicSafety string   `json:"public_safety"`
	Limitations  []string `json:"limitations"`
}

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
	return os.WriteFile(filepath.Join(dest, "TASK.md"), task, 0o644)
}

func copyCandidate(src, dst string) error { return copyTree(src, dst, false) }

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
			if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") {
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

type goTestEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

type trustedTestState struct {
	ran     bool
	passed  bool
	failed  bool
	skipped bool
}

func verifyTrustedTestEvents(out []byte, expected []string) error {
	states := make(map[string]*trustedTestState, len(expected))
	for _, name := range expected {
		states[name] = &trustedTestState{}
	}

	s := bufio.NewScanner(strings.NewReader(string(out)))
	// Test output can contain long lines; keep the parser bounded but well above these tiny cases.
	s.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		var ev goTestEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return fmt.Errorf("malformed go test -json output: %w", err)
		}
		st, ok := states[ev.Test]
		if !ok {
			continue
		}
		switch ev.Action {
		case "run":
			st.ran = true
		case "pass":
			st.passed = true
		case "fail":
			st.failed = true
		case "skip":
			st.skipped = true
		}
	}
	if err := s.Err(); err != nil {
		return err
	}

	var incomplete []string
	for _, name := range expected {
		st := states[name]
		if !st.ran || !st.passed || st.failed || st.skipped {
			incomplete = append(incomplete, name)
		}
	}
	if len(incomplete) != 0 {
		return fmt.Errorf("trusted grader assertions did not all run and pass: %s", strings.Join(incomplete, ", "))
	}
	return nil
}

func gradeUnsandboxed(root, id, solution string) (string, error) {
	c, dir, err := findCase(root, id)
	if err != nil {
		return "", err
	}
	expectedTests, err := trustedTestNames(filepath.Join(dir, c.Grader))
	if err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp("", "proof-fence-grade-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	if err := copyCandidate(solution, tmp); err != nil {
		return "", err
	}
	_ = os.Remove(filepath.Join(tmp, "TASK.md"))
	if err := copyTree(filepath.Join(dir, c.Grader), tmp, true); err != nil {
		return "", err
	}
	cmd := exec.Command("go", "test", "-json", ".")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "GOWORK=off")
	b, runErr := cmd.CombinedOutput()

	assertionErr := verifyTrustedTestEvents(b, expectedTests)
	passed := runErr == nil && assertionErr == nil
	out := fmt.Sprintf("[%s] %s\n%s", c.ID, map[bool]string{true: "PASS", false: "FAIL"}[passed], string(b))
	if runErr != nil {
		return out, runErr
	}
	if assertionErr != nil {
		return out, assertionErr
	}
	return out, nil
}

func Grade(root, id, solution string) (string, error) {
	if os.Getenv("PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE") != "1" {
		return "", errors.New("refusing to execute candidate code without explicit opt-in; run in a disposable sandbox and set PROOF_FENCE_ALLOW_UNSANDBOXED_GRADE=1")
	}
	return gradeUnsandboxed(root, id, solution)
}

func SelfTest(root string, w io.Writer) error {
	cases, err := LoadCases(root)
	if err != nil {
		return err
	}
	for _, c := range cases {
		tmp, err := os.MkdirTemp("", "proof-fence-selftest-*")
		if err != nil {
			return err
		}
		if err := Materialize(root, c.ID, filepath.Join(tmp, "starter")); err != nil {
			os.RemoveAll(tmp)
			return err
		}
		if out, err := gradeUnsandboxed(root, c.ID, filepath.Join(tmp, "starter")); err == nil {
			os.RemoveAll(tmp)
			return fmt.Errorf("%s starter unexpectedly passed:\n%s", c.ID, out)
		}
		_, dir, _ := findCase(root, c.ID)
		refDir := filepath.Join(tmp, "reference")
		if err := copyTree(filepath.Join(dir, c.Starter), refDir, false); err != nil {
			os.RemoveAll(tmp)
			return err
		}
		if err := copyTree(filepath.Join(dir, c.Reference), refDir, true); err != nil {
			os.RemoveAll(tmp)
			return err
		}
		out, err := gradeUnsandboxed(root, c.ID, refDir)
		if err != nil {
			os.RemoveAll(tmp)
			return fmt.Errorf("%s reference failed:\n%s", c.ID, out)
		}
		fmt.Fprintf(w, "%s starter=FAIL reference=PASS\n", c.ID)
		os.RemoveAll(tmp)
	}
	return nil
}
