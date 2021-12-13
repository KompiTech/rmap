package rmap

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

//RmapsToCSV takes multiple Rmap instances and returns as CSV bytes with header
//nested keys are stored as l1.l2.l3
func RmapsToCSV(rmaps []Rmap, separator string) ([]byte, error) {
	header := map[string]interface{}{}
	//Get header from first element
	collectKeys(rmaps[0], nil, &header)
	//Get sorted header keys
	headerKeys := NewFromMap(header).KeysSliceString()
	sort.Strings(headerKeys)
	output := bytes.Buffer{}

	//write sorted header to csv
	output.Write(writeHeader(headerKeys, separator))
	output.WriteString("\n")

	for _, rm := range rmaps {
		//each row starts with copy of header with all values set to struct{}
		row := NewFromMap(header).Copy()

		//fill row with values
		if err := collectValues(rm, nil, &row.Mapa); err != nil {
			return nil, err
		}

		//generate bytes for one row, sorted by header keys
		rowBytes, err := writeValues(row, headerKeys, separator)
		if err != nil {
			return nil, err
		}

		output.Write(rowBytes)
		output.WriteString("\n")
	}

	return output.Bytes(), nil
}

func writeHeader(keys []string, separator string) []byte {
	return []byte(strings.Join(keys, separator))
}

func writeValues(input Rmap, headerKeys []string, separator string) ([]byte, error) {
	rowData := make([]string, len(headerKeys))

	for idx, key := range headerKeys {
		val, err := input.Get(key)
		if err != nil {
			return nil, err
		}

		valS, isString := val.(string)
		if isString {
			if strings.Index(valS, separator) != -1 {
				//, in string, wrap in ""
				if strings.Index(valS, `"`) != -1 {
					// " in string remove
					valS = strings.Replace(valS, `"`, ``, -1)
				}

				rowData[idx] = `"` + valS + `"`
			} else {
				rowData[idx] = fmt.Sprintf("%v", val)
			}
		} else {
			rowData[idx] = fmt.Sprintf("%v", val)
		}
	}

	return []byte(strings.Join(rowData, separator)), nil
}

func collectValues(input Rmap, path []string, row *map[string]interface{}) error {
	for k, v := range input.Mapa {
		switch v.(type) {
		case Rmap:
			//nested Rmap, recurse
			if err := collectValues(v.(Rmap), append(path, k), row); err != nil {
				return err
			}
		case map[string]interface{}:
			//nested map, recurse
			if err := collectValues(NewFromMap(v.(map[string]interface{})), append(path, k), row); err != nil {
				return err
			}
		default:
			if err := processValue(v, append(path, k), row); err != nil {
				return err
			}
		}
	}

	return nil
}

func processValue(value interface{}, path []string, row *map[string]interface{}) error {
	key := strings.Join(path, ".")

	_, exists := (*row)[key]
	if !exists {
		return fmt.Errorf("unexpected key: %s, not found in header", key)
	}

	switch value.(type) {
	case string:
		(*row)[key] = strings.Replace(value.(string), "\n", "", -1)
	case float64:
		(*row)[key] = value.(float64)
	case int:
		(*row)[key] = value.(int)
	default:
		//fallback
		(*row)[key] = fmt.Sprintf("%v", value)
	}

	return nil
}

//fill keys map with keys present in input
//nested keys are returned in format a.b.c
func collectKeys(input Rmap, path []string, keys *map[string]interface{}) {
	for k, v := range input.Mapa {
		switch v.(type) {
		case Rmap:
			//nested Rmap, recurse
			collectKeys(v.(Rmap), append(path, k), keys)
		case map[string]interface{}:
			//nested map, recurse
			collectKeys(NewFromMap(v.(map[string]interface{})), append(path, k), keys)
		default:
			//anything else is just a key
			(*keys)[strings.Join(append(path, k), ".")] = struct{}{}
		}
	}
}
