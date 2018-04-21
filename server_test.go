package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/meiraka/gompd/mpd"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestApiImage(t *testing.T) {
	m := new(MockMusic)
	s := Server{Music: m, MusicDirectory: "assets/"}
	handler := s.makeHandle()
	ts := httptest.NewServer(handler)
	testsets := []struct {
		desc            string
		path            string
		ret             int
		ifModifiedSince time.Time
	}{
		{desc: "404: not music_directory", path: "/api/images/app.png?width=100&height=100", ret: 404},
		{desc: "404: file not found", path: "/api/images/music_directory/notfound.png?width=100&height=100", ret: 404},
		{desc: "500: unsupported format", path: "/api/images/music_directory/app.svg?width=100&height=100", ret: 500},
		{desc: "400: missing queries", path: "/api/images/music_directory/app.png", ret: 400},
		{desc: "200: ok", path: "/api/images/music_directory/app.png?width=100&height=100", ret: 200},
		{
			desc:            "304: not modified",
			path:            "/api/images/music_directory/app.png?width=100&height=100",
			ifModifiedSince: time.Now().UTC(),
			ret:             304,
		},
	}
	for _, tt := range testsets {
		req, _ := http.NewRequest("GET", ts.URL+tt.path, nil)
		req.Header.Set("If-Modified-Since", tt.ifModifiedSince.Format(http.TimeFormat))
		client := new(http.Client)
		res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
		if res.StatusCode != tt.ret {
			t.Errorf("[%s] unexpected status. actual:%d expect:%d", tt.desc, res.StatusCode, tt.ret)
		}
	}
}

func TestApiMusicControl(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		m := new(MockMusic)
		s := Server{Music: m}
		handler := s.makeHandle()
		ts := httptest.NewServer(handler)
		url := ts.URL + "/api/music/status"
		defer ts.Close()
		expect := MakeStatus(mpd.Attrs{})
		m.StatusRet1 = expect
		m.StatusRet2 = time.Unix(0, 0)
		testsets := []struct {
			desc            string
			ret             int
			ifModifiedSince time.Time
		}{
			{desc: "200 ok", ret: 200},
		}
		for _, tt := range testsets {
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("If-Modified-Since", tt.ifModifiedSince.Format(http.TimeFormat))
			client := new(http.Client)
			res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
			if res.StatusCode != tt.ret {
				t.Errorf("[%s] unexpected status. actual:%d expect:%d", tt.desc, res.StatusCode, tt.ret)
			}
			if tt.ret != 200 {
				continue
			}
			defer res.Body.Close()
			body, _ := ioutil.ReadAll(res.Body)
			actual := struct {
				Data  Status `json:"data"`
				Error string `json:"error"`
			}{Status{}, ""}
			json.Unmarshal(body, &actual)
			if !reflect.DeepEqual(expect, actual.Data) || actual.Error != "" {
				t.Errorf("unexpected body: %s", body)
			}
		}
	})
	t.Run("post", func(t *testing.T) {
		testsets := []struct {
			desc        string
			ret         int
			input       string
			volumearg1  int
			repeatarg1  bool
			singlearg1  bool
			randomarg1  bool
			playcalled  int
			pausecalled int
			nextcalled  int
			prevcalled  int
			errstr      string
		}{
			{desc: "parse error 400 bad request", ret: 400, input: "hoge", errstr: "failed to get request parameters: invalid character 'h' looking for beginning of value"},
			{desc: "volume 200 ok", ret: 200, input: "{\"volume\": 1}", volumearg1: 1},
			{desc: "repeat 200 ok", ret: 200, input: "{\"repeat\": true}", repeatarg1: true},
			{desc: "single 200 ok", ret: 200, input: "{\"single\": true}", singlearg1: true},
			{desc: "random 200 ok", ret: 200, input: "{\"random\": true}", randomarg1: true},
			{desc: "play 200 ok", ret: 200, input: "{\"state\": \"play\"}", playcalled: 1},
			{desc: "pause 200 ok", ret: 200, input: "{\"state\": \"pause\"}", pausecalled: 1},
			{desc: "next 200 ok", ret: 200, input: "{\"state\": \"next\"}", nextcalled: 1},
			{desc: "prev 200 ok", ret: 200, input: "{\"state\": \"prev\"}", prevcalled: 1},
			{desc: "unknown state 400 bad request", ret: 400, input: "{\"state\": \"unknown\"}", errstr: "unknown state value: unknown"},
		}
		for _, tt := range testsets {
			m := new(MockMusic)
			s := Server{Music: m}
			handler := s.makeHandle()
			ts := httptest.NewServer(handler)
			url := ts.URL + "/api/music/status"
			defer ts.Close()
			j := strings.NewReader(tt.input)
			res, err := http.Post(url, "application/json", j)
			if err != nil {
				t.Errorf("[%s] unexpected request error: %s", tt.desc, err.Error())
			}
			if m.VolumeArg1 != tt.volumearg1 {
				t.Errorf("[%s] unexpected Music.Volume arguments. actual:%d expect:%d", tt.desc, m.VolumeArg1, tt.volumearg1)
			}
			if m.RepeatArg1 != tt.repeatarg1 {
				t.Errorf("[%s] unexpected Music.Repeat arguments. actual:%t expect:%t", tt.desc, m.RepeatArg1, tt.repeatarg1)
			}
			if m.SingleArg1 != tt.singlearg1 {
				t.Errorf("[%s] unexpected Music.Single arguments. actual:%t expect:%t", tt.desc, m.SingleArg1, tt.singlearg1)
			}
			if m.RandomArg1 != tt.randomarg1 {
				t.Errorf("[%s] unexpected Music.Random arguments. actual:%t expect:%t", tt.desc, m.RandomArg1, tt.randomarg1)
			}
			if m.PlayCalled != tt.playcalled {
				t.Errorf("[%s] unexpected Music.Play callcount. actual:%d expect:%d", tt.desc, m.PlayCalled, tt.playcalled)
			}
			if m.PauseCalled != tt.pausecalled {
				t.Errorf("[%s] unexpected Music.Pause callcount. actual:%d expect:%d", tt.desc, m.PauseCalled, tt.pausecalled)
			}
			if m.NextCalled != tt.nextcalled {
				t.Errorf("[%s] unexpected Music.Next callcount. actual:%d expect:%d", tt.desc, m.NextCalled, tt.nextcalled)
			}
			if m.PrevCalled != tt.prevcalled {
				t.Errorf("[%s] unexpected Music.Prev callcount. actual:%d expect:%d", tt.desc, m.PrevCalled, tt.prevcalled)
			}
			if res.StatusCode != 404 {
				defer res.Body.Close()
				b, err := decodeJSONError(res.Body)
				if err != nil {
					t.Errorf("[%s] failed to decode json: %s", tt.desc, err.Error())
				}
				if b.Error != tt.errstr {
					t.Errorf("[%s] unexpected error. actual:%s expect:%s", tt.desc, b.Error, tt.errstr)
				}
			}
		}
	})
}

func TestApiMusicLibrary(t *testing.T) {
	m := new(MockMusic)
	s := Server{Music: m}
	handler := s.makeHandle()
	ts := httptest.NewServer(handler)
	defer ts.Close()
	t.Run("get", func(t *testing.T) {
		lastModified := time.Unix(100, 0)
		m.LibraryRet1 = []Song{Song{"foo": {"bar"}}}
		m.LibraryRet2 = lastModified
		testsets := []struct {
			desc           string
			ret            int
			header         map[string]string
			path           string
			expectSong     Song
			expectSongList []Song
		}{
			{desc: "200 ok", ret: 200, path: "", expectSongList: []Song{Song{"foo": {"bar"}}}},
			{desc: "200 ok", ret: 200, path: "/", expectSongList: []Song{Song{"foo": {"bar"}}}},
			{desc: "200 ok", ret: 200, path: "/0", expectSong: Song{"foo": {"bar"}}},
			{desc: "200 ok not gzip", ret: 200, path: "", header: map[string]string{"Accept-Encoding": "identity"}},
			{desc: "304 not modified", ret: 304, path: "", header: map[string]string{"If-Modified-Since": lastModified.Format(http.TimeFormat)}},
			{desc: "304 not modified", ret: 304, path: "/", header: map[string]string{"If-Modified-Since": lastModified.Format(http.TimeFormat)}},
			{desc: "304 not modified", ret: 304, path: "/0", header: map[string]string{"If-Modified-Since": lastModified.Format(http.TimeFormat)}},
			{desc: "404 not found(out of range)", ret: 404, path: "/1"},
			{desc: "404 not found(not int)", ret: 404, path: "/foobar"},
		}
		for _, tt := range testsets {
			res := testHTTPGet(t, ts.URL+"/api/music/library"+tt.path, tt.header)
			if res.StatusCode != tt.ret {
				t.Errorf("[%s] unexpected status. actual:%d expect:%d", tt.desc, res.StatusCode, tt.ret)
			}
			if tt.expectSong != nil {
				defer res.Body.Close()
				body, actual := decodeJSONSong(res.Body)
				if !reflect.DeepEqual(tt.expectSong, actual.Data) || actual.Error != "" {
					t.Errorf("[%s] unexpected body: %s", tt.desc, body)
				}
			}
			if tt.expectSongList != nil {
				defer res.Body.Close()
				body, actual := decodeJSONSongList(res.Body)
				if !reflect.DeepEqual(tt.expectSongList, actual.Data) || actual.Error != "" {
					t.Errorf("[%s] unexpected body: %s", tt.desc, body)
				}
			}
		}
	})
	t.Run("post", func(t *testing.T) {
		testsets := []struct {
			desc   string
			input  string
			ret    int
			errstr string
		}{
			{desc: "200 ok", ret: 200, input: "{\"action\": \"rescan\"}"},
			{desc: "unknown action 400 bad request", ret: 400, input: "{\"action\": \"unknown\"}", errstr: "unknown action: unknown"},
			{desc: "parse error 400 bad request", ret: 400, input: "action=rescaon", errstr: "failed to get request parameters: invalid character 'a' looking for beginning of value"},
		}
		for _, tt := range testsets {
			j := strings.NewReader(tt.input)
			res, err := http.Post(ts.URL+"/api/music/library", "application/json", j)
			if err != nil {
				t.Errorf("[%s] unexpected error %s", tt.desc, err.Error())
			}
			if res.StatusCode != tt.ret {
				t.Errorf("[%s] unexpected status. actual:%d expect:%d", tt.desc, res.StatusCode, tt.ret)
			}
			if res.StatusCode != 404 {
				defer res.Body.Close()
				b, err := decodeJSONError(res.Body)
				if err != nil {
					t.Errorf("[%s] failed to decode json: %s", tt.desc, err.Error())
				}
				if b.Error != tt.errstr {
					t.Errorf("[%s] unexpected error. actual:%s expect:%s", tt.desc, b.Error, tt.errstr)
				}
			}
		}
	})
}

func TestNotify(t *testing.T) {
	m := new(MockMusic)
	s := Server{Music: m}
	handler := s.makeHandle()
	ts := httptest.NewServer(handler)
	url := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/music/notify"
	defer ts.Close()
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Errorf("failed to connect websocket: %s", url)
		return
	}
	defer c.Close()
	for _, ch := range m.Subscribers {
		ch <- "test"
	}
	_, message, err := c.ReadMessage()
	if string(message) != "test" {
		t.Errorf("unexpected receive message. expect: %s, actual: %s", "test", message)
	}
}

func TestApiMusicOutputs(t *testing.T) {
	m := new(MockMusic)
	s := Server{Music: m}
	handler := s.makeHandle()
	ts := httptest.NewServer(handler)
	url := ts.URL + "/api/music/outputs"
	defer ts.Close()
	t.Run("get", func(t *testing.T) {
		m.OutputsRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.OutputsRet2 = time.Unix(100, 0)
		testsets := []struct {
			desc            string
			ret             int
			ifModifiedSince time.Time
		}{
			{desc: "200 ok", ret: 200},
			{desc: "304 not modified", ret: 304, ifModifiedSince: m.OutputsRet2},
		}
		for _, tt := range testsets {
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("If-Modified-Since", tt.ifModifiedSince.Format(http.TimeFormat))
			client := new(http.Client)
			res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
			if res.StatusCode != tt.ret {
				t.Errorf("[%s] unexpected status. actual:%d expect:%d", tt.desc, res.StatusCode, tt.ret)
			}
			if tt.ret != 200 {
				continue
			}
			defer res.Body.Close()
			body, st := decodeJSONAttrList(res.Body)
			if !reflect.DeepEqual(m.OutputsRet1, st.Data) || st.Error != "" {
				t.Errorf("[%s] unexpected body: %s", tt.desc, body)
			}
		}
	})
	t.Run("post", func(t *testing.T) {
		testsets := []struct {
			desc   string
			addr   string
			input  string
			ret    int
			arg1   int
			arg2   bool
			errstr string
		}{
			{desc: "404 not found", ret: 404, addr: "", input: "{\"outputenabled\": true}"},
			{desc: "400 bad request", ret: 400, addr: "/1", input: "[\"outputenabled\", true]", errstr: "failed to get request parameters: json: cannot unmarshal array into Go value of type struct { OutputEnabled bool \"json:\\\"outputenabled\\\"\" }"},
			{desc: "200 ok", ret: 200, addr: "/1", input: "{\"outputenabled\": true}", arg1: 1, arg2: true},
		}
		for _, tt := range testsets {
			j := strings.NewReader(tt.input)
			res, err := http.Post(url+tt.addr, "application/json", j)
			if err != nil {
				t.Errorf("[%s] unexpected error %s", tt.desc, err.Error())
			}
			if res.StatusCode != tt.ret {
				t.Errorf("[%s] unexpected status. actual:%d expect:%d", tt.desc, res.StatusCode, tt.ret)
			}
			if m.OutputArg1 != tt.arg1 || m.OutputArg2 != tt.arg2 {
				t.Errorf("unexpected arguments. actual:%d expect:%d, actual:%t expect:%t", m.OutputArg1, tt.arg1, m.OutputArg2, tt.arg2)
			}
			if res.StatusCode != 404 {
				defer res.Body.Close()
				b, err := decodeJSONError(res.Body)
				if err != nil {
					t.Errorf("[%s] failed to decode json: %s", tt.desc, err.Error())
				}
				if b.Error != tt.errstr {
					t.Errorf("[%s] unexpected error. actual:%s expect:%s", tt.desc, b.Error, tt.errstr)
				}
			}
		}
	})
}

func TestApiMusicPlaylistOne(t *testing.T) {
	m := new(MockMusic)
	s := Server{Music: m}
	handler := s.makeHandle()
	ts := httptest.NewServer(handler)
	defer ts.Close()
	t.Run("get", func(t *testing.T) {
		lastModified := time.Unix(100, 0)
		m.PlaylistRet1 = []Song{Song{"foo": {"bar"}}}
		m.PlaylistRet2 = lastModified
		m.PlaylistIsSortedRet4 = lastModified
		m.CurrentRet1 = Song{"hoge": {"fuga"}}
		m.CurrentRet2 = lastModified
		testsets := []struct {
			desc            string
			ret             int
			ifModifiedSince time.Time
			path            string
			expectSong      Song
			expectSongList  []Song
		}{
			{desc: "200 ok", ret: 200, path: "", expectSongList: []Song{Song{"foo": {"bar"}}}},
			{desc: "200 ok", ret: 200, path: "/", expectSongList: []Song{Song{"foo": {"bar"}}}},
			{desc: "200 ok", ret: 200, path: "/0", expectSong: Song{"foo": {"bar"}}},
			{desc: "200 ok", ret: 200, path: "/current", expectSong: Song{"hoge": {"fuga"}}},
			{desc: "200 ok", ret: 200, path: "/sort"},
			{desc: "304 not modified", ret: 304, path: "", ifModifiedSince: lastModified},
			{desc: "304 not modified", ret: 304, path: "/", ifModifiedSince: lastModified},
			{desc: "304 not modified", ret: 304, path: "/0", ifModifiedSince: lastModified},
			{desc: "304 not modified", ret: 304, path: "/current", ifModifiedSince: lastModified},
			{desc: "404 not found(out of range)", ret: 404, path: "/1"},
			{desc: "404 not found(not int)", ret: 404, path: "/foobar"},
		}
		for _, tt := range testsets {
			req, _ := http.NewRequest("GET", ts.URL+"/api/music/playlist"+tt.path, nil)
			req.Header.Set("If-Modified-Since", tt.ifModifiedSince.Format(http.TimeFormat))
			client := new(http.Client)
			res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
			if res.StatusCode != tt.ret {
				t.Errorf("[%s] unexpected status. actual:%d expect:%d", tt.desc, res.StatusCode, tt.ret)
			}
			if tt.expectSong != nil {
				defer res.Body.Close()
				body, actual := decodeJSONSong(res.Body)
				if !reflect.DeepEqual(tt.expectSong, actual.Data) || actual.Error != "" {
					t.Errorf("[%s] unexpected body: %s", tt.desc, body)
				}
			}
			if tt.expectSongList != nil {
				defer res.Body.Close()
				body, actual := decodeJSONSongList(res.Body)
				if !reflect.DeepEqual(tt.expectSongList, actual.Data) || actual.Error != "" {
					t.Errorf("[%s] unexpected body: %s", tt.desc, body)
				}
			}
		}
	})
	t.Run("post", func(t *testing.T) {
		m.SortPlaylistErr = nil
		testsets := []struct {
			desc   string
			input  string
			ret    int
			errstr string
		}{
			{desc: "200 ok", ret: 200, input: "{\"action\": \"sort\", \"keys\": [\"file\"], \"pos\": 0, \"filters\": [[\"key\", \"value\"]]}"},
			{desc: "400 missing field", ret: 400, input: "{\"action\": \"sort\", \"pos\": 0, \"filters\": [[\"key\", \"value\"]]}", errstr: "failed to get request parameters. missing fields: keys or/and filters"},
			{desc: "400 json decode failed", ret: 400, input: "{\"value\"]]}", errstr: "failed to get request parameters: invalid character ']' after object key"},
		}
		for _, tt := range testsets {
			j := strings.NewReader(tt.input)
			res, err := http.Post(ts.URL+"/api/music/playlist/sort", "application/json", j)
			if err != nil {
				t.Errorf("[%s] unexpected error %s", tt.desc, err.Error())
			}
			if res.StatusCode != tt.ret {
				t.Errorf("[%s] unexpected status. actual:%d expect:%d", tt.desc, res.StatusCode, tt.ret)
			}
			if res.StatusCode != 404 {
				defer res.Body.Close()
				b, err := decodeJSONError(res.Body)
				if err != nil {
					t.Errorf("[%s] failed to decode json: %s", tt.desc, err.Error())
				}
				if b.Error != tt.errstr {
					t.Errorf("[%s] unexpected error. actual:%s expect:%s", tt.desc, b.Error, tt.errstr)
				}
			}
		}
	})

}

func TestApiMusicStats(t *testing.T) {
	m := new(MockMusic)
	s := Server{Music: m}
	handler := s.makeHandle()
	ts := httptest.NewServer(handler)
	defer ts.Close()
	m.StatsRet1 = mpd.Attrs{"foo": "bar"}
	m.StatsRet2 = time.Unix(60, 0)
	testsets := []struct {
		desc            string
		ret             int
		ifModifiedSince time.Time
		expect          mpd.Attrs
	}{
		{desc: "200 ok", ret: 200, expect: mpd.Attrs{"foo": "bar"}},
		{desc: "304 not modified", ret: 304, ifModifiedSince: time.Unix(60, 0)},
	}
	for _, tt := range testsets {
		req, _ := http.NewRequest("GET", ts.URL+"/api/music/stats", nil)
		req.Header.Set("If-Modified-Since", tt.ifModifiedSince.Format(http.TimeFormat))
		client := new(http.Client)
		res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
		if res.StatusCode != tt.ret {
			t.Errorf("[%s] unexpected status. actual:%d expect:%d", tt.desc, res.StatusCode, tt.ret)
		}
		if tt.expect != nil {
			defer res.Body.Close()
			body, actual := decodeJSONAttr(res.Body)
			if !reflect.DeepEqual(tt.expect, actual.Data) {
				t.Errorf("unexpected body: %s", body)
			}
		}
	}
}

func TestApiVersion(t *testing.T) {
	lastModified := time.Unix(100, 0)
	m := new(MockMusic)
	var testsets = []struct {
		desc            string
		ifModifiedSince time.Time
		version         string
		debug           bool
		ret             int
		vvVersion       string
	}{
		{desc: "return staticVersion if vvVersion is empty", debug: false, ret: 200, vvVersion: staticVersion},
		{desc: "return vvVersion if vvVersion is not empty", debug: false, ret: 200, version: "v0.0.0", vvVersion: "v0.0.0"},
		{desc: "return version with 'dev mode' if debug is true", debug: true, ret: 200, vvVersion: staticVersion + " dev mode"},
		{desc: "304 not modified if Last-Modified == If-Modified-Since", ifModifiedSince: lastModified, ret: 304},
	}
	for _, tt := range testsets {
		version = tt.version
		s := Server{Music: m, StartTime: lastModified, debug: tt.debug}
		handler := s.makeHandle()
		ts := httptest.NewServer(handler)
		defer ts.Close()
		req, _ := http.NewRequest("GET", ts.URL+"/api/version", nil)
		req.Header.Set("If-Modified-Since", tt.ifModifiedSince.Format(http.TimeFormat))
		client := new(http.Client)
		res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
		if res.StatusCode != tt.ret {
			t.Errorf("[%s] unexpected status %d", tt.desc, res.StatusCode)
		}
		if tt.ret != 200 {
			continue
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data  map[string]string `json:"data"`
			Error string            `json:"error"`
		}{map[string]string{}, ""}
		json.Unmarshal(body, &st)
		actual := st.Data["vv"]
		if actual != tt.vvVersion {
			t.Errorf("[%s] unexpected vv version, actual: %s expect: %s", tt.desc, actual, tt.vvVersion)
		}
		actual = st.Data["go"]
		expect := fmt.Sprintf("%s %s %s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		if actual != expect {
			t.Errorf("[%s] unexpected go version, actual: %s expect: %s", tt.desc, actual, expect)
		}

	}
}

func TestAssets(t *testing.T) {
	assets := []string{"/", "/assets/app.css", "/assets/app.js"}
	testsets := []struct {
		desc   string
		debug  bool
		header map[string]string
	}{
		{desc: "use bindata", debug: false},
		{desc: "use bindata, nogzip", debug: false, header: map[string]string{"Accept-Encoding": "identity"}},
		{desc: "use local file", debug: true},
	}
	for _, tt := range testsets {
		m := new(MockMusic)
		s := Server{Music: m, debug: tt.debug}
		handler := s.makeHandle()
		ts := httptest.NewServer(handler)
		defer ts.Close()
		for i := range assets {
			res := testHTTPGet(t, ts.URL+assets[i], tt.header)
			if res.StatusCode != 200 {
				t.Errorf("[%s] unexpected status for \"%s\", %d", tt.desc, assets[i], res.StatusCode)
			}
		}
	}
}

func TestMusicDirectory(t *testing.T) {
	m := new(MockMusic)
	s := Server{Music: m, MusicDirectory: "./"}
	handler := s.makeHandle()
	ts := httptest.NewServer(handler)
	defer ts.Close()
	res := checkRequestError(t, func() (*http.Response, error) { return http.Get(ts.URL + "/music_directory/server_test.go") })
	if res.StatusCode != 200 {
		t.Errorf("unexpected status %d", res.StatusCode)
	}
}

func TestRoot(t *testing.T) {
	testsets := []struct {
		desc      string
		status    int
		debug     bool
		addr      string
		reqHeader map[string]string
		resHeader map[string][]string
	}{
		{
			desc:      "use bindata",
			status:    200,
			reqHeader: map[string]string{"Accept-Encoding": "gzip"},
			resHeader: map[string][]string{
				"Vary":             {"Accept-Encoding, Accept-Language"},
				"Content-Language": {"en-US"},
				"Content-Length":   nil,
				"Content-Type":     {"text/html; charset=utf-8"},
				"Content-Encoding": {"gzip"},
				"Cache-Control":    {"max-age=86400"},
				"Last-Modified":    nil,
			},
		},
		{
			desc:      "use bindata, nogzip",
			status:    200,
			reqHeader: map[string]string{"Accept-Encoding": "identity"},
			resHeader: map[string][]string{
				"Vary":             {"Accept-Encoding, Accept-Language"},
				"Content-Language": {"en-US"},
				"Content-Length":   nil,
				"Content-Type":     {"text/html; charset=utf-8"},
				"Cache-Control":    {"max-age=86400"},
				"Last-Modified":    nil,
			},
		},
		{
			desc:      "use bindata, If-Modified-Since",
			status:    304,
			reqHeader: map[string]string{"Accept-Encoding": "gzip", "If-Modified-Since": mustAssetInfo("assets/app.html").ModTime().Format(http.TimeFormat)},
		},
		{
			desc:      "use bindata, lang ja",
			status:    200,
			reqHeader: map[string]string{"Accept-Encoding": "gzip", "Accept-Language": "ja,en-US;q=0.9,en;q=0.8"},
			resHeader: map[string][]string{
				"Vary":             {"Accept-Encoding, Accept-Language"},
				"Content-Language": {"ja"},
				"Content-Length":   nil,
				"Content-Type":     {"text/html; charset=utf-8"},
				"Content-Encoding": {"gzip"},
				"Cache-Control":    {"max-age=86400"},
				"Last-Modified":    nil,
			},
		},
		{
			desc:      "use bindata, address lang ja",
			addr:      "ja/",
			status:    200,
			reqHeader: map[string]string{"Accept-Encoding": "gzip"},
			resHeader: map[string][]string{
				"Vary":             {"Accept-Encoding"},
				"Content-Language": {"ja"},
				"Content-Length":   nil,
				"Content-Type":     {"text/html; charset=utf-8"},
				"Content-Encoding": {"gzip"},
				"Cache-Control":    {"max-age=86400"},
				"Last-Modified":    nil,
			},
		},
		{
			desc:      "use bindata, address unknown lang",
			addr:      "foobar/",
			status:    404,
			reqHeader: map[string]string{"Accept-Encoding": "gzip"},
		},
		{
			desc:      "use local file",
			status:    200,
			reqHeader: map[string]string{"Accept-Encoding": "identity"},
			resHeader: map[string][]string{
				"Vary":             {"Accept-Encoding, Accept-Language"},
				"Content-Language": {"en-US"},
				"Content-Length":   nil,
				"Content-Type":     {"text/html; charset=utf-8"},
				"Last-Modified":    nil,
			},
			debug: true,
		},
		{
			desc:      "use local file, If-Modified-Since",
			status:    304,
			reqHeader: map[string]string{"Accept-Encoding": "identity", "If-Modified-Since": time.Now().Format(http.TimeFormat)},
			debug:     true,
		},
	}
	for _, tt := range testsets {
		m := new(MockMusic)
		s := Server{Music: m, debug: tt.debug}
		handler := s.makeHandle()
		ts := httptest.NewServer(handler)
		defer ts.Close()
		res := testHTTPGet(t, ts.URL+"/"+tt.addr, tt.reqHeader)
		if res.StatusCode != tt.status {
			t.Errorf("[%s] unexpected status: %d expect: %d", tt.desc, res.StatusCode, tt.status)
		}
		if res.ContentLength < 0 {
			t.Errorf("[%s] response header does not contain: Content-Length", tt.desc)
		}
		if tt.resHeader != nil {
			for key, value := range tt.resHeader {
				actual, ok := res.Header[key]
				if !ok {
					t.Errorf("[%s] response header does not contain: %s", tt.desc, key)
					continue
				}
				if value != nil && !reflect.DeepEqual(value, actual) {
					t.Errorf("[%s] response header %s has: %s, expect: %s", tt.desc, key, actual, value)
				}

			}
		}

	}
}

type MockMusic struct {
	PlayErr              error
	PlayCalled           int
	PauseErr             error
	PauseCalled          int
	NextErr              error
	NextCalled           int
	PrevErr              error
	PrevCalled           int
	VolumeArg1           int
	VolumeErr            error
	RepeatArg1           bool
	RepeatErr            error
	SingleArg1           bool
	SingleErr            error
	RandomArg1           bool
	RandomErr            error
	PlaylistRet1         []Song
	PlaylistRet2         time.Time
	LibraryRet1          []Song
	LibraryRet2          time.Time
	RescanLibraryRet1    error
	OutputsRet1          []mpd.Attrs
	OutputsRet2          time.Time
	OutputArg1           int
	OutputArg2           bool
	OutputRet1           error
	CurrentRet1          Song
	CurrentRet2          time.Time
	CommentsRet1         mpd.Attrs
	CommentsRet2         time.Time
	StatusRet1           Status
	StatusRet2           time.Time
	StatsRet1            mpd.Attrs
	StatsRet2            time.Time
	SortPlaylistArg1     []string
	SortPlaylistArg2     [][]string
	SortPlaylistArg3     int
	SortPlaylistErr      error
	PlaylistIsSortedRet1 bool
	PlaylistIsSortedRet2 []string
	PlaylistIsSortedRet3 [][]string
	PlaylistIsSortedRet4 time.Time
	Subscribers          []chan string
}

func (p *MockMusic) Play() error {
	// TODO: lock
	p.PlayCalled++
	return p.PlayErr
}

func (p *MockMusic) Pause() error {
	p.PauseCalled++
	return p.PauseErr
}
func (p *MockMusic) Next() error {
	p.NextCalled++
	return p.NextErr
}
func (p *MockMusic) Prev() error {
	p.PrevCalled++
	return p.PrevErr
}
func (p *MockMusic) Volume(i int) error {
	p.VolumeArg1 = i
	return p.VolumeErr
}
func (p *MockMusic) Repeat(b bool) error {
	p.RepeatArg1 = b
	return p.RepeatErr
}
func (p *MockMusic) Single(b bool) error {
	p.SingleArg1 = b
	return p.SingleErr
}
func (p *MockMusic) Random(b bool) error {
	p.RandomArg1 = b
	return p.RandomErr
}
func (p *MockMusic) Comments() (mpd.Attrs, time.Time) {
	return p.CommentsRet1, p.CommentsRet2
}
func (p *MockMusic) Current() (Song, time.Time) {
	return p.CurrentRet1, p.CurrentRet2
}
func (p *MockMusic) Library() ([]Song, time.Time) {
	return p.LibraryRet1, p.LibraryRet2
}
func (p *MockMusic) RescanLibrary() error {
	return p.RescanLibraryRet1
}
func (p *MockMusic) Outputs() ([]mpd.Attrs, time.Time) {
	return p.OutputsRet1, p.OutputsRet2
}
func (p *MockMusic) Output(id int, on bool) error {
	p.OutputArg1, p.OutputArg2 = id, on
	return p.OutputRet1
}
func (p *MockMusic) Playlist() ([]Song, time.Time) {
	return p.PlaylistRet1, p.PlaylistRet2
}
func (p *MockMusic) Status() (Status, time.Time) {
	return p.StatusRet1, p.StatusRet2
}
func (p *MockMusic) Stats() (mpd.Attrs, time.Time) {
	return p.StatsRet1, p.StatsRet2
}
func (p *MockMusic) SortPlaylist(s []string, t [][]string, u int) error {
	p.SortPlaylistArg1 = s
	p.SortPlaylistArg2 = t
	p.SortPlaylistArg3 = u
	return p.SortPlaylistErr
}
func (p *MockMusic) PlaylistIsSorted() (bool, []string, [][]string, time.Time) {
	return p.PlaylistIsSortedRet1, p.PlaylistIsSortedRet2, p.PlaylistIsSortedRet3, p.PlaylistIsSortedRet4
}
func (p *MockMusic) Subscribe(s chan string) {
	p.Subscribers = []chan string{s}
}
func (p *MockMusic) Unsubscribe(s chan string) {
	p.Subscribers = []chan string{}
}

func checkRequestError(t *testing.T, f func() (*http.Response, error)) *http.Response {
	r, err := f()
	if err != nil {
		t.Fatalf("failed to request: %s", err.Error())
	}
	return r
}

type jsonAttr struct {
	Data  mpd.Attrs `json:"data"`
	Error string    `json:"error"`
}

func decodeJSONAttr(b io.Reader) (body []byte, st jsonAttr) {
	body, _ = ioutil.ReadAll(b)
	st = jsonAttr{mpd.Attrs{}, ""}
	json.Unmarshal(body, &st)
	return
}

type jsonAttrList struct {
	Data  []mpd.Attrs `json:"data"`
	Error string      `json:"error"`
}

func decodeJSONAttrList(b io.Reader) (body []byte, st jsonAttrList) {
	body, _ = ioutil.ReadAll(b)
	st = jsonAttrList{[]mpd.Attrs{}, ""}
	json.Unmarshal(body, &st)
	return
}

type jsonError struct {
	Error string `json:"error"`
}

func decodeJSONError(b io.Reader) (jsonError, error) {
	d := jsonError{}
	body, err := ioutil.ReadAll(b)
	if err != nil {
		return d, err
	}
	err = json.Unmarshal(body, &d)
	if err != nil {
		return d, err
	}
	return d, nil
}

type jsonSong struct {
	Data  Song   `json:"data"`
	Error string `json:"error"`
}

func decodeJSONSong(b io.Reader) (body []byte, st jsonSong) {
	body, _ = ioutil.ReadAll(b)
	st = jsonSong{Song{}, ""}
	json.Unmarshal(body, &st)
	return
}

type jsonSongList struct {
	Data  []Song `json:"data"`
	Error string `json:"error"`
}

func decodeJSONSongList(b io.Reader) (body []byte, st jsonSongList) {
	body, _ = ioutil.ReadAll(b)
	st = jsonSongList{[]Song{}, ""}
	json.Unmarshal(body, &st)
	return
}

func testHTTPGet(t *testing.T, url string, header map[string]string) (res *http.Response) {
	t.Helper()
	var err error
	if header == nil {
		res, err = http.Get(url)
	} else {
		req, _ := http.NewRequest("GET", url, nil)
		for k, v := range header {
			req.Header.Set(k, v)
		}
		client := new(http.Client)
		res, err = client.Do(req)
	}
	if err != nil {
		t.Fatalf("failed to request %s", url)
	}
	return res
}
