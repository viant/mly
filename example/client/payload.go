package client

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"github.com/viant/mly/shared/client"
	"github.com/viant/mly/shared/common"
)

type CliPayload struct {
	Data  map[string]interface{}
	Batch int
}

func (c *CliPayload) Iterator(pair common.Pair) error {
	for field, values := range c.Data {
		err := pair(field, values)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *CliPayload) Pair(key string, value interface{}) error {
	return nil

}

func (c *CliPayload) SetBatch(msg *client.Message) {
	if c.Batch > 0 {
		msg.SetBatchSize(c.Batch)
	}
}

func (c *CliPayload) Bind(k string, value interface{}, msg *client.Message) error {
	if c.Batch > 0 {
		switch v := value.(type) {
		case []int:
			msg.IntsKey(k, v)
		case []float32:
			msg.FloatsKey(k, v)
		case []string:
			msg.StringsKey(k, v)
		}
	} else {
		switch v := value.(type) {
		case []int:
			msg.IntKey(k, v[0])
		case []float32:
			msg.FloatKey(k, v[0])
		case []string:
			msg.StringKey(k, v[0])
		}
	}

	return nil
}

func Parse(p string, cp *CliPayload) error {
	data := make(map[string]interface{})
	c := None

	chunks := strings.Split(p, ";")
	for _, chunk := range chunks {
		def := strings.Split(chunk, ":")
		if len(def) != 2 {
			return fmt.Errorf("chunk \"%s\" missing or has more than one \":\"", chunk)
		}

		field := def[0]
		vals, err := csv.NewReader(strings.NewReader(def[1])).Read()
		if err != nil {
			return fmt.Errorf("csv error for field %s: %v", field, err)
		}

		valLen := len(vals)
		if valLen > 1 {
			if c == Single {
				return fmt.Errorf("inconsistent batching at field %s", field)
			}

			c = Batch
			if valLen > cp.Batch {
				cp.Batch = valLen
			}
		} else {
			if c == Batch {
				return fmt.Errorf("inconsistent batching at field %s", field)
			}
			c = Single
		}

		typed := strings.Split(field, "|")
		if len(typed) == 2 {
			field = typed[0]
			switch typed[1] {
			case "int":
				vs := make([]int, valLen)
				for i, v := range vals {
					cv, err := strconv.Atoi(v)
					if err != nil {
						return err
					}

					vs[i] = cv
				}
				data[field] = vs
			case "float32":
				vs := make([]float32, valLen)
				for i, v := range vals {
					cv, err := strconv.ParseFloat(v, 32)
					if err != nil {
						return err
					}

					vs[i] = float32(cv)
				}
				data[field] = vs
			default:
				return fmt.Errorf("unknown type:%s", typed[1])
			}

		} else {
			data[field] = vals
		}
	}

	cp.Data = data
	return nil
}
