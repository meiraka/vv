package main

import (
	"github.com/fhs/gompd/mpd"
	"sync"
	"time"
)

type connMessageType int

const (
	syncLibrary connMessageType = iota
	syncPlaylist
	syncCurrent
	prev
	play
	next
)

type connMessage struct {
	request connMessageType
	err     chan error
}

type connRequest int

/*Dial Connects to mpd server.*/
func Dial(network, addr string) (p *Player, err error) {
	// connect to mpd
	p = new(Player)
	p.m = new(sync.Mutex)
	p.stop = make(chan bool)
	p.c = make(chan *connMessage)
	p.network = network
	p.addr = addr
	err = p.connect()
	if err != nil {
		return p, err
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
	go p.connDaemon()
	go p.watch()
	return
}

/*Player represents mpd control interface.*/
type Player struct {
	network          string
	addr             string
	conn             mpd.Client
	w                mpd.Watcher
	m                *sync.Mutex
	stop             chan bool
	c                chan *connMessage
	r                chan *connRequest
	current          mpd.Attrs
	currentModified  time.Time
	comments         mpd.Attrs
	commentsModified time.Time
	library          []mpd.Attrs
	libraryModified  time.Time
	playlist         []mpd.Attrs
	playlistModified time.Time
}

func (p *Player) connDaemon() {
loop:
	for {
		select {
		case <-p.stop:
			break loop
		case m := <-p.c:
			switch m.request {
			case prev:
				m.err <- p.conn.Previous()
			case play:
				m.err <- p.conn.Play(-1)
			case next:
				m.err <- p.conn.Next()
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
	p.w.Close()
	p.conn.Close()
	return p.connect()
}

func (p *Player) connect() error {
	conn, err := mpd.Dial(p.network, p.addr)
	if err != nil {
		return err
	}
	p.conn = *conn
	w, err := mpd.NewWatcher(p.network, p.addr, "")
	if err != nil {
		return err
	}
	p.w = *w
	return nil
}

func (p *Player) watch() {
	for subsystem := range p.w.Event {
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
	p.m.Lock()
	defer p.m.Unlock()
	song, err := p.conn.CurrentSong()
	if err != nil {
		return err
	}
	status, err := p.conn.Status()
	if err != nil {
		return err
	}
	for k, v := range status {
		song[k] = v
	}
	p.currentModified = time.Now()
	if p.comments == nil || p.current["file"] != song["file"] {
		comments, err := p.conn.ReadComments(song["file"])
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
	p.m.Lock()
	defer p.m.Unlock()
	library, err := p.conn.ListAllInfo("/")
	if err != nil {
		return err
	}
	p.library = library
	p.libraryModified = time.Now()
	return nil
}

/*Library returns mpd library song list.*/
func (p *Player) Library() ([]mpd.Attrs, time.Time) {
	p.m.Lock()
	defer p.m.Unlock()
	return p.library, p.libraryModified
}

func (p *Player) syncPlaylist() error {
	p.m.Lock()
	defer p.m.Unlock()
	playlist, err := p.conn.PlaylistInfo(-1, -1)
	if err != nil {
		return err
	}
	p.playlist = playlist
	p.playlistModified = time.Now()
	return nil
}

/*Playlist returns json string mpd playlist.*/
func (p *Player) Playlist() ([]mpd.Attrs, time.Time) {
	p.m.Lock()
	defer p.m.Unlock()
	return p.playlist, p.playlistModified
}

/*Current returns json string mpd current song.*/
func (p *Player) Current() (mpd.Attrs, time.Time) {
	p.m.Lock()
	defer p.m.Unlock()
	return p.current, p.currentModified
}

/*Comments returns json string mpd current song comments.*/
func (p *Player) Comments() (mpd.Attrs, time.Time) {
	p.m.Lock()
	defer p.m.Unlock()
	return p.comments, p.commentsModified
}

func (p *Player) request(req connMessageType) error {
	r := new(connMessage)
	r.request = req
	r.err = make(chan error)
	p.c <- r
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
	return p.conn.Pause(true)
}

/*Next song.*/
func (p *Player) Next() error {
	return p.request(next)
}

/*Close mpd connection.*/
func (p *Player) Close() error {
	p.stop <- true
	return p.conn.Close()
}
