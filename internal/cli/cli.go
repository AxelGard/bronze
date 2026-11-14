// Package cli implements the bronze command: run, check and version.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/check"
	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/interp"
	"github.com/AxelGard/bronze/internal/lsp"
	"github.com/AxelGard/bronze/internal/parser"
)

// Version is the version of Bronze this tool implements.
const Version = "0.0.0-dev"

// PanicExit is the exit code of a program that panics.
const PanicExit = 101

const usage = `bronze: the Bronze language tool

Usage:
  bronze run <file.brz>     check, then run a program
  bronze check <file.brz>   check syntax and types only
  bronze lsp                run the language server on stdin and stdout
  bronze version            print the version
`

// Main runs the bronze command with the given arguments (without the
// program name) and returns the exit code.
func Main(args []string, stdout, stderr io.Writer) int {
	return MainIO(args, os.Stdin, stdout, stderr)
}

// MainIO is Main with stdin, which `bronze lsp` reads.
func MainIO(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}

	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, "bronze", Version)
		return 0
	case "lsp":
		return lsp.Serve(stdin, stdout, stderr, Version)
	case "run", "check":
		if len(args) != 2 {
			fmt.Fprint(stderr, usage)
			return 2
		}
		file, src, ok := load(args[1], stderr)
		if !ok {
			return 1
		}
		if args[0] == "check" {
			return 0
		}
		code, p := interp.Run(file, src, stdout)
		if p != nil {
			p.Format(stderr)
			return PanicExit
		}
		return code
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "bronze: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

// load reads, parses and checks a file. It prints all diagnostics to
// stderr and reports whether the file is free of errors.
func load(path string, stderr io.Writer) (*ast.File, *diag.Source, bool) {
	text, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "bronze: %v\n", err)
		return nil, nil, false
	}
	src := diag.NewSource(path, text)
	file, diags := parser.Parse(src)
	_, cdiags := check.Check(file, src)
	diags = append(diags, cdiags...)
	if len(diags) > 0 {
		diag.Format(stderr, src, diags)
		return nil, nil, false
	}
	return file, src, true
}
