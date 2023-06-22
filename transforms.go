package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func SendIndex(res http.ResponseWriter) error {
	resp, err := JsonGet[EngineInfo](config.Node)

	if err != nil {
		return json.NewEncoder(res).Encode(EngineInfoError{
			Error:   err.Error(),
			Success: false,
		})
	}

	resp.DisabledMethods.Message = strings.Replace(resp.DisabledMethods.Message, "h-e", "h-e and cadengine", 1) + ". Source code and licence available at https://github.com/CADawg/BlockFlock."
	resp.Success = true
	resp.Domain = "https://engine.rishipanthee.com/"

	res.Header().Set("Content-Type", "application/json")

	return json.NewEncoder(res).Encode(resp)
}
