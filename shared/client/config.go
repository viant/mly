package client

import (
	"fmt"
	"time"

	"github.com/viant/mly/shared/client/config"
)

const (
	defaultLatencyBreakerRollingWindow       = time.Second
	defaultLatencyBreakerKConsecutive        = 3
	defaultLatencyBreakerPassThroughFraction = 0.01
)

type latencyBreakerSettings struct {
	latest, rolling, window time.Duration
	k                       int
	fraction                float64
}

// Config represents a client config
type Config struct {
	Hosts []*Host
	Model string

	CacheSizeMb int

	// CacheScope limits which caches are available to this client.
	CacheScope *CacheScope

	Datastore *config.Remote

	// MaxRetry defines the maximum number of HTTP requests that should be sent
	// during a shared/client.(*Service).Run()
	MaxRetry int

	Debug              bool
	DictHashValidation bool

	// LatencyBreaker fields are passed through to circut.LatencyBreaker
	// at Service init time. When both LatencyBreakerLatestThreshold and
	// LatencyBreakerRollingThreshold are zero, the LatencyBreaker is not
	// constructed and the host's IsUp() reflects only the connection
	// breaker -- backward-compatible default.
	//
	// LatestThreshold is the per-attempt latency above which a single
	// observation is enough to trip into the shedding state. The caller
	// is expected to size this near (or just below) its own request
	// timeout so the breaker fires before requests would have failed.
	LatencyBreakerLatestThreshold time.Duration

	// RollingThreshold is the rolling-average latency above which the
	// breaker trips. Detects sustained slow-creep that no single
	// observation crosses LatestThreshold for.
	LatencyBreakerRollingThreshold time.Duration

	// RollingWindow is the duration over which the rolling average is
	// computed. Default 1s if zero.
	LatencyBreakerRollingWindow time.Duration

	// KConsecutive is the number of consecutive observations satisfying
	// (latest < LatestThreshold AND rolling < RollingThreshold) needed
	// to transition from ON back to OFF. Higher = more conservative
	// recovery, prevents flap on outliers. Default 3 if zero.
	LatencyBreakerKConsecutive int

	// PassThroughFraction is the probability that a request is allowed
	// through while the breaker is ON, to drive recovery sensing.
	// Default 0.01 (1%). Set higher for low-QPS models that need more
	// observations to recover. Valid range: [0, 1]. A zero value means
	// use the default.
	LatencyBreakerPassThroughFraction float64
}

func (c *Config) latencyBreakerSettings() (latencyBreakerSettings, bool, error) {
	settings := latencyBreakerSettings{
		latest:   c.LatencyBreakerLatestThreshold,
		rolling:  c.LatencyBreakerRollingThreshold,
		window:   c.LatencyBreakerRollingWindow,
		k:        c.LatencyBreakerKConsecutive,
		fraction: c.LatencyBreakerPassThroughFraction,
	}

	if settings.latest < 0 {
		return settings, false, fmt.Errorf("LatencyBreakerLatestThreshold must be >= 0, got %s", settings.latest)
	}
	if settings.rolling < 0 {
		return settings, false, fmt.Errorf("LatencyBreakerRollingThreshold must be >= 0, got %s", settings.rolling)
	}
	if settings.window < 0 {
		return settings, false, fmt.Errorf("LatencyBreakerRollingWindow must be >= 0, got %s", settings.window)
	}
	if settings.k < 0 {
		return settings, false, fmt.Errorf("LatencyBreakerKConsecutive must be >= 0, got %d", settings.k)
	}
	if settings.fraction < 0 || settings.fraction > 1 {
		return settings, false, fmt.Errorf("LatencyBreakerPassThroughFraction must be in [0, 1], got %v", settings.fraction)
	}

	enabled := settings.latest > 0 || settings.rolling > 0
	if !enabled {
		return settings, false, nil
	}

	if settings.window == 0 {
		settings.window = defaultLatencyBreakerRollingWindow
	}
	if settings.k == 0 {
		settings.k = defaultLatencyBreakerKConsecutive
	}
	if settings.fraction == 0 {
		settings.fraction = defaultLatencyBreakerPassThroughFraction
	}

	return settings, true, nil
}

// CacheSize returns cache size
func (c *Config) CacheSize() int {
	if c.CacheSizeMb == 0 {
		return 0
	}
	return 1024 * 1024 * (c.CacheSizeMb)
}

func (c *Config) updateCache() {
	if c.Datastore == nil {
		return
	}
	if c.CacheSizeMb > 0 {
		if c.Datastore != nil && c.Datastore.Cache != nil {
			c.Datastore.Cache.SizeMb = c.CacheSizeMb
		}
	}

	scope := c.CacheScope
	if scope == nil {
		return
	}
	if !scope.IsLocal() {
		c.Datastore = nil
		return
	}
	if !scope.IsL2() {
		c.Datastore.Datastore.L2 = nil
	}
	if !scope.IsL1() {
		c.Datastore.Datastore.Connection = ""
		c.Datastore.Connections = nil
	}
}
