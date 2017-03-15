package main

import (
	"github.com/fhs/gompd/mpd"
	"sync"
	"time"
)

type mpcMessageType int

const (
	syncLibrary mpcMessageType = iota
	syncPlaylist
	syncCurrent
	prev
	play
	next
)

type mpcMessage struct {
	request mpcMessageType
	err     chan error
}

/*Dial Connects to mpd server.*/
func Dial(network, addr string) (*Player, error) {
	// connect to mpd
	p := new(Player)
	p.mutex = new(sync.Mutex)
	p.daemonStop = make(chan bool)
	p.daemonRequest = make(chan *mpcMessage)
	p.network = network
	p.addr = addr
	return p, p.start()
}

/*Player represents mpd control interface.*/
type Player struct {
	network          string
	addr             string
	mpc              mpd.Client
	watcher          mpd.Watcher
	daemonStop       chan bool
	daemonRequest    chan *mpcMessage
	mutex            *sync.Mutex
	current          mpd.Attrs
	currentModified  time.Time
	comments         mpd.Attrs
	commentsModified time.Time
	library          []mpd.Attrs
	libraryModified  time.Time
	playlist         []mpd.Attrs
	playlistModified time.Time
}

func (p *Player) start() (err error) {
	err = p.connect()
	if err != nil {
		return err
	}

	// initialize library
	err = p.syncLibrary()
	if err != nil {
		p.Close()
		return
	}
	// initialize playlist
	err = p.syncPlaylist()
	if err != nil {
		p.Close()
		return
	}
	err = p.syncCurrent()
	if err != nil {
		p.Close()
		return
	}
	go p.daemon()
	go p.watch()
	return
}

func (p *Player) daemon() {
loop:
	for {
		select {
		case <-p.daemonStop:
			break loop
		case m := <-p.daemonRequest:
			switch m.request {
			case prev:
				m.err <- p.mpc.Previous()
			case play:
				m.err <- p.mpc.Play(-1)
			case next:
				m.err <- p.mpc.Next()
			case syncLibrary:
				m.err <- p.syncLibrary()
			case syncPlaylist:
				m.err <- p.syncPlaylist()
			case syncCurrent:
				m.err <- p.syncCurrent()
			}
		}
	}
}

func (p *Player) reconnect() error {
	p.watcher.Close()
	p.mpc.Close()
	return p.connect()
}

func (p *Player) connect() error {
	mpc, err := mpd.Dial(p.network, p.addr)
	if err != nil {
		return err
	}
	p.mpc = *mpc
	watcher, err := mpd.NewWatcher(p.network, p.addr, "")
	if err != nil {
		mpc.Close()
		return err
	}
	p.watcher = *watcher
	return nil
}

func (p *Player) watch() {
	for subsystem := range p.watcher.Event {
		switch subsystem {
		case "database":
			p.request(syncLibrary)
		case "playlist":
			p.request(syncPlaylist)
		case "player":
			p.request(syncCurrent)
		}
	}
}

func (p *Player) syncCurrent() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	song, err := p.mpc.CurrentSong()
	if err != nil {
		return err
	}
	status, err := p.mpc.Status()
	if err != nil {
		return err
	}
	for k, v := range status {
		song[k] = v
	}
	p.currentModified = time.Now()
	if p.comments == nil || p.current["file"] != song["file"] {
		comments, err := p.mpc.ReadComments(song["file"])
		if err != nil {
			return err
		}
		p.commentsModified = time.Now()
		p.comments = comments
	}

	p.current = song
	return nil
}

func (p *Player) syncLibrary() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	library, err := p.mpc.ListAllInfo("/")
	if err != nil {
		return err
	}
	p.library = library
	p.libraryModified = time.Now()
	return nil
}

/*Library returns mpd library song list.*/
func (p *Player) Library() ([]mpd.Attrs, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.library, p.libraryModified
}

func (p *Player) syncPlaylist() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	playlist, err := p.mpc.PlaylistInfo(-1, -1)
	if err != nil {
		return err
	}
	p.playlist = playlist
	p.playlistModified = time.Now()
	return nil
}

/*Playlist returns json string mpd playlist.*/
func (p *Player) Playlist() ([]mpd.Attrs, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.playlist, p.playlistModified
}

/*Current returns json string mpd current song.*/
func (p *Player) Current() (mpd.Attrs, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.current, p.currentModified
}

/*Comments returns json string mpd current song comments.*/
func (p *Player) Comments() (mpd.Attrs, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.comments, p.commentsModified
}

func (p *Player) request(req mpcMessageType) error {
	r := new(mpcMessage)
	r.request = req
	r.err = make(chan error)
	p.daemonRequest <- r
	return <-r.err
}

/*Prev song.*/
func (p *Player) Prev() error {
	return p.request(prev)
}

/*Play or resume song.*/
func (p *Player) Play() error {
	return p.request(play)
}

/*Pause song.*/
func (p *Player) Pause() error {
	return p.mpc.Pause(true)
}

/*Next song.*/
func (p *Player) Next() error {
	return p.request(next)
}

/*Close mpd connection.*/
func (p *Player) Close() error {
	p.daemonStop <- true
	p.mpc.Close()
	return p.watcher.Close()
}
