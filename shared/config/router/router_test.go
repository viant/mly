package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSON_EncodeDecode_WithGlobal(t *testing.T) {
	cfg := &RouterConfig{
		EntityMapping: []EntityKV{
			{EntityID: 12345, ModelName: "roas_model_12345_202511121116"},
			{EntityID: 12347, ModelName: "roas_model_12347_202511111116"},
		},
		GlobalModelName: "roas_global_202511111116",
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	expected := `{"entityMapping":[{"entityID":12345,"modelName":"roas_model_12345_202511121116"},{"entityID":12347,"modelName":"roas_model_12347_202511111116"}],"globalModelName":"roas_global_202511111116"}`
	require.Equal(t, expected, string(data))

	var decoded RouterConfig
	require.NoError(t, json.Unmarshal(data, &decoded))

	require.Equal(t, cfg.GlobalModelName, decoded.GlobalModelName)
	require.Len(t, decoded.EntityMapping, 2)
	require.Equal(t, 12345, decoded.EntityMapping[0].EntityID)
	require.Equal(t, "roas_model_12345_202511121116", decoded.EntityMapping[0].ModelName)
	require.Equal(t, 12347, decoded.EntityMapping[1].EntityID)
	require.Equal(t, "roas_model_12347_202511111116", decoded.EntityMapping[1].ModelName)
}

func TestJSON_Decode_NoGlobal(t *testing.T) {
	data := []byte(`{"entityMapping":[{"entityID":1,"modelName":"m1"}]}`)
	var cfg RouterConfig
	require.NoError(t, json.Unmarshal(data, &cfg))
	require.Empty(t, cfg.GlobalModelName)
	require.Len(t, cfg.EntityMapping, 1)
	require.Equal(t, 1, cfg.EntityMapping[0].EntityID)
	require.Equal(t, "m1", cfg.EntityMapping[0].ModelName)
}

func TestJSON_Decode_EmptyArray(t *testing.T) {
	data := []byte(`{"entityMapping":[]}`)
	var cfg RouterConfig
	require.NoError(t, json.Unmarshal(data, &cfg))
	require.NotNil(t, cfg.EntityMapping)
	require.Len(t, cfg.EntityMapping, 0)
}

func TestJSON_Decode_InvalidEntityID(t *testing.T) {
	data := []byte(`{"entityMapping":[{"entityID":"oops","modelName":"x"}]}`)
	var cfg RouterConfig
	require.Error(t, json.Unmarshal(data, &cfg))
}
