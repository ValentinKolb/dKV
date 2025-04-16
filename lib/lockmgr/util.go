package lockmgr

import (
	"crypto/rand"
)

const (
	bitLength = 256
)

// generateOwnerID creates a new unique owner ID
// The owner ID is a random byte slice of length 256.
func generateOwnerID() ([]byte, error) {
	randomBytes := make([]byte, bitLength)
	_, err := rand.Read(randomBytes)
	return randomBytes, err
}
