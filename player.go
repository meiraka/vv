package main

import "github.com/fhs/gompd/mpd"

/*Dial Connects to mpd server.*/
func Dial(network, addr string) (p *Player, err error) {
	// connect to mpd
	conn, err := mpd.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	p = new(Player)
	p.conn = conn

	// initialize library
	err = p.SyncLibrary()
	if err != nil {
		return nil, err
	}
	// initialize playlist
	err = p.SyncPlaylist()
	if err != nil {
		return nil, err
	}
	return
}

/*Player represents mpd control interface.*/
type Player struct {
	conn     *mpd.Client
	library  []mpd.Attrs
	playlist []mpd.Attrs
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

/*Close mpd connection.*/
func (p *Player) Close() error {
	return p.conn.Close()
}
