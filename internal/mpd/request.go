package mpd

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func request(w io.Writer, cmd string, args ...interface{}) error {
	if _, err := io.WriteString(w, cmd); err != nil {
		return err
	}
	for i := range args {
		if _, err := io.WriteString(w, " "); err != nil {
			return err
		}
		switch v := args[i].(type) {
		case string:
			if _, err := io.WriteString(w, quote(v)); err != nil {
				return err
			}
		case bool:
			if _, err := io.WriteString(w, btoa(v, "1", "0")); err != nil {
				return err
			}
		case int:
			if _, err := io.WriteString(w, strconv.Itoa(v)); err != nil {
				return err
			}
		case float64:
			if _, err := io.WriteString(w, strconv.FormatFloat(v, 'g', -1, 64)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("mpd: fixme: unsupported arguments type: %#v", v)
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func srequest(cmd string, args ...interface{}) (string, error) {
	b := &strings.Builder{}
	if err := request(b, cmd, args...); err != nil {
		return "", err
	}
	return b.String(), nil
}

var quoter = strings.NewReplacer(
	"\\", "\\\\",
	`"`, `\"`,
	"\n", "",
)

// quote escaping strings values for mpd.
func quote(s string) string {
	return `"` + quoter.Replace(s) + `"`
}

func btoa(s bool, t string, f string) string {
	if s {
		return t
	}
	return f
}
