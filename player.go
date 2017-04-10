package main

import (
	"fmt"
	"github.com/fhs/gompd/mpd"
	"sort"
	"sync"
	"time"
)

/*Dial Connects to mpd server.*/
func Dial(network, addr, passwd string) (*Player, error) {
	p := new(Player)
	p.network = network
	p.addr = addr
	p.passwd = passwd
	return p, p.initIfNot()
}

/*Player represents mpd control interface.*/
type Player struct {
	network          string
	addr             string
	passwd           string
	mpc              mpdClient
	watcher          mpd.Watcher
	watcherResponse  chan error
	daemonStop       chan bool
	daemonRequest    chan *playerMessage
	init             sync.Mutex
	mutex            sync.Mutex
	current          mpd.Attrs
	currentModified  time.Time
	status           PlayerStatus
	library          []mpd.Attrs
	libraryModified  time.Time
	playlist         []mpd.Attrs
	playlistModified time.Time
	outputs          []mpd.Attrs
	outputsModified  time.Time
}

/*Close mpd connection.*/
func (p *Player) Close() error {
	p.daemonStop <- true
	p.mpc.Close()
	return p.watcher.Close()
}

/*Current returns mpd current song data.*/
func (p *Player) Current() (mpd.Attrs, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.current, p.currentModified
}

/*Status returns mpd current song data.*/
func (p *Player) Status() (PlayerStatus, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.status, time.Unix(p.status.LastModified, 0)
}

/*Library returns mpd library song data list.*/
func (p *Player) Library() ([]mpd.Attrs, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.library, p.libraryModified
}

/*Playlist returns mpd playlist song data list.*/
func (p *Player) Playlist() ([]mpd.Attrs, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.playlist, p.playlistModified
}

/*Pause song.*/
func (p *Player) Pause() error {
	return p.request(func() error { return p.mpc.Pause(true) })
}

/*Play or resume song.*/
func (p *Player) Play() error {
	return p.request(func() error { return p.mpc.Play(-1) })
}

/*Prev song.*/
func (p *Player) Prev() error {
	return p.request(p.mpc.Previous)
}

/*Next song.*/
func (p *Player) Next() error {
	return p.request(p.mpc.Next)
}

/*Volume set player volume.*/
func (p *Player) Volume(v int) error {
	return p.request(func() error { return p.mpc.SetVolume(v) })
}

/*Repeat enable if true*/
func (p *Player) Repeat(on bool) error {
	return p.request(func() error { return p.mpc.Repeat(on) })
}

/*Random enable if true*/
func (p *Player) Random(on bool) error {
	return p.request(func() error { return p.mpc.Random(on) })
}

/*Outputs return output device list.*/
func (p *Player) Outputs() ([]mpd.Attrs, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.outputs, p.outputsModified
}

/*Output enable output if true.*/
func (p *Player) Output(id int, on bool) error {
	if on {
		return p.request(func() error { return p.mpc.EnableOutput(id) })
	}
	return p.request(func() error { return p.mpc.DisableOutput(id) })
}

/*SortPlaylist sorts playlist by song tag name.*/
func (p *Player) SortPlaylist(keys []string, uri string) (err error) {
	return p.request(func() error { return p.sortPlaylist(keys, uri) })
}

func (p *Player) sortPlaylist(keys []string, uri string) (err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	err = nil
	l := make([]mpd.Attrs, len(p.library))
	copy(l, p.library)
	sort.Slice(l, func(i, j int) bool {
		return songSortKey(l[i], keys) < songSortKey(l[j], keys)
	})
	update := false
	if len(l) != len(p.playlist) {
		update = true
		fmt.Printf("length not match")
	} else {
		for i := range l {
			n := l[i]["file"]
			o := p.playlist[i]["file"]
			if n != o {
				fmt.Printf("index %d not match:\n'new:%s'\n'old:%s'", i, n, o)
				update = true
				break
			}
		}
	}
	if update {
		cl := p.mpc.BeginCommandList()
		cl.Clear()
		for i := range l {
			cl.Add(l[i]["file"])
		}
		err = cl.End()
	}
	if err != nil {
		return
	}
	for i := range l {
		if l[i]["file"] == uri {
			err = p.mpc.Play(i)
			return
		}
	}
	return
}

type playerMessage struct {
	request func() error
	err     chan error
}

type mpdClient interface {
	Play(int) error
	SetVolume(int) error
	Pause(bool) error
	Previous() error
	Next() error
	Ping() error
	Close() error
	Repeat(bool) error
	Random(bool) error
	CurrentSong() (mpd.Attrs, error)
	Status() (mpd.Attrs, error)
	ListAllInfo(string) ([]mpd.Attrs, error)
	PlaylistInfo(int, int) ([]mpd.Attrs, error)
	BeginCommandList() *mpd.CommandList
	ListOutputs() ([]mpd.Attrs, error)
	DisableOutput(int) error
	EnableOutput(int) error
}

func (p *Player) initIfNot() error {
	p.init.Lock()
	defer p.init.Unlock()
	if p.daemonStop == nil {
		p.daemonStop = make(chan bool)
		p.daemonRequest = make(chan *playerMessage)
		fs := []func() error{p.connect, p.updateLibrary, p.updatePlaylist, p.updateCurrentSong, p.updateStatus, p.updateOutputs}
		for i := range fs {
			err := fs[i]()
			if err != nil {
				if i != 0 {
					p.Close()
				}
				return err
			}
		}
		go p.daemon()
		go p.watch()
		go p.ping()
	}
	return nil
}

func (p *Player) daemon() {
	sendErr := func(ec chan error, err error) {
		if ec != nil {
			ec <- err
		}
	}
loop:
	for {
		select {
		case <-p.daemonStop:
			break loop
		case m := <-p.daemonRequest:
			sendErr(m.err, m.request())
		}
	}
}

func (p *Player) ping() {
	for {
		time.Sleep(1)
		p.request(p.mpc.Ping)
	}
}

func (p *Player) watch() {
	for subsystem := range p.watcher.Event {
		switch subsystem {
		case "database":
			p.requestAsync(p.updateLibrary, p.watcherResponse)
		case "playlist":
			p.requestAsync(p.updatePlaylist, p.watcherResponse)
		case "player", "mixer", "options":
			p.requestAsync(p.updateCurrentSong, p.watcherResponse)
			p.requestAsync(p.updateStatus, p.watcherResponse)
		case "output":
			p.requestAsync(p.updateOutputs, p.watcherResponse)
		}
	}
}

func (p *Player) reconnect() error {
	p.watcher.Close()
	p.mpc.Close()
	return p.connect()
}

func playerRealMpdDial(net, addr, passwd string) (mpdClient, error) {
	return mpd.DialAuthenticated(net, addr, passwd)
}

func playerRealMpdNewWatcher(net, addr, passwd string) (*mpd.Watcher, error) {
	return mpd.NewWatcher(net, addr, passwd)
}

var playerMpdDial = playerRealMpdDial
var playerMpdNewWatcher = playerRealMpdNewWatcher

func (p *Player) connect() error {
	mpc, err := playerMpdDial(p.network, p.addr, p.passwd)
	if err != nil {
		return err
	}
	p.mpc = mpc
	watcher, err := playerMpdNewWatcher(p.network, p.addr, p.passwd)
	if err != nil {
		mpc.Close()
		return err
	}
	p.watcher = *watcher
	return nil
}
func (p *Player) request(f func() error) error {
	ec := make(chan error)
	p.requestAsync(f, ec)
	return <-ec
}

func (p *Player) requestAsync(f func() error, ec chan error) {
	r := new(playerMessage)
	r.request = f
	r.err = ec
	p.daemonRequest <- r
}

func (p *Player) updateCurrentSong() error {
	song, err := p.mpc.CurrentSong()
	if err != nil {
		return err
	}
	if p.current["file"] != song["file"] {
		p.mutex.Lock()
		defer p.mutex.Unlock()
		p.current = songAddReadableData(song)
		p.currentModified = time.Now()
	}
	return nil
}

func (p *Player) updateStatus() error {
	status, err := p.mpc.Status()
	if err != nil {
		return err
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.status = convStatus(status, time.Now().Unix())
	return nil
}

func (p *Player) updateLibrary() error {
	library, err := p.mpc.ListAllInfo("/")
	if err != nil {
		return err
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.library = songsAddReadableData(library)
	p.libraryModified = time.Now()
	return nil
}

func (p *Player) updatePlaylist() error {
	playlist, err := p.mpc.PlaylistInfo(-1, -1)
	if err != nil {
		return err
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.playlist = songsAddReadableData(playlist)
	p.playlistModified = time.Now()
	return nil
}

func (p *Player) updateOutputs() error {
	outputs, err := p.mpc.ListOutputs()
	if err != nil {
		return err
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.outputs = outputs
	p.outputsModified = time.Now()
	return nil
}
