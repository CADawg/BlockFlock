package main

import (
	"bytes"
	"io"
	"net/http"

	"github.com/goccy/go-json"
)

// ReadJRPCBody reads the body of a json rpc request
// Returns (requests, error, isSingleRequestWithoutBrackets)
func ReadJRPCBody(req *http.Request) ([]JSONRPCRequest, error, bool) {
	body, err := io.ReadAll(req.Body)

	if err != nil {
		return nil, err, false
	}

	var request JSONRPCRequest

	err = json.Unmarshal(body, &request)

	if err == nil {
		return []JSONRPCRequest{request}, nil, true
	}

	var requests []JSONRPCRequest

	// need to try reading it as an array
	err = json.Unmarshal(body, &requests)

	if err != nil {
		return nil, err, false
	}

	return requests, nil, false
}

// JsonGet gets json from a url
func JsonGet[T any](url string) (*T, error) {
	var decodeInto = new(T)

	resp, err := http.Get(url)

	if err != nil {
		return new(T), err
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)

	if err != nil {
		return new(T), err
	}

	err = json.Unmarshal(data, decodeInto)

	if err != nil {
		return new(T), err
	}

	return decodeInto, nil
}

func SendUnparsableErrorResponse(w http.ResponseWriter, err string) {
	json.NewEncoder(w).Encode(map[string]string{
		"error": err,
	})
}

type JSONRPCErrorResponse struct {
	Version string       `json:"jsonrpc"`
	ID      *int         `json:"id"`
	Error   JSONRPCError `json:"error"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func SendErrorResponse(w http.ResponseWriter, requests []JSONRPCRequest, err error) {
	var errors []JSONRPCErrorResponse

	for _, request := range requests {
		errors = append(errors, JSONRPCErrorResponse{
			Version: request.Version,
			ID:      request.ID,
			Error: JSONRPCError{
				Code:    -69,
				Message: err.Error(),
			},
		})
	}

	json.NewEncoder(w).Encode(errors)
}

func MakeErrorResponse(request JSONRPCRequest, message string, code int) ([]byte, error) {
	response := JSONRPCErrorResponse{
		Version: request.Version,
		ID:      request.ID,
		Error: JSONRPCError{
			Code:    code,
			Message: message,
		},
	}

	return json.Marshal(response)
}

type JSONRPCResponse struct {
	Version string          `json:"jsonrpc"`
	ID      *int            `json:"id"`
	Result  json.RawMessage `json:"result"`
}

func (j *JSONRPCResponse) IsNullResult() bool {
	return bytes.Equal(j.Result, []byte("null"))
}

func DoPost(url string, body []byte) (*http.Response, error) {
	// send request to upstream
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "Mozilla/5.0 AppleWebKit/537.36 Safari/537.36 Chrome/41.0.2272.96 (KHTML, like Gecko; compatible; BlockFlock/1.0; +http://github.com/CADawg/BlockFlock)")

	return http.DefaultClient.Do(req)
}
