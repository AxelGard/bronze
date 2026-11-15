package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/AxelGard/bronze/editors"
)

const setupUsage = `Usage:
  bronze setup nvim [--dir <dir>]

Sets up Neovim (0.11 or later) for Bronze: highlighting with the
tree-sitter parser, and the language server. It installs a package into
Neovim's data directory (<data>/site/pack/bronze/start/bronze), and
compiles the parser there with the C compiler (cc, or $CC).
nvim-treesitter is not needed.

  --dir <dir>   install the package into <dir> instead

Run it again after updating bronze.
`

// setupMarker is the file that marks a package directory as made by
// `bronze setup`, so that it can be replaced safely.
const setupMarker = "version.txt"

func setup(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "nvim" {
		fmt.Fprint(stderr, setupUsage)
		return 2
	}
	dir := ""
	rest := args[1:]
	for len(rest) > 0 {
		switch {
		case rest[0] == "--dir" && len(rest) > 1:
			dir, rest = rest[1], rest[2:]
		case rest[0] == "-h" || rest[0] == "--help":
			fmt.Fprint(stdout, setupUsage)
			return 0
		default:
			fmt.Fprint(stderr, setupUsage)
			return 2
		}
	}

	nvim, _ := exec.LookPath("nvim")
	if dir == "" {
		data, err := nvimDataDir(nvim)
		if err != nil {
			fmt.Fprintf(stderr, "bronze setup: %v\n", err)
			return 1
		}
		dir = filepath.Join(data, "site", "pack", "bronze", "start", "bronze")
	}

	if err := installNvim(dir, compileParser); err != nil {
		fmt.Fprintf(stderr, "bronze setup: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "installed Bronze for Neovim into %s\n", dir)

	// Hints about the rest of the setup.
	if _, err := exec.LookPath("bronze"); err != nil {
		fmt.Fprintf(stdout, "\nnote: `bronze` is not on your PATH, so Neovim can't start the language server.\n"+
			"Add the directory it is in to PATH, for example in ~/.zshrc or ~/.bashrc:\n    export PATH=\"%s:$PATH\"\n", binDir())
	}
	if nvim == "" {
		fmt.Fprintln(stdout, "\nnote: `nvim` is not on your PATH; the package is ready for when it is.")
		return 0
	}
	for _, other := range otherParsers(nvim, dir) {
		fmt.Fprintf(stdout, "\nnote: Neovim also finds another Bronze parser, which may be used instead of this one:\n    %s\n", other)
		if strings.Contains(other, "nvim-treesitter") {
			fmt.Fprintln(stdout, "It is from nvim-treesitter; remove it with :TSUninstall bronze, and remove the old\neditors/nvim plugin from your plugin manager.")
		}
	}
	fmt.Fprintln(stdout, "\nOpen a .brz file to try it: :InspectTree shows the syntax tree, and\n:checkhealth vim.lsp the language server.")
	return 0
}

// binDir is the directory of the running bronze, for the PATH hint.
func binDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "$HOME/.local/bin"
}

// nvimDataDir returns Neovim's data directory (stdpath("data")), asking
// nvim if it is installed.
func nvimDataDir(nvim string) (string, error) {
	if nvim != "" {
		out, err := runNvim(nvim, true, `io.stdout:write(vim.fn.stdpath("data"))`)
		if err == nil && strings.TrimSpace(out) != "" {
			return strings.TrimSpace(out), nil
		}
	}
	name := os.Getenv("NVIM_APPNAME")
	if name == "" {
		name = "nvim"
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, name), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("can't find the home directory; use --dir")
	}
	return filepath.Join(home, ".local", "share", name), nil
}

// otherParsers returns the Bronze parsers Neovim finds on its runtime path
// outside dir, with the user's configuration loaded.
func otherParsers(nvim, dir string) []string {
	out, err := runNvim(nvim, false, `for _, p in ipairs(vim.api.nvim_get_runtime_file("parser/bronze.so", true)) do io.stdout:write(p, "\n") end`)
	if err != nil {
		return nil
	}
	want, _ := filepath.EvalSymlinks(dir)
	var others []string
	for _, p := range strings.Split(strings.TrimSpace(out), "\n") {
		if p == "" {
			continue
		}
		real, _ := filepath.EvalSymlinks(p)
		if !strings.HasPrefix(real, want+string(filepath.Separator)) && !strings.HasPrefix(p, dir+string(filepath.Separator)) {
			others = append(others, p)
		}
	}
	return others
}

// runNvim runs a Lua snippet in a headless Neovim and returns what it
// writes to stdout. clean skips the user's configuration.
func runNvim(nvim string, clean bool, lua string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	args := []string{"--headless"}
	if clean {
		args = append(args, "--clean")
	}
	args = append(args, "-c", "lua "+lua, "-c", "qa!")
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, nvim, args...)
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

// installNvim writes the Neovim package into dir: the plugin, the
// highlight queries, the parser (compiled by compile) and version.txt. An
// existing package made by setup is replaced.
func installNvim(dir string, compile func(src, out string) error) error {
	if _, err := os.Stat(dir); err == nil {
		if _, err := os.Stat(filepath.Join(dir, setupMarker)); err != nil {
			return fmt.Errorf("%s already exists, and was not made by `bronze setup`; remove it first, or use --dir", dir)
		}
	}
	// Build into a new directory next to it, then swap, so that a failed
	// compile leaves a working old setup in place.
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, ".bronze-setup-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	files := map[string]string{ // embedded path -> path in the package
		"nvim/plugin/bronze.lua":                      "plugin/bronze.lua",
		"nvim/ftdetect/bronze.lua":                    "ftdetect/bronze.lua",
		"tree-sitter-bronze/queries/highlights.scm":   "queries/bronze/highlights.scm",
		"tree-sitter-bronze/src/parser.c":             "src/parser.c",
		"tree-sitter-bronze/src/tree_sitter/parser.h": "src/tree_sitter/parser.h",
		"tree-sitter-bronze/src/tree_sitter/alloc.h":  "src/tree_sitter/alloc.h",
		"tree-sitter-bronze/src/tree_sitter/array.h":  "src/tree_sitter/array.h",
	}
	for from, to := range files {
		data, err := fs.ReadFile(editors.Files, from)
		if err != nil {
			return err
		}
		path := filepath.Join(tmp, to)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Join(tmp, "parser"), 0o755); err != nil {
		return err
	}
	if err := compile(filepath.Join(tmp, "src"), filepath.Join(tmp, "parser", "bronze.so")); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(tmp, "src")); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, setupMarker), []byte("bronze "+Version+"\n"), 0o644); err != nil {
		return err
	}

	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.Rename(tmp, dir)
}

// compileParser compiles the parser's C source in src into a shared
// library at out, with $CC or cc.
func compileParser(src, out string) error {
	cc := os.Getenv("CC")
	if cc == "" {
		cc = "cc"
	}
	if _, err := exec.LookPath(cc); err != nil {
		help := "install a C compiler"
		switch runtime.GOOS {
		case "darwin":
			help = "install the Command Line Tools: xcode-select --install"
		case "linux":
			help = "install gcc or clang (for example: sudo apt install build-essential)"
		}
		return fmt.Errorf("the parser needs a C compiler, and %q was not found; %s", cc, help)
	}
	var log bytes.Buffer
	cmd := exec.Command(cc, "-O2", "-shared", "-fPIC", "-I", src, "-o", out, filepath.Join(src, "parser.c"))
	cmd.Stdout, cmd.Stderr = &log, &log
	if err := cmd.Run(); err != nil {
		msg := fmt.Sprintf("compiling the parser failed:\n%s", log.String())
		if runtime.GOOS == "darwin" && (strings.Contains(log.String(), "tapi") || strings.Contains(log.String(), "unknown architecture")) {
			msg += "\nThe selected Xcode is older than the macOS SDK. Select the Command Line Tools, or update Xcode:\n" +
				"    sudo xcode-select -s /Library/Developer/CommandLineTools"
		}
		return errors.New(msg)
	}
	return nil
}
