package main

import (
	"fmt"
	"os"

	"github.com/wesamabed/proof-fence/internal/bench"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: proof-fence <list|materialize|grade|selftest> [args]")
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
			fmt.Printf("%s\t%s\t%s\n", c.ID, c.Category, c.Title)
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
		out, err := bench.Grade(root, os.Args[2], os.Args[3])
		fmt.Print(out)
		if err != nil {
			os.Exit(1)
		}
	case "selftest":
		if err := bench.SelfTest(root, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}
