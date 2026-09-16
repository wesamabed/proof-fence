package bench

import (
	"encoding/json"
	"errors"
	"fmt"
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

func gradeUnsandboxed(root, id, solution string) (string, error) {
	c, dir, err := findCase(root, id)
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
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "GOWORK=off")
	b, runErr := cmd.CombinedOutput()
	return fmt.Sprintf("[%s] %s\n%s", c.ID, map[bool]string{true: "PASS", false: "FAIL"}[runErr == nil], string(b)), runErr
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
