// Command bronze is the Bronze language tool: it checks and runs .brz files.
package main

import (
	"os"

	"github.com/AxelGard/bronze/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
