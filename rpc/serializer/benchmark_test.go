package serializer

import (
	"github.com/ValentinKolb/dKV/rpc/common"
	"testing"
)

// benchmarkMessages returns a set of messages for targeted benchmarking
func benchmarkMessages() map[string]common.Message {
	return map[string]common.Message{
		"Empty": {
			MsgType: common.MsgTSuccess,
		},
		"SmallKeyOnly": {
			MsgType: common.MsgTKVGet,
			Key:     "k",
		},
		"MediumKeyOnly": {
			MsgType: common.MsgTKVGet,
			Key:     "medium-length-key-for-testing",
		},
		"LargeKeyOnly": {
			MsgType: common.MsgTKVGet,
			Key:     "this-is-a-very-large-key-that-could-be-used-for-storing-data-or-as-a-document-id-in-some-cases",
		},
		"SmallValue": {
			MsgType: common.MsgTKVSet,
			Key:     "key",
			Value:   []byte("v"),
		},
		"MediumValue": {
			MsgType: common.MsgTKVSet,
			Key:     "key",
			Value:   []byte("medium length value for testing serialization"),
		},
		"LargeValue": {
			MsgType: common.MsgTKVSet,
			Key:     "key",
			Value:   make([]byte, 1024), // 1KB of data
		},
		"VeryLargeValue": {
			MsgType: common.MsgTKVSet,
			Key:     "key",
			Value:   make([]byte, 1024*16), // 16KB of data
		},
		"CompleteMessage": {
			MsgType:  common.MsgTLCKAcquire,
			Key:      "complete-test-key",
			ExpireIn: 10000,
			DeleteIn: 20000,
			Value:    []byte("test-value-data"),
			Ok:       true,
			Err:      "This is a test error message",
			Meta:     []byte("test-meta-data-for-benchmarking"),
		},
		"ErrorMessage": {
			MsgType: common.MsgTError,
			Err:     "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
		},
	}
}

// BenchmarkSerialize benchmarks serialization for all implementations with various message types
func BenchmarkSerialize(b *testing.B) {
	messages := benchmarkMessages()

	for name, factory := range testSerializers {
		for msgName, msg := range messages {
			b.Run(name+"_"+msgName, func(b *testing.B) {
				serializer := factory()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					_, err := serializer.Serialize(msg)
					if err != nil {
						b.Fatalf("Failed to serialize: %v", err)
					}
				}
			})
		}
	}
}

// BenchmarkDeserialize benchmarks deserialization for all implementations with various message types
func BenchmarkDeserialize(b *testing.B) {
	messages := benchmarkMessages()
	serializedData := make(map[string]map[string][]byte)

	// Pre-serialize all messages with all serializers
	for name, factory := range testSerializers {
		serializer := factory()
		serializedData[name] = make(map[string][]byte)

		for msgName, msg := range messages {
			data, err := serializer.Serialize(msg)
			if err != nil {
				b.Fatalf("Failed to serialize %s with %s: %v", msgName, name, err)
			}
			serializedData[name][msgName] = data
		}
	}

	// Benchmark deserialization
	for name, factory := range testSerializers {
		for msgName := range messages {
			b.Run(name+"_"+msgName, func(b *testing.B) {
				serializer := factory()
				data := serializedData[name][msgName]
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					var msg common.Message
					err := serializer.Deserialize(data, &msg)
					if err != nil {
						b.Fatalf("Failed to deserialize: %v", err)
					}
				}
			})
		}
	}
}

// BenchmarkSize measures and reports the serialized size for each message type
func BenchmarkSize(b *testing.B) {
	messages := benchmarkMessages()

	for name, factory := range testSerializers {
		serializer := factory()

		for msgName, msg := range messages {
			b.Run(name+"_"+msgName, func(b *testing.B) {
				data, err := serializer.Serialize(msg)
				if err != nil {
					b.Fatalf("Failed to serialize: %v", err)
				}

				// Report the size as a custom metric
				b.ReportMetric(float64(len(data)), "bytes")

				// Minimal loop to satisfy benchmark requirements
				for i := 0; i < b.N; i++ {
					_ = data
				}
			})
		}
	}
}
