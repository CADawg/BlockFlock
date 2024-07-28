package normalise

import (
	"github.com/CADawg/BlockFlock/internal/jsonrpc"
	"github.com/goccy/go-json"
	"io"
)

func Normalise(requestBody io.ReadCloser, endpoint string) ([]jsonrpc.Request, error) {
	var requests []jsonrpc.Request
	var request jsonrpc.Request

	var requestsData []byte

	requestsData, err := io.ReadAll(requestBody)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(requestsData, &requests)

	if err != nil {
		err = json.Unmarshal(requestsData, &request)

		if err != nil {
			return nil, err
		}

		// so that we know not to send it back inside an array
		request.Single = true
		requests = append(requests, request)
	}

	if endpoint == "" {
		return requests, nil
	}

	for i, request := range requests {
		request.Method = endpoint + "." + request.Method
		requests[i] = request
	}

	return requests, nil
}
