package main

import (
	"encoding/json"
	"github.com/fhs/gompd/mpd"
	"reflect"
	"sync"
	"testing"
)

func TestPlayerPlay(t *testing.T) {
	p, m := mockDial("tcp", "localhost:6600")
	m.err = new(mockError)
	err := p.Play()
	if m.playcalled != 1 {
		t.Errorf("Client.Play does not called")
	}
	if err != m.err {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.playarg1 != -1 {
		t.Errorf("unexpected Client.Play arguments: %d", m.playarg1)
	}

	m.err = nil
	err = p.Play()

	if m.playcalled != 2 {
		t.Errorf("Client.Play does not called")
	}
	if err != m.err {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.playarg1 != -1 {
		t.Errorf("unexpected Client.Play arguments: %d", m.playarg1)
	}
}

func TestPlayerPause(t *testing.T) {
	p, m := mockDial("tcp", "localhost:6600")
	m.err = new(mockError)
	err := p.Pause()
	if m.pausecalled != 1 {
		t.Errorf("Client.Pause does not called")
	}
	if err != m.err {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.pausearg1 != true {
		t.Errorf("unexpected Client.Pause arguments: %t", m.pausearg1)
	}

	m.err = nil
	err = p.Pause()

	if m.pausecalled != 2 {
		t.Errorf("Client.Pause does not called")
	}
	if err != m.err {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.pausearg1 != true {
		t.Errorf("unexpected Client.Pause arguments: %t", m.pausearg1)
	}
}

func TestPlayerNext(t *testing.T) {
	p, m := mockDial("tcp", "localhost:6600")
	m.err = new(mockError)
	err := p.Next()
	if m.nextcalled != 1 {
		t.Errorf("Client.Next does not called")
	}
	if err != m.err {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	m.err = nil
	err = p.Next()
	if m.nextcalled != 2 {
		t.Errorf("Client.Next does not called")
	}
	if err != m.err {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestPlayerPrevious(t *testing.T) {
	p, m := mockDial("tcp", "localhost:6600")
	m.err = new(mockError)
	err := p.Prev()
	if m.previouscalled != 1 {
		t.Errorf("Client.Previous does not called")
	}
	if err != m.err {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	m.err = nil
	err = p.Prev()
	if m.previouscalled != 2 {
		t.Errorf("Client.Previous does not called")
	}
	if err != m.err {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestPlayerPlaylist(t *testing.T) {
	p, m := mockDial("tcp", "localhost:6600")
	m.err = nil
	m.playlistinforet = []mpd.Attrs{{"foo": "bar"}}
	expect := songsAddReadableData((m.playlistinforet))
	// if mpd.Watcher.Event recieve "playlist"
	p.watcher.Event <- "playlist"
	if err := <-p.watcherResponse; err != nil {
		t.Errorf("unexpected watcher error: %s", err.Error())
	}

	// mpd.Client.PlaylistInfo was called
	if m.playlistinfocalled != 1 {
		t.Errorf("Client.PlaylistInfo does not called")
	}
	if m.playlistinfoarg1 != -1 || m.playlistinfoarg2 != -1 {
		t.Errorf("unexpected Client.PlaylistInfo arguments: %d %d", m.playlistinfoarg1, m.playlistinfoarg2)
	}
	if !reflect.DeepEqual(expect, p.playlist) {
		t.Errorf("unexpected stored playlist")
	}
	// Player.Playlist returns mpd.Client.PlaylistInfo result
	playlist, _ := p.Playlist()
	if !reflect.DeepEqual(expect, playlist) {
		t.Errorf("unexpected get playlist")
	}
}

func TestPlayerLibrary(t *testing.T) {
	p, m := mockDial("tcp", "localhost:6600")
	m.err = nil
	m.listallinforet = []mpd.Attrs{{"foo": "bar"}}
	expect := songsAddReadableData((m.listallinforet))
	// if mpd.Watcher.Event recieve "database"
	p.watcher.Event <- "database"
	if err := <-p.watcherResponse; err != nil {
		t.Errorf("unexpected watcher error: %s", err.Error())
	}

	// mpd.Client.ListAllInfo was called
	if m.listallinfocalled != 1 {
		t.Errorf("Client.ListAllInfo does not called")
	}
	if m.listallinfoarg1 != "/" {
		t.Errorf("unexpected Client.ListAllInfo arguments: %s", m.listallinfoarg1)
	}
	if !reflect.DeepEqual(expect, p.library) {
		t.Errorf("unexpected stored library")
	}
	// Player.Library returns mpd.Client.ListAllInfo result
	library, _ := p.Library()
	if !reflect.DeepEqual(expect, library) {
		t.Errorf("unexpected get library")
	}
}

func TestConvStatus(t *testing.T) {
	var lastModifiedOverRide int64
	lastModifiedOverRide = 0
	candidates := []struct {
		song   mpd.Attrs
		status mpd.Attrs
		expect PlayerStatus
	}{
		{
			mpd.Attrs{},
			mpd.Attrs{},
			PlayerStatus{
				-1, false, false, false, false,
				"stopped", 0, 0.0, 0, lastModifiedOverRide,
			},
		},
		{
			mpd.Attrs{
				"Time": "121",
			},
			mpd.Attrs{
				"volume":  "100",
				"repeat":  "1",
				"random":  "0",
				"single":  "1",
				"consume": "0",
				"state":   "playing",
				"song":    "1",
				"elapsed": "10.1",
			},
			PlayerStatus{
				100, true, false, true, false,
				"playing", 1, 10.1, 121, lastModifiedOverRide,
			},
		},
	}
	for _, c := range candidates {
		r := convStatus(c.song, c.status)
		r.LastModified = lastModifiedOverRide
		if !reflect.DeepEqual(c.expect, r) {
			jr, _ := json.Marshal(r)
			je, _ := json.Marshal(c.expect)
			t.Errorf(
				"unexpected. input: %s %s\nexpected: %s\nactual:   %s",
				songString(c.song),
				songString(c.status),
				je, jr,
			)
		}
	}
}

func TestPlayerCurrent(t *testing.T) {
	errret := new(mockError)
	candidates := []struct {
		currentSongRet1    mpd.Attrs
		currentSongRet2    error
		currentSongCalled  int
		currentRet         mpd.Attrs
		statusRet1         mpd.Attrs
		statusRet2         error
		statusCalled       int
		statusRet          PlayerStatus
		readCommentsRet1   mpd.Attrs
		readCommentsRet2   error
		readCommentsCalled int
		commentsRet        mpd.Attrs
		watcherRet         error
	}{
		// dont update if mpd.CurrentSong returns error
		{
			mpd.Attrs{}, errret, 1,
			nil,
			mpd.Attrs{}, nil, 0,
			PlayerStatus{},
			mpd.Attrs{}, nil, 0,
			nil,
			errret,
		},
		// dont update if mpd.Status returns error
		{
			mpd.Attrs{"Artist": "foo"}, nil, 2,
			nil,
			mpd.Attrs{}, errret, 1,
			PlayerStatus{},
			mpd.Attrs{}, nil, 0,
			nil,
			errret,
		},
		// dont update if mpd.ReadComments returns error
		{
			mpd.Attrs{"file": "p"}, nil, 3,
			nil,
			mpd.Attrs{}, nil, 2,
			PlayerStatus{},
			mpd.Attrs{}, errret, 1,
			nil,
			errret,
		},
		// update current/status/comments
		{
			mpd.Attrs{"file": "p"}, nil, 4,
			songAddReadableData(mpd.Attrs{"file": "p"}),
			mpd.Attrs{}, nil, 3,
			convStatus(mpd.Attrs{"file": "p"}, mpd.Attrs{}),
			mpd.Attrs{}, nil, 2,
			mpd.Attrs{},
			nil,
		},
		// dont call mpd.ReadComments if mpd.CurrentSong returns same song
		{
			mpd.Attrs{"file": "p"}, nil, 5,
			songAddReadableData(mpd.Attrs{"file": "p"}),
			mpd.Attrs{}, nil, 4,
			convStatus(mpd.Attrs{"file": "p"}, mpd.Attrs{}),
			mpd.Attrs{}, nil, 2,
			mpd.Attrs{},
			nil,
		},
	}
	p, m := mockDial("tcp", "localhost:6600")
	for _, c := range candidates {
		m.currentSongRet1 = c.currentSongRet1
		m.currentSongRet2 = c.currentSongRet2
		m.statusRet1 = c.statusRet1
		m.statusRet2 = c.statusRet2
		m.readCommentsRet1 = c.readCommentsRet1
		m.readCommentsRet2 = c.readCommentsRet2
		p.watcher.Event <- "player"
		if err := <-p.watcherResponse; err != c.watcherRet {
			t.Errorf("unexpected watcher error")
		}
		if m.currentsongcalled != c.currentSongCalled {
			t.Errorf("unexpected function call")
		}
		current, _ := p.Current()
		if !reflect.DeepEqual(current, c.currentRet) {
			t.Errorf(
				"unexpected Player.Current()\nexpect: %s\nactual:   %s",
				songString(c.currentRet),
				songString(current),
			)
		}
		if m.statuscalled != c.statusCalled {
			t.Errorf("unexpected function call")
		}
		status, _ := p.Status()
		if !reflect.DeepEqual(status, c.statusRet) {
			sj, _ := json.Marshal(status)
			ej, _ := json.Marshal(c.statusRet)
			t.Errorf(
				"unexpected Player.Status()\nexpect: %s\nactual:   %s",
				ej, sj,
			)
		}
		if m.readcommentscalled != c.readCommentsCalled {
			t.Errorf("unexpected function call")
		}
		comments, _ := p.Comments()
		if !reflect.DeepEqual(comments, c.commentsRet) {
			t.Errorf(
				"unexpected Player.Comments()\nexpect: %s\nactual:   %s",
				songString(c.commentsRet),
				songString(comments),
			)
		}
	}
}

func mockDial(network, addr string) (p *Player, m *mockMpc) {
	p = new(Player)
	p.mutex = new(sync.Mutex)
	p.daemonStop = make(chan bool)
	p.daemonRequest = make(chan *mpcMessage)
	p.network = network
	p.addr = addr
	m = new(mockMpc)
	p.mpc = m
	p.watcher = *new(mpd.Watcher)
	p.watcherResponse = make(chan error)
	p.watcher.Event = make(chan string)
	go p.daemon()
	go p.watch()
	return
}

type mockMpc struct {
	err                    error
	playcalled             int
	playarg1               int
	pausecalled            int
	pausearg1              bool
	nextcalled             int
	previouscalled         int
	closecalled            int
	playlistinfocalled     int
	playlistinfoarg1       int
	playlistinfoarg2       int
	playlistinforet        []mpd.Attrs
	listallinfocalled      int
	listallinfoarg1        string
	listallinforet         []mpd.Attrs
	readcommentscalled     int
	readcommentsarg1       string
	readCommentsRet1       mpd.Attrs
	readCommentsRet2       error
	currentsongcalled      int
	currentSongRet1        mpd.Attrs
	currentSongRet2        error
	statuscalled           int
	statusRet1             mpd.Attrs
	statusRet2             error
	pingcalled             int
	begincommandlistcalled int
}

func (p *mockMpc) Play(playarg1 int) error {
	p.playcalled++
	p.playarg1 = playarg1
	return p.err
}
func (p *mockMpc) Pause(pausearg1 bool) error {
	p.pausecalled++
	p.pausearg1 = pausearg1
	return p.err
}
func (p *mockMpc) Next() error {
	p.nextcalled++
	return p.err
}
func (p *mockMpc) Previous() error {
	p.previouscalled++
	return p.err
}
func (p *mockMpc) Close() error {
	p.closecalled++
	return p.err
}
func (p *mockMpc) Ping() error {
	p.pingcalled++
	return p.err
}
func (p *mockMpc) CurrentSong() (mpd.Attrs, error) {
	p.currentsongcalled++
	return p.currentSongRet1, p.currentSongRet2
}
func (p *mockMpc) Status() (mpd.Attrs, error) {
	p.statuscalled++
	return p.statusRet1, p.statusRet2
}
func (p *mockMpc) ReadComments(readcommentsarg1 string) (mpd.Attrs, error) {
	p.readcommentscalled++
	p.readcommentsarg1 = readcommentsarg1
	return p.readCommentsRet1, p.readCommentsRet2
}
func (p *mockMpc) PlaylistInfo(playlistinfoarg1, playlistinfoarg2 int) ([]mpd.Attrs, error) {
	p.playlistinfocalled++
	p.playlistinfoarg1 = playlistinfoarg1
	p.playlistinfoarg2 = playlistinfoarg2
	return p.playlistinforet, p.err
}
func (p *mockMpc) ListAllInfo(listallinfoarg1 string) ([]mpd.Attrs, error) {
	p.listallinfocalled++
	p.listallinfoarg1 = listallinfoarg1
	return p.listallinforet, p.err
}

func (p *mockMpc) BeginCommandList() *mpd.CommandList {
	p.begincommandlistcalled++
	return nil
}

type mockError struct{}

func (m *mockError) Error() string { return "err" }
