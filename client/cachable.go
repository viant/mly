package client

//Cachable returns a cacheable key
type Cachable interface {
	CacheKey() string
}
