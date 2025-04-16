package serializer

import (
	"github.com/ValentinKolb/dKV/rpc/common"
	"reflect"
	"testing"
)

// testSerializers is a map of serializer name to factory function
var testSerializers = map[string]func() IRPCSerializer{
	"JSON":   NewJSONSerializer,
	"GOB":    NewGOBSerializer,
	"Binary": NewBinarySerializer,
}

// testMessages creates a set of test messages with different fields filled
func testMessages() []common.Message {
	return []common.Message{
		// Basic message with just a type
		{MsgType: common.MsgTSuccess},

		// Set request
		{
			MsgType: common.MsgTKVSet,
			Key:     "test-key",
			Value:   []byte("test-value"),
		},

		// Get response
		{
			MsgType: common.MsgTKVGet,
			Key:     "test-key",
			Value:   []byte("test-value"),
			Ok:      true,
		},

		// Error response
		{
			MsgType: common.MsgTError,
			Err:     "test error message",
		},

		// Message with all fields filled
		{
			MsgType:  common.MsgTLCKAcquire,
			Key:      "test-lock-key",
			ExpireIn: 60,
			DeleteIn: 300,
			Value:    []byte("test-lock-value"),
			Ok:       true,
			Err:      "",
			Meta:     []byte("test-meta-data"),
		},
	}
}

// TestSerializerRoundTrip tests that messages can be serialized and deserialized correctly
func TestSerializerRoundTrip(t *testing.T) {
	messages := testMessages()

	for name, factory := range testSerializers {
		t.Run(name, func(t *testing.T) {
			serializer := factory()

			for i, msg := range messages {
				// Serialize
				data, err := serializer.Serialize(msg)
				if err != nil {
					t.Errorf("Failed to serialize message %d: %v", i, err)
					continue
				}

				// Deserialize
				var result common.Message
				err = serializer.Deserialize(data, &result)
				if err != nil {
					t.Errorf("Failed to deserialize message %d: %v", i, err)
					continue
				}

				// Compare
				if !reflect.DeepEqual(msg, result) {
					t.Errorf("Message %d doesn't match after round trip:\nOriginal: %+v\nResult: %+v",
						i, msg, result)
				}
			}
		})
	}
}

// TestMessageTypes tests each message type with each serializer
func TestMessageTypes(t *testing.T) {
	for name, factory := range testSerializers {
		t.Run(name, func(t *testing.T) {
			serializer := factory()

			// Test each message type (don't test for MsgTUnknown since this should raise an error)
			for msgType := common.MsgTSuccess; msgType <= common.MsgTCustom; msgType++ {
				msg := common.Message{MsgType: msgType}

				// Serialize
				data, err := serializer.Serialize(msg)
				if err != nil {
					t.Errorf("Failed to serialize message type %s: %v", msgType.String(), err)
					continue
				}

				// Deserialize
				var result common.Message
				err = serializer.Deserialize(data, &result)
				if err != nil {
					t.Errorf("Failed to deserialize message type %s: %v", msgType.String(), err)
					continue
				}

				// Check type
				if result.MsgType != msgType {
					t.Errorf("Message type doesn't match after round trip: Expected %s, got %s",
						msgType.String(), result.MsgType.String())
				}
			}
		})
	}
}

// TestBinarySerializerSpecific tests specific edge cases for the binary serializer
func TestBinarySerializerSpecific(t *testing.T) {
	serializer := NewBinarySerializer()

	// Test cases for empty or zero values
	testCases := []struct {
		name string
		msg  common.Message
	}{
		{
			name: "Empty message",
			msg:  common.Message{},
		},
		{
			name: "Message with empty strings and zero values",
			msg: common.Message{
				MsgType:  common.MsgTKVSet,
				Key:      "",
				ExpireIn: 0,
				DeleteIn: 0,
				Value:    []byte{},
				Ok:       false,
				Err:      "",
				Meta:     []byte{},
			},
		},
		{
			name: "Message with empty strings but Ok=true",
			msg: common.Message{
				MsgType: common.MsgTKVGet,
				Key:     "",
				Ok:      true,
				Value:   nil,
			},
		},
		{
			name: "Message with empty value slice but not nil",
			msg: common.Message{
				MsgType: common.MsgTKVSet,
				Key:     "test",
				Value:   []byte{},
			},
		},
		{
			name: "Message with empty meta slice but not nil",
			msg: common.Message{
				MsgType: common.MsgTCustom,
				Meta:    []byte{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize
			data, err := serializer.Serialize(tc.msg)
			if err != nil {
				t.Fatalf("Failed to serialize: %v", err)
			}

			// Deserialize
			var result common.Message
			err = serializer.Deserialize(data, &result)
			if err != nil {
				t.Fatalf("Failed to deserialize: %v", err)
			}

			// Verify key
			if tc.msg.Key != result.Key {
				t.Errorf("Key mismatch: expected '%s', got '%s'", tc.msg.Key, result.Key)
			}

			// Verify ExpireIn
			if tc.msg.ExpireIn != result.ExpireIn {
				t.Errorf("ExpireIn mismatch: expected %d, got %d", tc.msg.ExpireIn, result.ExpireIn)
			}

			// Verify DeleteIn
			if tc.msg.DeleteIn != result.DeleteIn {
				t.Errorf("DeleteIn mismatch: expected %d, got %d", tc.msg.DeleteIn, result.DeleteIn)
			}

			// Verify Ok
			if tc.msg.Ok != result.Ok {
				t.Errorf("Ok mismatch: expected %v, got %v", tc.msg.Ok, result.Ok)
			}

			// Verify Err
			if tc.msg.Err != result.Err {
				t.Errorf("Err mismatch: expected '%s', got '%s'", tc.msg.Err, result.Err)
			}

			// Verify MsgType
			if tc.msg.MsgType != result.MsgType {
				t.Errorf("MsgType mismatch: expected %v, got %v", tc.msg.MsgType, result.MsgType)
			}

			// Special handling for byte slices that may be nil or empty
			if (tc.msg.Value == nil) != (result.Value == nil) {
				t.Errorf("Value nil/non-nil mismatch: expected %v, got %v", tc.msg.Value, result.Value)
			} else if tc.msg.Value != nil && result.Value != nil {
				if len(tc.msg.Value) != len(result.Value) {
					t.Errorf("Value length mismatch: expected %d, got %d", len(tc.msg.Value), len(result.Value))
				} else {
					for i := 0; i < len(tc.msg.Value); i++ {
						if tc.msg.Value[i] != result.Value[i] {
							t.Errorf("Value content mismatch at index %d", i)
							break
						}
					}
				}
			}

			// Same for Meta
			if (tc.msg.Meta == nil) != (result.Meta == nil) {
				t.Errorf("Meta nil/non-nil mismatch: expected %v, got %v", tc.msg.Meta, result.Meta)
			} else if tc.msg.Meta != nil && result.Meta != nil {
				if len(tc.msg.Meta) != len(result.Meta) {
					t.Errorf("Meta length mismatch: expected %d, got %d", len(tc.msg.Meta), len(result.Meta))
				} else {
					for i := 0; i < len(tc.msg.Meta); i++ {
						if tc.msg.Meta[i] != result.Meta[i] {
							t.Errorf("Meta content mismatch at index %d", i)
							break
						}
					}
				}
			}
		})
	}
}

// TestInvalidBinaryData tests how the binary serializer handles corrupt or invalid data
func TestInvalidBinaryData(t *testing.T) {
	serializer := NewBinarySerializer()

	testCases := []struct {
		name        string
		data        []byte
		expectError bool
	}{
		{
			name:        "Empty data",
			data:        []byte{},
			expectError: true,
		},
		{
			name:        "Too short header",
			data:        []byte{1}, // Only message type, no flags
			expectError: true,
		},
		{
			name:        "Valid header only",
			data:        []byte{1, 0}, // Message type 1, no flags
			expectError: false,
		},
		{
			name:        "Invalid length for key",
			data:        []byte{1, 1, 0, 0, 0, 5, 'a', 'b', 'c'}, // Claims key length 5 but only 3 bytes provided
			expectError: true,
		},
		{
			name:        "Invalid length for value",
			data:        []byte{1, 8, 0, 0, 0, 10}, // Claims value length 10 but no bytes provided
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var msg common.Message
			err := serializer.Deserialize(tc.data, &msg)

			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
			} else if !tc.expectError && err != nil {
				t.Errorf("Did not expect error but got: %v", err)
			}
		})
	}
}
