package stat

import "github.com/viant/gmetric/counter"

const (
	// Not deprecated since still used in other metrics
	NoSuchKey = "noKey"

	// Deprecated due to complications
	HasValue = "hasValue"
	CacheHit = "cacheHit"

	CacheCollision = "collision"
	CacheExpired   = "expired"
	LocalHasValue  = "localHasValue"
	LocalNoSuchKey = "localNoKey"

	L1NoSuchKey = "L1NoKey"
	L1HasValue  = "L1HasValue"
	L2NoSuchKey = "L2NoKey"
	L1Copy      = "L1Copy"

	// Aerospike Errors - still complicated
	Timeout = "timeout"
	Down    = "down"
)

type cache struct{}

func (p cache) Keys() []string {
	return []string{
		ErrorKey,
		NoSuchKey,
		CacheCollision,
		CacheHit,
		CacheExpired,
		HasValue,
		L2NoSuchKey,
		L1Copy,
		Timeout,
		Down,
		LocalHasValue,
		LocalNoSuchKey,
		L1NoSuchKey,
		L1HasValue,
	}
}

func (p cache) Map(value interface{}) int {
	if value == nil {
		return -1
	}

	switch val := value.(type) {
	case error:
		return 0
	case string:
		switch val {
		case NoSuchKey:
			return 1
		case CacheCollision:
			return 2
		case CacheHit:
			return 3
		case CacheExpired:
			return 4
		case HasValue:
			return 5
		case L2NoSuchKey:
			return 6
		case L1Copy:
			return 7
		case Timeout:
			return 8
		case Down:
			return 9
		case LocalHasValue:
			return 10
		case LocalNoSuchKey:
			return 11
		case L1NoSuchKey:
			return 12
		case L1HasValue:
			return 13
		}
	}
	return -1
}

func NewCache() counter.Provider {
	return &cache{}
}
