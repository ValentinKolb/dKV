package base

import (
	"encoding/binary"
	"io"
	"net"
)

// writeFrame writes a frame to the connection with the format:
// - 8 bytes: shardId (uint64, big endian)
// - 8 bytes: requestID (uint64, big endian)
// - 4 bytes: data length (uint32, big endian)
// - N bytes: data payload
func writeFrame(conn net.Conn, shardID uint64, requestID uint64, data []byte) error {
	// Create the header (8 bytes for shardId + 8 bytes for requestID + 4 bytes for content length)
	header := make([]byte, 20)
	binary.BigEndian.PutUint64(header[:8], shardID)
	binary.BigEndian.PutUint64(header[8:16], requestID)
	binary.BigEndian.PutUint32(header[16:20], uint32(len(data)))

	b := net.Buffers{header, data}
	_, err := b.WriteTo(conn)
	return err
}

// readFrame reads a frame from the connection using the provided buffer
// If the buffer is too small, it will allocate a new temporary buffer for the data
func readFrame(conn net.Conn, buf []byte) (uint64, uint64, []byte, error) {
	// Check if buffer is large enough for header
	if buf == nil || len(buf) < 20 {
		buf = make([]byte, 20) // create header buffer
	}

	// Read header
	if _, err := io.ReadFull(conn, buf[:20]); err != nil {
		return 0, 0, nil, err
	}

	// Parse header
	shardID := binary.BigEndian.Uint64(buf[:8])
	requestID := binary.BigEndian.Uint64(buf[8:16])
	contentLength := binary.BigEndian.Uint32(buf[16:20])

	// If no data, return empty slice
	if contentLength == 0 {
		return shardID, requestID, []byte{}, nil
	}

	// Check if buffer is large enough for data
	if len(buf) < int(contentLength) {
		buf = make([]byte, contentLength)
	}

	// Read data
	if _, err := io.ReadFull(conn, buf[:contentLength]); err != nil {
		return 0, 0, nil, err
	}

	// Return data
	return shardID, requestID, buf[:contentLength], nil
}
