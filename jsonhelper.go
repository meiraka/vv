package main

import (
	"encoding/json"
	"errors"
	"io"
)

type jsonMap map[string]interface{}

func parseSimpleJSON(b io.Reader) (jsonMap, error) {
	decoder := json.NewDecoder(b)
	s := jsonMap{}
	return s, decoder.Decode(&s)
}

func (j *jsonMap) execIfInt(key string, f func(int) error) error {
	d := *j
	if v, exist := d[key]; exist {
		switch v.(type) {
		case float64:
			return f(int(v.(float64)))
		default:
			return errors.New("unexpected type for " + key)
		}
	}
	return nil
}

func (j *jsonMap) execIfBool(key string, f func(bool) error) error {
	d := *j
	if v, exist := d[key]; exist {
		switch v.(type) {
		case bool:
			return f(v.(bool))
		default:
			return errors.New("unexpected type for " + key)
		}
	}
	return nil
}

func (j *jsonMap) execIfString(key string, f func(string) error) error {
	d := *j
	if v, exist := d[key]; exist {
		switch v.(type) {
		case string:
			return f(v.(string))
		default:
			return errors.New("unexpected type for " + key)
		}
	}
	return nil
}
