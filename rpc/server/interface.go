package server

import (
	"github.com/ValentinKolb/dKV/lib/store"
	"github.com/ValentinKolb/dKV/rpc/common"
)

// IRPCServerAdapter is the interface for all RPC server adapters
// It is responsible for handling requests and responses
type IRPCServerAdapter interface {
	// Handle handles a request and returns a response
	// It takes a Message and a store as parameters.
	// It returns a Message as a response
	// If an error occurs, it should be set in the response
	Handle(req *common.Message, store store.IStore) (resp *common.Message)
}
