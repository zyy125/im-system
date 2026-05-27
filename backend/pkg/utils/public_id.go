package utils

import (
	"crypto/rand"
	"encoding/binary"
)

const (
	minPublicID uint64 = 100000000
	maxPublicID uint64 = 999999999
)

func GeneratePublicID() (uint64, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, err
	}

	span := maxPublicID - minPublicID + 1
	value := binary.BigEndian.Uint64(buf[:]) % span
	return minPublicID + value, nil
}
