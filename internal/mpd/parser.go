package mpd

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// parser errors.
var (
	ErrParse                = errors.New("parse error")
	ErrParseNoStartResponse = fmt.Errorf("%w: no start response", ErrParse)
	ErrParseNoEndResponse   = fmt.Errorf("%w: no end response", ErrParse)
	ErrParseNoKey           = fmt.Errorf("%w: no key and colon", ErrParse)
)

func addCommandInfo(err error, cmd string) error {
	cmdErr := &CommandError{}
	if !errors.As(err, &cmdErr) {
		return fmt.Errorf("mpd: %s: %w", cmd, err)
	}
	return err
}

func parseCommandError(s string) error {
	if !strings.HasPrefix(s, "ACK [") {
		return fmt.Errorf("ack: %w: got %q, want %q", ErrParse, s, "ACK [")
	}

	u := s[5:]
	at := strings.IndexRune(u, '@')
	if at < 0 {
		return fmt.Errorf("ack: %w: got %q, want %q", ErrParse, s, '@')
	}
	id, err := strconv.Atoi(u[:at])
	if err != nil {
		return fmt.Errorf("ack: %w: %v: got %q", ErrParse, err, s)
	}
	u = u[at+1:]

	sqBEnd := strings.IndexRune(u, ']')
	if sqBEnd < 0 {
		return fmt.Errorf("ack: %w: got %q, want %q", ErrParse, s, ']')
	}
	index, err := strconv.Atoi(u[:sqBEnd])
	if err != nil {
		return fmt.Errorf("ack: %w: %v: got %q", ErrParse, err, s)
	}
	u = u[sqBEnd+1:]

	crBStart := strings.IndexRune(u, '{')
	if crBStart < 0 {
		return fmt.Errorf("ack: %w: got %q, want %q", ErrParse, s, '{')
	}
	u = u[crBStart+1:]

	crBEnd := strings.IndexRune(u, '}')
	if crBEnd < 0 {
		return fmt.Errorf("ack: %w: got %q, want %q", ErrParse, s, '}')
	}
	cmd := u[:crBEnd]
	u = u[crBEnd+1:]

	return &CommandError{
		ID:      AckError(id),
		Index:   index,
		Command: cmd,
		Message: strings.TrimSpace(u),
	}
}

type connReader interface {
	ReadString(delim byte) (string, error)
	io.Reader
}

func readln(c connReader) (string, error) {
	s, err := c.ReadString('\n')
	if err != nil {
		return s, err
	}
	return s[0 : len(s)-1], nil
}

// isEnd checks line is equal to end or failed response.
func isEnd(line string, end string) (bool, error) {
	if line == end {
		return true, nil
	}
	if strings.HasPrefix(line, "ACK [") {
		return true, parseCommandError(line)
	}
	return false, nil
}

func parseEnd(conn connReader, end string) error {
	line, err := readln(conn)
	if err != nil {
		return err
	}
	ok, err := isEnd(line, end)
	if !ok {
		return fmt.Errorf("%w: got: %q; want: %q", ErrParseNoEndResponse, line, end)
	}
	if err != nil {
		return err
	}
	return nil
}

func parseBinary(r connReader, end string) (map[string]string, []byte, error) {
	m := map[string]string{}
	var key, value string
	for {
		line, err := readln(r)
		if err != nil {
			return nil, nil, err
		}
		if ok, err := isEnd(line, end); ok {
			if err != nil {
				return nil, nil, err
			}
			return m, nil, nil
		}
		i := strings.Index(line, ": ")
		if i < 0 {
			return nil, nil, fmt.Errorf("%w: %s", ErrParseNoKey, line)
		}
		key = line[0:i]
		value = line[i+2:]
		m[key] = value
		if key == "binary" {
			length, err := strconv.Atoi(value)
			if err != nil {
				return nil, nil, err
			}
			// binary
			b := make([]byte, length)
			_, err = io.ReadFull(r, b)
			if err != nil {
				return nil, nil, err
			}
			// newline
			_, err = r.ReadString('\n')
			if err != nil {
				return nil, nil, err
			}
			// OK
			if err := parseEnd(r, end); err != nil {
				return nil, nil, err
			}
			return m, b, nil
		}
	}
}

func parseList(conn connReader, end string, label string) ([]string, error) {
	prefix := label + ": "
	ret := []string{}
	for {
		line, err := readln(conn)
		if err != nil {
			return nil, err
		}
		if ok, err := isEnd(line, end); ok {
			if err != nil {
				return nil, err
			}
			return ret, nil
		}
		s := strings.TrimPrefix(line, prefix)
		if s == line {
			return nil, fmt.Errorf("%w: %s", ErrParseNoKey, line)
		}
		ret = append(ret, s)
	}

}

func parseSong(conn connReader, end string) (map[string][]string, error) {
	song := map[string][]string{}
	for {
		line, err := readln(conn)
		if err != nil {
			return nil, err
		}
		if ok, err := isEnd(line, end); ok {
			if err != nil {
				return nil, err
			}
			return song, nil
		}
		i := strings.Index(line, ": ")
		if i < 0 {
			return nil, fmt.Errorf("%w: %s", ErrParseNoKey, line)
		}
		key := line[0:i]
		song[key] = append(song[key], line[i+2:])
	}
}

func parseSongs(conn connReader, end string) ([]map[string][]string, error) {
	songs := []map[string][]string{}
	var song map[string][]string
	in := true
	for {
		line, err := readln(conn)
		if err != nil {
			return nil, err
		}
		if ok, err := isEnd(line, end); ok {
			if err != nil {
				return nil, err
			}
			return songs, nil
		}
		if strings.HasPrefix(line, "file: ") {
			song = map[string][]string{}
			songs = append(songs, song)
			in = true
		} else if strings.HasPrefix(line, "directory: ") { // skip listallinfo directory info
			in = false
		}
		if in {
			if len(songs) == 0 {
				// song is not initialized.
				return nil, fmt.Errorf("%w: got: %q; want: %q", ErrParseNoStartResponse, line, "file: ")
			}
			i := strings.Index(line, ": ")
			if i < 0 {
				return nil, fmt.Errorf("%w: %s", ErrParseNoKey, line)
			}
			key := line[0:i]
			song[key] = append(song[key], line[i+2:])
		}
	}
}

func parseMap(conn connReader, end string) (map[string]string, error) {
	m := map[string]string{}
	for {
		line, err := readln(conn)
		if err != nil {
			return nil, err
		}
		if ok, err := isEnd(line, end); ok {
			if err != nil {
				return nil, err
			}
			return m, nil
		}
		i := strings.Index(line, ": ")
		if i < 0 {
			return nil, fmt.Errorf("%w: %s", ErrParseNoKey, line)
		}
		m[line[0:i]] = line[i+2:]
	}
}

func parseListMap(conn connReader, end string, newKey string) ([]map[string]string, error) {
	nkPrefix := newKey + ": "
	l := []map[string]string{}
	var m map[string]string
	for {
		line, err := readln(conn)
		if err != nil {
			return nil, err
		}
		if ok, err := isEnd(line, end); ok {
			if err != nil {
				return nil, err
			}
			return l, nil
		}
		if strings.HasPrefix(line, nkPrefix) {
			m = map[string]string{}
			l = append(l, m)
		}
		if m == nil {
			return nil, fmt.Errorf("%w: got: %q; want: %q", ErrParseNoStartResponse, line, nkPrefix)
		}
		i := strings.Index(line, ": ")
		if i < 0 {
			return nil, fmt.Errorf("%w: %s", ErrParseNoKey, line)
		}
		m[line[0:i]] = line[i+2:]
	}
}

func parseOutputs(conn connReader, end string) ([]*Output, error) {
	outputs := []*Output{}
	var output *Output
	for {
		line, err := readln(conn)
		if err != nil {
			return nil, err
		}
		if ok, err := isEnd(line, end); ok {
			if err != nil {
				return nil, err
			}
			return outputs, nil
		}
		if strings.HasPrefix(line, "outputid: ") {
			output = &Output{}
			outputs = append(outputs, output)
		}
		if output == nil {
			return nil, fmt.Errorf("%w: got: %q; want: %q", ErrParseNoStartResponse, line, "outputid: ")
		}
		i := strings.Index(line, ": ")
		if i < 0 {
			return nil, fmt.Errorf("%w: %s", ErrParseNoKey, line)
		}
		key, value := line[0:i], line[i+2:]
		if key == "outputid" {
			output.ID = value
		} else if key == "outputname" {
			output.Name = value
		} else if key == "outputenabled" {
			output.Enabled = (value == "1")
		} else if key == "plugin" {
			output.Plugin = value
		} else if key == "attribute" {
			i := strings.Index(value, "=")
			if i < 0 {
				continue
			}
			if output.Attributes == nil {
				output.Attributes = make(map[string]string)
			}
			output.Attributes[value[0:i]] = value[i+1:]
		}
	}
}
