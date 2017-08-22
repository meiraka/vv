package main

import (
	"encoding/json"
	"github.com/fhs/gompd/mpd"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

var assets = []string{"/", "/assets/app.css", "/assets/app.js"}

func TestAssets(t *testing.T) {
	m := new(MockMusic)
	handler := makeHandle(m, Config{}, false)
	ts := httptest.NewServer(handler)
	defer ts.Close()
	for i := range assets {
		res := checkRequestError(t, func() (*http.Response, error) { return http.Get(ts.URL + assets[i]) })
		if res.StatusCode != 200 {
			t.Errorf("unexpected status for \"%s\", %d", assets[i], res.StatusCode)
		}
	}
}

func TestAssetsBinary(t *testing.T) {
	m := new(MockMusic)
	handler := makeHandle(m, Config{}, true)
	ts := httptest.NewServer(handler)
	defer ts.Close()
	for i := range assets {
		res := checkRequestError(t, func() (*http.Response, error) { return http.Get(ts.URL + assets[i]) })
		if res.StatusCode != 200 {
			t.Errorf("unexpected status for \"%s\", %d", assets[i], res.StatusCode)
		}
	}
}

func TestMusicDirectory(t *testing.T) {
	m := new(MockMusic)
	handler := makeHandle(m, Config{Mpd: MpdConfig{MusicDirectory: "./"}}, false)
	ts := httptest.NewServer(handler)
	defer ts.Close()
	res := checkRequestError(t, func() (*http.Response, error) { return http.Get(ts.URL + "/music_directory/server_test.go") })
	if res.StatusCode != 200 {
		t.Errorf("unexpected status %d", res.StatusCode)
	}
}

func TestVersion(t *testing.T) {
	m := new(MockMusic)
	var testsets = []struct {
		lastModified    time.Time
		ifModifiedSince time.Time
		bindata         bool
		ret             int
		vvVersion       string
	}{
		{bindata: true, ret: 200, vvVersion: version},
		{bindata: false, ret: 200, vvVersion: version + " dev mode"},
		{lastModified: time.Unix(100, 0), ifModifiedSince: time.Unix(100, 0), ret: 304},
	}
	for _, tt := range testsets {
		startTime = tt.lastModified
		handler := makeHandle(m, Config{}, tt.bindata)
		ts := httptest.NewServer(handler)
		defer ts.Close()
		req, _ := http.NewRequest("GET", ts.URL+"/api/version", nil)
		if !tt.ifModifiedSince.IsZero() {
			req.Header.Set("If-Modified-Since", tt.ifModifiedSince.Format(http.TimeFormat))
		}
		client := new(http.Client)
		res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
		if res.StatusCode != tt.ret {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		if len(tt.vvVersion) > 0 {
			defer res.Body.Close()
			body, _ := ioutil.ReadAll(res.Body)
			st := struct {
				Data  map[string]string `json:"data"`
				Error string            `json:"error"`
			}{map[string]string{}, ""}
			json.Unmarshal(body, &st)
			actual := st.Data["vv"]
			if actual != tt.vvVersion {
				t.Errorf("unexpected vv version, actual: %s expect: %s", actual, tt.vvVersion)
			}
		}
	}
}

func TestLibrary(t *testing.T) {
	m := new(MockMusic)
	handler := makeHandle(m, Config{}, false)
	ts := httptest.NewServer(handler)
	url := ts.URL + "/api/music/library"
	defer ts.Close()
	t.Run("no parameter", func(t *testing.T) {
		m.LibraryRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.LibraryRet2 = time.Unix(0, 0)
		res := checkRequestError(t, func() (*http.Response, error) { return http.Get(url) })
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data  []mpd.Attrs `json:"data"`
			Error string      `json:"error"`
		}{[]mpd.Attrs{}, ""}
		json.Unmarshal(body, &st)
		if !reflect.DeepEqual(m.LibraryRet1, st.Data) {
			t.Errorf("unexpected body: %s", body)
		}
		if st.Error != "" {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		m.LibraryRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.LibraryRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", ts.URL+"/api/music/library", nil)
		req.Header.Set("If-Modified-Since", m.LibraryRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("rescan", func(t *testing.T) {
		m.SortPlaylistErr = nil
		j := strings.NewReader(
			"{\"action\": \"rescan\"}",
		)
		res, err := http.Post(url, "application/json", j)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		b, err := decodeJSONError(res.Body)
		if res.StatusCode != 200 || err != nil || b.Error != "" {
			t.Errorf("unexpected response")
		}
	})
}
func TestLibraryOne(t *testing.T) {
	m := new(MockMusic)
	handler := makeHandle(m, Config{}, false)
	ts := httptest.NewServer(handler)
	url := ts.URL + "/api/music/library"
	defer ts.Close()
	t.Run("not found", func(t *testing.T) {
		m.LibraryRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.LibraryRet2 = time.Unix(0, 0)
		res := checkRequestError(t, func() (*http.Response, error) { return http.Get(url + "/hoge") })
		if res.StatusCode != 404 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("not found(out of range)", func(t *testing.T) {
		m.LibraryRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.LibraryRet2 = time.Unix(0, 0)
		res := checkRequestError(t, func() (*http.Response, error) { return http.Get(url + "/1") })
		if res.StatusCode != 404 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("ok", func(t *testing.T) {
		m.LibraryRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.LibraryRet2 = time.Unix(0, 0)
		res := checkRequestError(t, func() (*http.Response, error) { return http.Get(url + "/0") })
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		m.LibraryRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.LibraryRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", ts.URL+"/api/music/library/0", nil)
		req.Header.Set("If-Modified-Since", m.LibraryRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
}
func TestPlaylist(t *testing.T) {
	m := new(MockMusic)
	handler := makeHandle(m, Config{}, false)
	ts := httptest.NewServer(handler)
	url := ts.URL + "/api/music/songs"
	defer ts.Close()
	t.Run("no parameter", func(t *testing.T) {
		m.PlaylistRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.PlaylistRet2 = time.Unix(0, 0)
		res := checkRequestError(t, func() (*http.Response, error) { return http.Get(url) })
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data  []mpd.Attrs `json:"data"`
			Error string      `json:"error"`
		}{[]mpd.Attrs{}, ""}
		json.Unmarshal(body, &st)
		if !reflect.DeepEqual(m.PlaylistRet1, st.Data) {
			t.Errorf("unexpected body: %s", body)
		}
		if st.Error != "" {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		m.PlaylistRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.PlaylistRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("If-Modified-Since", m.PlaylistRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("sort", func(t *testing.T) {
		m.SortPlaylistErr = nil
		j := strings.NewReader(
			"{\"action\": \"sort\", \"keys\": [\"file\"], \"uri\": \"path\"}",
		)
		res, err := http.Post(url, "application/json", j)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		b, err := decodeJSONError(res.Body)
		if res.StatusCode != 200 || err != nil || b.Error != "" {
			t.Errorf("unexpected response")
		}
	})
}

func TestPlaylistOne(t *testing.T) {
	m := new(MockMusic)
	handler := makeHandle(m, Config{}, false)
	ts := httptest.NewServer(handler)
	url := ts.URL + "/api/music/songs"
	defer ts.Close()
	t.Run("not found", func(t *testing.T) {
		m.PlaylistRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.PlaylistRet2 = time.Unix(0, 0)
		res := checkRequestError(t, func() (*http.Response, error) { return http.Get(url + "/hoge") })
		if res.StatusCode != 404 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("not found(out of range)", func(t *testing.T) {
		m.PlaylistRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.PlaylistRet2 = time.Unix(0, 0)
		res := checkRequestError(t, func() (*http.Response, error) { return http.Get(url + "/1") })
		if res.StatusCode != 404 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("ok", func(t *testing.T) {
		m.PlaylistRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.PlaylistRet2 = time.Unix(0, 0)
		res := checkRequestError(t, func() (*http.Response, error) { return http.Get(url + "/0") })
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		m.PlaylistRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.PlaylistRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", url+"/0", nil)
		req.Header.Set("If-Modified-Since", m.PlaylistRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
}
func TestOutput(t *testing.T) {
	m := new(MockMusic)
	handler := makeHandle(m, Config{}, false)
	ts := httptest.NewServer(handler)
	url := ts.URL + "/api/music/outputs"
	defer ts.Close()
	t.Run("no parameter", func(t *testing.T) {
		m.OutputsRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.OutputsRet2 = time.Unix(0, 0)
		res := checkRequestError(t, func() (*http.Response, error) { return http.Get(url) })
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data  []mpd.Attrs `json:"data"`
			Error string      `json:"error"`
		}{[]mpd.Attrs{}, ""}
		json.Unmarshal(body, &st)
		if !reflect.DeepEqual(m.OutputsRet1, st.Data) {
			t.Errorf("unexpected body: %s", body)
		}
		if st.Error != "" {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		m.OutputsRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.OutputsRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("If-Modified-Since", m.OutputsRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("enable", func(t *testing.T) {
		j := strings.NewReader(
			"{\"outputenabled\": true}",
		)
		res, err := http.Post(url+"/1", "application/json", j)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		if m.OutputArg1 != 1 || m.OutputArg2 != true {
			t.Errorf("unexpected arguments: %d, %t", m.OutputArg1, m.OutputArg2)
		}
		defer res.Body.Close()
		b, err := decodeJSONError(res.Body)
		if res.StatusCode != 200 || err != nil || b.Error != "" {
			t.Errorf("unexpected response")
		}
	})
}
func TestCurrent(t *testing.T) {
	m := new(MockMusic)
	handler := makeHandle(m, Config{}, false)
	ts := httptest.NewServer(handler)
	url := ts.URL + "/api/music/songs/current"
	defer ts.Close()
	t.Run("no parameter", func(t *testing.T) {
		m.CurrentRet1 = mpd.Attrs{"foo": "bar"}
		m.CurrentRet2 = time.Unix(0, 0)
		res := checkRequestError(t, func() (*http.Response, error) { return http.Get(url) })
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data  mpd.Attrs `json:"data"`
			Error string    `json:"error"`
		}{mpd.Attrs{}, ""}
		json.Unmarshal(body, &st)
		if !reflect.DeepEqual(m.CurrentRet1, st.Data) {
			t.Errorf("unexpected body: %s", body)
		}
		if st.Error != "" {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		m.CurrentRet1 = mpd.Attrs{"foo": "bar"}
		m.CurrentRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("If-Modified-Since", m.CurrentRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
}
func TestStats(t *testing.T) {
	m := new(MockMusic)
	handler := makeHandle(m, Config{}, false)
	ts := httptest.NewServer(handler)
	url := ts.URL + "/api/music/stats"
	defer ts.Close()
	m.StatsRet1 = mpd.Attrs{"foo": "bar"}
	m.StatsRet2 = time.Unix(60, 0)
	res := checkRequestError(t, func() (*http.Response, error) { return http.Get(url) })
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	st := struct {
		Data  mpd.Attrs `json:"data"`
		Error string    `json:"error"`
	}{mpd.Attrs{}, ""}
	json.Unmarshal(body, &st)
	if !reflect.DeepEqual(m.StatsRet1, st.Data) {
		t.Errorf("unexpected body: %s", body)
	}
}
func TestControl(t *testing.T) {
	m := new(MockMusic)
	handler := makeHandle(m, Config{}, false)
	ts := httptest.NewServer(handler)
	url := ts.URL + "/api/music/control"
	defer ts.Close()
	t.Run("no parameter", func(t *testing.T) {
		s := convStatus(mpd.Attrs{}, 0)
		m.StatusRet1 = s
		m.StatusRet2 = time.Unix(0, 0)
		res := checkRequestError(t, func() (*http.Response, error) { return http.Get(url) })
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data  PlayerStatus `json:"data"`
			Error string       `json:"error"`
		}{PlayerStatus{}, ""}
		json.Unmarshal(body, &st)
		if !reflect.DeepEqual(s, st.Data) {
			t.Errorf("unexpected body: %s", body)
		}
		if st.Error != "" {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		s := convStatus(mpd.Attrs{}, 60)
		m.StatusRet1 = s
		m.StatusRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("If-Modified-Since", m.StatusRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res := checkRequestError(t, func() (*http.Response, error) { return client.Do(req) })
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("volume", func(t *testing.T) {
		j := strings.NewReader(
			"{\"volume\": 1}",
		)
		res, err := http.Post(url, "application/json", j)
		if err != nil {
			t.Errorf("unexpected request error: %s", err.Error())
			return
		}
		if m.VolumeArg1 != 1 {
			t.Errorf("unexpected arguments: %d", m.VolumeArg1)
		}
		defer res.Body.Close()
		b, err := decodeJSONError(res.Body)
		if res.StatusCode != 200 || err != nil || b.Error != "" {
			t.Errorf("unexpected response")
		}
	})
	t.Run("repeat", func(t *testing.T) {
		j := strings.NewReader(
			"{\"repeat\": true}",
		)
		res, err := http.Post(url, "application/json", j)
		if err != nil {
			t.Errorf("unexpected request error: %s", err.Error())
			return
		}
		if m.RepeatArg1 != true {
			t.Errorf("unexpected arguments: %t", m.RepeatArg1)
		}
		defer res.Body.Close()
		b, err := decodeJSONError(res.Body)
		if res.StatusCode != 200 || err != nil || b.Error != "" {
			t.Errorf("unexpected response")
		}
	})
	t.Run("random", func(t *testing.T) {
		j := strings.NewReader(
			"{\"random\": true}",
		)
		res, err := http.Post(url, "application/json", j)
		if err != nil {
			t.Errorf("unexpected request error: %s", err.Error())
			return
		}
		if m.RandomArg1 != true {
			t.Errorf("unexpected arguments: %t", m.RandomArg1)
		}
		defer res.Body.Close()
		b, err := decodeJSONError(res.Body)
		if res.StatusCode != 200 || err != nil || b.Error != "" {
			t.Errorf("unexpected response")
		}
	})
	t.Run("state", func(t *testing.T) {
		candidates := []struct {
			input string
		}{
			{
				"{\"state\": \"play\"}",
			},
			{
				"{\"state\": \"pause\"}",
			},
			{
				"{\"state\": \"next\"}",
			},
			{
				"{\"state\": \"prev\"}",
			},
		}
		for _, c := range candidates {
			j := strings.NewReader(c.input)
			res, err := http.Post(url, "application/json", j)
			if err != nil {
				t.Errorf("unexpected request error: %s", err.Error())
				return
			}
			defer res.Body.Close()
			b, err := decodeJSONError(res.Body)
			if res.StatusCode != 200 || err != nil || b.Error != "" {
				t.Errorf("unexpected response")
			}
		}
		if m.PlayCalled != 1 || m.PauseCalled != 1 || m.NextCalled != 1 || m.PrevCalled != 1 {
			t.Errorf("unexpected function call")
		}
	})
	t.Run("state unknown", func(t *testing.T) {
		j := strings.NewReader("{\"state\": \"unknown\"}")
		res, err := http.Post(url, "application/json", j)
		if err != nil {
			t.Errorf("unexpected request error: %s", err.Error())
			return
		}
		defer res.Body.Close()
		b, err := decodeJSONError(res.Body)
		if res.StatusCode != 200 || err != nil || b.Error != "unknown state value: unknown" {
			t.Errorf("unexpected response")
		}

	})
}

func TestNotify(t *testing.T) {
	m := new(MockMusic)
	handler := makeHandle(m, Config{}, false)
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

func checkRequestError(t *testing.T, f func() (*http.Response, error)) *http.Response {
	r, err := f()
	if err != nil {
		t.Fatalf("failed to request: %s", err.Error())
	}
	return r
}

type MockMusic struct {
	PlayErr           error
	PlayCalled        int
	PauseErr          error
	PauseCalled       int
	NextErr           error
	NextCalled        int
	PrevErr           error
	PrevCalled        int
	VolumeArg1        int
	VolumeErr         error
	RepeatArg1        bool
	RepeatErr         error
	RandomArg1        bool
	RandomErr         error
	PlaylistRet1      []mpd.Attrs
	PlaylistRet2      time.Time
	LibraryRet1       []mpd.Attrs
	LibraryRet2       time.Time
	RescanLibraryRet1 error
	OutputsRet1       []mpd.Attrs
	OutputsRet2       time.Time
	OutputArg1        int
	OutputArg2        bool
	OutputRet1        error
	CurrentRet1       mpd.Attrs
	CurrentRet2       time.Time
	CommentsRet1      mpd.Attrs
	CommentsRet2      time.Time
	StatusRet1        PlayerStatus
	StatusRet2        time.Time
	StatsRet1         mpd.Attrs
	StatsRet2         time.Time
	SortPlaylistArg1  []string
	SortPlaylistArg2  string
	SortPlaylistErr   error
	Subscribers       []chan string
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
func (p *MockMusic) Random(b bool) error {
	p.RandomArg1 = b
	return p.RandomErr
}
func (p *MockMusic) Comments() (mpd.Attrs, time.Time) {
	return p.CommentsRet1, p.CommentsRet2
}
func (p *MockMusic) Current() (mpd.Attrs, time.Time) {
	return p.CurrentRet1, p.CurrentRet2
}
func (p *MockMusic) Library() ([]mpd.Attrs, time.Time) {
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
func (p *MockMusic) Playlist() ([]mpd.Attrs, time.Time) {
	return p.PlaylistRet1, p.PlaylistRet2
}
func (p *MockMusic) Status() (PlayerStatus, time.Time) {
	return p.StatusRet1, p.StatusRet2
}
func (p *MockMusic) Stats() (mpd.Attrs, time.Time) {
	return p.StatsRet1, p.StatsRet2
}
func (p *MockMusic) SortPlaylist(s []string, u string) error {
	p.SortPlaylistArg1 = s
	p.SortPlaylistArg2 = u
	return p.SortPlaylistErr
}

func (p *MockMusic) Subscribe(s chan string) {
	p.Subscribers = []chan string{s}
}

func (p *MockMusic) Unsubscribe(s chan string) {
	p.Subscribers = []chan string{}
}
