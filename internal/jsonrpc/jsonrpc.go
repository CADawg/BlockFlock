package jsonrpc

import (
	"bytes"
	"github.com/goccy/go-json"
	"io"
	"net/http"
	"os"
	"strings"
)

// JsonGet gets json from a url
func JsonGet[T any](url string) (*T, error) {
	var decodeInto = new(T)

	resp, err := http.Get(url)

	if err != nil {
		return new(T), err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

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

func JsonPost[T any](urlString string, data []byte) (*T, error) {
	var decodeInto = new(T)

	// create a new request
	req, err := http.NewRequest("POST", urlString, bytes.NewBuffer(data))

	if err != nil {
		return new(T), err
	}

	// Look like a browser
	req.Header.Set("Content-Type", "application/json")

	// get other headers from env format: Key=Value\nKey2=Value2
	headers := strings.Split(os.Getenv("HEADERS"), "\n")

	for _, header := range headers {
		parts := strings.Split(header, "=")

		if len(parts) == 2 {
			req.Header.Set(parts[0], parts[1])
		}
	}

	// send the request
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return new(T), err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	data, err = io.ReadAll(resp.Body)

	if err != nil {
		return new(T), err
	}

	err = json.Unmarshal(data, decodeInto)

	if err != nil {
		return new(T), err
	}

	return decodeInto, nil
}
