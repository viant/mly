package client

type customMakerRegistry struct {
	registry map[string]func() interface{}
}

func (c *customMakerRegistry) Register(k string, gen func() interface{}) (old func() interface{}, ok bool) {
	old, ok = c.registry[k]
	c.registry[k] = gen

	return old, ok
}
