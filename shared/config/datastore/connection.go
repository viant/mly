package datastore

import "fmt"

//Connection represents a connection
type Connection struct {
	ID string
	//Hosts coma separated list of hostnmae
	Hostnames string
	Port      int
	Timeout   *Timeout
}

//Init initialises connection
func (c *Connection) Init() {
	if c.Port == 0 {
		c.Port = 3000
	}
	if c.Timeout == nil {
		c.Timeout = &Timeout{
			Total:      450,
			Socket:     400,
			Connection: 800,
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
