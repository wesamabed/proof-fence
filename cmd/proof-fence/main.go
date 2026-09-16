package main

import (
	"fmt"
	"os"

	"github.com/wesamabed/proof-fence/internal/bench"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: proof-fence <list|materialize|grade|selftest|mutation|toolchain> [args]")
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
		// INFRASTRUCTURE is reported with its own exit code so a study can drop the
		// trial instead of scoring it as a model failure.
		switch res.Verdict {
		case bench.VerdictPass:
			os.Exit(0)
		case bench.VerdictFail:
			os.Exit(1)
		default:
			os.Exit(3)
		}
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
