package datastore

import "fmt"

type Connection struct {
	ID string
	//Hosts coma separated list of hostnmae
	Hostnames string
	Port      int
	Timeout   *Timeout
}

func (c *Connection) Init() {
	if c.Port == 0 {
		c.Port = 3000
	}
	if c.Timeout == nil {
		c.Timeout = &Timeout{
			Total:      45,
			Socket:     40,
			Connection: 400,
		}
	}
}

func (c *Connection) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("connection.ID was empty")
	}
	if c.Hostnames == "" {
		return fmt.Errorf("connection.Hostnames was empty")
	}
	return nil
}
