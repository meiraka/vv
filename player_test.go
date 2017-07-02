package main

import (
	"encoding/json"
	"errors"
	"github.com/fhs/gompd/mpd"
	"reflect"
	"testing"
	"time"
)

func initMock(dialError, newWatcherError error) *mockMpc {
	m := new(mockMpc)
	m.ListAllInfoRet1 = []mpd.Attrs{}
	m.PlaylistInfoRet1 = []mpd.Attrs{}
	m.StatusRet1 = mpd.Attrs{}
	m.ReadCommentsRet1 = mpd.Attrs{}
	m.CurrentSongRet1 = mpd.Attrs{}
	m.ListOutputsRet1 = []mpd.Attrs{}
	playerMpdDial = func(n, a, s string) (mpdClient, error) {
		m.DialCalled++
		return m, dialError
	}
	playerMpdNewWatcher = func(n, a, s string) (*mpd.Watcher, error) {
		w := new(mpd.Watcher)
		w.Event = make(chan string)
		m.NewWatcherCalled++
		return w, newWatcherError
	}
	playerMpdWatcherClose = func(w mpd.Watcher) error {
		close(w.Event)
		w.Event = nil
		return nil
	}
	return m
}

func TestDial(t *testing.T) {
	m := initMock(nil, nil)
	p, err := Dial("tcp", "localhost:6600", "", "./")
	if err != nil {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.DialCalled != 1 {
		t.Errorf("mpd.Dial was not called: %d", m.DialCalled)
	}
	if m.NewWatcherCalled != 1 {
		t.Errorf("mpd.NewWatcher was not called: %d", m.NewWatcherCalled)
	}
	p.Close()
	me := new(mockError)
	m = initMock(me, nil)
	p, err = Dial("tcp", "localhost:6600", "", "./")
	if m.DialCalled != 1 {
		t.Errorf("mpd.Dial was not called: %d", m.DialCalled)
	}
	if m.NewWatcherCalled != 0 {
		t.Errorf("mpd.NewWatcher was not called: %d", m.NewWatcherCalled)
	}
	p.Close()

	m = initMock(nil, me)
	p, err = Dial("tcp", "localhost:6600", "", "./")
	if m.DialCalled != 1 {
		t.Errorf("mpd.Dial was not called: %d", m.DialCalled)
	}
	if m.NewWatcherCalled != 1 {
		t.Errorf("mpd.NewWatcher was not called: %d", m.NewWatcherCalled)
	}
	if m.CloseCalled != 1 {
		t.Errorf("mpd.Client.Close was not called: %d", m.CloseCalled)
	}
	p.Close()
}

func TestPlayerPlay(t *testing.T) {
	m := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
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
	p.Close()
}

func TestPlayerRescanLibrary(t *testing.T) {
	m := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	m.UpdateRet2 = nil
	err := p.RescanLibrary()
	if m.UpdateCalled != 1 {
		t.Errorf("Client.Update did not Called")
	}
	if m.UpdateArg1 != "" {
		t.Errorf("unexpected argument 1: %s", m.UpdateArg1)
	}
	if err != nil {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestPlayerPause(t *testing.T) {
	m := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
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
	m := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
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
	m := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
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
	m := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	err := p.Volume(1)
	if m.SetVolumeCalled != 1 {
		t.Errorf("Client.SetVolume does not Called")
	}
	if err != nil {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestPlayerRepeat(t *testing.T) {
	m := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	err := p.Repeat(true)
	if m.RepeatCalled != 1 {
		t.Errorf("Client.Repeat does not Called")
	}
	if m.RepeatArg1 != true {
		t.Errorf("unexpected argument: %t", m.RepeatArg1)
	}
	if err != m.RepeatRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestPlayerRandom(t *testing.T) {
	m := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	err := p.Random(true)
	if m.RandomCalled != 1 {
		t.Errorf("Client.Random does not Called")
	}
	if m.RandomArg1 != true {
		t.Errorf("unexpected argument: %t", m.RandomArg1)
	}
	if err != m.RandomRet1 {
		t.Errorf("unexpected return error: %s", err.Error())
	}
}

func TestPlayerPlaylist(t *testing.T) {
	m := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	p.watcherResponse = make(chan error)
	m.PlaylistInfoCalled = 0
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

func TestPlayerStats(t *testing.T) {
	m := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	p.watcherResponse = make(chan error)
	var testsets = []struct {
		ret1   mpd.Attrs
		ret2   error
		expect mpd.Attrs
		event  string
	}{
		{nil, errors.New("hoge"), nil, "database"},
		{mpd.Attrs{"foo": "bar"}, nil, mpd.Attrs{"foo": "bar"}, "database"},
		{nil, errors.New("hoge"), mpd.Attrs{"foo": "bar"}, "database"},
	}
	for _, tt := range testsets {
		m.StatsRet1 = tt.ret1
		m.StatsRet2 = tt.ret2
		m.StatsCalled = 0
		p.watcher.Event <- tt.event
		if err := <-p.watcherResponse; err != nil {
			t.Errorf("unexpected watcher error: %s from Library", err.Error())
		}
		if err := <-p.watcherResponse; err != tt.ret2 {
			t.Errorf("unexpected watcher error from Stats")
		}
		actual, _ := p.Stats()
		if m.StatsCalled != 1 {
			t.Errorf("mpd.Client.Stats does not called")
		}
		if !reflect.DeepEqual(tt.expect, actual) {
			t.Errorf("unexpected get stats")
		}
	}
}

func TestPlayerLibrary(t *testing.T) {
	m := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	p.watcherResponse = make(chan error)
	m.ListAllInfoCalled = 0
	m.ListAllInfoRet1 = []mpd.Attrs{{"foo": "bar"}}
	m.ListAllInfoRet2 = nil
	expect := songsAddReadableData((m.ListAllInfoRet1))
	// if mpd.Watcher.Event recieve "database"
	p.watcher.Event <- "database"
	for i := 0; i < 2; i++ {
		if err := <-p.watcherResponse; err != nil {
			t.Errorf("unexpected watcher error: %s", err.Error())
		}
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

func TestPlayerCurrent(t *testing.T) {
	m := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	p.watcherResponse = make(chan error)
	m.CurrentSongCalled = 0
	m.StatusCalled = 0
	errret := new(mockError)
	candidates := []struct {
		CurrentSongRet1   mpd.Attrs
		CurrentSongRet2   error
		CurrentSongCalled int
		currentRet        mpd.Attrs
		StatusRet1        mpd.Attrs
		StatusRet2        error
		StatusCalled      int
		StatusRet         PlayerStatus
	}{
		// dont update if mpd.CurrentSong returns error
		{
			mpd.Attrs{}, errret, 1,
			p.current,
			mpd.Attrs{}, errret, 1,
			p.status,
		},
		// update current/status/comments
		{
			mpd.Attrs{"file": "p"}, nil, 2,
			songFindCover(songAddReadableData(mpd.Attrs{"file": "p"}), p.musicDirectory, p.coverCache),
			mpd.Attrs{}, nil, 2,
			convStatus(mpd.Attrs{}, time.Now().Unix()),
		},
	}
	for _, c := range candidates {
		m.CurrentSongRet1 = c.CurrentSongRet1
		m.CurrentSongRet2 = c.CurrentSongRet2
		m.StatusRet1 = c.StatusRet1
		m.StatusRet2 = c.StatusRet2
		p.watcher.Event <- "player"
		if err := <-p.watcherResponse; err != c.CurrentSongRet2 {
			t.Errorf("unexpected watcher error from CurrentSong")
		}
		if err := <-p.watcherResponse; err != c.StatusRet2 {
			t.Errorf("unexpected watcher error from Status")
		}
		if err := <-p.watcherResponse; err != nil {
			t.Errorf("unexpected watcher error from Stats")
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
	}
}

func TestPlayerOutputs(t *testing.T) {
	m := initMock(nil, nil)
	p, _ := Dial("tcp", "localhost:6600", "", "./")
	defer p.Close()
	p.watcherResponse = make(chan error)
	m.ListOutputsCalled = 0
	m.ListOutputsRet1 = []mpd.Attrs{{"foo": "bar"}}
	m.ListOutputsRet2 = nil
	// if mpd.Watcher.Event recieve "output"
	p.watcher.Event <- "output"
	if err := <-p.watcherResponse; err != nil {
		t.Errorf("unexpected watcher error: %s", err.Error())
	}

	// mpd.Client.ListOutputs was Called
	if m.ListOutputsCalled != 1 {
		t.Errorf("Client.ListOutputs does not Called")
	}
	if !reflect.DeepEqual(m.ListOutputsRet1, p.outputs) {
		t.Errorf("unexpected stored outputs")
	}
	// Player.Library returns mpd.Client.ListOutputs result
	outputs, _ := p.Outputs()
	if !reflect.DeepEqual(m.ListOutputsRet1, outputs) {
		t.Errorf("unexpected get outputs")
	}
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
	RepeatCalled           int
	RepeatArg1             bool
	RepeatRet1             error
	RandomCalled           int
	RandomArg1             bool
	RandomRet1             error
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
	StatsCalled            int
	StatsRet1              mpd.Attrs
	StatsRet2              error
	PingCalled             int
	PingRet1               error
	ListOutputsCalled      int
	ListOutputsRet1        []mpd.Attrs
	ListOutputsRet2        error
	DisableOutputCalled    int
	DisableOutputArg1      int
	DisableOutputRet1      error
	EnableOutputCalled     int
	EnableOutputArg1       int
	EnableOutputRet1       error
	UpdateCalled           int
	UpdateArg1             string
	UpdateRet1             int
	UpdateRet2             error
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
func (p *mockMpc) Repeat(b bool) error {
	p.RepeatCalled++
	p.RepeatArg1 = b
	return p.RepeatRet1
}
func (p *mockMpc) Random(b bool) error {
	p.RandomCalled++
	p.RandomArg1 = b
	return p.RandomRet1
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
func (p *mockMpc) Stats() (mpd.Attrs, error) {
	p.StatsCalled++
	return p.StatsRet1, p.StatsRet2
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

func (p *mockMpc) ListOutputs() ([]mpd.Attrs, error) {
	p.ListOutputsCalled++
	return p.ListOutputsRet1, p.ListOutputsRet2
}

func (p *mockMpc) DisableOutput(arg1 int) error {
	p.DisableOutputCalled++
	p.DisableOutputArg1 = arg1
	return p.DisableOutputRet1
}

func (p *mockMpc) EnableOutput(arg1 int) error {
	p.EnableOutputCalled++
	p.EnableOutputArg1 = arg1
	return p.EnableOutputRet1
}

func (p *mockMpc) BeginCommandList() *mpd.CommandList {
	p.begincommandlistCalled++
	return nil
}

func (p *mockMpc) Update(arg1 string) (int, error) {
	p.UpdateCalled++
	p.UpdateArg1 = arg1
	return p.UpdateRet1, p.UpdateRet2
}

type mockError struct{}

func (m *mockError) Error() string { return "err" }
