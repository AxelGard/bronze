package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func fakeCompile(src, out string) error {
	if _, err := os.Stat(filepath.Join(src, "parser.c")); err != nil {
		return err
	}
	return os.WriteFile(out, []byte("so"), 0o644)
}

func TestInstallNvim(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pack", "bronze", "start", "bronze")
	if err := installNvim(dir, fakeCompile); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"plugin/bronze.lua", "ftdetect/bronze.lua", "queries/bronze/highlights.scm", "parser/bronze.so", "version.txt"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("missing %s", f)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "src")); err == nil {
		t.Errorf("the C source should not be left in the package")
	}

	// Running it again replaces the package.
	os.WriteFile(filepath.Join(dir, "stale.txt"), nil, 0o644)
	if err := installNvim(dir, fakeCompile); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "stale.txt")); err == nil {
		t.Errorf("the old package was not replaced")
	}

	// A failed compile keeps the old package.
	if err := installNvim(dir, func(string, string) error { return errors.New("no cc") }); err == nil {
		t.Fatal("no error from a failed compile")
	}
	if _, err := os.Stat(filepath.Join(dir, "parser/bronze.so")); err != nil {
		t.Errorf("a failed compile removed the old package")
	}
	if entries, _ := os.ReadDir(filepath.Dir(dir)); len(entries) != 1 {
		t.Errorf("temporary directories left behind: %v", entries)
	}
}

func TestInstallNvimRefusesOtherDirs(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mine.lua"), nil, 0o644)
	err := installNvim(dir, fakeCompile)
	if err == nil || !strings.Contains(err.Error(), "was not made by `bronze setup`") {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mine.lua")); err != nil {
		t.Errorf("an unrelated directory was changed")
	}
}

func TestCompileParser(t *testing.T) {
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("no C compiler")
	}
	dir := filepath.Join(t.TempDir(), "bronze")
	if err := installNvim(dir, compileParser); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(dir, "parser/bronze.so")); err != nil || info.Size() == 0 {
		t.Fatalf("no parser: %v", err)
	}
}

func TestSetupUsage(t *testing.T) {
	var out, errOut strings.Builder
	if code := Main([]string{"setup"}, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "bronze setup nvim") {
		t.Errorf("code %d, stderr %q", code, errOut.String())
	}
	if code := Main([]string{"setup", "vscode"}, &out, &errOut); code != 2 {
		t.Errorf("unknown editor: code %d", code)
	}
}
