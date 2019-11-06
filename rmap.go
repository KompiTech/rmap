package rmap

import (
	"encoding/json"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/pkg/errors"
	"github.com/qri-io/jsonschema"
	jsonptr "github.com/xeipuuv/gojsonpointer"
	"golang.org/x/crypto/blake2b"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

// Rmap is map[string]interface{} with additional functionality
type Rmap struct {
	Mapa map[string]interface{}
}

// which key in asset stores version
const VersionKey = "xxx_version"

// which key in asset stores primary key
const IdKey = "uuid"

// which key in asset stores document type
const DocTypeKey = "docType"

// which key stores fingerprint for identity assets
const FingerprintKey = "fingerprint"

// which keys are constituting an asset
var ServiceKeys = [...]string{VersionKey, IdKey, DocTypeKey}

// ConvertSliceToMaps converts slice of []Rmap to []interface{} containing map[string]interface{}, so it can be marshalled
func ConvertSliceToMaps(slice []Rmap) []interface{} {
	outputSlice := make([]interface{}, 0, len(slice))

	for _, elem := range slice {
		outputSlice = append(outputSlice, elem.Mapa)
	}

	return outputSlice
}

func NewFromBytes(bytes []byte) (Rmap, error) {
	mapa := map[string]interface{}{}

	if err := json.Unmarshal(bytes, &mapa); err != nil {
		return Rmap{}, errors.Wrap(err, "json.Unmarshal() failed")
	}

	return NewFromMap(mapa), nil
}

func NewFromMap(mapa map[string]interface{}) Rmap {
	return Rmap{mapa}
}

func NewFromYAMLMap(mapa map[interface{}]interface{}) Rmap {
	return NewFromMap(jsonify(mapa))
}

func NewFromInterface(value interface{}) (Rmap, error) {
	switch value.(type) {
	case Rmap:
		return value.(Rmap), nil
	case map[string]interface{}:
		return NewFromMap(value.(map[string]interface{})), nil
	case []byte:
		return NewFromBytes(value.([]byte))
	default:
		return Rmap{}, fmt.Errorf("unable to create Rmap from interface{}, type is: %T", value)
	}
}

func NewFromYAMLBytes(data []byte) (Rmap, error) {
	out := map[interface{}]interface{}{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		return Rmap{}, errors.Wrapf(err, "yaml.Unmarshal() failed")
	}

	return NewFromMap(jsonify(out)), nil
}

func NewFromYAMLFile(path string) (Rmap, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return Rmap{}, errors.Wrapf(err, "ioutil.ReadFile() failed")
	}

	return NewFromYAMLBytes(data)
}

func MustNewFromYAMLFile(path string) Rmap {
	rm, err := NewFromYAMLFile(path)
	if err != nil {
		panic(err)
	}
	return rm
}

func MustNewFromYAMLBytes(data []byte) Rmap {
	rm, err := NewFromYAMLBytes(data)
	if err != nil {
		panic(err)
	}

	return rm
}

func MustNewFromInterface(value interface{}) Rmap {
	rm, err := NewFromInterface(value)
	if err != nil {
		panic(err)
	}
	return rm
}

func NewEmpty() Rmap {
	return NewFromMap(map[string]interface{}{})
}

// NewFromSlice creates Rmap with keys from input. Values are always empty struct{}. Do NOT attempt to marshal this.
// Useful only for set operations.
func NewFromSlice(input []interface{}) (Rmap, error) {
	output := NewEmpty()

	for index, keyI := range input {
		key, ok := keyI.(string)
		if !ok {
			return Rmap{}, fmt.Errorf("input slice key with index: %d is not a STRING but: %T", index, keyI)
		}

		output.Mapa[key] = struct{}{}
	}

	return output, nil
}

func (r Rmap) IsEmpty() bool {
	return len(r.Mapa) == 0
}

func (r Rmap) Bytes() []byte {
	byt, _ := json.Marshal(r)
	return byt
}

func (r Rmap) BytesRef() *[]byte {
	byt := r.Bytes()
	return &byt
}

func (r Rmap) Copy() Rmap {
	rm, _ := NewFromBytes(r.Bytes())
	return rm
}

func (r Rmap) WrappedResultBytesRef() *[]byte {
	wrapper := NewFromMap(map[string]interface{}{
		"result": r.Mapa,
	})

	byts := wrapper.Bytes()
	return &byts
}

func (r Rmap) String() string {
	return string(r.Bytes())
}

// ValidateSchema checks if Rmap satisfies JSONSchema (bytes form) in argument
func (r Rmap) ValidateSchemaBytes(schema []byte) error {
	// load schema
	rSchema := &jsonschema.RootSchema{}
	if err := json.Unmarshal(schema, rSchema); err != nil {
		return errors.Wrapf(err, "json.Unmarshal() failed")
	}

	// if any errors are present, concat them into error
	if errs, _ := rSchema.ValidateBytes(r.Bytes()); len(errs) > 0 {
		allErrors := strings.Builder{}
		for _, err := range errs {
			allErrors.WriteString(err.Error())
		}
		return errors.New(allErrors.String())
	}

	return nil
}

func (r Rmap) DeleteJPtr(jptr string) error {
	ptr, err := jsonptr.NewJsonPointer(jptr)
	if err != nil {
		return errors.Wrapf(err, "jsonptr.NewJsonPointer() failed")
	}

	_, err = ptr.Delete(r.Mapa)
	if err != nil {
		return errors.Wrapf(err, "ptr.Delete() failed")
	}

	return nil
}

func (r Rmap) MustDeleteJPtr(jptr string) {
	if err := r.DeleteJPtr(jptr); err != nil {
		panic(err)
	}
}

// GetJPtr gets something from Rmap using JSONPointer, no type is asserted
func (r Rmap) GetJPtr(path string) (interface{}, error) {
	ptr, err := jsonptr.NewJsonPointer(path)
	if err != nil {
		return nil, errors.Wrapf(err, "jsonptr.NewJsonPointer() failed")
	}

	value, _, err := ptr.Get(r.Mapa)
	if err != nil {
		return nil, errors.Wrapf(err, "ptr.Get() failed")
	}

	return value, nil
}

func (r Rmap) MustGetJPtr(path string) interface{} {
	value, err := r.GetJPtr(path)
	if err != nil {
		panic(err)
	}

	return value
}

// SetJPtr sets something in Rmap using JSONPointer
func (r Rmap) SetJPtr(path string, value interface{}) error {
	ptr, err := jsonptr.NewJsonPointer(path)
	if err != nil {
		return errors.Wrapf(err, "jsonptr.NewJsonPointer() failed")
	}

	rm, ok := value.(Rmap)
	if ok {
		// if value is Rmap, store its backing map
		value = rm.Mapa
	}

	if _, err := ptr.Set(r.Mapa, value); err != nil {
		return errors.Wrapf(err, "ptr.Set() failed")
	}

	return nil
}

func (r Rmap) HasServiceKey() bool {
	_, hasDocType := r.Mapa[DocTypeKey]
	_, hasVersion := r.Mapa[VersionKey]
	_, hasID := r.Mapa[IdKey]
	_, hasFingerprint := r.Mapa[FingerprintKey]

	if hasDocType || hasVersion || hasID || hasFingerprint {
		return true
	}

	return false
}

func (r Rmap) MustSetJPtr(jptr string, value interface{}) {
	err := r.SetJPtr(jptr, value)
	if err != nil {
		panic(err)
	}
}

func (r Rmap) Exists(key string) bool {
	_, exists := r.Mapa[key]
	return exists
}

// ExistsJPtr checks if some key (even nested), exists
func (r Rmap) ExistsJPtr(path string) (bool, error) {
	ptr, err := jsonptr.NewJsonPointer(path)
	if err != nil {
		return false, errors.Wrapf(err, "jsonptr.NewJsonPointer() failed")
	}

	if _, _, err := ptr.Get(r.Mapa); err != nil {
		if strings.HasPrefix(err.Error(), "Object has no key") {
			return false, nil
		}
		return false, errors.Wrapf(err, "ptr.Get() failed")
	}

	return true, nil
}

func (r Rmap) MustExistsJPtr(path string) bool {
	result, err := r.ExistsJPtr(path)
	if err != nil {
		panic(err)
	}

	return result
}

func (r Rmap) GetJPtrString(path string) (string, error) {
	val, err := r.GetJPtr(path)
	if err != nil {
		return "", errors.Wrapf(err, "r.GetJPtr() failed")
	}
	valS, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("JSONPointer path: %s is not a STRING in object: %s", path, r.String())
	}
	return valS, nil
}

func (r Rmap) MustGetJPtrString(path string) string {
	result, err := r.GetJPtrString(path)
	if err != nil {
		panic(err)
	}

	return result
}

func (r Rmap) GetJPtrBool(path string) (bool, error) {
	val, err := r.GetJPtr(path)
	if err != nil {
		return false, errors.Wrapf(err, "r.GetJPtr() failed")
	}
	valB, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("JSONPointer path: %s is not a BOOLEAN in object: %s", path, r.String())
	}
	return valB, nil
}

func (r Rmap) GetJPtrInt(path string) (int, error) {
	val, err := r.GetJPtr(path)
	if err != nil {
		return -1, errors.Wrapf(err, "r.GetJPtr() failed")
	}

	// integers in JSON does not exist, it only knows float64, so those will be in something unmarshalled
	switch val.(type) {
	case float64:
		return int(val.(float64)), nil
	case int:
		return val.(int), nil
	default:
		return -1, fmt.Errorf("JSONPointer path: %s is not an INT or FLOAT64 in object: %s, but: %T", path, r.String(), val)
	}
}

func (r Rmap) MustGetJPtrInt(path string) int {
	value, err := r.GetJPtrInt(path)
	if err != nil {
		panic(err)
	}

	return value
}

// jptr must point to something iterable, usually []string
func (r Rmap) ContainsJPtr(jptr string, needle interface{}) (bool, error) {
	haystack, err := r.GetJPtrIterable(jptr)
	if err != nil {
		return false, errors.Wrapf(err, "r.GetJPtrIterable() failed")
	}

	for _, elem := range haystack {
		if elem == needle {
			return true, nil
		}
	}

	return false, nil
}

// ContainsJPtrKV expects array of objects at jptr. It is iterated a looks for jptrKey, value to be present in at least one array member
func (r Rmap) ContainsJPtrKV(jptr, jptrKey string, value interface{}) (bool, error) {
	iter, err := r.GetJPtrIterable(jptr)
	if err != nil {
		return false, errors.Wrapf(err, "r.GetJPtrIterable() failed")
	}

	for _, objI := range iter {
		obj, err := NewFromInterface(objI)
		if err != nil {
			return false, errors.Wrapf(err, "rmap.NewFromInterface() failed")
		}

		keyExists, err := obj.ExistsJPtr(jptrKey)
		if err != nil {
			return false, errors.Wrapf(err, "obj.ExistsJPtr() failed")
		}

		if keyExists {
			keyVal, err := obj.GetJPtr(jptrKey)
			if err != nil {
				return false, errors.Wrapf(err, "obj.GetJPtr() failed")
			}
			// TODO allow more types than just string
			if keyVal.(string) == value.(string) {
				return true, nil
			}
		}
	}

	return false, nil
}



func (r Rmap) GetJPtrRmap(path string) (Rmap, error) {
	val, err := r.GetJPtr(path)
	if err != nil {
		return Rmap{}, errors.Wrapf(err, "r.GetJPtr() failed")
	}
	switch val.(type) {
	case map[string]interface{}:
		return NewFromMap(val.(map[string]interface{})), nil
	case Rmap:
		return val.(Rmap), nil
	default:
		return Rmap{}, fmt.Errorf("JSONPointer path: %s is not an OBJECT in object: %s, but: %T", path, r.String(), val)
	}
}

func (r Rmap) MustGetJPtrRmap(jptr string) Rmap {
	val, err := r.GetJPtrRmap(jptr)
	if err != nil {
		panic(err)
	}
	return val
}

func (r Rmap) GetJPtrIterable(path string) ([]interface{}, error) {
	val, err := r.GetJPtr(path)
	if err != nil {
		return []interface{}{}, errors.Wrapf(err, "r.GetJPtr() failed")
	}
	valIterable, ok := val.([]interface{})
	if !ok {
		return []interface{}{}, fmt.Errorf("JSONPointer path: %s is not an ARRAY in object: %s, but: %T", path, r.String(), val)
	}
	return valIterable, nil
}

func (r Rmap) GetIterable(key string) ([]interface{}, error) {
	valI := r.Mapa[key]

	valIterable, ok := valI.([]interface{})
	if !ok {
		return []interface{}{}, fmt.Errorf("key: %s is not an ARRAY in object: %s, but: %T", key, r.String(), valI)
	}

	return valIterable, nil
}

func (r Rmap) MustGetJPtrIterable(jptr string) []interface{} {
	val, err := r.GetJPtrIterable(jptr)
	if err != nil {
		panic(err)
	}
	return val
}

func (r Rmap) Hash() [32]byte {
	return blake2b.Sum256(r.Bytes())
}

func (r Rmap) GetVersion() (int, error) {
	return r.GetJPtrInt("/" + VersionKey)
}

func (r Rmap) MustGetVersion() int {
	val, err := r.GetVersion()
	if err != nil {
		panic(err)
	}
	return val
}

func (r Rmap) SetVersion(version int) {
	r.Mapa[VersionKey] = version
}

func (r Rmap) GetDocType() (string, error) {
	val, err := r.GetJPtrString("/" + DocTypeKey)
	if err != nil {
		return "", errors.Wrapf(err, "r.GetJPtrString() failed")
	}

	return strings.ToLower(val), nil
}

func (r Rmap) GetIDKey() (string, error) {
	docType, err := r.GetDocType()
	if err != nil {
		return "", errors.Wrapf(err, "GetDocType() failed")
	}

	if strings.ToLower(docType) == "identity" {
		return "fingerprint", nil
	}
	return IdKey, nil
}

func (r Rmap) GetID() (string, error) {
	idKey, err := r.GetIDKey()
	if err != nil {
		return "", errors.Wrapf(err, "r.GetIDKey() failed")
	}

	value, err := r.GetJPtrString("/" + idKey)
	if err != nil {
		return "", errors.Wrapf(err, "r.GetJPtrString() failed")
	}

	return strings.ToLower(value), nil
}

func (r Rmap) MustGetID() string {
	val, err := r.GetID()
	if err != nil {
		panic(err)
	}

	return val
}

func (r Rmap) GetCasbinObject() (string, error) {
	docType, err := r.GetDocType()
	if err != nil {
		return "", errors.Wrapf(err, "r.GetDocType() failed")
	}

	name, err := r.GetID()
	if err != nil {
		return "", errors.Wrapf(err, "r.GetID() failed")
	}

	return strings.ToLower(fmt.Sprintf("/%s/%s", docType, name)), nil
}

func (r Rmap) IsAsset() (bool, error) {
	if r.Exists(DocTypeKey) {
		docType, err := r.GetDocType()
		if err != nil {
			return false, errors.Wrapf(err, "r.GetDocType() failed")
		}

		var keysToCheck []string
		if docType != "identity" {
			// anything else than identity uses standard service keys
			keysToCheck = append(keysToCheck, ServiceKeys[:]...)
		} else {
			// identity is also asset, but doesnt have uuid, has fingerprint
			keysToCheck = []string{"fingerprint", VersionKey, DocTypeKey}
		}

		for _, key := range keysToCheck {
			exists, err := r.ExistsJPtr("/" + key)
			if err != nil {
				return false, errors.Wrapf(err, "r.ExistsJPtr() failed")
			}
			if !exists {
				return false, nil
			}
		}
		return true, nil
	}
	// no docType - cannot be an asset
	return false, nil
}

// Inject puts keys from value into this Rmap in path
// Creates target path if it doesnt exist (but only one level)
// Silently overwrites existing values
func (r Rmap) Inject(path string, value Rmap) error {
	targetExists, err := r.ExistsJPtr(path)
	if err != nil {
		return errors.Wrapf(err, "r.ExistsJPtr() failed")
	}

	if !targetExists {
		// target key doesn't exist, initialize with empty Rmap
		if err := r.SetJPtr(path, NewEmpty()) ; err != nil {
			return errors.Wrapf(err, "r.SetJPtr() failed")
		}
	}

	for k,v := range value.Mapa {
		keyPath := path + "/" + k
		if err := r.SetJPtr(keyPath, v); err != nil {
			return errors.Wrapf(err, "r.SetJPtr() failed")
		}
	}
	return nil
}

func (r *Rmap) ApplyMergePatchBytes(patch []byte) error {
	patchedBytes, err := jsonpatch.MergePatch(r.Bytes(), patch)
	if err != nil {
		return errors.Wrapf(err, "jsonpatch.MergePatch() failed")
	}

	// r.Mapa must be replaced in-place
	patched, err := NewFromBytes(patchedBytes)
	if err != nil {
		return errors.Wrapf(err, "rmap.NewFromBytes() failed")
	}

	r.Mapa = patched.Mapa
	return nil
}

func (r *Rmap) ApplyMergePatch(patch Rmap) error {
	if err := r.ApplyMergePatchBytes(patch.Bytes()); err != nil {
		return errors.Wrapf(err, "r.ApplyMergePatchBytes() failed")
	}

	return nil
}

func (r Rmap) CreateMergePatch(changed Rmap) ([]byte, error) {
	return jsonpatch.CreateMergePatch(r.Bytes(), changed.Bytes())
}

func (r Rmap) GetTXSKey() (string, error) {
	docType, err := r.GetDocType()
	if err != nil {
		return "", errors.Wrapf(err, "r.GetDocType() failed")
	}

	id, err := r.GetID()
	if err != nil {
		return "", errors.Wrapf(err, "r.GetID() failed")
	}

	return strings.ToLower(fmt.Sprintf("%s:%s", docType, id)), nil
}

// KeysSlice returns r.Mapa keys as slice
func (r Rmap) KeysSlice() []interface{} {
	output := make([]interface{}, 0, len(r.Mapa))
	for key, _ := range r.Mapa {
		output = append(output, key)
	}
	return output
}

// MarshalJSON implements Marshaller interface to produce correct JSON without Mapa encapsulation
func (r Rmap) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Mapa)
}

func (r Rmap) YAMLBytes() ([]byte, error) {
	return yaml.Marshal(r.Mapa)
}

func (r Rmap) MustYAMLBytes() []byte {
	byt, err := r.YAMLBytes()
	if err != nil {
		panic(err)
	}

	return byt
}

// Jsonify converts map[interface{}]interface{} (YAML) to map[string]interface{} (JSON)
func jsonify(m map[interface{}]interface{}) map[string]interface{} {
	res := map[string]interface{}{}
	for k, v := range m {
		switch v2 := v.(type) {
		case map[interface{}]interface{}:
			res[fmt.Sprint(k)] = jsonify(v2)
		case []interface{}:
			array := make([]interface{}, len(v2))
			for idx, v3 := range v2 {
				if m, ok := v3.(map[interface{}]interface{}); !ok {
					//default
					array[idx] = v3
				} else {
					array[idx] = jsonify(m)
				}
			}
			res[fmt.Sprint(k)] = array
		default:
			res[fmt.Sprint(k)] = v
		}
	}
	return res
}
