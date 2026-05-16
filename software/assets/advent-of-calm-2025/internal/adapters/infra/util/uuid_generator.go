package util

import (
	"crypto/rand"
	"fmt"
)

type UUIDGenerator struct{}

func (g *UUIDGenerator) GenerateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback or panic in extreme cases, but for this simplified version:
		return "00000000-0000-0000-0000-000000000000"
	}
	// UUID v4 variant
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
