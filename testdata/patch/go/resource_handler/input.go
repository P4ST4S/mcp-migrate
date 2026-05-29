package handler

import "encoding/json"

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func handleResourcesRead(uri string) (any, *rpcError) {
	res, ok := resourceStore[uri]
	if !ok {
		return nil, &rpcError{Code: -32002, Message: "resource not found"}
	}
	return res, nil
}

func handleResourcesRead2(uri string) (any, *rpcError) {
	if _, exists := resourceStore[uri]; !exists {
		return nil, &rpcError{
			Code:    -32002,
			Message: "not found",
		}
	}
	return resourceStore[uri], nil
}

var resourceStore = map[string]json.RawMessage{}
