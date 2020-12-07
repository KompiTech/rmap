package rmap

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetJPtrRmap(t *testing.T) {
	r := NewFromMap(map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": "lol",
		},
	})

	_, err := r.GetJPtrRmap("/level1")
	assert.Nil(t, err)
}

func TestExistsJPtr(t *testing.T) {
	r, err := NewFromBytes([]byte(`{"additionalProperties":false,"description":"Identity stores data related to current user's identity","properties":{"docType":{"type":"string"},"fingerprint":{"description":"SHA-512 fingerprint in hexstring format (no leading 0x, lowercase)","type":"string"},"is_enabled":{"description":"If this is set to false, this identity cannot access anything","type":"boolean"},"org_name":{"description":"Copied from cert.Issuer.CommonName","type":"string"},"overrides":{"description":"Overrides for this Identity","items":{"$ref":"#/definitions/override"},"type":"array"},"roles":{"description":"REF-\u003eROLE granted roles","items":{"type":"string"},"type":"array","uniqueItems":true},"users":{"description":"REF-\u003eUSER user details","items":{"type":"string"},"type":"array","uniqueItems":true},"xxx_version":{"minimum":1,"type":"integer"}},"required":["fingerprint","is_enabled","docType","xxx_version"]}`))
	assert.Nil(t, err)

	res, err := r.ExistsJPtr("/properties/roles/items/type")
	assert.True(t, res)
}

func TestCreateMergePatch(t *testing.T) {
	original := NewFromMap(map[string]interface{}{"existing": "value"})
	changed := NewFromMap(map[string]interface{}{"new": "value"})
	expectedPatch := []byte(`{"existing":null,"new":"value"}`)

	patch, err := original.CreateMergePatch(changed)
	assert.Nil(t, err)

	assert.Zero(t, bytes.Compare(patch, expectedPatch))
}

func TestNestedRmapMarshal(t *testing.T) {
	refMap := map[string]interface{}{
		"nested": map[string]interface{}{
			"value": "foobar",
		},
	}

	testRmap := Rmap{
		Mapa: map[string]interface{}{
			"nested": Rmap{
				Mapa: map[string]interface{}{
					"value": "foobar",
				},
			},
		},
	}

	jsonMap := map[string]interface{}{}

	err := json.Unmarshal(testRmap.Bytes(), &jsonMap)
	assert.Nil(t, err)

	assert.Equal(t, refMap, jsonMap)
}

func TestYAMLBytes(t *testing.T) {
	refMap := map[string]interface{}{
		"nested": map[string]interface{}{
			"value": "foobar",
		},
	}

	refYAML := []byte(`nested:
  value: foobar
`)

	testRmap := NewFromMap(refMap)

	yamlBytes, err := testRmap.YAMLBytes()
	assert.Nil(t, err)

	assert.Equal(t, refYAML, yamlBytes)
}

func TestGetJPtrTime(t *testing.T) {
	now := time.Now().UTC()
	now = now.Truncate(time.Second) // RFC3339 has second precision

	rm := NewEmpty()
	rm.Mapa["time"] = now.Format(time.RFC3339)

	parsed, err := rm.GetJPtrTime("/time")
	assert.Nil(t, err)

	assert.Equal(t, now, parsed)
}

func TestGetJPtrFloat64(t *testing.T) {
	value := 1.337
	rm := NewEmpty()
	rm.Mapa["float"] = value

	parsed, err := rm.GetJPtrFloat64("/float")
	assert.Nil(t, err)

	assert.Equal(t, value, parsed)
}

func TestVerboseErrors(t *testing.T) {
	schema := []byte(`{
      "title": "Person",
      "type": "object",
      "properties": {
          "firstName": {
              "type": "string"
          },
          "lastName": {
              "type": "string"
          }
      },
      "required": ["firstName", "lastName"],
      "additionalProperties": false
    }`)

	data := []byte(`{}`)
	rm, err := NewFromBytes(data)
	assert.Nil(t, err)

	err = rm.ValidateSchemaBytes(schema)
	expectedErr := "InvalidValue: map[], PropertyPath: /, RulePath: , Message: \"firstName\" value is required" + "\n" +
		"InvalidValue: map[], PropertyPath: /, RulePath: , Message: \"lastName\" value is required"
	assert.NotNil(t, err)
	assert.Equal(t, expectedErr, err.Error())

	data = []byte(`{"extraData": "bar"}`)
	rm, err = NewFromBytes(data)
	assert.Nil(t, err)

	err = rm.ValidateSchemaBytes(schema)
	expectedErr = `InvalidValue: bar, PropertyPath: /extraData, RulePath: , Message: cannot match schema` + "\n" + `InvalidValue: map[extraData:bar], PropertyPath: /, RulePath: , Message: "firstName" value is required` + "\n" +
		`InvalidValue: map[extraData:bar], PropertyPath: /, RulePath: , Message: "lastName" value is required`

	assert.NotNil(t, err)
	assert.Equal(t, expectedErr, err.Error())
}

func TestNewFromStringSlice(t *testing.T) {
	keys := []string{"a", "b", "c"}
	rmap := NewFromStringSlice(keys)

	for _, key := range keys {
		val, exists := rmap.Mapa[key]
		assert.Equal(t, struct{}{}, val)
		assert.True(t, exists)
	}
}

func TestGetIterable(t *testing.T) {
	rm := NewFromMap(map[string]interface{}{
		"key": []interface{}{
			map[string]interface{}{"a": "a"},
			map[string]interface{}{"b": "b"},
		},
	})

	iter, err := rm.GetIterable("key")
	assert.Nil(t, err)

	assert.Equal(t, 2, len(iter))
}

func TestGetIterable2(t *testing.T) {
	rm := NewFromMap(map[string]interface{}{
		"key": []map[string]interface{}{
			{"a": "a"},
			{"b": "b"},
		},
	})

	iter, err := rm.GetIterable("key")
	assert.Nil(t, err)

	assert.Equal(t, 2, len(iter))
}

func TestGetIterable3(t *testing.T) {
	rm := NewFromMap(map[string]interface{}{
		"key": []Rmap{
			NewFromMap(map[string]interface{}{"a": "a"}),
			NewFromMap(map[string]interface{}{"b": "b"}),
		},
	})

	iter, err := rm.GetIterable("key")
	assert.Nil(t, err)

	assert.Equal(t, 2, len(iter))
}

func TestRmap_IsValidJSONSchema(t *testing.T) {
	rm, err := NewFromYAMLBytes([]byte(`{"additionalProperties":false,"type":"object"}`))
	assert.Nil(t, err)
	assert.True(t, rm.IsValidJSONSchema())

	rm2, err2 := NewFromYAMLBytes([]byte(`{"hello":false,"world":"object"}`))
	assert.Nil(t, err2)
	assert.False(t, rm2.IsValidJSONSchema())
}

func TestConvertSliceToMaps(t *testing.T) {
	map0 := map[string]interface{}{
		"hello1": "world1",
	}

	map1 := map[string]interface{}{
		"hello2": "world2",
	}

	rmaps := []Rmap{
		NewFromMap(map0),
		NewFromMap(map1),
	}

	maps := ConvertSliceToMaps(rmaps)

	assert.Len(t, maps, 2)
	assert.Equal(t, maps[0], map0)
	assert.Equal(t, maps[1], map1)
}

func TestNewFromYAMLMap(t *testing.T) {
	ymap := map[interface{}]interface{}{
		"hello": "world",
	}

	rm := NewFromYAMLMap(ymap)

	assert.Equal(t, rm.Mapa, map[string]interface{}{
		"hello": "world",
	})
}

func TestNewFromYAMLFile(t *testing.T) {
	rm, err := NewFromYAMLFile("testdata/test.yaml")
	assert.Nil(t, err)
	assert.Equal(t, rm.Mapa, map[string]interface{}{
		"hello": "world",
	})
}

func TestMustNewFromYAMLFile(t *testing.T) {
	rm := MustNewFromYAMLFile("testdata/test.yaml")
	assert.Equal(t, rm.Mapa, map[string]interface{}{
		"hello": "world",
	})
}

func TestMustNewFromYAMLBytes(t *testing.T) {
	byts := []byte(`hello: world`)
	rm, err := NewFromYAMLBytes(byts)
	assert.Nil(t, err)
	assert.Equal(t, rm.Mapa, map[string]interface{}{
		"hello": "world",
	})

	rm = MustNewFromYAMLBytes(byts)
	assert.Equal(t, rm.Mapa, map[string]interface{}{
		"hello": "world",
	})
}

func TestNewFromInterfaceMap(t *testing.T) {
	var iface interface{}
	mapa := map[string]interface{}{"hello": "world"}

	iface = mapa

	rm, err := NewFromInterface(iface)
	assert.Nil(t, err)
	assert.Equal(t, rm.Mapa, mapa)

	rm = MustNewFromInterface(iface)
	assert.Equal(t, rm.Mapa, mapa)
}

func TestNewFromInterfaceBytes(t *testing.T) {
	var iface interface{}
	byts := []byte(`{"hello": "world"}`)
	mapa := map[string]interface{}{"hello": "world"}

	iface = byts
	rm, err := NewFromInterface(iface)
	assert.Nil(t, err)
	assert.Equal(t, rm.Mapa, mapa)

	rm = MustNewFromInterface(iface)
	assert.Equal(t, rm.Mapa, mapa)
}

func TestNewFromSlice(t *testing.T) {
	slice := []interface{}{"hello", "world"}

	rm, err := NewFromSlice(slice)
	assert.Nil(t, err)

	assert.Len(t, rm.Mapa, 2)
	assert.True(t, rm.Exists("hello"))
	assert.True(t, rm.Exists("world"))
}

func TestGetIterableJPtr(t *testing.T) {
	obj := NewFromMap(map[string]interface{}{
		"submap1": map[string]interface{}{
			"submap2": map[string]interface{}{
				"iter": []interface{}{"1", "2", "3"},
			},
		},
	})

	iter, err := obj.GetIterableJPtr("/submap1/submap2/iter")
	assert.Nil(t, err)

	assert.Equal(t, 3, len(iter))
}

func TestGetIterableRmap(t *testing.T) {
	obj := NewFromMap(map[string]interface{}{
		"submaps": []map[string]interface{}{
			{"subvalue": "hello"},
			{"subvalue": "world"},
		},
	})

	submaps, err := obj.GetIterableRmap("submaps")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(submaps))

	assert.Equal(t, "hello", submaps[0].MustGetString("subvalue"))
	assert.Equal(t, "world", submaps[1].MustGetString("subvalue"))
}

func TestGetIterableRmapJPtr(t *testing.T) {
	obj := NewFromMap(map[string]interface{}{
		"submaps": []map[string]interface{}{
			{"subvalue": "hello"},
			{"subvalue": "world"},
		},
	})

	submaps, err := obj.GetIterableRmapJPtr("/submaps")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(submaps))

	assert.Equal(t, "hello", submaps[0].MustGetString("subvalue"))
	assert.Equal(t, "world", submaps[1].MustGetString("subvalue"))
}

func TestGetIterableString(t *testing.T) {
	obj := NewFromMap(map[string]interface{}{
		"iter": []interface{}{"hello", "world"},
	})

	iterS, err := obj.GetIterableString("iter")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(iterS))
	assert.Equal(t, "hello", iterS[0])
	assert.Equal(t, "world", iterS[1])
}

func TestGetIterableStringJPtr(t *testing.T) {
	obj := NewFromMap(map[string]interface{}{
		"iter": []interface{}{"hello", "world"},
	})

	iterS, err := obj.GetIterableStringJPtr("/iter")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(iterS))
	assert.Equal(t, "hello", iterS[0])
	assert.Equal(t, "world", iterS[1])
}

func TestSetJPtrRecursiveCreate(t *testing.T) {
	obj := NewEmpty()
	jptr := "/very/deep/obj"
	value := "world"

	err := obj.SetJPtrRecursive(jptr, value)
	assert.Nil(t, err)
	assert.Equal(t, value, obj.MustGetJPtrString(jptr))
}