package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseJWT(t *testing.T) {

	tests := []struct {
		name      string
		secret    string
		expectErr bool
	}{
		{"correct secret", "test-secret", false},
		{"wrong secret", "wrong-secret", true},
	}

	token, _, _ := GenerateJWT("123", "test-secret", time.Hour)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := ParseJWT(token, tt.secret)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

		})
	}
}
