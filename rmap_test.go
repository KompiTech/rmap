package rmap

import (
	"testing"
)

func TestGetJPtrRmap(t *testing.T) {
	r := NewFromMap(map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": "lol",
		},
	})

	_, err := r.GetJPtrRmap("/level1")
	if err != nil {
		t.Error(`r.GetJPtrRmap("/level1")`)
	}
}

func TestExistsJPtr(t *testing.T) {
	r, err := NewFromBytes([]byte(`{"additionalProperties":false,"description":"Identity stores data related to current user's identity","properties":{"docType":{"type":"string"},"fingerprint":{"description":"SHA-512 fingerprint in hexstring format (no leading 0x, lowercase)","type":"string"},"is_enabled":{"description":"If this is set to false, this identity cannot access anything","type":"boolean"},"org_name":{"description":"Copied from cert.Issuer.CommonName","type":"string"},"overrides":{"description":"Overrides for this Identity","items":{"$ref":"#/definitions/override"},"type":"array"},"roles":{"description":"REF-\u003eROLE granted roles","items":{"type":"string"},"type":"array","uniqueItems":true},"users":{"description":"REF-\u003eUSER user details","items":{"type":"string"},"type":"array","uniqueItems":true},"xxx_version":{"minimum":1,"type":"integer"}},"required":["fingerprint","is_enabled","docType","xxx_version"]}`))
	if err != nil {
		t.Error("NewFromBytes()")
		return
	}

	res, err := r.ExistsJPtr("/properties/roles/items/type")
	if res != true {
		t.Error("res is false")
		return
	}
}