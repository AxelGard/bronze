// Package tests runs the golden tests described in README.md.
package tests

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AxelGard/bronze/internal/cli"
)

// pending lists golden tests that need features not implemented yet. They
// are skipped; remove them from here as the features land.
var pending = map[string]string{}

func TestRun(t *testing.T) {
	golden(t, "run", func(t *testing.T, base string) {
		stdout, stderr, code := bronze("run", base+".brz")
		if code != 0 {
			t.Errorf("exit code %d, want 0\nstderr:\n%s", code, stderr)
		}
		if want := read(t, base+".out"); stdout != want {
			t.Errorf("stdout:\n%s\nwant:\n%s", stdout, want)
		}
	})
}

func TestErrors(t *testing.T) {
	golden(t, "errors", func(t *testing.T, base string) {
		stdout, stderr, code := bronze("check", base+".brz")
		if code != 1 {
			t.Errorf("exit code %d, want 1", code)
		}
		if want := read(t, base+".err"); !strings.HasPrefix(stderr, want) {
			t.Errorf("stderr:\n%s\nshould start with:\n%s", stderr, want)
		}
		if stdout != "" {
			t.Errorf("unexpected stdout:\n%s", stdout)
		}
	})
}

func TestPanics(t *testing.T) {
	golden(t, "panics", func(t *testing.T, base string) {
		stdout, stderr, code := bronze("run", base+".brz")
		if code != cli.PanicExit {
			t.Errorf("exit code %d, want %d\nstderr:\n%s", code, cli.PanicExit, stderr)
		}
		if want := read(t, base+".out"); stdout != want {
			t.Errorf("stdout:\n%s\nwant:\n%s", stdout, want)
		}
		if want := read(t, base+".err"); !strings.HasPrefix(stderr, want) {
			t.Errorf("stderr:\n%s\nshould start with:\n%s", stderr, want)
		}
	})
}

// golden runs test for every .brz file in dir, passing the path without
// the extension.
func golden(t *testing.T, dir string, test func(t *testing.T, base string)) {
	files, err := filepath.Glob(filepath.Join(dir, "*.brz"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatalf("no tests in %s", dir)
	}
	for _, f := range files {
		base := strings.TrimSuffix(f, ".brz")
		name := filepath.Base(base)
		t.Run(name, func(t *testing.T) {
			if why, ok := pending[filepath.ToSlash(base)]; ok {
				t.Skipf("pending: needs %s", why)
			}
			test(t, base)
		})
	}
}

// bronze runs the bronze command in-process. File paths in the output are
// shortened to their base name.
func bronze(cmd, path string) (stdout, stderr string, code int) {
	var out, errOut bytes.Buffer
	code = cli.Main([]string{cmd, path}, &out, &errOut)
	short := func(s string) string { return strings.ReplaceAll(s, path, filepath.Base(path)) }
	return short(out.String()), short(errOut.String()), code
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestExamplesInSync checks that the golden copies of the 0.1.0 target
// programs are the same as the examples.
func TestExamplesInSync(t *testing.T) {
	for copy, example := range map[string]string{
		"run/core.brz":    "../examples/core.brz",
		"run/counter.brz": "../examples/projects/counter.brz",
		"run/bank.brz":    "../examples/projects/bank.brz",
	} {
		if read(t, copy) != read(t, example) {
			t.Errorf("tests/%s is not the same as %s", copy, strings.TrimPrefix(example, "../"))
		}
	}
}

// TestExamples runs every program in examples/. Each must end with a
// comment saying what it prints:
//
//	/*
//	    Expected output:
//
//	    hello
//
//	    Exit code: 0
//	*/
//
// so that examples/ only holds programs that work. Design files that
// don't run belong in doc/explorations/.
func TestExamples(t *testing.T) {
	var files []string
	err := filepath.WalkDir("../examples", func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".brz") {
			files = append(files, path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		t.Run(strings.TrimPrefix(path, "../examples/"), func(t *testing.T) {
			wantOut, wantCode, ok := expected(read(t, path))
			if !ok {
				t.Fatalf("%s has no `Expected output:` ... `Exit code: N` comment (programs that don't run belong in doc/explorations/)", path)
			}
			stdout, stderr, code := bronze("run", path)
			if code != wantCode {
				t.Errorf("exit code %d, want %d\nstderr:\n%s", code, wantCode, stderr)
			}
			if stdout != wantOut {
				t.Errorf("stdout:\n%s\nwant:\n%s", stdout, wantOut)
			}
		})
	}
}

// expected reads the `Expected output:` comment of an example.
func expected(src string) (out string, code int, ok bool) {
	_, rest, found := strings.Cut(src, "Expected output:")
	if !found {
		return "", 0, false
	}
	body, tail, found := strings.Cut(rest, "Exit code:")
	if !found {
		return "", 0, false
	}
	if _, err := fmt.Sscanf(strings.TrimSpace(tail), "%d", &code); err != nil {
		return "", 0, false
	}
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		lines = append(lines, strings.TrimPrefix(line, "    "))
	}
	out = strings.Trim(strings.Join(lines, "\n"), "\n")
	if out != "" {
		out += "\n"
	}
	return out, code, true
}
