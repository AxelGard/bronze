package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/textproto"
	"strconv"
	"sync"
)

// message is a JSON-RPC 2.0 request, notification or response.
type message struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  *json.RawMessage `json:"result,omitempty"` // a response has a result, even null, or an error
	Error   *responseError   `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error codes from JSON-RPC and the LSP.
const (
	codeParseError     = -32700
	codeInvalidParams  = -32602
	codeMethodNotFound = -32601
	codeRequestFailed  = -32803
)

// conn reads and writes LSP messages: a Content-Length header, a blank
// line, then the JSON body.
type conn struct {
	in  *bufio.Reader
	out io.Writer
	mu  sync.Mutex
}

func (c *conn) read() (*message, error) {
	headers, err := textproto.NewReader(c.in).ReadMIMEHeader()
	if err != nil {
		return nil, err
	}
	n, err := strconv.Atoi(headers.Get("Content-Length"))
	if err != nil {
		return nil, fmt.Errorf("bad Content-Length: %v", err)
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(c.in, body); err != nil {
		return nil, err
	}
	var m message
	if err := json.Unmarshal(body, &m); err != nil {
		return &message{}, fmt.Errorf("bad message: %v", err)
	}
	return &m, nil
}

func (c *conn) write(m *message) error {
	m.JSONRPC = "2.0"
	body, err := json.Marshal(m)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := fmt.Fprintf(c.out, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = c.out.Write(body)
	return err
}

func (c *conn) notify(method string, params any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return c.write(&message{Method: method, Params: raw})
}
