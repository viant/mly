package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_LatencyBreakerSettings_Disabled(t *testing.T) {
	settings, enabled, err := (&Config{}).latencyBreakerSettings()
	require.NoError(t, err)
	assert.False(t, enabled)
	assert.Zero(t, settings.latest)
	assert.Zero(t, settings.rolling)
}

func TestConfig_LatencyBreakerSettings_Defaults(t *testing.T) {
	cfg := &Config{LatencyBreakerLatestThreshold: 40 * time.Millisecond}

	settings, enabled, err := cfg.latencyBreakerSettings()
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, 40*time.Millisecond, settings.latest)
	assert.Equal(t, defaultLatencyBreakerRollingWindow, settings.window)
	assert.Equal(t, defaultLatencyBreakerKConsecutive, settings.k)
	assert.Equal(t, defaultLatencyBreakerPassThroughFraction, settings.fraction)
}

func TestConfig_LatencyBreakerSettings_ExplicitValues(t *testing.T) {
	cfg := &Config{
		LatencyBreakerRollingThreshold:    20 * time.Millisecond,
		LatencyBreakerRollingWindow:       500 * time.Millisecond,
		LatencyBreakerKConsecutive:        5,
		LatencyBreakerPassThroughFraction: 0.25,
	}

	settings, enabled, err := cfg.latencyBreakerSettings()
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Equal(t, 20*time.Millisecond, settings.rolling)
	assert.Equal(t, 500*time.Millisecond, settings.window)
	assert.Equal(t, 5, settings.k)
	assert.Equal(t, 0.25, settings.fraction)
}

func TestConfig_LatencyBreakerSettings_Invalid(t *testing.T) {
	cases := []struct {
		name   string
		config Config
	}{
		{name: "negative latest threshold", config: Config{LatencyBreakerLatestThreshold: -time.Millisecond}},
		{name: "negative rolling threshold", config: Config{LatencyBreakerRollingThreshold: -time.Millisecond}},
		{name: "negative rolling window", config: Config{LatencyBreakerRollingWindow: -time.Millisecond}},
		{name: "negative consecutive count", config: Config{LatencyBreakerKConsecutive: -1}},
		{name: "negative pass-through fraction", config: Config{LatencyBreakerPassThroughFraction: -0.1}},
		{name: "pass-through fraction above one", config: Config{LatencyBreakerPassThroughFraction: 1.1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := tc.config.latencyBreakerSettings()
			assert.Error(t, err)
		})
	}
}

func TestNew_InvalidLatencyBreakerOptionReturnsError(t *testing.T) {
	_, err := New(
		"invalid-latency-breaker",
		[]*Host{NewHost("localhost", 8080)},
		WithLatencyBreaker(-time.Millisecond, 0, 0, 0, 0),
	)
	assert.ErrorContains(t, err, "LatencyBreakerLatestThreshold")
}
