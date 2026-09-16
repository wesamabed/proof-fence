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
