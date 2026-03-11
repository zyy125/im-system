package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifyPassword(t *testing.T) {
	password := "test-password"
	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	err = VerifyPassword(password, hash)
	assert.NoError(t, err)
}