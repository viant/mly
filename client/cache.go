package client

type CacheScope int

func (c CacheScope) IsLocal() bool {
	return c&CacheScopeLocal != 0
}

func (c CacheScope) IsL1() bool {
	return c&CacheScopeL1 != 0
}

func (c CacheScope) IsL2() bool {
	return c&CacheScopeL2 != 0
}


const (
	CacheScopeLocal = CacheScope(1)
	CacheScopeL1    = CacheScope(2)
	CacheScopeL2    = CacheScope(4)
)
