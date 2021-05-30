package mpd

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestParseCommandError(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want error
	}{
		{in: `ACK [50@1] {play} song doesn't exist: "10240"`,
			want: &CommandError{ID: 50, Index: 1, Command: "play", Message: `song doesn't exist: "10240"`},
		},
		{in: ``, want: ErrParse},
		{in: `ACK [`, want: ErrParse},
		{in: `ACK [1@1]`, want: ErrParse},
		{in: `ACK [@] {}`, want: ErrParse},
		{in: `ACK [1@] {}`, want: ErrParse},
		{in: `ACK [1@1] {}`, want: &CommandError{ID: 1, Index: 1}},
		{in: `ACK [1@1] {} `, want: &CommandError{ID: 1, Index: 1}},
		{in: `ACK [1@b] {}`, want: ErrParse},
		{in: `ACK [c@a] {}`, want: ErrParse},
	} {
		got := parseCommandError(tt.in)
		if !errors.Is(got, tt.want) {
			t.Errorf("parseCommandError(%q)=%q; want %q", tt.in, got, tt.want)
		} else {
			t.Log(got)
		}
	}
}

func TestParseEnd(t *testing.T) {
	for label, tt := range map[string]struct {
		in  string
		end string
		err error
	}{
		"ok": {
			in:  "OK\n",
			end: responseOK,
		},
		"listok": {
			in:  "list_OK\n",
			end: responseListOK,
		},
		"error/ack": {
			in:  "ACK [5@1] {errendack} foo: bar\n",
			end: responseOK,
			err: &CommandError{Command: "errendack", ID: 5, Index: 1, Message: "foo: bar"},
		},
		"error:parse": {
			in:  "file: foo/bar.flac\nOK\n",
			end: responseOK,
			err: ErrParseNoEndResponse,
		},
	} {
		t.Run(label, func(t *testing.T) {
			err := parseEnd(bufio.NewReader(strings.NewReader(tt.in)), tt.end)
			if !errors.Is(err, tt.err) {
				t.Errorf("got %v; want %v", err, tt.err)
			}

		})

	}
}

func TestParseBinary(t *testing.T) {
	for label, tt := range map[string]struct {
		in string
		// end string
		want   map[string]string
		binary []byte
		err    error
	}{
		"ok:empty": {
			in:   "OK\n",
			want: map[string]string{},
		},
		"ok:binary": {
			in:     "foo: bar\nbinary: 3\nabc\nOK\n",
			want:   map[string]string{"foo": "bar", "binary": "3"},
			binary: []byte("abc"),
		},
		"error:ack": {
			in:  "ACK [5@1] {errbinaryack} foo: bar\n",
			err: &CommandError{Command: "errbinaryack", ID: 5, Index: 1, Message: "foo: bar"},
		},
		"error:parse:map": {
			in:  "file foo/bar.flac\nOK\n",
			err: ErrParseNoKey,
		},
		"error:parse:binary:eof": {
			in:  "foo: bar\nbinary: 3\nab",
			err: io.ErrUnexpectedEOF,
		},
		"error:parse:binary:noend": {
			in:  "foo: bar\nbinary: 3\nabc\nnot ok\n",
			err: ErrParseNoEndResponse,
		},
	} {
		t.Run(label, func(t *testing.T) {
			m, b, err := parseBinary(bufio.NewReader(strings.NewReader(tt.in)), responseOK)
			if !reflect.DeepEqual(m, tt.want) || !bytes.Equal(b, tt.binary) || !errors.Is(err, tt.err) {
				t.Errorf("got %v, %q, %v; want %v, %q, %v", m, b, err, tt.want, tt.binary, tt.err)
			}
		})
	}
}

func TestParseList(t *testing.T) {
	for label, tt := range map[string]struct {
		in     string
		end    string
		prefix string
		want   []string
		err    error
	}{
		"ok": {
			in: strings.Join([]string{
				"command: stop",
				"command: subscribe",
				"command: swap",
				"command: swapid",
				"command: tagtypes",
				"command: toggleoutput",
				"command: unmount",
				"command: unsubscribe",
				"command: update",
				"command: urlhandlers",
				"command: volume",
				"OK",
			}, "\n") + "\n",
			end:    responseOK,
			prefix: "command",
			want: []string{
				"stop",
				"subscribe",
				"swap",
				"swapid",
				"tagtypes",
				"toggleoutput",
				"unmount",
				"unsubscribe",
				"update",
				"urlhandlers",
				"volume",
			},
		},
		"error:ack": {
			in:  "ACK [5@1] {errlistack} foo: bar\n",
			end: responseOK,
			err: &CommandError{Command: "errlistack", ID: 5, Index: 1, Message: "foo: bar"},
		},
		"error:parse": {
			in:  "file foo/bar.flac\nOK\n",
			end: responseOK,
			err: ErrParseNoKey,
		},
	} {
		t.Run(label, func(t *testing.T) {
			got, err := parseList(bufio.NewReader(strings.NewReader(tt.in)), tt.end, tt.prefix)
			if !reflect.DeepEqual(got, tt.want) || !errors.Is(err, tt.err) {
				t.Errorf("got %q, %v; want %q, %v", got, err, tt.want, tt.err)
			}
		})
	}

}

func TestParseSong(t *testing.T) {
	for label, tt := range map[string]struct {
		in   string
		end  string
		want map[string][]string
		err  error
	}{
		"ok": {
			in: strings.Join([]string{
				"file: foo/bar.flac",
				"Last-Modified: 2015-07-29T05:08:47Z",
				"Album: foo",
				"Artist: 1",
				"Artist: 2",
				"Date: 2009",
				"Title: bar",
				"Track: 4",
				"Time: 392",
				"duration: 392.000",
				"Pos: 23",
				"Id: 311",
				"OK",
			}, "\n") + "\n",
			end: responseOK,
			want: map[string][]string{
				"file":          {"foo/bar.flac"},
				"Last-Modified": {"2015-07-29T05:08:47Z"},
				"Album":         {"foo"},
				"Artist":        {"1", "2"},
				"Date":          {"2009"},
				"Title":         {"bar"},
				"Track":         {"4"},
				"Time":          {"392"},
				"duration":      {"392.000"},
				"Pos":           {"23"},
				"Id":            {"311"},
			},
		},
		"listok": {
			in: strings.Join([]string{
				"file: foo/bar.flac",
				"Last-Modified: 2015-07-29T05:08:47Z",
				"Album: foo",
				"Artist: 1",
				"Artist: 2",
				"Date: 2009",
				"Title: bar",
				"Track: 4",
				"Time: 392",
				"duration: 392.000",
				"Pos: 23",
				"Id: 311",
				"list_OK",
			}, "\n") + "\n",
			end: responseListOK,
			want: map[string][]string{
				"file":          {"foo/bar.flac"},
				"Last-Modified": {"2015-07-29T05:08:47Z"},
				"Album":         {"foo"},
				"Artist":        {"1", "2"},
				"Date":          {"2009"},
				"Title":         {"bar"},
				"Track":         {"4"},
				"Time":          {"392"},
				"duration":      {"392.000"},
				"Pos":           {"23"},
				"Id":            {"311"},
			},
		},
		"error:ack": {
			in:  "ACK [5@1] {errsongack} foo: bar\n",
			end: responseOK,
			err: &CommandError{Command: "errsongack", ID: 5, Index: 1, Message: "foo: bar"},
		},
		"error:parse": {
			in:  "file foo/bar.flac\nOK\n",
			end: responseOK,
			err: ErrParseNoKey,
		},
	} {
		t.Run(label, func(t *testing.T) {
			got, err := parseSong(bufio.NewReader(strings.NewReader(tt.in)), tt.end)
			if !reflect.DeepEqual(got, tt.want) || !errors.Is(err, tt.err) {
				t.Errorf("got %q, %v; want %q, %v", got, err, tt.want, tt.err)
			}
		})
	}
}

func TestParseSongs(t *testing.T) {
	for label, tt := range map[string]struct {
		in   string
		end  string
		want []map[string][]string
		err  error
	}{
		"ok:empty": {
			in:   "OK\n",
			end:  responseOK,
			want: []map[string][]string{},
		},
		"ok": {
			in: strings.Join([]string{
				"directory: foo",
				"Last-Modified: 2015-07-29T05:08:49Z",
				"file: foo/bar.flac",
				"Last-Modified: 2015-07-29T05:08:47Z",
				"Album: foo",
				"Artist: 1",
				"Artist: 2",
				"Date: 2009",
				"Title: bar",
				"Track: 4",
				"Time: 392",
				"duration: 392.000",
				"Pos: 23",
				"Id: 311",
				"directory: baz",
				"Last-Modified: 2015-07-29T05:08:48Z",
				"file: baz/bar.flac",
				"Last-Modified: 2015-07-29T05:08:46Z",
				"OK",
			}, "\n") + "\n",
			end: responseOK,
			want: []map[string][]string{{
				"file":          {"foo/bar.flac"},
				"Last-Modified": {"2015-07-29T05:08:47Z"},
				"Album":         {"foo"},
				"Artist":        {"1", "2"},
				"Date":          {"2009"},
				"Title":         {"bar"},
				"Track":         {"4"},
				"Time":          {"392"},
				"duration":      {"392.000"},
				"Pos":           {"23"},
				"Id":            {"311"},
			}, {"file": {"baz/bar.flac"}, "Last-Modified": {"2015-07-29T05:08:46Z"}}},
		},
		"ok:listok": {
			in: strings.Join([]string{
				"directory: foo",
				"Last-Modified: 2015-07-29T05:08:49Z",
				"file: foo/bar.flac",
				"Last-Modified: 2015-07-29T05:08:47Z",
				"Album: foo",
				"Artist: 1",
				"Artist: 2",
				"Date: 2009",
				"Title: bar",
				"Track: 4",
				"Time: 392",
				"duration: 392.000",
				"Pos: 23",
				"Id: 311",
				"list_OK",
			}, "\n") + "\n",
			end: responseListOK,
			want: []map[string][]string{{
				"file":          {"foo/bar.flac"},
				"Last-Modified": {"2015-07-29T05:08:47Z"},
				"Album":         {"foo"},
				"Artist":        {"1", "2"},
				"Date":          {"2009"},
				"Title":         {"bar"},
				"Track":         {"4"},
				"Time":          {"392"},
				"duration":      {"392.000"},
				"Pos":           {"23"},
				"Id":            {"311"},
			}},
		},
		"error:ack": {
			in:  "ACK [5@1] {errsongsack} foo: bar\n",
			end: responseOK,
			err: &CommandError{Command: "errsongsack", ID: 5, Index: 1, Message: "foo: bar"},
		},
		"error:parse:nostart": {
			in:  "file foo/bar.flac\nOK\n",
			end: responseOK,
			err: ErrParseNoStartResponse,
		},
		"error:parse:nokey": {
			in:  "file: foo/bar.flac\nfile foo/bar.flac\nOK\n",
			end: responseOK,
			err: ErrParseNoKey,
		},
	} {
		t.Run(label, func(t *testing.T) {
			got, err := parseSongs(bufio.NewReader(strings.NewReader(tt.in)), tt.end)
			if !reflect.DeepEqual(got, tt.want) || !errors.Is(err, tt.err) {
				t.Errorf("got %q, %v; want %q, %v", got, err, tt.want, tt.err)
			}
		})
	}
}

func TestParseMap(t *testing.T) {
	for label, tt := range map[string]struct {
		in   string
		end  string
		want map[string]string
		err  error
	}{
		"ok:empty": {
			in:   "OK\n",
			end:  responseOK,
			want: map[string]string{},
		},
		"ok": {
			in: strings.Join([]string{
				"uptime: 1041403",
				"playtime: 85296",
				"artists: 981",
				"albums: 597",
				"songs: 6411",
				"db_playtime: 1659296",
				"db_update: 1610585747",
				"OK",
			}, "\n") + "\n",
			end: responseOK,
			want: map[string]string{
				"uptime":      "1041403",
				"playtime":    "85296",
				"artists":     "981",
				"albums":      "597",
				"songs":       "6411",
				"db_playtime": "1659296",
				"db_update":   "1610585747",
			},
		},
		"ok:listok": {
			in: strings.Join([]string{
				"uptime: 1041403",
				"playtime: 85296",
				"artists: 981",
				"albums: 597",
				"songs: 6411",
				"db_playtime: 1659296",
				"db_update: 1610585747",
				"list_OK",
			}, "\n") + "\n",
			end: responseListOK,
			want: map[string]string{
				"uptime":      "1041403",
				"playtime":    "85296",
				"artists":     "981",
				"albums":      "597",
				"songs":       "6411",
				"db_playtime": "1659296",
				"db_update":   "1610585747",
			},
		},
		"error:ack": {
			in:  "ACK [5@1] {errmapack} foo: bar\n",
			end: responseOK,
			err: &CommandError{Command: "errmapack", ID: 5, Index: 1, Message: "foo: bar"},
		},
		"error:parse": {
			in:  "file foo/bar.flac\nOK\n",
			end: responseOK,
			err: ErrParseNoKey,
		},
	} {
		t.Run(label, func(t *testing.T) {
			got, err := parseMap(bufio.NewReader(strings.NewReader(tt.in)), tt.end)
			if !reflect.DeepEqual(got, tt.want) || !errors.Is(err, tt.err) {
				t.Errorf("got %q, %v; want %q, %v", got, err, tt.want, tt.err)
			}
		})
	}

}

func TestParseListMap(t *testing.T) {
	for label, tt := range map[string]struct {
		in     string
		end    string
		newkey string
		want   []map[string]string
		err    error
	}{
		"ok:empty": {
			in:     "OK\n",
			end:    responseOK,
			newkey: "mount",
			want:   []map[string]string{},
		},
		"ok": {
			in: strings.Join([]string{
				"mount: ",
				"mount: storage",
				"storage: nfs://storage.local/Music",
				"OK",
			}, "\n") + "\n",
			newkey: "mount",
			end:    responseOK,
			want: []map[string]string{
				{"mount": ""},
				{"mount": "storage", "storage": "nfs://storage.local/Music"},
			},
		},
		"ok:listok": {
			in: strings.Join([]string{
				"mount: ",
				"mount: storage",
				"storage: nfs://storage.local/Music",
				"list_OK",
			}, "\n") + "\n",
			newkey: "mount",
			end:    responseListOK,
			want: []map[string]string{
				{"mount": ""},
				{"mount": "storage", "storage": "nfs://storage.local/Music"},
			},
		},
		"error:ack": {
			in:     "ACK [5@1] {errmapack} foo: bar\n",
			newkey: "mount",
			end:    responseOK,
			err:    &CommandError{Command: "errmapack", ID: 5, Index: 1, Message: "foo: bar"},
		},
		"error:parse:nostart": {
			in:     "storage: nfs://storage.local/Music\nOK\n",
			newkey: "mount",
			end:    responseOK,
			err:    ErrParseNoStartResponse,
		},
		"error:parse:nokey": {
			in:     "mount: \nstorage\nOK\n",
			newkey: "mount",
			end:    responseOK,
			err:    ErrParseNoKey,
		},
	} {
		t.Run(label, func(t *testing.T) {
			got, err := parseListMap(bufio.NewReader(strings.NewReader(tt.in)), tt.end, tt.newkey)
			if !reflect.DeepEqual(got, tt.want) || !errors.Is(err, tt.err) {
				t.Errorf("got %q, %v; want %q, %v", got, err, tt.want, tt.err)
			}
		})
	}
}

func TestParseOutputs(t *testing.T) {
	for label, tt := range map[string]struct {
		in   string
		end  string
		want []*Output
		err  error
	}{
		"ok:empty": {
			in:   "OK\n",
			end:  responseOK,
			want: []*Output{},
		},
		"ok:listok:empty": {
			in:   "list_OK\n",
			end:  responseListOK,
			want: []*Output{},
		},
		"ok": {
			in: strings.Join([]string{
				"outputid: 0",
				"outputname: My ALSA Device",
				"plugin: alsa",
				"outputenabled: 0",
				"attribute: allowed_formats=",
				"attribute: dop=0",
				"outputid: 1",
				"outputname: OGG / HTTP Stream",
				"plugin: httpd",
				"outputenabled: 0",
				"outputid: 2",
				"outputname: FLAC HTTP Stream",
				"plugin: httpd",
				"outputenabled: 0",
				"outputid: 3",
				"outputname: MP3 Stream",
				"plugin: httpd",
				"outputenabled: 1",
				"outputid: 4",
				"outputname: My Pulse Output",
				"plugin: pulse",
				"outputenabled: 0",
				"OK",
			}, "\n") + "\n",
			end: responseOK,
			want: []*Output{
				{
					ID:      "0",
					Name:    "My ALSA Device",
					Plugin:  "alsa",
					Enabled: false,
					Attributes: map[string]string{
						"allowed_formats": "",
						"dop":             "0",
					},
				},
				{
					ID:      "1",
					Name:    "OGG / HTTP Stream",
					Plugin:  "httpd",
					Enabled: false,
				},
				{
					ID:      "2",
					Name:    "FLAC HTTP Stream",
					Plugin:  "httpd",
					Enabled: false,
				},
				{
					ID:      "3",
					Name:    "MP3 Stream",
					Plugin:  "httpd",
					Enabled: true,
				},
				{
					ID:      "4",
					Name:    "My Pulse Output",
					Plugin:  "pulse",
					Enabled: false,
				},
			},
		},
		"error:ack": {
			in:  "ACK [5@1] {errmapack} foo: bar\n",
			end: responseOK,
			err: &CommandError{Command: "errmapack", ID: 5, Index: 1, Message: "foo: bar"},
		},
		"error:parse:nostart": {
			in:  "foobar\n",
			end: responseOK,
			err: ErrParseNoStartResponse,
		},
		"error:parse:nokey": {
			in:  "outputid: \nfoobar\nOK\n",
			end: responseOK,
			err: ErrParseNoKey,
		},
	} {
		t.Run(label, func(t *testing.T) {
			got, err := parseOutputs(bufio.NewReader(strings.NewReader(tt.in)), tt.end)
			if !reflect.DeepEqual(got, tt.want) || !errors.Is(err, tt.err) {
				t.Errorf("got %v, %v; want %v, %v", got, err, tt.want, tt.err)
			}
		})
	}
}
