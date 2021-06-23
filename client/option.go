package client

type Option interface {
	Apply(c *Client)
}

type cacheSizeOpt struct {
	sizeMB int
}

func (o *cacheSizeOpt) Apply(c *Client) {
	c.Config.CacheSize = o.sizeMB
}

//NewCacheSize returns cache size MB
func NewCacheSize(sizeMB int) Option {
	return &cacheSizeOpt{sizeMB: sizeMB}
}