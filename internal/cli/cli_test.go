package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func bronze(args ...string) (string, string, int) {
	var out, errOut strings.Builder
	code := Main(args, &out, &errOut)
	return out.String(), errOut.String(), code
}

func TestVersion(t *testing.T) {
	out, _, code := bronze("version")
	if code != 0 || out != "bronze "+Version+"\n" {
		t.Errorf("got %q, %d", out, code)
	}
}

func TestUsage(t *testing.T) {
	if _, errOut, code := bronze(); code != 2 || !strings.Contains(errOut, "Usage:") {
		t.Errorf("no args: %d %q", code, errOut)
	}
	if _, errOut, code := bronze("build"); code != 2 || !strings.Contains(errOut, `unknown command "build"`) {
		t.Errorf("unknown command: %d %q", code, errOut)
	}
	if _, _, code := bronze("run"); code != 2 {
		t.Errorf("run without file: %d", code)
	}
}

func TestMissingFile(t *testing.T) {
	_, errOut, code := bronze("run", "does-not-exist.brz")
	if code != 1 || !strings.HasPrefix(errOut, "bronze: ") {
		t.Errorf("got %d %q", code, errOut)
	}
}

func TestRunAndCheck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.brz")
	src := "func main() -> Int\n{\n    print(\"hi\")\n    return 7\n}\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, _, code := bronze("run", path); code != 7 || out != "hi\n" {
		t.Errorf("run: %q %d", out, code)
	}
	if out, errOut, code := bronze("check", path); code != 0 || out != "" || errOut != "" {
		t.Errorf("check: %q %q %d", out, errOut, code)
	}
}
