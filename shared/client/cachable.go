package client

//Cachable returns a cacheable key
type Cachable interface {
	CacheKey() string
	CacheKeyAt(index int) string
	BatchSize() int
	FlagCacheHit(index int)
	CacheHit(index int) bool
}
