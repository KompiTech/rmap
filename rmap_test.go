package rmap

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
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