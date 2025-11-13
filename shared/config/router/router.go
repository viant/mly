package config

type RouterConfig struct {
	EntityMapping []EntityKV `json:"entityMapping" yaml:"entityMapping"`

	GlobalModelName string `json:"globalModelName" yaml:"globalModelName"`
}

type EntityKV struct {
	EntityID  int    `json:"entityID" yaml:"entityID"`
	ModelName string `json:"modelName" yaml:"modelName"`
}
