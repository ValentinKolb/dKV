package common

import (
	"encoding/json"
	"fmt"
)

// --------------------------------------------------------------------------
// Message Structure
// --------------------------------------------------------------------------

// Message represents a single message used for both requests and responses.
// Which fields are used depends on the type of message.
type Message struct {
	// Type of message
	MsgType MessageType `json:"msg_type"`

	// General fields
	Key      string `json:"key,omitempty"`      // Used for: Set, Get, Has, Expire, Delete, Acquire, Release
	ExpireIn uint64 `json:"expireIn,omitempty"` // Used for: Set operations
	DeleteIn uint64 `json:"deleteIn,omitempty"` // Used for: Set, Acquire operations
	Value    []byte `json:"value,omitempty"`    // Used for: Set (request), Get (response), Acquire (response)

	// Response only fields
	Ok  bool   `json:"ok,omitempty"`  // Used for: Get, Has, Acquire, Release responses
	Err string `json:"err,omitempty"` // Empty if no error, otherwise contains the error message

	// Meta information
	Meta []byte `json:"meta,omitempty"` // Unused, can be used for additional Adapters
}

// --------------------------------------------------------------------------
// Message Factory Functions
// --------------------------------------------------------------------------

// NewSetRequest creates a new Set request
func NewSetRequest(key string, value []byte) *Message {
	return &Message{
		MsgType: MsgTKVSet,
		Key:     key,
		Value:   value,
	}
}

// NewSetResponse creates a new Set response
func NewSetResponse(err error) *Message {
	msg := &Message{
		MsgType: MsgTKVSet,
	}
	if err != nil {
		msg.Err = err.Error()
	}
	return msg
}

// NewSetERequest creates a new SetE request
func NewSetERequest(key string, value []byte, expireIn, deleteIn uint64) *Message {
	return &Message{
		MsgType:  MsgTKVSetE,
		Key:      key,
		Value:    value,
		ExpireIn: expireIn,
		DeleteIn: deleteIn,
	}
}

// NewSetEResponse creates a new SetE response
func NewSetEResponse(err error) *Message {
	msg := &Message{
		MsgType: MsgTKVSetE,
	}
	if err != nil {
		msg.Err = err.Error()
	}
	return msg
}

// NewSetEIfUnsetRequest creates a new SetEIfUnset request
func NewSetEIfUnsetRequest(key string, value []byte, expireIn, deleteIn uint64) *Message {
	return &Message{
		MsgType:  MsgTKVSetEIfUnset,
		Key:      key,
		Value:    value,
		ExpireIn: expireIn,
		DeleteIn: deleteIn,
	}
}

// NewSetEIfUnsetResponse creates a new SetEIfUnset response
func NewSetEIfUnsetResponse(err error) *Message {
	msg := &Message{
		MsgType: MsgTKVSetEIfUnset,
	}
	if err != nil {
		msg.Err = err.Error()
	}
	return msg
}

// NewExpireRequest creates a new Expire request
func NewExpireRequest(key string) *Message {
	return &Message{
		MsgType: MsgTKVExpire,
		Key:     key,
	}
}

// NewExpireResponse creates a new Expire response
func NewExpireResponse(err error) *Message {
	msg := &Message{
		MsgType: MsgTKVExpire,
	}
	if err != nil {
		msg.Err = err.Error()
	}
	return msg
}

// NewDeleteRequest creates a new Delete request
func NewDeleteRequest(key string) *Message {
	return &Message{
		MsgType: MsgTKVDelete,
		Key:     key,
	}
}

// NewDeleteResponse creates a new Delete response
func NewDeleteResponse(err error) *Message {
	msg := &Message{
		MsgType: MsgTKVDelete,
	}
	if err != nil {
		msg.Err = err.Error()
	}
	return msg
}

// NewGetRequest creates a new Get request
func NewGetRequest(key string) *Message {
	return &Message{
		MsgType: MsgTKVGet,
		Key:     key,
	}
}

// NewGetResponse creates a new Get response
func NewGetResponse(value []byte, ok bool, err error) *Message {
	msg := &Message{
		MsgType: MsgTKVGet,
		Ok:      ok,
		Value:   value,
	}
	if err != nil {
		msg.Err = err.Error()
	}
	return msg
}

// NewHasRequest creates a new Has request
func NewHasRequest(key string) *Message {
	return &Message{
		MsgType: MsgTKVHas,
		Key:     key,
	}
}

// NewHasResponse creates a new Has response
func NewHasResponse(ok bool, err error) *Message {
	msg := &Message{
		MsgType: MsgTKVHas,
		Ok:      ok,
	}
	if err != nil {
		msg.Err = err.Error()
	}
	return msg
}

// NewAcquireRequest creates a new Acquire request
func NewAcquireRequest(key string, deleteIn uint64) *Message {
	return &Message{
		MsgType:  MsgTLCKAcquire,
		Key:      key,
		DeleteIn: deleteIn,
	}
}

// NewAcquireResponse creates a new Acquire response
func NewAcquireResponse(ok bool, value []byte, err error) *Message {
	msg := &Message{
		MsgType: MsgTLCKAcquire,
		Ok:      ok,
		Value:   value,
	}
	if err != nil {
		msg.Err = err.Error()
	}
	return msg
}

// NewReleaseRequest creates a new Release request
func NewReleaseRequest(key string, ownerId []byte) *Message {
	return &Message{
		MsgType: MsgTLCKRelease,
		Key:     key,
		Value:   ownerId,
	}
}

// NewReleaseResponse creates a new Release response
func NewReleaseResponse(ok bool, err error) *Message {
	msg := &Message{
		MsgType: MsgTLCKRelease,
		Ok:      ok,
	}
	if err != nil {
		msg.Err = err.Error()
	}
	return msg
}

// NewCustomRequest creates a new Custom request
func NewCustomRequest(meta []byte) *Message {
	return &Message{
		MsgType: MsgTCustom,
		Meta:    meta,
	}
}

// NewCustomResponse creates a new Custom response
func NewCustomResponse(meta []byte, err error) *Message {
	msg := &Message{
		MsgType: MsgTCustom,
		Meta:    meta,
	}
	if err != nil {
		msg.Err = err.Error()
	}
	return msg
}

// NewErrorResponse creates a new Error response
func NewErrorResponse(err string) *Message {
	return &Message{
		MsgType: MsgTError,
		Err:     err,
	}
}

// --------------------------------------------------------------------------
// Message Type Definition
// --------------------------------------------------------------------------

// MessageType defines the type of message used in RPC communication.
type MessageType uint8

// String returns the string representation of a MessageType.
func (t MessageType) String() string {
	switch t {
	case MsgTKVSet:
		return "set"
	case MsgTKVSetE:
		return "setE"
	case MsgTKVSetEIfUnset:
		return "setEIfUnset"
	case MsgTKVExpire:
		return "expire"
	case MsgTKVDelete:
		return "delete"
	case MsgTKVGet:
		return "get"
	case MsgTKVHas:
		return "has"
	case MsgTLCKAcquire:
		return "acquire"
	case MsgTLCKRelease:
		return "release"
	case MsgTCustom:
		return "custom"
	case MsgTError:
		return "error"
	case MsgTSuccess:
		return "success"
	default:
		return "unknown"
	}
}

// MarshalJSON implements the json.Marshaller interface for MessageType.
// This allows MessageType to be serialized as a string in JSON.
func (t MessageType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface for MessageType.
// This allows MessageType to be deserialized from a string in JSON.
func (t *MessageType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// Convert string back to MessageType
	switch s {
	case "set":
		*t = MsgTKVSet
	case "setE":
		*t = MsgTKVSetE
	case "setEIfUnset":
		*t = MsgTKVSetEIfUnset
	case "expire":
		*t = MsgTKVExpire
	case "delete":
		*t = MsgTKVDelete
	case "get":
		*t = MsgTKVGet
	case "has":
		*t = MsgTKVHas
	case "acquire":
		*t = MsgTLCKAcquire
	case "release":
		*t = MsgTLCKRelease
	case "custom":
		*t = MsgTCustom
	case "error":
		*t = MsgTError
	case "success":
		*t = MsgTSuccess
	default:
		return fmt.Errorf("unknown message type: %s", s)
	}

	return nil
}

// --------------------------------------------------------------------------
// Message Type Constants
// --------------------------------------------------------------------------

const (
	// General message types

	MsgTUnknown MessageType = iota
	MsgTSuccess             // Indicates a successful operation
	MsgTError               // Indicates an error occurred

	// IStore operations

	MsgTKVSet         // Set a key-value pair
	MsgTKVSetE        // Set a key-value pair with expiration
	MsgTKVSetEIfUnset // Set a key-value pair if not already set
	MsgTKVExpire      // Expire a key
	MsgTKVDelete      // Delete a key-value pair
	MsgTKVGet         // Get a value by key
	MsgTKVHas         // Check if a key exists

	// ILockProvider operations

	MsgTLCKAcquire // Acquire a lock
	MsgTLCKRelease // Release a lock

	// Custom operations

	MsgTCustom // Custom operation type
)
