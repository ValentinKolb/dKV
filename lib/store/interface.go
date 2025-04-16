package store

import (
	"fmt"
	"github.com/ValentinKolb/dKV/lib/db"
)

// --------------------------------------------------------------------------
// Interface Definition
// --------------------------------------------------------------------------

// DBFactory is a function type that creates a new db used by the store.
// This is used to abstract the creation of the db from the store implementation.
type DBFactory func() db.KVDB

// IStore is the generic interface for interacting with a key–value store.
// All write operations return only a *Error (nil on success),
// while read operations return the requested data along with a *Error (nil on success).
type IStore interface {
	// Set inserts or updates a key–value pair.
	Set(key string, value []byte) (err error)
	// SetE inserts or updates a key–value pair with expiration and or deletion timestamps.
	// A zero value for expireIn and deleteIn means no expiration or deletion.
	SetE(key string, value []byte, expireIn, deleteIn uint64) (err error)
	// SetEIfUnset inserts a key–value pair if the key does not exist.
	// If the key already exists, the old value is not updated, no matter the value of expireIn and deleteIn.
	// No error is returned if the key already exists.
	SetEIfUnset(key string, value []byte, expireIn, deleteIn uint64) (err error)
	// Expire expired the value for a key. The key should still be findable with the Has() method.
	Expire(key string) (err error)
	// Delete deletes a key–value pair. The key should be removed from the store.
	Delete(key string) (err error)
	// Get return the value for a key. The boolean return value indicates whether a value for the key was found.
	Get(key string) (value []byte, loaded bool, err error)
	// Has returns whether a key exists in the store. The method should return true even if the value for the key is expired.
	Has(key string) (loaded bool, err error)
	// GetDBInfo returns metadata about the database underlying the store.
	// It is not guaranteed that all fields are filled in or that the information is up-to-date!
	GetDBInfo() (info db.DatabaseInfo, err error)
}

// --------------------------------------------------------------------------
// Custom Error Type
// --------------------------------------------------------------------------

// Error is a custom error type that wraps a return code (of type RetCode)
// and an error message.
type Error struct {
	Code RetCode // The return code
	Msg  string  // The error message.
}

// Error implements the error interface.
func (e *Error) Error() string {
	fmt.Println("code:", e.Code)
	errorCode := ""
	switch e.Code {
	case RetCInternalError:
		errorCode = "RetCInternalError"
	case RetCInvalidOperation:
		errorCode = "InvalidOperation"
	default:
		errorCode = "Unknown"
	}

	return fmt.Sprintf("KVStoreError (code %s): %s", errorCode, e.Msg)
}

// NewError creates a new KVStoreError with the given code and message.
func NewError(code RetCode, msg string) *Error {
	return &Error{
		Code: code,
		Msg:  msg,
	}
}

// --------------------------------------------------------------------------
// Return Codes
// --------------------------------------------------------------------------

type RetCode uint64

const (
	RetCSuccess              RetCode = iota // 0: Command executed successfully.
	RetCInternalError                       // 1: Command failed due to an internal error.
	RetCUnsupportedOperation                // 2: Operation is not supported by underlying database.
	RetCInvalidOperation                    // 3: Invalid operation.
)
