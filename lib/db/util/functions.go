package util

import (
	"crypto/rand"
	"encoding/binary"
	"time"
)

// --------------------------------------------------------------------------
// General Utility Functions
// --------------------------------------------------------------------------

// GenerateSeed creates a more robust random seed for internal hash distribution
func GenerateSeed() uint64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback mit aktueller Zeit, nur im äußersten Notfall
		return uint64(time.Now().UnixNano())
	}
	return binary.LittleEndian.Uint64(b[:])
}

// --------------------------------------------------------------------------
// Hash Functions
// --------------------------------------------------------------------------

// UintKey is an efficient key type based on uint64 for internal hash representation
type UintKey uint64

// HashString generates a hash value for a string with a seed
// This function uses the FNV-1a hash algorithm, which is fast and has good distribution
func HashString(s string, seed uint64) UintKey {

	// FNV-1a hash with seed incorporation
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)

	// Start with the offset combined with our seed for uniqueness
	hash := uint64(offset64) ^ seed

	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= prime64
	}

	return UintKey(hash)
}
