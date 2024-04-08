package main

import (
	"io"
	"net/url"

	"github.com/goccy/go-json"
)

func FetchFromUpstream(request JSONRPCRequest, endpoint string) ([]byte, error) {
	// serialize request
	data, err := json.Marshal(request)

	if err != nil {
		return nil, err
	}

	var endpointUrl string
	if endpoint == "blockchain" {
		endpointUrl = config.BlockchainEndpoint
	} else if endpoint == "contracts" {
		endpointUrl = config.ContractsEndpoint
	} else {
		// this means the method should include contracts. or blockchain. and so we don't need to specify a specific endpoint
		endpointUrl = ""
	}

	path, err := url.JoinPath(config.Node, endpointUrl)

	if err != nil {
		return nil, err
	}

	resp, err := DoPost(path, data)

	if err != nil {
		return nil, err
	}

	// read response
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	// close response body
	err = resp.Body.Close()

	if err != nil {
		return nil, err
	}

	return body, nil
}
