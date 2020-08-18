package rmap

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertKeyInMap(t *testing.T, m map[string]interface{}, key string) {
	_, ex := m[key]
	if !ex {
		t.Error(fmt.Sprintf("key: %s expected in map, but not found", key))
	}
}

func TestCollectKeys(t *testing.T) {
	input := NewFromMap(map[string]interface{}{
		"key1": "val1",
		"key2": 1.0,
		"key3": 3,
		"key4": []interface{}{"a", "b"},
		"key5": map[string]interface{}{
			"nkey1": 1,
			"nkey2": map[string]interface{}{
				"nkey3": 4,
			},
		},
	})

	keys := map[string]interface{}{}

	collectKeys(input, nil, &keys)

	assertKeyInMap(t, keys, "key1")
	assertKeyInMap(t, keys, "key2")
	assertKeyInMap(t, keys, "key3")
	assertKeyInMap(t, keys, "key4")
	assertKeyInMap(t, keys, "key5.nkey1")
	assertKeyInMap(t, keys, "key5.nkey2.nkey3")

	assert.Equal(t, 6, len(keys))
}

