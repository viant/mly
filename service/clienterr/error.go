package clienterr

// Used to bubble lower-level errors up to HTTP handler
type ClientError struct {
	e      string
	Parent error
}

func (c *ClientError) Error() string {
	return c.e
}

func Wrap(e error) *ClientError {
	c := new(ClientError)
	c.e = e.Error()
	c.Parent = e
	return c
}
