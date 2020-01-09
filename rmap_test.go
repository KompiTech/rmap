package rmap

import (
	"bytes"
	"encoding/json"
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
	assert.Equal(t, expectedErr, err.Error())

	data = []byte(`{"extraData": "bar"}`)
	rm, err = NewFromBytes(data)
	assert.Nil(t, err)

	err = rm.ValidateSchemaBytes(schema)
	expectedErr = `InvalidValue: map[extraData:bar], PropertyPath: /, RulePath: , Message: "firstName" value is required` + "\n" +
	`InvalidValue: map[extraData:bar], PropertyPath: /, RulePath: , Message: "lastName" value is required` + "\n" +
	`InvalidValue: bar, PropertyPath: /extraData, RulePath: , Message: cannot match schema`
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