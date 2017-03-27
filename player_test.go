package main

import (
	"encoding/json"
	"github.com/fhs/gompd/mpd"
	"reflect"
	"sync"
	"testing"
)

func initMock(dialError, newWatcherError error) *mockMpc {
	m := new(mockMpc)
	m.ListAllInfoRet1 = []mpd.Attrs{}
	m.PlaylistInfoRet1 = []mpd.Attrs{}
	m.StatusRet1 = mpd.Attrs{}
	m.ReadCommentsRet1 = mpd.Attrs{}
	m.CurrentSongRet1 = mpd.Attrs{}
	playerMpdDial = func(n, a, s string) (mpdClient, error) {
		m.DialCalled++
		return m, dialError
	}
	playerMpdNewWatcher = func(n, a, s string) (*mpd.Watcher, error) {
		m.NewWatcherCalled++
		return new(mpd.Watcher), newWatcherError
	}
	return m
}

func TestDial(t *testing.T) {
	m := initMock(nil, nil)
	_, err := Dial("tcp", "localhost:6600", "")
	if err != nil {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.DialCalled != 1 {
		t.Errorf("mpd.Dial was not called: %d", m.DialCalled)
	}
	if m.NewWatcherCalled != 1 {
		t.Errorf("mpd.NewWatcher was not called: %d", m.NewWatcherCalled)
	}

	me := new(mockError)
	m = initMock(me, nil)
	_, err = Dial("tcp", "localhost:6600", "")
	if err != me {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.DialCalled != 1 {
		t.Errorf("mpd.Dial was not called: %d", m.DialCalled)
	}
	if m.NewWatcherCalled != 0 {
		t.Errorf("mpd.NewWatcher was not called: %d", m.NewWatcherCalled)
	}

	m = initMock(nil, me)
	_, err = Dial("tcp", "localhost:6600", "")
	if err != me {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.DialCalled != 1 {
		t.Errorf("mpd.Dial was not called: %d", m.DialCalled)
	}
	if m.NewWatcherCalled != 1 {
		t.Errorf("mpd.NewWatcher was not called: %d", m.NewWatcherCalled)
	}
	if m.CloseCalled != 1 {
		t.Errorf("mpd.Client.Close was not called: %d", m.CloseCalled)
	}
}

func TestPlayerPlay(t *testing.T) {
	p, m := mockDial("tcp", "localhost:6600")
	m.PlayRet1 = new(mockError)
	err := p.Play()
	if m.PlayCalled != 1 {
		t.Errorf("Client.Play does not Called")
	}
	if err != m.PlayRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.PlayArg1 != -1 {
		t.Errorf("unexpected Client.Play Arguments: %d", m.PlayArg1)
	}

	m.PlayRet1 = nil
	err = p.Play()

	if m.PlayCalled != 2 {
		t.Errorf("Client.Play does not Called")
	}
	if err != m.PlayRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.PlayArg1 != -1 {
		t.Errorf("unexpected Client.Play Arguments: %d", m.PlayArg1)
	}
}

func TestPlayerPause(t *testing.T) {
	p, m := mockDial("tcp", "localhost:6600")
	m.PauseRet1 = new(mockError)
	err := p.Pause()
	if m.PauseCalled != 1 {
		t.Errorf("Client.Pause does not Called")
	}
	if err != m.PauseRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.PauseArg1 != true {
		t.Errorf("unexpected Client.Pause Arguments: %t", m.PauseArg1)
	}

	m.PauseRet1 = nil
	err = p.Pause()

	if m.PauseCalled != 2 {
		t.Errorf("Client.Pause does not Called")
	}
	if err != m.PauseRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.PauseArg1 != true {
		t.Errorf("unexpected Client.Pause Arguments: %t", m.PauseArg1)
	}
}

func TestPlayerNext(t *testing.T) {
	p, m := mockDial("tcp", "localhost:6600")
	m.NextRet1 = new(mockError)
	err := p.Next()
	if m.NextCalled != 1 {
		t.Errorf("Client.Next does not Called")
	}
	if err != m.NextRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	m.NextRet1 = nil
	err = p.Next()
	if m.NextCalled != 2 {
		t.Errorf("Client.Next does not Called")
	}
	if err != m.NextRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestPlayerPrevious(t *testing.T) {
	p, m := mockDial("tcp", "localhost:6600")
	m.PreviousRet1 = new(mockError)
	err := p.Prev()
	if m.PreviousCalled != 1 {
		t.Errorf("Client.Previous does not Called")
	}
	if err != m.PreviousRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	m.PreviousRet1 = nil
	err = p.Prev()
	if m.PreviousCalled != 2 {
		t.Errorf("Client.Previous does not Called")
	}
	if err != m.PreviousRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestPlayerSetVolume(t *testing.T) {
	p, m := mockDial("tcp", "localhost:6600")
	m.CurrentSongRet2 = new(mockError)
	err := p.Volume(1)
	if m.SetVolumeCalled != 1 {
		t.Errorf("Client.SetVolume does not Called")
	}
	if m.CurrentSongCalled != 1 {
		t.Errorf("Client.CurrentSong does not Called")
	}
	if err != m.CurrentSongRet2 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestPlayerPlaylist(t *testing.T) {
	p, m := mockDial("tcp", "localhost:6600")
	m.PlaylistInfoRet1 = []mpd.Attrs{{"foo": "bar"}}
	m.PlaylistInfoRet2 = nil
	expect := songsAddReadableData((m.PlaylistInfoRet1))
	// if mpd.Watcher.Event recieve "playlist"
	p.watcher.Event <- "playlist"
	if err := <-p.watcherResponse; err != nil {
		t.Errorf("unexpected watcher error: %s", err.Error())
	}

	// mpd.Client.PlaylistInfo was Called
	if m.PlaylistInfoCalled != 1 {
		t.Errorf("Client.PlaylistInfo does not Called")
	}
	if m.PlaylistInfoArg1 != -1 || m.PlaylistInfoArg2 != -1 {
		t.Errorf("unexpected Client.PlaylistInfo Arguments: %d %d", m.PlaylistInfoArg1, m.PlaylistInfoArg2)
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
	m.ListAllInfoRet1 = []mpd.Attrs{{"foo": "bar"}}
	m.ListAllInfoRet2 = nil
	expect := songsAddReadableData((m.ListAllInfoRet1))
	// if mpd.Watcher.Event recieve "database"
	p.watcher.Event <- "database"
	if err := <-p.watcherResponse; err != nil {
		t.Errorf("unexpected watcher error: %s", err.Error())
	}

	// mpd.Client.ListAllInfo was Called
	if m.ListAllInfoCalled != 1 {
		t.Errorf("Client.ListAllInfo does not Called")
	}
	if m.ListAllInfoArg1 != "/" {
		t.Errorf("unexpected Client.ListAllInfo Arguments: %s", m.ListAllInfoArg1)
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
		CurrentSongRet1    mpd.Attrs
		CurrentSongRet2    error
		CurrentSongCalled  int
		currentRet         mpd.Attrs
		StatusRet1         mpd.Attrs
		StatusRet2         error
		StatusCalled       int
		StatusRet          PlayerStatus
		ReadCommentsRet1   mpd.Attrs
		ReadCommentsRet2   error
		ReadCommentsCalled int
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
		m.CurrentSongRet1 = c.CurrentSongRet1
		m.CurrentSongRet2 = c.CurrentSongRet2
		m.StatusRet1 = c.StatusRet1
		m.StatusRet2 = c.StatusRet2
		m.ReadCommentsRet1 = c.ReadCommentsRet1
		m.ReadCommentsRet2 = c.ReadCommentsRet2
		p.watcher.Event <- "player"
		if err := <-p.watcherResponse; err != c.watcherRet {
			t.Errorf("unexpected watcher error")
		}
		if m.CurrentSongCalled != c.CurrentSongCalled {
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
		if m.StatusCalled != c.StatusCalled {
			t.Errorf("unexpected function call")
		}
		status, _ := p.Status()
		if !reflect.DeepEqual(status, c.StatusRet) {
			sj, _ := json.Marshal(status)
			ej, _ := json.Marshal(c.StatusRet)
			t.Errorf(
				"unexpected Player.Status()\nexpect: %s\nactual:   %s",
				ej, sj,
			)
		}
		if m.ReadCommentsCalled != c.ReadCommentsCalled {
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
	p.daemonRequest = make(chan *playerMessage)
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
	DialCalled             int
	NewWatcherCalled       int
	PlayCalled             int
	PlayArg1               int
	PlayRet1               error
	PauseCalled            int
	PauseArg1              bool
	PauseRet1              error
	NextCalled             int
	NextRet1               error
	PreviousCalled         int
	PreviousRet1           error
	CloseCalled            int
	CloseRet1              error
	SetVolumeCalled        int
	SetVolumeArg1          int
	SetVolumeRet1          error
	PlaylistInfoCalled     int
	PlaylistInfoArg1       int
	PlaylistInfoArg2       int
	PlaylistInfoRet1       []mpd.Attrs
	PlaylistInfoRet2       error
	ListAllInfoCalled      int
	ListAllInfoArg1        string
	ListAllInfoRet1        []mpd.Attrs
	ListAllInfoRet2        error
	ReadCommentsCalled     int
	ReadCommentsArg1       string
	ReadCommentsRet1       mpd.Attrs
	ReadCommentsRet2       error
	CurrentSongCalled      int
	CurrentSongRet1        mpd.Attrs
	CurrentSongRet2        error
	StatusCalled           int
	StatusRet1             mpd.Attrs
	StatusRet2             error
	PingCalled             int
	PingRet1               error
	begincommandlistCalled int
}

func (p *mockMpc) Play(PlayArg1 int) error {
	p.PlayCalled++
	p.PlayArg1 = PlayArg1
	return p.PlayRet1
}
func (p *mockMpc) Pause(PauseArg1 bool) error {
	p.PauseCalled++
	p.PauseArg1 = PauseArg1
	return p.PauseRet1
}
func (p *mockMpc) Next() error {
	p.NextCalled++
	return p.NextRet1
}
func (p *mockMpc) Previous() error {
	p.PreviousCalled++
	return p.PreviousRet1
}
func (p *mockMpc) Close() error {
	p.CloseCalled++
	return p.CloseRet1
}
func (p *mockMpc) SetVolume(i int) error {
	p.SetVolumeCalled++
	p.SetVolumeArg1 = i
	return p.SetVolumeRet1
}
func (p *mockMpc) Ping() error {
	p.PingCalled++
	return p.PingRet1
}
func (p *mockMpc) CurrentSong() (mpd.Attrs, error) {
	p.CurrentSongCalled++
	return p.CurrentSongRet1, p.CurrentSongRet2
}
func (p *mockMpc) Status() (mpd.Attrs, error) {
	p.StatusCalled++
	return p.StatusRet1, p.StatusRet2
}
func (p *mockMpc) ReadComments(ReadCommentsArg1 string) (mpd.Attrs, error) {
	p.ReadCommentsCalled++
	p.ReadCommentsArg1 = ReadCommentsArg1
	return p.ReadCommentsRet1, p.ReadCommentsRet2
}
func (p *mockMpc) PlaylistInfo(PlaylistInfoArg1, PlaylistInfoArg2 int) ([]mpd.Attrs, error) {
	p.PlaylistInfoCalled++
	p.PlaylistInfoArg1 = PlaylistInfoArg1
	p.PlaylistInfoArg2 = PlaylistInfoArg2
	return p.PlaylistInfoRet1, p.PlaylistInfoRet2
}
func (p *mockMpc) ListAllInfo(ListAllInfoArg1 string) ([]mpd.Attrs, error) {
	p.ListAllInfoCalled++
	p.ListAllInfoArg1 = ListAllInfoArg1
	return p.ListAllInfoRet1, p.ListAllInfoRet2
}

func (p *mockMpc) BeginCommandList() *mpd.CommandList {
	p.begincommandlistCalled++
	return nil
}

type mockError struct{}

func (m *mockError) Error() string { return "err" }
