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
	"sort"
	"strconv"
	"strings"
	"time"
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

const submissionBoundary = `## Submission boundary

ProofFence v0.1 is a source-edit benchmark. Modify only challenge.go.
Do not add files or modify go.mod. The grader reconstructs the module from
the trusted starter, accepts only challenge.go from the candidate, and applies
a small safe-import/source policy before executing trusted tests.

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
			if strings.HasPrefix(strings.TrimSpace(c.Text), "//go:") {
				return fmt.Errorf("candidate compiler directive is not allowed: %s", strings.TrimSpace(c.Text))
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
			return fmt.Errorf("candidate import %q is outside the v0.1 safe-import policy", path)
		}
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
			return fmt.Errorf("unexpected candidate path %q; v0.1 accepts only challenge.go with the trusted go.mod/TASK.md", e.Name())
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

func runTrustedTest(tmp, name string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pattern := "^" + regexp.QuoteMeta(name) + "$"
	cmd := exec.CommandContext(ctx, "go", "test", "-count=1", "-timeout=10s", "-run", pattern, ".")
	cmd.Dir = tmp
	cmd.Env = controlledGoEnv()
	b, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return b, fmt.Errorf("trusted test %s exceeded controller timeout", name)
	}
	if err != nil {
		return b, fmt.Errorf("trusted test %s failed: %w", name, err)
	}
	return b, nil
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

	if err := copyValidatedCandidateSource(solution, filepath.Join(dir, c.Starter), tmp); err != nil {
		return fmt.Sprintf("[%s] FAIL\n", c.ID), err
	}
	if err := copyTree(filepath.Join(dir, c.Grader), tmp, true); err != nil {
		return "", err
	}

	var transcript strings.Builder
	for _, name := range expectedTests {
		b, err := runTrustedTest(tmp, name)
		fmt.Fprintf(&transcript, "=== TRUSTED %s ===\n%s", name, string(b))
		if err != nil {
			return fmt.Sprintf("[%s] FAIL\n%s", c.ID, transcript.String()), err
		}
	}
	return fmt.Sprintf("[%s] PASS\n%s", c.ID, transcript.String()), nil
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
