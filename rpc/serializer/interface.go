package serializer

import "github.com/ValentinKolb/dKV/rpc/common"

// IRPCSerializer IRPCServerAdapter is the interface for all Message Serializers
type IRPCSerializer interface {
	// Serialize serializes a Message into a byte array
	// It returns the serialized byte array and an error if any
	Serialize(msg common.Message) ([]byte, error)
	// Deserialize deserializes a byte array into a Message
	// It takes a byte array and a pointer to a Message as parameters
	// It returns an error if any
	Deserialize(b []byte, msg *common.Message) error
}
