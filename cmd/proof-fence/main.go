package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/wesamabed/proof-fence/internal/bench"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: proof-fence <list|materialize|grade|evaluate|evaluate-suite|manifest|distribution|selftest|mutation|toolchain> [args]")
	fmt.Fprintln(os.Stderr, "  grade <case-id> <solution-dir>            fail-fast single-case check (stops at the first failing fixture)")
	fmt.Fprintln(os.Stderr, "  evaluate <case-id> <solution-dir>         full-vector evaluation of one case (JSON)")
	fmt.Fprintln(os.Stderr, "  evaluate-suite <solutions-root>           full-vector evaluation of every case with a submission (JSONL)")
	fmt.Fprintln(os.Stderr, "  manifest                                  print a blank run manifest for an external runner to fill in")
	fmt.Fprintln(os.Stderr, "  distribution                              print the four-valued fixture distribution derived from the graders")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Formal evaluation must use evaluate/evaluate-suite. grade truncates the fixture")
	fmt.Fprintln(os.Stderr, "vector at the first semantic failure and cannot support a study record.")
}

// openOut returns the destination for a machine-readable record.
func openOut(path string) (io.WriteCloser, error) {
	if path == "" {
		return nopCloser{os.Stdout}, nil
	}
	return os.Create(path)
}

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

func loadManifestFlag(path string) *bench.RunManifest {
	if path == "" {
		return nil
	}
	m, err := bench.LoadManifest(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return m
}

// exitForVerdict maps a case verdict onto a process exit status. INFRASTRUCTURE
// gets its own code so a study can drop a trial instead of scoring it against a
// submission.
func exitForVerdict(v bench.Verdict) int {
	switch v {
	case bench.VerdictPass:
		return 0
	case bench.VerdictFail:
		return 1
	default:
		return 3
	}
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	root, err := bench.FindRepoRoot("")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	switch os.Args[1] {
	case "list":
		cases, err := bench.LoadCases(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, c := range cases {
			fmt.Printf("%s\t%s\t%s\t%s\n", c.ID, c.Category, c.DecisionModel, c.Title)
		}
	case "toolchain":
		tc, err := bench.DetectToolchain()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("%s\n", tc)
	case "manifest":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(bench.ManifestTemplate(root)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "materialize":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		if err := bench.Materialize(root, os.Args[2], os.Args[3]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "grade":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		res, err := bench.Grade(root, os.Args[2], os.Args[3])
		fmt.Print(res.Transcript)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(exitForVerdict(res.Verdict))
	case "evaluate":
		fs := flag.NewFlagSet("evaluate", flag.ExitOnError)
		manifestPath := fs.String("manifest", "", "run manifest JSON supplied by the external runner")
		outPath := fs.String("out", "", "write the record here instead of stdout")
		_ = fs.Parse(os.Args[2:])
		if fs.NArg() != 2 {
			usage()
			os.Exit(2)
		}
		m := loadManifestFlag(*manifestPath)
		e, err := bench.EvaluateCase(root, fs.Arg(0), fs.Arg(1))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if m != nil {
			m.CaseID = e.CaseID
			e.Manifest = m
		}
		w, err := openOut(*outPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(e); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		w.Close()
		fmt.Fprintf(os.Stderr, "[proof-fence CONTROLLER] %s verdict=%s fixtures=%d pass=%d fail=%d infrastructure=%d complete=%t\n",
			e.CaseID, e.Verdict, e.Counts.Total, e.Counts.Pass, e.Counts.Fail, e.Counts.Infrastructure, e.Complete)
		os.Exit(exitForVerdict(e.Verdict))
	case "evaluate-suite":
		fs := flag.NewFlagSet("evaluate-suite", flag.ExitOnError)
		manifestPath := fs.String("manifest", "", "run manifest JSON supplied by the external runner")
		outPath := fs.String("out", "", "write the JSONL record stream here instead of stdout")
		_ = fs.Parse(os.Args[2:])
		if fs.NArg() != 1 {
			usage()
			os.Exit(2)
		}
		m := loadManifestFlag(*manifestPath)
		w, err := openOut(*outPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		s, err := bench.EvaluateSuite(root, fs.Arg(0), m, w)
		w.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "[proof-fence CONTROLLER] evaluate-suite cases=%d passed=%d failed=%d infrastructure=%d fixtures=%d pass=%d fail=%d infra=%d\n",
			len(s.Cases), s.CasesPassed, s.CasesFailed, s.CasesInfra,
			s.Counts.Total, s.Counts.Pass, s.Counts.Fail, s.Counts.Infrastructure)
		switch {
		case s.CasesFailed > 0:
			os.Exit(1)
		case s.CasesInfra > 0:
			os.Exit(3)
		}
	case "distribution":
		t, err := bench.DistributionTable(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print(t)
	case "selftest":
		if err := bench.SelfTest(root, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "mutation":
		if err := bench.MutationSuite(root, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}
