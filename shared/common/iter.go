package common

type (
	//Iterator represents iterator
	Iterator func(pair Pair) error
	//Pair represents a pair
	Pair func(key string, value interface{}) error
)

//ToMap coverts iterator to map
func (r Iterator) ToMap() (map[string]interface{}, error) {
	var result = make(map[string]interface{})
	err := r(func(key string, value interface{}) error {
		result[key] = value
		return nil
	})
	return result, err
}

//MapToIterator create an iterator for supplied map
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
