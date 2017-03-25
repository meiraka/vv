package main

import (
	"encoding/json"
	"github.com/fhs/gompd/mpd"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestLibrary(t *testing.T) {
	m := new(MockMusic)
	api := apiHandler{m}
	ts := httptest.NewServer(http.HandlerFunc(api.library))
	defer ts.Close()
	t.Run("no parameter", func(t *testing.T) {
		m.LibraryRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.LibraryRet2 = time.Unix(0, 0)
		res, err := http.Get(ts.URL)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data   []mpd.Attrs `json:"data"`
			Errors error       `json:"errors"`
		}{[]mpd.Attrs{}, nil}
		json.Unmarshal(body, &st)
		if !reflect.DeepEqual(m.LibraryRet1, st.Data) {
			t.Errorf("unexpected body: %s", body)
		}
		if st.Errors != nil {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		m.LibraryRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.LibraryRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", ts.URL, nil)
		req.Header.Set("If-Modified-Since", m.LibraryRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res, err := client.Do(req)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
}
func TestPlaylist(t *testing.T) {
	m := new(MockMusic)
	api := apiHandler{m}
	ts := httptest.NewServer(http.HandlerFunc(api.playlist))
	defer ts.Close()
	t.Run("no parameter", func(t *testing.T) {
		m.PlaylistRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.PlaylistRet2 = time.Unix(0, 0)
		res, err := http.Get(ts.URL)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data   []mpd.Attrs `json:"data"`
			Errors error       `json:"errors"`
		}{[]mpd.Attrs{}, nil}
		json.Unmarshal(body, &st)
		if !reflect.DeepEqual(m.PlaylistRet1, st.Data) {
			t.Errorf("unexpected body: %s", body)
		}
		if st.Errors != nil {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		m.PlaylistRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.PlaylistRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", ts.URL, nil)
		req.Header.Set("If-Modified-Since", m.PlaylistRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res, err := client.Do(req)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
}
func TestControl(t *testing.T) {
	m := new(MockMusic)
	api := apiHandler{m}
	ts := httptest.NewServer(http.HandlerFunc(api.control))
	defer ts.Close()
	t.Run("no parameter", func(t *testing.T) {
		s := convStatus(mpd.Attrs{}, mpd.Attrs{})
		s.LastModified = 0
		m.StatusRet1 = s
		m.StatusRet2 = time.Unix(0, 0)
		res, err := http.Get(ts.URL)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data   PlayerStatus `json:"data"`
			Errors error        `json:"errors"`
		}{PlayerStatus{}, nil}
		json.Unmarshal(body, &st)
		if !reflect.DeepEqual(s, st.Data) {
			t.Errorf("unexpected body: %s", body)
		}
		if st.Errors != nil {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		s := convStatus(mpd.Attrs{}, mpd.Attrs{})
		s.LastModified = 60
		m.StatusRet1 = s
		m.StatusRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", ts.URL, nil)
		req.Header.Set("If-Modified-Since", m.StatusRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res, err := client.Do(req)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("action=play", func(t *testing.T) {
		res, err := http.Get(ts.URL + "?action=play")
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})

	t.Run("action=pause", func(t *testing.T) {
		res, err := http.Get(ts.URL + "?action=pause")
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})

	t.Run("action=next", func(t *testing.T) {
		res, err := http.Get(ts.URL + "?action=next")
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})

	t.Run("action=prev", func(t *testing.T) {
		res, err := http.Get(ts.URL + "?action=prev")
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})

}

type MockMusic struct {
	PlayErr         error
	PauseErr        error
	NextErr         error
	PrevErr         error
	PlaylistRet1    []mpd.Attrs
	PlaylistRet2    time.Time
	LibraryRet1     []mpd.Attrs
	LibraryRet2     time.Time
	CurrentRet1     mpd.Attrs
	CurrentRet2     time.Time
	CommentsRet1    mpd.Attrs
	CommentsRet2    time.Time
	StatusRet1      PlayerStatus
	StatusRet2      time.Time
	SortPlaylistErr error
}

func (p *MockMusic) Play() error {
	return p.PlayErr
}

func (p *MockMusic) Pause() error {
	return p.PauseErr
}
func (p *MockMusic) Next() error {
	return p.NextErr
}
func (p *MockMusic) Prev() error {
	return p.PrevErr
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
func (p *MockMusic) Playlist() ([]mpd.Attrs, time.Time) {
	return p.PlaylistRet1, p.PlaylistRet2
}
func (p *MockMusic) Status() (PlayerStatus, time.Time) {
	return p.StatusRet1, p.StatusRet2
}
func (p *MockMusic) SortPlaylist([]string, string) error {
	return p.SortPlaylistErr
}
