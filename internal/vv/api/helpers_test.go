package api_test

import (
	"errors"
	"testing"
)

var errTest = errors.New("api_test: test error")

func recieveMsg(c <-chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}

func mockBoolFunc(f string, want bool, ret error) func(t *testing.T, b bool) error {
	return func(t *testing.T, got bool) error {
		t.Helper()
		if got != want {
			t.Errorf("called "+f+"; want "+f, got, want)
		}
		return ret
	}
}

func mockIntFunc(f string, want int, ret error) func(t *testing.T, b int) error {
	return func(t *testing.T, got int) error {
		t.Helper()
		if got != want {
			t.Errorf("called "+f+"; want "+f, got, want)
		}
		return ret
	}
}

func mockStringFunc(f string, want string, ret error) func(t *testing.T, b string) error {
	return func(t *testing.T, got string) error {
		t.Helper()
		if got != want {
			t.Errorf("called "+f+"; want "+f, got, want)
		}
		return ret
	}
}

func mockFloat64Func(f string, want float64, ret error) func(t *testing.T, b float64) error {
	return func(t *testing.T, got float64) error {
		t.Helper()
		if got != want {
			t.Errorf("called "+f+"; want "+f, got, want)
		}
		return ret
	}
}
