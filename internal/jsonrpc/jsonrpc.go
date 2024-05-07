package jsonrpc

import (
	"bytes"
	"github.com/goccy/go-json"
	"io"
	"net/http"
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

func JsonPost[T any](url string, data []byte) (*T, error) {
	var decodeInto = new(T)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))

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
