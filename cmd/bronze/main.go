// Command bronze is the Bronze language tool: it checks and runs .brz files.
package main

import (
	"fmt"
	"os"
)

const version = "0.0.0-dev"

const usage = `bronze: the Bronze language tool

Usage:
  bronze run <file.brz>     check, then run a program
  bronze check <file.brz>   check syntax and types only
  bronze version            print the version
`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	switch args[0] {
	case "version":
		fmt.Println("bronze", version)
		return 0
	case "run", "check":
		if len(args) != 2 {
			fmt.Fprint(os.Stderr, usage)
			return 2
		}
		fmt.Fprintf(os.Stderr, "bronze %s: not implemented yet\n", args[0])
		return 1
	case "help", "-h", "--help":
		fmt.Print(usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "bronze: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}
