package jsonrpc

import (
	"bytes"
	"github.com/goccy/go-json"
)

type Request struct {
	Version string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Single  bool            `json:"-"`
}

type Response struct {
	Version string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
	Single  bool            `json:"-"`
}

type ErrorResponse struct {
	Version string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Error   Error  `json:"error"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (j *Response) IsNullResult() bool {
	return bytes.Equal(j.Result, []byte("null"))
}
