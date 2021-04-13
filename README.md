Rmap (recursive map) is wrapper around Go type ```map[string]interface{}``` which usually represents JSON data. Compared to pure Go, error handling code is greatly reduced.

All relevant methods also have Must variant which doesn't return error and immediately panics.

# Getters

## Get{Type}

Gets key with given type. Error is returned if key does not exist, or the type is invalid.

Example:
```
m := map[string]interface{}{
  "stringValue": "hello world",
  "intValue": 42,
}

r := rmap.NewFromMap(m)

val, err := r.GetString("stringValue")
// val is "hello world"
// err is nil

val2, err := r.GetString("intValue")
// val2 is ""
// err string is: intValue is not of type: STRING in object: {"intValue":42,"stringValue":"hello world"}, but: int
```

## GetJPtr{Type}

Get key with given type by using JSONPointer selector. This is very useful when working with nested objects. Error is returned if key does not exist, type is invalid or JSONPointer is not correct.

Example:
```
m := map[string]interface{}{
  "nestedArray": []interface{}{
    map[string]interface{}{
      "nestedObj": map[string]interface{}{
        "stringValue": "hello world",
      },
    },
  },
}

r := NewFromMap(m)

val, err := r.GetJPtrString("/nestedArray/0/nestedObj/stringValue")
// val is "hello world"
// err is nil

val, err := r.GetJPtrString("/nestedArray/1/nestedObj/stringValue")
// val is ""
// err string is : r.GetJPtr() failed: ptr.Get() failed: Out of bound array[0,1] index '1'
```

## Get(JPtr)Iterable

Gets key that must be iterable. Error is returned if key does not exist or value is not iterable.

Example:
```
m := map[string]interface{}{
  "array": []interface{}{
    map[string]interface{}{
      "position": "first",
    },
    map[string]interface{}{
      "position": "second",
    },
  },
}

r := NewFromMap(m)

val, err := r.GetIterable("array")
// val is []interface{}
// err is nil

for _, objI := range val {
  obj, err := NewFromInterface(objI)
  // obj is next array element in every iteration
  // err is nil
}
```

## And many more (check the code!)

# Constructors

JSON content is assumed in String/Bytes, unless YAML is specified. YAML must be able to be converted to JSON.

- NewEmpty
- NewFromInterface
- NewFromMap
- NewFromBytes
- NewFromString
- NewFromReader  
- NewFromYAMLBytes
- NewFromYAMLFile