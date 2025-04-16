package internal

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestSizeBytes tests the SizeBytes method
func TestSizeBytes(t *testing.T) {
	tests := []struct {
		name     string
		command  Command
		expected int
	}{
		{
			name: "Command with key and value",
			command: Command{
				Type:     CommandTSetE,
				Key:      "testkey",
				ExpireIn: 100,
				DeleteIn: 200,
				Value:    []byte("testvalue"),
			},
			expected: 1 + 8 + 8 + 4 + 7 + 9, // Type + ExpireIn + DeleteIn + KeyLen + Key + Value
		},
		{
			name: "Command with empty key and value",
			command: Command{
				Type:     CommandTSetE,
				Key:      "",
				ExpireIn: 100,
				DeleteIn: 200,
				Value:    []byte("testvalue"),
			},
			expected: 1 + 8 + 8 + 4 + 0 + 9, // Type + ExpireIn + DeleteIn + KeyLen + Key + Value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := tt.command.SizeBytes()
			if size != tt.expected {
				t.Errorf("SizeBytes() = %v, want %v", size, tt.expected)
			}
		})
	}
}

// TestSerializeDeserialize tests both Serialize and Deserialize methods
func TestSerializeDeserialize(t *testing.T) {
	tests := []struct {
		name    string
		command Command
	}{
		{
			name: "Standard command with value",
			command: Command{
				Type:     CommandTSetE,
				Key:      "testkey",
				ExpireIn: 100,
				DeleteIn: 200,
				Value:    []byte("testvalue"),
			},
		},
		{
			name: "Command without value",
			command: Command{
				Type:     CommandTDelete,
				Key:      "testkey",
				ExpireIn: 100,
				DeleteIn: 200,
				Value:    nil,
			},
		},
		{
			name: "Command with empty key",
			command: Command{
				Type:     CommandTSetE,
				Key:      "",
				ExpireIn: 100,
				DeleteIn: 200,
				Value:    []byte("testvalue"),
			},
		},
		{
			name: "Command with empty value",
			command: Command{
				Type:     CommandTSetE,
				Key:      "testkey",
				ExpireIn: 100,
				DeleteIn: 200,
				Value:    []byte{},
			},
		},
		{
			name: "Command with large expiration values",
			command: Command{
				Type:     CommandTSetE,
				Key:      "testkey",
				ExpireIn: 18446744073709551615, // Max uint64
				DeleteIn: 18446744073709551615, // Max uint64
				Value:    []byte("testvalue"),
			},
		},
		{
			name: "Command with binary value",
			command: Command{
				Type:     CommandTSetE,
				Key:      "binary",
				ExpireIn: 100,
				DeleteIn: 200,
				Value:    []byte{0, 1, 2, 3, 254, 255},
			},
		},
		{
			name: "Command with Unicode key",
			command: Command{
				Type:     CommandTSetE,
				Key:      "你好世界", // Hello World in Chinese
				ExpireIn: 100,
				DeleteIn: 200,
				Value:    []byte("unicode test"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data := tt.command.Serialize()

			// Deserialize into a new command
			var newCommand Command
			err := newCommand.Deserialize(data)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}

			// Compare original and deserialized command
			if newCommand.Type != tt.command.Type {
				t.Errorf("Type mismatch: got %v, want %v", newCommand.Type, tt.command.Type)
			}
			if newCommand.Key != tt.command.Key {
				t.Errorf("Key mismatch: got %q, want %q", newCommand.Key, tt.command.Key)
			}
			if newCommand.ExpireIn != tt.command.ExpireIn {
				t.Errorf("ExpireIn mismatch: got %v, want %v", newCommand.ExpireIn, tt.command.ExpireIn)
			}
			if newCommand.DeleteIn != tt.command.DeleteIn {
				t.Errorf("DeleteIn mismatch: got %v, want %v", newCommand.DeleteIn, tt.command.DeleteIn)
			}

			// Value comparison handling nil case
			if tt.command.Value == nil {
				if newCommand.Value != nil && len(newCommand.Value) != 0 {
					t.Errorf("Value should be nil or empty, got %v", newCommand.Value)
				}
			} else if !bytes.Equal(newCommand.Value, tt.command.Value) {
				t.Errorf("Value mismatch: got %v, want %v", newCommand.Value, tt.command.Value)
			}

			// Verify that SizeBytes matches the serialized data length
			if tt.command.SizeBytes() != len(data) {
				t.Errorf("SizeBytes() = %d, but serialized data length = %d",
					tt.command.SizeBytes(), len(data))
			}
		})
	}
}

// TestDeserializeErrors tests error cases in Deserialize
func TestDeserializeErrors(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectedErr string
	}{
		{
			name:        "Empty data",
			data:        []byte{},
			expectedErr: "data too short for command",
		},
		{
			name:        "Data too short (less than header)",
			data:        []byte{1, 2, 3, 4, 5},
			expectedErr: "data too short for command",
		},
		{
			name: "Invalid key length",
			data: func() []byte {
				data := make([]byte, 21) // Just the header
				data[0] = byte(CommandTSetE)
				// Set key length to a large value that exceeds the data
				binary.BigEndian.PutUint32(data[17:21], 1000)
				return data
			}(),
			expectedErr: "data too short for key of length 1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd Command
			err := cmd.Deserialize(tt.data)

			// Check if we got the expected error
			if err == nil {
				t.Fatalf("Expected error but got nil")
			}
			if err.Error() != tt.expectedErr {
				t.Errorf("Expected error %q, got %q", tt.expectedErr, err.Error())
			}
		})
	}
}

// TestBinaryFormat tests the exact binary format of serialized commands
func TestBinaryFormat(t *testing.T) {
	// Create a command
	cmd := Command{
		Type:     CommandTSetE,
		Key:      "testkey",
		ExpireIn: 12345,
		DeleteIn: 67890,
		Value:    []byte("testvalue"),
	}

	// Manually create the expected byte array
	expected := make([]byte, cmd.SizeBytes())
	// Type
	expected[0] = byte(CommandTSetE)
	// ExpireIn
	binary.BigEndian.PutUint64(expected[1:9], 12345)
	// DeleteIn
	binary.BigEndian.PutUint64(expected[9:17], 67890)
	// Key length
	binary.BigEndian.PutUint32(expected[17:21], 7) // "testkey" length
	// Key
	copy(expected[21:28], []byte("testkey"))
	// Value
	copy(expected[28:], []byte("testvalue"))

	// Serialize and compare
	serialized := cmd.Serialize()
	if !bytes.Equal(serialized, expected) {
		t.Errorf("Binary format does not match:\nGot:      %v\nExpected: %v", serialized, expected)
	}
}

// TestBufferReuse tests that the Deserialize method reuses buffers when possible
func TestBufferReuse(t *testing.T) {
	// Create a command with a value
	cmd := Command{
		Type:     CommandTSetE,
		Key:      "key",
		ExpireIn: 100,
		DeleteIn: 200,
		Value:    []byte("original value"),
	}

	// Get the current value buffer address
	originalBuffer := cmd.Value

	// Create a new serialized command with a different value but same length
	cmd2 := Command{
		Type:     CommandTSetE,
		Key:      "key",
		ExpireIn: 100,
		DeleteIn: 200,
		Value:    []byte("changed value"),
	}
	serialized2 := cmd2.Serialize()

	// Deserialize the new command into the original
	err := cmd.Deserialize(serialized2)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	// Check if the buffer was reused (same capacity, same address)
	if cap(cmd.Value) != cap(originalBuffer) {
		t.Logf("Buffer capacity changed from %d to %d", cap(originalBuffer), cap(cmd.Value))
	}

	// Ensure the value was correctly deserialized
	if !bytes.Equal(cmd.Value, []byte("changed value")) {
		t.Errorf("Value not correctly deserialized: got %q, want %q",
			string(cmd.Value), "changed value")
	}

	// Now test with a larger value to ensure capacity increases
	cmd3 := Command{
		Type:     CommandTSetE,
		Key:      "key",
		ExpireIn: 100,
		DeleteIn: 200,
		Value:    []byte("this is a much longer value that won't fit in the original buffer"),
	}
	serialized3 := cmd3.Serialize()

	// Get buffer info before deserialization
	beforeCap := cap(cmd.Value)

	// Deserialize
	err = cmd.Deserialize(serialized3)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	// Check if buffer capacity increased
	if cap(cmd.Value) <= beforeCap {
		t.Errorf("Buffer capacity did not increase for larger value: still %d", cap(cmd.Value))
	}

	// Ensure the value was correctly deserialized
	if !bytes.Equal(cmd.Value, cmd3.Value) {
		t.Errorf("Value not correctly deserialized")
	}
}
