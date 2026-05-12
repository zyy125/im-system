package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitRealtimeBuildsHubWithoutBroker(t *testing.T) {
	rt, err := initRealtime(nil, &repositories{}, &services{})

	assert.NoError(t, err)
	if assert.NotNil(t, rt) {
		assert.NotNil(t, rt.hub)
		assert.NotNil(t, rt.hub.Lifecycle)
	}
}
