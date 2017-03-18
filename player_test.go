package main

import (
	"github.com/fhs/gompd/mpd"
	"reflect"
	"sync"
	"testing"
)

func TestConvSong(t *testing.T) {
	i := mpd.Attrs{"Title": "foo", "file": "path"}
	r := convSong(i)
	if r.AlbumArtist != "[no Artist]" {
		t.Errorf("unexpected Song.AlbumArtist: %s", r.AlbumArtist)
	}
	if r.Album != "[no Album]" {
		t.Errorf("unexpected Song.Album: %s", r.Album)
	}
	if r.Artist != "[no Artist]" {
		t.Errorf("unexpected Song.Artist: %s", r.Artist)
	}
	if r.Genre != "[no Genre]" {
		t.Errorf("unexpected Song.Genre: %s", r.Genre)
	}
	if r.Track != -1 {
		t.Errorf("unexpected Song.Track: %d", r.Track)
	}
	if r.TrackNo != "[no Track]" {
		t.Errorf("unexpected Song.TrackNo: %s", r.TrackNo)
	}
	if r.Album != "[no Album]" {
		t.Errorf("unexpected Song.Album: %s", r.Album)
	}
	if r.Title != "foo" {
		t.Errorf("unexpected Song.Title: %s", r.Title)
	}
}

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
	expect := convSongs(m.playlistinforet)
	// if mpd.Watcher.Event recieve "playlist"
	p.watcher.Event <- "playlist"
	p.Nop()

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
	expect := convSongs(m.listallinforet)
	// if mpd.Watcher.Event recieve "database"
	p.watcher.Event <- "database"
	p.Nop()

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

func TestPlayerCurrent(t *testing.T) {
	p, m := mockDial("tcp", "localhost:6600")
	m.err = nil
	m.currentsongret = mpd.Attrs{"foo": "bar"}
	m.statusret = mpd.Attrs{"hoge": "fuga"}
	m.readcommentsret = mpd.Attrs{"baz": "piyo"}
	// if mpd.Watcher.Event recieve "database"
	p.watcher.Event <- "player"
	p.Nop()

	// mpd.Client.CurrentSong was called
	if m.currentsongcalled != 1 {
		t.Errorf("Client.CurrentSong does not called")
	}
	// mpd.Client.Status was called
	if m.statuscalled != 1 {
		t.Errorf("Client.Status does not called")
	}
	// mpd.Client.ReadComments was called
	if m.readcommentscalled != 1 {
		t.Errorf("Client.ReadComments does not called")
	}
	// Player.Current returns converted mpd.Client.CurrentSong result
	current, _ := p.Current()
	if !reflect.DeepEqual(convSong(m.currentsongret), current) {
		t.Errorf("unexpected get Current")
	}
	// Player.Status returns converted mpd.Client.Status result
	status, _ := p.Status()
	if status.Volume != -1 {
		t.Errorf("unexpected PlayerStatus.Volume")
	}
	if status.Repeat != false {
		t.Errorf("unexpected PlayerStatus.Repeat")
	}
	if status.Random != false {
		t.Errorf("unexpected PlayerStatus.Random")
	}
	if status.Single != false {
		t.Errorf("unexpected PlayerStatus.Single")
	}
	if status.Consume != false {
		t.Errorf("unexpected PlayerStatus.Consume")
	}
	if status.State != "stopped" {
		t.Errorf("unexpected PlayerStatus.State")
	}
	if status.SongPos != 0 {
		t.Errorf("unexpected PlayerStatus.SongPos: %d", status.SongPos)
	}
	if status.SongElapsed != 0.0 {
		t.Errorf("unexpected PlayerStatus.SongElapsed")
	}
	// Player.Current returns mpd.Client.ReadComments result
	comments, _ := p.Comments()
	if !reflect.DeepEqual(m.readcommentsret, comments) {
		t.Errorf("unexpected get comments")
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
	p.watcher.Event = make(chan string)
	go p.daemon()
	go p.watch()
	return
}

type mockMpc struct {
	err                error
	playcalled         int
	playarg1           int
	pausecalled        int
	pausearg1          bool
	nextcalled         int
	previouscalled     int
	closecalled        int
	playlistinfocalled int
	playlistinfoarg1   int
	playlistinfoarg2   int
	playlistinforet    []mpd.Attrs
	listallinfocalled  int
	listallinfoarg1    string
	listallinforet     []mpd.Attrs
	readcommentscalled int
	readcommentsarg1   string
	readcommentsret    mpd.Attrs
	currentsongcalled  int
	currentsongret     mpd.Attrs
	statuscalled       int
	statusret          mpd.Attrs
	pingcalled         int
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
	return p.currentsongret, p.err
}
func (p *mockMpc) Status() (mpd.Attrs, error) {
	p.statuscalled++
	return p.statusret, p.err
}
func (p *mockMpc) ReadComments(readcommentsarg1 string) (mpd.Attrs, error) {
	p.readcommentscalled++
	p.readcommentsarg1 = readcommentsarg1
	return p.readcommentsret, p.err
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

type mockError struct{}

func (m *mockError) Error() string { return "err" }
