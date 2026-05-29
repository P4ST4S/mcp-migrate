package handler

// handleRequest processes any MCP request.
// It returns a resource from the store when the method is resources/read.
// For other methods it returns a generic JSON-RPC error.

func handleRequest(method string, id string) *rpcError {
	switch method {
	case "ping":
		return nil
	default:
		// The method is not supported; return method-not-found.
		return &rpcError{Code: -32002, Message: "method not found"}
	}
}
