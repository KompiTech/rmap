package rmap

import (
	"encoding/json"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/pkg/errors"
	"github.com/qri-io/jsonschema"
	"github.com/shopspring/decimal"
	jsonptr "github.com/xeipuuv/gojsonpointer"
	"golang.org/x/crypto/blake2b"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
	"time"
)

// Rmap is map[string]interface{} with additional functionality
type Rmap struct {
	Mapa map[string]interface{}
}

const (
	VersionKey        = "xxx_version" // which key in asset stores version
	IdKey             = "uuid" // which key in asset stores primary key
	DocTypeKey        = "docType" // which key in asset stores document type
	FingerprintKey    = "fingerprint" // which key stores fingerprint for identity assets
	errInvalidKeyType = "key: %s is not of type: %s in object: %s, but: %T"
	errInvalidJPtrType = "JSONPointer: %s is not of type: %s in object: %s, but: %T"
)

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

func NewFromStringSlice(slice []string) Rmap {
	output := NewEmpty()
	for _, key := range slice {
		output.Mapa[key] = struct{}{}
	}
	return output
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
		errorStrings := make([]string, 0, len(errs))

		for _, err := range errs {
			errorStrings = append(errorStrings, fmt.Sprintf("InvalidValue: %+v, PropertyPath: %s, RulePath: %s, Message: %s", err.InvalidValue, err.PropertyPath, err.RulePath, err.Message))
		}
		return errors.New(strings.Join(errorStrings, "\n"))
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

func (r Rmap) ExistsMany(keys []string) bool {
	for _, key := range keys {
		if exists := r.Exists(key); !exists {
			return false
		}
	}
	return true
}

// ExistsJPtr checks if some key (even nested), exists
func (r Rmap) ExistsJPtr(path string) (bool, error) {
	ptr, err := jsonptr.NewJsonPointer(path)
	if err != nil {
		return false, errors.Wrapf(err, "jsonptr.NewJsonPointer() failed")
	}

	if _, _, err := ptr.Get(r.Mapa); err != nil {
		// TODO this is not ideal, do not check error message, but no other API is available
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

func (r Rmap) MustGetJPtrBool(jptr string) bool {
	val, err := r.GetJPtrBool(jptr)
	if err != nil {
		panic(err)
	}

	return val
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

// ContainsJPtr gets some JPtr path (it must be iterable) and checks if needle is contained
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

// Contains gets some key (it must be iterable) and checks if needle is contained
// key must point to something iterable, usually []string
func (r Rmap) Contains(key string, needle interface{}) (bool, error) {
	haystack, err := r.GetIterable(key)
	if err != nil {
		return false, errors.Wrapf(err, "r.GetIterable() failed")
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
	valI, err := r.GetJPtr(path)
	if err != nil {
		return Rmap{}, errors.Wrapf(err, "r.GetJPtr() failed")
	}
	switch valI.(type) {
	case map[string]interface{}:
		return NewFromMap(valI.(map[string]interface{})), nil
	case Rmap:
		return valI.(Rmap), nil
	default:
		return Rmap{}, fmt.Errorf(errInvalidJPtrType, path, "OBJECT", r.String(), valI)
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
	valI, err := r.GetJPtr(path)
	if err != nil {
		return []interface{}{}, errors.Wrapf(err, "r.GetJPtr() failed")
	}

	valIterable, ok := valI.([]interface{})
	if !ok {
		return []interface{}{}, fmt.Errorf(errInvalidJPtrType, path, "ARRAY", r.String(), valI)
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

func (r Rmap) GetJPtrTime(jptr string) (time.Time, error) {
	val, err := r.GetJPtrString(jptr)
	if err != nil {
		return time.Time{}, err
	}

	parsed, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return time.Time{}, err
	}

	return parsed, nil
}

func (r Rmap) MustGetJPtrTime(jptr string) time.Time {
	val, err := r.GetJPtrTime(jptr)
	if err != nil {
		panic(err)
	}

	return val
}

func (r Rmap) GetJPtrFloat64(jptr string) (float64, error) {
	valI, err := r.GetJPtr(jptr)
	if err != nil {
		return -1.0, errors.Wrapf(err, "r.GetJPtr() failed")
	}

	switch valI.(type) {
	case float64:
		return valI.(float64), nil
	default:
		return -1.0, fmt.Errorf(errInvalidJPtrType, jptr, "FLOAT64", r.String(), valI)
	}
}

func (r Rmap) MustGetJPtrFloat64(jptr string) float64 {
	val, err := r.GetJPtrFloat64(jptr)
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

// KeysSliceString returns r.Mapa keys as slice
func (r Rmap) KeysSliceString() []string {
	output := make([]string, 0, len(r.Mapa))
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

func (r Rmap) get(key string) (interface{}, error) {
	if val, exists := r.Mapa[key]; exists {
		return val, nil
	}
	return nil, fmt.Errorf("key: %s does not exist in object: %s", key, r.String())
}

func (r Rmap) GetBool(key string) (bool, error) {
	valI, err := r.get(key)
	if err != nil {
		return false, errors.Wrap(err, "r.get() failed")
	}

	valB, ok := valI.(bool)
	if !ok {
		return false, fmt.Errorf(errInvalidKeyType, key, "BOOLEAN", r.String(), valI)
	}
	return valB, nil
}

func (r Rmap) MustGetBool(key string) bool {
	val, err := r.GetBool(key)
	if err != nil {
		panic(err)
	}
	return val
}

func (r Rmap) GetFloat64(key string) (float64, error) {
	valI, err := r.get(key)
	if err != nil {
		return -1.0, errors.Wrap(err, "r.get() failed")
	}

	valF, ok := valI.(float64)
	if !ok {
		return -1.0, fmt.Errorf(errInvalidKeyType, key, "FLOAT64", r.String(), valI)
	}
	return valF, nil
}

func (r Rmap) MustGetFloat64(key string) float64 {
	val, err := r.GetFloat64(key)
	if err != nil {
		panic(err)
	}
	return val
}

func (r Rmap) GetInt(key string) (int, error) {
	valI, err := r.get(key)
	if err != nil {
		return -1, errors.Wrap(err, "r.get() failed")
	}

	switch valI.(type) {
	case float64:
		return int(valI.(float64)), nil
	case int:
		return valI.(int), nil
	default:
		return -1, fmt.Errorf(errInvalidKeyType, key, "INT or FLOAT64", r.String(), valI)
	}
}

func (r Rmap) MustGetInt(key string) int {
	val, err := r.GetInt(key)
	if err != nil {
		panic(err)
	}
	return val
}

func (r Rmap) GetIterable(key string) ([]interface{}, error) {
	valI, err := r.get(key)
	if err != nil {
		return nil, errors.Wrap(err, "r.get() failed")
	}

	var valIter []interface{}
	var ok bool

	valIter, ok = valI.([]interface{})
	if !ok {
		tmpI, ok := valI.([]map[string]interface{})
		if !ok {
			tmpI2, ok := valI.([]Rmap)
			if !ok {
				return nil, fmt.Errorf(errInvalidKeyType, key, "ARRAY", r.String(), valI)
			} else {
				for _, xI := range tmpI2 {
					valIter = append(valIter, xI)
				}
			}
		} else {
			for _, xI := range tmpI {
				valIter = append(valIter, xI)
			}
		}
	}


	return valIter, nil
}

func (r Rmap) MustGetIterable(key string) []interface{} {
	val, err := r.GetIterable(key)
	if err != nil {
		panic(err)
	}
	return val
}

func (r Rmap) GetRmap(key string) (Rmap, error) {
	valI, err := r.get(key)
	if err != nil {
		return Rmap{}, errors.Wrap(err, "r.get() failed")
	}

	switch valI.(type) {
	case map[string]interface{}:
		return NewFromMap(valI.(map[string]interface{})), nil
	case Rmap:
		return valI.(Rmap), nil
	default:
		return Rmap{}, fmt.Errorf(errInvalidKeyType, key, "OBJECT", r.String(), valI)
	}
}

func (r Rmap) MustGetRmap(key string) Rmap {
	val, err := r.GetRmap(key)
	if err != nil {
		panic(err)
	}
	return val
}

func (r Rmap) GetString(key string) (string, error) {
	valI, err := r.get(key)
	if err != nil {
		return "", errors.Wrap(err, "r.get() failed")
	}

	valS, ok := valI.(string)
	if !ok {
		return "", fmt.Errorf(errInvalidKeyType, key, "STRING", r.String(), valI)
	}
	return valS, nil
}

func (r Rmap) MustGetString(key string) string {
	val, err := r.GetString(key)
	if err != nil {
		panic(err)
	}
	return val
}

func (r Rmap) GetTime(key string) (time.Time, error) {
	valS, err := r.GetString(key)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "r.GetString() failed")
	}

	parsed, err := time.Parse(time.RFC3339, valS)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "time.Parse() failed")
	}

	return parsed, nil
}

func (r Rmap) MustGetTime(key string) time.Time {
	val, err := r.GetTime(key)
	if err != nil {
		panic(err)
	}
	return val
}

func (r Rmap) GetDecimal(key string) (decimal.Decimal, error) {
	valS, err := r.GetString(key)
	if err != nil {
		return decimal.Zero, errors.Wrap(err, "r.GetString() failed")
	}

	val, err := decimal.NewFromString(valS)
	if err != nil {
		return decimal.Zero, errors.Wrap(err, "decimal.NewFromString() failed")
	}

	return val, nil
}

func (r Rmap) MustGetDecimal(key string) decimal.Decimal {
	val, err := r.GetDecimal(key)
	if err != nil {
		panic(err)
	}

	return val
}

func (r Rmap) GetJPtrDecimal(jptr string) (decimal.Decimal, error) {
	valS, err := r.GetJPtrString(jptr)
	if err != nil {
		return decimal.Zero, errors.Wrap(err, "r.GetJPtrString() failed")
	}

	val, err := decimal.NewFromString(valS)
	if err != nil {
		return decimal.Zero, errors.Wrap(err, "decimal.NewFromString() failed")
	}

	return val, nil
}

func (r Rmap) MustGetJPtrDecimal(jptr string) decimal.Decimal {
	val, err := r.GetJPtrDecimal(jptr)
	if err != nil {
		panic(err)
	}

	return val
}