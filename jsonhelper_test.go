package main

import (
	"strings"
	"testing"
)

func TestParseSimpleJson(t *testing.T) {
	t.Run("parse Error", func(t *testing.T) {
		r := strings.NewReader("")
		_, err := parseSimpleJSON(r)
		if err == nil {
			t.Errorf("unexpected nil error")
		}
	})
	t.Run("parse OK", func(t *testing.T) {
		r := strings.NewReader("{\"int\": 1, \"bool\": true, \"string\": \"foo\"}")
		j, err := parseSimpleJSON(r)
		if err != nil {
			t.Errorf("unexpected error: %s", err.Error())
		}
		t.Run("execIfInt", func(t *testing.T) {
			store := 0
			execFunc := func(i int) error {
				store = i
				return nil
			}
			t.Run("not found", func(t *testing.T) {
				err = j.execIfInt("foo", execFunc)
				if err != nil {
					t.Errorf("unexpected error: %s", err.Error())
				}
				if store != 0 {
					t.Errorf("unexpected function call")
				}
			})
			t.Run("unexpected type", func(t *testing.T) {
				err = j.execIfInt("bool", execFunc)
				if err == nil {
					t.Errorf("unexpected nil error")
				}
				if store != 0 {
					t.Errorf("unexpected function call")
				}
			})
			t.Run("called", func(t *testing.T) {
				err = j.execIfInt("int", execFunc)
				if err != nil {
					t.Errorf("unexpected error: %s", err.Error())
				}
				if store != 1 {
					t.Errorf("expected function call")
				}
			})
		})

		t.Run("execIfBool", func(t *testing.T) {
			store := false
			execFunc := func(b bool) error {
				store = b
				return nil
			}
			t.Run("not found", func(t *testing.T) {
				err = j.execIfBool("foo", execFunc)
				if err != nil {
					t.Errorf("unexpected error: %s", err.Error())
				}
				if store != false {
					t.Errorf("unexpected function call")
				}
			})
			t.Run("unexpected type", func(t *testing.T) {
				err = j.execIfBool("int", execFunc)
				if err == nil {
					t.Errorf("unexpected nil error")
				}
				if store != false {
					t.Errorf("unexpected function call")
				}
			})
			t.Run("called", func(t *testing.T) {
				err = j.execIfBool("bool", execFunc)
				if err != nil {
					t.Errorf("unexpected error: %s", err.Error())
				}
				if store != true {
					t.Errorf("expected function call")
				}
			})
		})

		t.Run("execIfString", func(t *testing.T) {
			store := ""
			execFunc := func(s string) error {
				store = s
				return nil
			}
			t.Run("not found", func(t *testing.T) {
				err = j.execIfString("foo", execFunc)
				if err != nil {
					t.Errorf("unexpected error: %s", err.Error())
				}
				if store != "" {
					t.Errorf("unexpected function call")
				}
			})
			t.Run("unexpected type", func(t *testing.T) {
				err = j.execIfString("int", execFunc)
				if err == nil {
					t.Errorf("unexpected nil error")
				}
				if store != "" {
					t.Errorf("unexpected function call")
				}
			})
			t.Run("called", func(t *testing.T) {
				err = j.execIfString("string", execFunc)
				if err != nil {
					t.Errorf("unexpected error: %s", err.Error())
				}
				if store != "foo" {
					t.Errorf("expected function call")
				}
			})
		})
	})
}
