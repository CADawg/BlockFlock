package upstream

import (
	"errors"
	"github.com/CADawg/BlockFlock/internal/cache"
	"github.com/CADawg/BlockFlock/internal/hive_engine"
	"github.com/CADawg/BlockFlock/internal/jsonrpc"
	"github.com/goccy/go-json"
	"strconv"
)

const TypeBlock rune = 'b'

type Upstream struct {
	Cache cache.Cache
	Node  string
}

func NewUpstream(cache cache.Cache, node string) *Upstream {
	return &Upstream{
		Cache: cache,
		Node:  node,
	}
}

func (u *Upstream) HandleRequests(requests []jsonrpc.Request, latestSafeBlock int64) ([]jsonrpc.Response, error) {
	var interestingIds []int

	var uncachedRequests []jsonrpc.Request
	var cachedResponses []*jsonrpc.Response
	var responses *[]jsonrpc.Response
	var isSingle = false

	for i, request := range requests {
		if i == 0 && request.Single {
			isSingle = true
		}

		if i != 0 {
			request.Single = false
		}

		if request.Method == "blockchain.getBlockInfo" {
			var blockInfoParams hive_engine.GetBlockInfoParams

			err := json.Unmarshal(request.Params, &blockInfoParams)

			if err != nil {
				uncachedRequests = append(uncachedRequests, request)
				cachedResponses = append(cachedResponses, nil)
				continue
			}

			// check if we have it cached
			cached, err := u.Cache.Has(TypeBlock, strconv.FormatInt(blockInfoParams.BlockNumber, 10))

			if err != nil {
				uncachedRequests = append(uncachedRequests, request)
				cachedResponses = append(cachedResponses, nil)
				continue
			}

			if cached {
				// get it from cache
				data, err := u.Cache.Get(TypeBlock, strconv.FormatInt(blockInfoParams.BlockNumber, 10))

				if err != nil {
					uncachedRequests = append(uncachedRequests, request)
					cachedResponses = append(cachedResponses, nil)
					continue
				}

				var jsonResponse jsonrpc.Response

				err = json.Unmarshal(data, &jsonResponse)

				if err != nil {
					uncachedRequests = append(uncachedRequests, request)
					cachedResponses = append(cachedResponses, nil)
					continue
				}

				jsonResponse.ID = request.ID

				cachedResponses = append(cachedResponses, &jsonResponse)
			} else {
				uncachedRequests = append(uncachedRequests, request)
				cachedResponses = append(cachedResponses, nil)
				interestingIds = append(interestingIds, request.ID)
			}
		} else {
			uncachedRequests = append(uncachedRequests, request)
			cachedResponses = append(cachedResponses, nil)
		}
	}

	if len(uncachedRequests) != 0 {
		// marshal all requests
		requestsData, err := json.Marshal(uncachedRequests)

		if err != nil {
			return nil, err
		}

		// send all together to the parent node
		responses, err = jsonrpc.JsonPost[[]jsonrpc.Response](u.Node, requestsData)

		if err != nil {
			return nil, err
		}

		if responses == nil {
			return nil, errors.New("no responses")
		}

		for _, response := range *responses {
			for _, id := range interestingIds {
				if response.ID == id {
					var blockInfoParams hive_engine.GetBlockInfoParams

					err = json.Unmarshal(response.Result, &blockInfoParams)

					if err != nil {
						continue
					}

					if blockInfoParams.BlockNumber > latestSafeBlock {
						// we can't cache this, it's not necessarily final
						continue
					}

					// check if we have it cached
					cached, err := u.Cache.Has(TypeBlock, strconv.FormatInt(blockInfoParams.BlockNumber, 10))

					if err != nil {
						continue
					}

					if !cached {
						// marshal it
						data, err := json.Marshal(response)

						if err != nil {
							continue
						}

						// cache it
						err = u.Cache.Set(TypeBlock, strconv.FormatInt(blockInfoParams.BlockNumber, 10), data)

						if err != nil {
							continue
						}
					}
				}
			}
		}

		var finalResponses []jsonrpc.Response

		var cachedEncountered = 0
		for i, cachedResponse := range cachedResponses {
			if cachedResponse != nil {
				cachedResponse.Single = isSingle
				finalResponses = append(finalResponses, *cachedResponse)
				cachedEncountered++
			} else {
				// the actual responses don't know which ones were cached so have to account for them
				(*responses)[i-cachedEncountered].Single = isSingle
				finalResponses = append(finalResponses, (*responses)[i-cachedEncountered])
			}
		}

		return finalResponses, nil
	} else {
		var finalResponses []jsonrpc.Response

		// need to deref all the responses
		for _, resp := range cachedResponses {
			if resp != nil {
				resp.Single = isSingle
				finalResponses = append(finalResponses, *resp)
			}
		}

		return finalResponses, nil
	}
}
