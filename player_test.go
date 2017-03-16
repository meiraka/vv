package main

import (
	"github.com/fhs/gompd/mpd"
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
	if p.Play() != m.err {
		t.Errorf("unexpected return error: %s", err.Error())
	}
	if m.playarg1 != -1 {
		t.Errorf("unexpected Client.Play arguments: %d", m.playarg1)
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
