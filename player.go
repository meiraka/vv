package main

import "github.com/fhs/gompd/mpd"

type connMessageType int

const (
	syncLibrary connMessageType = iota
	syncPlaylist
	prev
	play
	next
)

type connMessage struct {
	request connMessageType
	err     chan error
}

/*Dial Connects to mpd server.*/
func Dial(network, addr string) (p *Player, err error) {
	// connect to mpd
	p = new(Player)
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
	go p.connDaemon()
	return
}

/*Player represents mpd control interface.*/
type Player struct {
	network  string
	addr     string
	conn     *mpd.Client
	stop     chan bool
	c        chan *connMessage
	library  []mpd.Attrs
	playlist []mpd.Attrs
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
			}
		}
	}
}

func (p *Player) connect() error {
	conn, err := mpd.Dial(p.network, p.addr)
	if err != nil {
		return err
	}
	p.conn = conn
	return nil
}

func (p *Player) syncLibrary() error {
	library, err := p.conn.ListAllInfo("/")
	if err != nil {
		return err
	}
	p.library = library
	return nil
}

/*Library returns mpd library song list.*/
func (p *Player) Library() []mpd.Attrs {
	return p.library
}

func (p *Player) syncPlaylist() error {
	playlist, err := p.conn.PlaylistInfo(-1, -1)
	if err != nil {
		return err
	}
	p.playlist = playlist
	return nil
}

/*Playlist returns json string mpd playlist.*/
func (p *Player) Playlist() []mpd.Attrs {
	return p.playlist
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
