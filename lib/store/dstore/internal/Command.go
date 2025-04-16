package internal

import (
	"encoding/binary"
	"fmt"
	"github.com/ValentinKolb/dKV/lib/db"
)

// CommandType defines the possible operations for the state machine.
type CommandType uint8

const (
	CommandTSet        CommandType = iota // Insert or update an entry.
	CommandTSetE                          // Insert or update an entry with expiration and deletion times.
	CommandTSetIfUnset                    // Insert an entry if it does not exist.
	CommandTExpire                        // Expire the value of an entry immediately.
	CommandTDelete                        // Delete an entry.
)

func (ct CommandType) String() string {
	switch ct {
	case CommandTSet:
		return "Set"
	case CommandTSetE:
		return "SetE"
	case CommandTSetIfUnset:
		return "SetIfUnset"
	case CommandTExpire:
		return "Expire"
	case CommandTDelete:
		return "Delete"
	default:
		return fmt.Sprintf("Unknown(%d)", ct)
	}
}

// ToDBFeature converts a CommandType to the corresponding db.Feature.
// This can be used for checking if the database supports a certain operation.
func (ct CommandType) ToDBFeature() (db.Feature, error) {
	switch ct {
	case CommandTSet:
		return db.FeatureSet, nil
	case CommandTSetE:
		return db.FeatureSetE, nil
	case CommandTSetIfUnset:
		return db.FeatureSetEIfUnset, nil
	case CommandTExpire:
		return db.FeatureExpire, nil
	case CommandTDelete:
		return db.FeatureDelete, nil
	default:
		return 0, fmt.Errorf("unknown command type %d", ct)
	}
}

// Command represents a command to be executed by the state machine (a single entry in the raft log)
type Command struct {
	Type     CommandType
	Key      string
	ExpireIn uint64
	DeleteIn uint64
	Value    []byte
}

// SizeBytes returns the exact number of bytes needed to serialize this command
func (command *Command) SizeBytes() int {
	size := 1 + 8 + 8 + 4 + len(command.Key) // Type + ExpireIn + DeleteIn + KeyLen + Key
	if command.Value != nil {
		size += len(command.Value)
	}
	return size
}

// Serialize serializes a command into a byte array with the format:
// 1 byte for operation type,
// 8 bytes for expireIn,
// 8 bytes for deleteIn,
// 4 bytes for key length (big endian),
// N bytes for key data,
// N bytes for value data (optional)
func (command *Command) Serialize() []byte {
	// Use SizeBytes to calculate the total size needed
	totalSize := command.SizeBytes()

	result := make([]byte, totalSize)

	// Set operation type
	result[0] = byte(command.Type)

	// Set expireIn and deleteIn
	binary.BigEndian.PutUint64(result[1:9], command.ExpireIn)
	binary.BigEndian.PutUint64(result[9:17], command.DeleteIn)

	// Set key length (4 bytes, big endian)
	binary.BigEndian.PutUint32(result[17:21], uint32(len(command.Key)))

	// Copy key bytes
	keyBytes := []byte(command.Key)
	copy(result[21:21+len(keyBytes)], keyBytes)

	// Copy value if present
	if command.Value != nil {
		copy(result[21+len(keyBytes):], command.Value)
	}

	return result
}

// Deserialize extracts all Command fields from a byte array.
func (command *Command) Deserialize(data []byte) error {
	// Minimum size: 1 (Type) + 8 (ExpireIn) + 8 (DeleteIn) + 4 (KeyLen) = 21 bytes
	if len(data) < 21 {
		return fmt.Errorf("data too short for command")
	}

	// Extract operation type
	command.Type = CommandType(data[0])

	// Extract expireIn and deleteIn
	command.ExpireIn = binary.BigEndian.Uint64(data[1:9])
	command.DeleteIn = binary.BigEndian.Uint64(data[9:17])

	// Extract key length
	keyLen := binary.BigEndian.Uint32(data[17:21])

	// Validate key length
	if len(data) < 21+int(keyLen) {
		return fmt.Errorf("data too short for key of length %d", keyLen)
	}

	// Extract key
	command.Key = string(data[21 : 21+keyLen])

	// Extract value if present
	if len(data) > 21+int(keyLen) {
		valueLen := len(data) - (21 + int(keyLen))
		// Reuse existing buffer if possible to reduce allocations
		if command.Value == nil || cap(command.Value) < valueLen {
			command.Value = make([]byte, valueLen)
		} else {
			command.Value = command.Value[:valueLen]
		}
		copy(command.Value, data[21+int(keyLen):])
	} else {
		command.Value = nil
	}

	return nil
}
