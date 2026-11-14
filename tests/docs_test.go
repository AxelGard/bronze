package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/AxelGard/bronze/internal/cli"
)

// TestDocs runs every ```bronze block in the website (website/docs), so
// that the documentation only shows code that works.
//
//   - A block with top-level declarations (a line starting with `func`,
//     `class` or `enum`) is a program. If it has no `main`, one that
//     returns 0 is added at the end.
//   - Any other block is a list of statements, and is run inside a `main`
//     that returns 0.
//
// If the next fenced block is ```text title="Output", the program's stdout
// followed by its stderr must be exactly that text, with the file called
// main.brz. Otherwise the program must check, run, and exit with 0.
func TestDocs(t *testing.T) {
	var pages []string
	err := filepath.WalkDir("../website/docs", func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".md") {
			pages = append(pages, path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) == 0 {
		t.Fatal("no pages in website/docs")
	}
	for _, page := range pages {
		blocks := fences(read(t, page))
		for i, b := range blocks {
			if b.lang != "bronze" {
				continue
			}
			var want *fence
			if i+1 < len(blocks) && blocks[i+1].lang == "text" && blocks[i+1].isOutput() && blocks[i+1].follows(b) {
				want = &blocks[i+1]
			}
			name := strings.TrimPrefix(page, "../website/docs/") + ":" + strconv.Itoa(b.line)
			t.Run(name, func(t *testing.T) {
				docBlock(t, b.body, want)
			})
		}
	}
}

func docBlock(t *testing.T, body string, want *fence) {
	path := filepath.Join(t.TempDir(), "main.brz")
	if err := os.WriteFile(path, []byte(program(body)), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := cli.Main([]string{"run", path}, &out, &errOut)
	got := strings.ReplaceAll(out.String()+errOut.String(), path, "main.brz")
	if want == nil {
		if code != 0 {
			t.Errorf("exit code %d, want 0\noutput:\n%s", code, got)
		}
		return
	}
	if got != want.body {
		t.Errorf("output:\n%s\nwant:\n%s", got, want.body)
	}
}

// program turns a block into a whole program.
func program(body string) string {
	decls := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "func ") || strings.HasPrefix(line, "class ") || strings.HasPrefix(line, "enum ") {
			decls = true
		}
	}
	if decls {
		if strings.Contains(body, "func main(") {
			return body
		}
		return body + "\n\nfunc main() -> Int\n{\n    return 0\n}\n"
	}
	var b strings.Builder
	b.WriteString("func main() -> Int\n{\n")
	for _, line := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
		if line != "" {
			b.WriteString("    " + line)
		}
		b.WriteString("\n")
	}
	b.WriteString("    return 0\n}\n")
	return b.String()
}

type fence struct {
	lang, info string
	body       string
	line, end  int // the opening and closing ``` lines, 1-based
}

func (f fence) isOutput() bool { return strings.Contains(f.info, `title="Output"`) }

// follows reports whether f comes right after p, with only blank lines
// between them.
func (f fence) follows(p fence) bool { return f.line-p.end <= 2 }

// fences finds the fenced code blocks of a Markdown page. Fences must
// start at the beginning of a line.
func fences(src string) []fence {
	var out []fence
	var cur *fence
	var body []string
	for i, line := range strings.Split(src, "\n") {
		if !strings.HasPrefix(line, "```") {
			if cur != nil {
				body = append(body, line)
			}
			continue
		}
		if cur == nil {
			info := strings.TrimSpace(strings.TrimPrefix(line, "```"))
			lang, rest, _ := strings.Cut(info, " ")
			cur = &fence{lang: lang, info: rest, line: i + 1}
			body = nil
			continue
		}
		cur.end = i + 1
		cur.body = strings.Join(body, "\n") + "\n"
		out = append(out, *cur)
		cur = nil
	}
	return out
}
