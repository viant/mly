package common

type (
	Iterator func(pair Pair) error
	Pair     func(key string, value interface{}) error
)

func (r Iterator) ToMap() (map[string]interface{}, error) {
	var result = make(map[string]interface{})
	err := r(func(key string, value interface{}) error {
		result[key] = value
		return nil
	})
	return result, err
}

func MapToIterator(aMap map[string]interface{}) Iterator {
	return func(pair Pair) error {
		for k, v := range aMap {
			if err := pair(k, v); err != nil {
				return err
			}
		}
		return nil
	}
}
