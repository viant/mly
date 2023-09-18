package client

type customMakerRegistry struct {
	registry map[string]func() interface{}
}

func (c *customMakerRegistry) Register(k string, gen func() interface{}) (old func() interface{}, ok bool) {
	if c.registry == nil {
		c.registry = make(map[string]func() interface{})
	}

	old, ok = c.registry[k]
	c.registry[k] = gen

	return old, ok
}
