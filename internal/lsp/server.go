// Package lsp is the Bronze language server: `bronze lsp` talks the
// Language Server Protocol over stdin and stdout.
//
// It supports diagnostics, go to definition and rename. Every document is
// checked on its own, since a 0.1 program is one file.
package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"runtime/debug"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/check"
	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/lexer"
	"github.com/AxelGard/bronze/internal/parser"
)

// Serve runs the language server until the client says exit, and returns
// the exit code. Problems are logged to logw.
func Serve(in io.Reader, out io.Writer, logw io.Writer, version string) int {
	s := &server{
		conn:     &conn{in: bufio.NewReader(in), out: out},
		log:      logw,
		version:  version,
		docs:     map[string]*document{},
		encoding: "utf-16",
	}
	return s.run()
}

type server struct {
	conn     *conn
	log      io.Writer
	version  string
	docs     map[string]*document
	encoding string // "utf-16" or "utf-8": how positions count characters
	shutdown bool
}

// document is an open file and what the checker found in it.
type document struct {
	uri         string
	src         *diag.Source
	info        *check.Info
	syntaxError bool
}

func (s *server) logf(format string, args ...any) {
	fmt.Fprintf(s.log, "bronze lsp: "+format+"\n", args...)
}

func (s *server) run() int {
	for {
		m, err := s.conn.read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 1 // the client went away without `exit`
			}
			s.logf("%v", err)
			if m == nil {
				return 1
			}
			continue
		}
		if m.Method == "exit" {
			if s.shutdown {
				return 0
			}
			return 1
		}
		s.handle(m)
	}
}

// handle answers one message. A panic in the checker is a bug in bronze;
// it is logged, and the server keeps running.
func (s *server) handle(m *message) {
	defer func() {
		if r := recover(); r != nil {
			s.logf("internal error handling %s: %v\n%s", m.Method, r, debug.Stack())
			if m.ID != nil {
				s.replyError(m.ID, codeRequestFailed, fmt.Sprintf("internal error in bronze: %v", r))
			}
		}
	}()

	result, rerr := s.dispatch(m)
	if m.ID == nil {
		return // a notification: no reply
	}
	if rerr != nil {
		s.replyError(m.ID, rerr.Code, rerr.Message)
		return
	}
	raw, err := json.Marshal(result)
	if err != nil {
		s.replyError(m.ID, codeRequestFailed, err.Error())
		return
	}
	r := json.RawMessage(raw)
	s.send(&message{ID: m.ID, Result: &r})
}

func (s *server) send(m *message) {
	if err := s.conn.write(m); err != nil {
		s.logf("write: %v", err)
	}
}

func (s *server) replyError(id *json.RawMessage, code int, msg string) {
	s.send(&message{ID: id, Error: &responseError{Code: code, Message: msg}})
}

func (s *server) dispatch(m *message) (any, *responseError) {
	decode := func(v any) *responseError {
		if err := json.Unmarshal(m.Params, v); err != nil {
			return &responseError{Code: codeInvalidParams, Message: err.Error()}
		}
		return nil
	}

	switch m.Method {
	case "initialize":
		var p initializeParams
		if err := decode(&p); err != nil {
			return nil, err
		}
		return s.initialize(p), nil
	case "initialized", "$/cancelRequest", "$/setTrace", "textDocument/didSave":
		return nil, nil
	case "shutdown":
		s.shutdown = true
		return nil, nil

	case "textDocument/didOpen":
		var p didOpenParams
		if err := decode(&p); err != nil {
			return nil, err
		}
		s.update(p.TextDocument.URI, p.TextDocument.Text)
		return nil, nil
	case "textDocument/didChange":
		var p didChangeParams
		if err := decode(&p); err != nil {
			return nil, err
		}
		if n := len(p.ContentChanges); n > 0 {
			s.update(p.TextDocument.URI, p.ContentChanges[n-1].Text) // full sync
		}
		return nil, nil
	case "textDocument/didClose":
		var p didCloseParams
		if err := decode(&p); err != nil {
			return nil, err
		}
		delete(s.docs, p.TextDocument.URI)
		s.publish(p.TextDocument.URI, nil, nil)
		return nil, nil

	case "textDocument/definition":
		var p textDocumentPositionParams
		if err := decode(&p); err != nil {
			return nil, err
		}
		return s.definition(p), nil
	case "textDocument/hover":
		var p textDocumentPositionParams
		if err := decode(&p); err != nil {
			return nil, err
		}
		return s.hover(p), nil
	case "textDocument/completion":
		var p textDocumentPositionParams
		if err := decode(&p); err != nil {
			return nil, err
		}
		return s.completion(p), nil
	case "textDocument/prepareRename":
		var p textDocumentPositionParams
		if err := decode(&p); err != nil {
			return nil, err
		}
		return s.prepareRename(p)
	case "textDocument/rename":
		var p renameParams
		if err := decode(&p); err != nil {
			return nil, err
		}
		return s.rename(p)
	}
	if m.ID == nil {
		return nil, nil // unknown notifications are ignored
	}
	return nil, &responseError{Code: codeMethodNotFound, Message: "bronze lsp does not support " + m.Method}
}

func (s *server) initialize(p initializeParams) any {
	for _, e := range p.Capabilities.General.PositionEncodings {
		if e == "utf-8" {
			s.encoding = "utf-8"
		}
	}
	return map[string]any{
		"capabilities": map[string]any{
			"positionEncoding":   s.encoding,
			"textDocumentSync":   map[string]any{"openClose": true, "change": 1},
			"definitionProvider": true,
			"hoverProvider":      true,
			"completionProvider": map[string]any{"triggerCharacters": []string{"."}},
			"renameProvider":     map[string]any{"prepareProvider": true},
		},
		"serverInfo": map[string]any{"name": "bronze", "version": s.version},
	}
}

// update checks a document's new text and publishes its diagnostics.
func (s *server) update(uri, text string) {
	src := diag.NewSource(fileName(uri), []byte(text))
	file, diags := parser.Parse(src)
	info, cdiags := check.Check(file, src)
	s.docs[uri] = &document{uri: uri, src: src, info: info, syntaxError: len(diags) > 0 || hasBadDecl(file)}
	s.publish(uri, src, append(diags, cdiags...))
}

func hasBadDecl(f *ast.File) bool {
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.BadDecl:
			return true
		case *ast.FuncDecl:
			if d.Bad {
				return true
			}
		}
	}
	return false
}

// fileName is the name used in messages: the base name of the file.
func fileName(uri string) string {
	if u, err := url.Parse(uri); err == nil && u.Path != "" {
		return filepath.Base(u.Path)
	}
	return uri
}

func (s *server) publish(uri string, src *diag.Source, diags diag.List) {
	out := []diagnostic{}
	diags.Sort()
	for _, d := range diags {
		msg := d.Message
		if d.Label != "" {
			msg += " (" + d.Label + ")"
		}
		for _, h := range d.Help {
			msg += "\nhelp: " + h
		}
		for _, n := range d.Notes {
			msg += "\nnote: " + n
		}
		out = append(out, diagnostic{Range: s.toRange(src, d.Span), Severity: 1, Source: "bronze", Message: msg})
	}
	if err := s.conn.notify("textDocument/publishDiagnostics", publishDiagnosticsParams{URI: uri, Diagnostics: out}); err != nil {
		s.logf("write: %v", err)
	}
}

// ---------------------------------------------------------------------
// Definition, hover and rename

// refAt returns the document and the reference at a position, if any.
func (s *server) refAt(uri string, pos position) (*document, *check.Ref) {
	doc := s.docs[uri]
	if doc == nil {
		return nil, nil
	}
	return doc, doc.info.RefAt(s.toOffset(doc.src, pos))
}

func (s *server) definition(p textDocumentPositionParams) any {
	doc, ref := s.refAt(p.TextDocument.URI, p.Position)
	if ref == nil || ref.Sym.Builtin {
		return nil
	}
	return location{URI: doc.uri, Range: s.toRange(doc.src, ref.Sym.Decl)}
}

// hover shows what the name under the cursor is: its declaration, as a
// bronze code block, and for fields and methods what they belong to.
func (s *server) hover(p textDocumentPositionParams) any {
	doc, ref := s.refAt(p.TextDocument.URI, p.Position)
	if ref == nil || ref.Sym.Hover == "" {
		return nil
	}
	text := "```bronze\n" + ref.Sym.Hover + "\n```"
	if ref.Sym.In != "" {
		text += "\n\nin `" + ref.Sym.In + "`"
	}
	if ref.Sym.Doc != "" {
		text += "\n\n" + ref.Sym.Doc
	}
	r := s.toRange(doc.src, ref.Span)
	return hoverResult{Contents: markupContent{Kind: "markdown", Value: text}, Range: &r}
}

func (s *server) prepareRename(p textDocumentPositionParams) (any, *responseError) {
	doc, ref := s.refAt(p.TextDocument.URI, p.Position)
	if ref == nil {
		return nil, nil
	}
	if err := renamable(doc, ref); err != nil {
		return nil, err
	}
	return map[string]any{"range": s.toRange(doc.src, ref.Span), "placeholder": ref.Sym.Name}, nil
}

func (s *server) rename(p renameParams) (any, *responseError) {
	doc, ref := s.refAt(p.TextDocument.URI, p.Position)
	if ref == nil {
		return nil, &responseError{Code: codeRequestFailed, Message: "there is no name to rename here"}
	}
	if err := renamable(doc, ref); err != nil {
		return nil, err
	}
	if msg := checkNewName(p.NewName); msg != "" {
		return nil, &responseError{Code: codeRequestFailed, Message: msg}
	}
	var edits []textEdit
	for _, r := range doc.info.RefsTo(ref.Sym) {
		edits = append(edits, textEdit{Range: s.toRange(doc.src, r.Span), NewText: p.NewName})
	}
	return workspaceEdit{Changes: map[string][]textEdit{doc.uri: edits}}, nil
}

// renamable says why a symbol can't be renamed, or nil if it can.
func renamable(doc *document, ref *check.Ref) *responseError {
	switch {
	case ref.Sym.Builtin:
		return &responseError{Code: codeRequestFailed, Message: fmt.Sprintf("`%s` is built in and can't be renamed", ref.Sym.Name)}
	case doc.syntaxError:
		return &responseError{Code: codeRequestFailed, Message: "fix the syntax errors first: rename needs to see the whole file"}
	}
	return nil
}

// checkNewName returns why name can't be used, or "". A good name lexes
// as exactly one identifier.
func checkNewName(name string) string {
	toks, diags := lexer.Lex(diag.NewSource("", []byte(name)))
	switch {
	case len(diags) > 0 || len(toks) != 3 || toks[0].Text != name || name == "_":
		return fmt.Sprintf("`%s` is not a valid name", name)
	case toks[0].Kind != lexer.IDENT:
		return fmt.Sprintf("`%s` is a keyword or a reserved word", name)
	}
	switch name {
	case "Int", "Float", "Bool", "String", "Error", "print", "panic", "assert", "init":
		return fmt.Sprintf("`%s` is a built-in name", name)
	}
	return ""
}
