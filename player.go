package main

import "github.com/fhs/gompd/mpd"

type playerRequestType int

const (
	syncLibrary playerRequestType = iota
	syncPlaylist
	prev
	play
	next
)

type playerRequest struct {
	request playerRequestType
	err     chan error
}

/*Dial Connects to mpd server.*/
func Dial(network, addr string) (p *Player, err error) {
	// connect to mpd
	p = new(Player)
	p.stop = make(chan bool)
	p.requestChannel = make(chan *playerRequest)
	p.network = network
	p.addr = addr
	err = p.connect()
	if err != nil {
		return p, err
	}

	// initialize library
	err = p.SyncLibrary()
	if err != nil {
		p.Close()
		return
	}
	// initialize playlist
	err = p.SyncPlaylist()
	go p.daemon()
	return
}

/*Player represents mpd control interface.*/
type Player struct {
	network        string
	addr           string
	conn           *mpd.Client
	stop           chan bool
	requestChannel chan *playerRequest
	library        []mpd.Attrs
	playlist       []mpd.Attrs
}

func (p *Player) daemon() {
loop:
	for {
		select {
		case <-p.stop:
			break loop
		case r := <-p.requestChannel:
			switch r.request {
			case prev:
				r.err <- p.conn.Previous()
			case play:
				r.err <- p.conn.Play(-1)
			case next:
				r.err <- p.conn.Next()
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

/*SyncLibrary updates Player.library*/
func (p *Player) SyncLibrary() error {
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

/*SyncPlaylist updates Player.playlist*/
func (p *Player) SyncPlaylist() error {
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

func (p *Player) request(req playerRequestType) error {
	r := new(playerRequest)
	r.request = req
	r.err = make(chan error)
	p.requestChannel <- r
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
