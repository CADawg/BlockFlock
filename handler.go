package main

import (
	"fmt"

	"github.com/goccy/go-json"
)

type IJSONRPCResponse interface {
	ToJSON() ([]byte, error)
	GetVersion() string
	GetID() *int
}

func (j *JSONRPCResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(j)
}

func (j *JSONRPCResponse) GetID() *int {
	return j.ID
}

func (j *JSONRPCResponse) GetVersion() string {
	return j.Version
}

func (j *JSONRPCErrorResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(j)
}

func (j *JSONRPCErrorResponse) GetID() *int {
	return j.ID
}

func (j *JSONRPCErrorResponse) GetVersion() string {
	return j.Version
}

func HandleRequests(requests []JSONRPCRequest, endpoint string, noBrackets bool) (json.RawMessage, error) {
	var responses = make([]json.RawMessage, len(requests))

	for i, request := range requests {
		response, err := HandleOneRequest(request, endpoint)

		if err != nil {
			// make error response (we don't want to ruin all requests because of one error)
			response, err := MakeErrorResponse(request, err.Error(), -69)

			if err != nil {
				return nil, err
			}

			responses[i] = response
		}

		responses[i] = response
	}

	if noBrackets && len(responses) == 1 {
		return responses[0], nil
	}

	return json.Marshal(responses)
}

// HandleOneRequest handles everything a request needs to do
// this includes checking the cache, populating the cache and forwarding to upstream
func HandleOneRequest(request JSONRPCRequest, endpoint string) ([]byte, error) {
	_, err := ValidateRequest(request)

	if err != nil {
		// return this as a single request error not a batch error
		return nil, fmt.Errorf("invalid request %w", err)
	}

	// Check if it is a getBlockInfo request
	if getBlockInfo.IsMethod(request.Method) {
		// check our cache
		value, err := ReadCacheForBlock(request)

		if err == nil {
			// return cached value
			return value, nil
		}

		// fetch from upstream
		response, err := FetchFromUpstream(request, endpoint)

		if err != nil {
			return nil, err
		}

		// write response to cache
		_ = WriteCacheForBlock(request, response)

		return response, nil
	} else {
		// forward onto full node
		return FetchFromUpstream(request, endpoint)
	}
}
