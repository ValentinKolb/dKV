package serializer

import (
	"encoding/binary"
	"fmt"
	"github.com/ValentinKolb/dKV/rpc/common"
)

// NewBinarySerializer creates a new serializer using a custom binary format
// optimized for speed and efficiency
func NewBinarySerializer() IRPCSerializer {
	return &binarySerializerImpl{}
}

// binarySerializerImpl implements IRPCSerializer using a custom binary format
type binarySerializerImpl struct {
}

// Bit flags to indicate which optional fields are present
const (
	hasKey      byte = 1 << 0
	hasExpireIn byte = 1 << 1
	hasDeleteIn byte = 1 << 2
	hasValue    byte = 1 << 3
	hasOk       byte = 1 << 4
	hasErr      byte = 1 << 5
	hasMeta     byte = 1 << 6
)

// --------------------------------------------------------------------------
// Interface Methods (docu see serializer.IRPCSerializer)
// --------------------------------------------------------------------------

func (b binarySerializerImpl) Serialize(msg common.Message) ([]byte, error) {
	// Calculate total size needed
	totalSize := b.sizeBytes(msg)
	result := make([]byte, totalSize)

	// Write message type
	result[0] = byte(msg.MsgType)

	// Initialize flags byte
	var flags byte = 0

	// Set position for writing
	pos := 2 // Start after MsgType and flags

	// Handle Key
	if msg.Key != "" {
		flags |= hasKey
		keyBytes := []byte(msg.Key)
		keyLen := len(keyBytes)

		// Write key length
		binary.BigEndian.PutUint32(result[pos:pos+4], uint32(keyLen))
		pos += 4

		// Write key data
		copy(result[pos:pos+keyLen], keyBytes)
		pos += keyLen
	}

	// Handle ExpireIn
	if msg.ExpireIn > 0 {
		flags |= hasExpireIn
		binary.BigEndian.PutUint64(result[pos:pos+8], msg.ExpireIn)
		pos += 8
	}

	// Handle DeleteIn
	if msg.DeleteIn > 0 {
		flags |= hasDeleteIn
		binary.BigEndian.PutUint64(result[pos:pos+8], msg.DeleteIn)
		pos += 8
	}

	// Handle Value
	if msg.Value != nil {
		flags |= hasValue
		valueLen := len(msg.Value)

		// Write value length
		binary.BigEndian.PutUint32(result[pos:pos+4], uint32(valueLen))
		pos += 4

		// Write value data
		if valueLen > 0 {
			copy(result[pos:pos+valueLen], msg.Value)
			pos += valueLen
		}
	}

	// Handle Ok
	if msg.Ok {
		flags |= hasOk
		if msg.Ok {
			result[pos] = 1
		} else {
			result[pos] = 0
		}
		pos += 1
	}

	// Handle Err
	if msg.Err != "" {
		flags |= hasErr
		errBytes := []byte(msg.Err)
		errLen := len(errBytes)

		// Write error length
		binary.BigEndian.PutUint32(result[pos:pos+4], uint32(errLen))
		pos += 4

		// Write error data
		copy(result[pos:pos+errLen], errBytes)
		pos += errLen
	}

	// Handle Meta
	if msg.Meta != nil {
		flags |= hasMeta
		metaLen := len(msg.Meta)

		// Write meta length
		binary.BigEndian.PutUint32(result[pos:pos+4], uint32(metaLen))
		pos += 4

		// Write meta data
		if metaLen > 0 {
			copy(result[pos:pos+metaLen], msg.Meta)
			pos += metaLen
		}
	}

	// Set flags byte after knowing which fields are present
	result[1] = flags

	return result, nil
}

func (b binarySerializerImpl) Deserialize(data []byte, msg *common.Message) error {
	// Check minimum size (MsgType + flags)
	if len(data) < 2 {
		return fmt.Errorf("data too short for message header")
	}

	// Read message type
	msg.MsgType = common.MessageType(data[0])

	// Read flags
	flags := data[1]

	// Initialize read position
	pos := 2

	// Read Key if present
	if flags&hasKey != 0 {
		if pos+4 > len(data) {
			return fmt.Errorf("data too short for key length")
		}

		// Read key length
		keyLen := binary.BigEndian.Uint32(data[pos : pos+4])
		pos += 4

		if pos+int(keyLen) > len(data) {
			return fmt.Errorf("data too short for key data")
		}

		// Read key data
		msg.Key = string(data[pos : pos+int(keyLen)])
		pos += int(keyLen)
	} else {
		msg.Key = ""
	}

	// Read ExpireIn if present
	if flags&hasExpireIn != 0 {
		if pos+8 > len(data) {
			return fmt.Errorf("data too short for ExpireIn")
		}

		msg.ExpireIn = binary.BigEndian.Uint64(data[pos : pos+8])
		pos += 8
	} else {
		msg.ExpireIn = 0
	}

	// Read DeleteIn if present
	if flags&hasDeleteIn != 0 {
		if pos+8 > len(data) {
			return fmt.Errorf("data too short for DeleteIn")
		}

		msg.DeleteIn = binary.BigEndian.Uint64(data[pos : pos+8])
		pos += 8
	} else {
		msg.DeleteIn = 0
	}

	// Read Value if present
	if flags&hasValue != 0 {
		if pos+4 > len(data) {
			return fmt.Errorf("data too short for value length")
		}

		// Read value length
		valueLen := binary.BigEndian.Uint32(data[pos : pos+4])
		pos += 4

		if pos+int(valueLen) > len(data) {
			return fmt.Errorf("data too short for value data")
		}

		// Read value data - create an empty slice (not nil) if length is 0
		// Allocate only if needed
		if msg.Value == nil || cap(msg.Value) < int(valueLen) {
			msg.Value = make([]byte, valueLen)
		} else {
			msg.Value = msg.Value[:valueLen]
		}

		if valueLen > 0 {
			copy(msg.Value, data[pos:pos+int(valueLen)])
		}
		pos += int(valueLen)
	} else {
		msg.Value = nil
	}

	// Read Ok if present
	if flags&hasOk != 0 {
		if pos+1 > len(data) {
			return fmt.Errorf("data too short for Ok flag")
		}

		msg.Ok = data[pos] != 0
		pos += 1
	} else {
		msg.Ok = false
	}

	// Read Err if present
	if flags&hasErr != 0 {
		if pos+4 > len(data) {
			return fmt.Errorf("data too short for error length")
		}

		// Read error length
		errLen := binary.BigEndian.Uint32(data[pos : pos+4])
		pos += 4

		if pos+int(errLen) > len(data) {
			return fmt.Errorf("data too short for error data")
		}

		// Read error data
		msg.Err = string(data[pos : pos+int(errLen)])
		pos += int(errLen)
	} else {
		msg.Err = ""
	}

	// Read Meta if present
	if flags&hasMeta != 0 {
		if pos+4 > len(data) {
			return fmt.Errorf("data too short for meta length")
		}

		// Read meta length
		metaLen := binary.BigEndian.Uint32(data[pos : pos+4])
		pos += 4

		if pos+int(metaLen) > len(data) {
			return fmt.Errorf("data too short for meta data")
		}

		// Read metadata - create an empty slice (not nil) if length is 0
		// Allocate only if needed
		if msg.Meta == nil || cap(msg.Meta) < int(metaLen) {
			msg.Meta = make([]byte, metaLen)
		} else {
			msg.Meta = msg.Meta[:metaLen]
		}

		if metaLen > 0 {
			copy(msg.Meta, data[pos:pos+int(metaLen)])
		}
		pos += int(metaLen)
	} else {
		msg.Meta = nil
	}

	return nil
}

// --------------------------------------------------------------------------
// Helper Methods
// --------------------------------------------------------------------------

// sizeBytes calculates the total size needed for serialization
func (b binarySerializerImpl) sizeBytes(msg common.Message) int {
	// 1 byte for MsgType + 1 byte for flags
	size := 2

	// Add sizes for fields that require length encoding
	if msg.Key != "" {
		size += 4 + len(msg.Key) // 4 bytes for length + key string
	}
	if msg.ExpireIn > 0 {
		size += 8 // uint64
	}
	if msg.DeleteIn > 0 {
		size += 8 // uint64
	}
	if msg.Value != nil {
		size += 4 + len(msg.Value) // 4 bytes for length + value bytes
	}
	if msg.Ok {
		size += 1 // 1 byte for boolean
	}
	if msg.Err != "" {
		size += 4 + len(msg.Err) // 4 bytes for length + error string
	}
	if msg.Meta != nil {
		size += 4 + len(msg.Meta) // 4 bytes for length + meta bytes
	}

	return size
}
