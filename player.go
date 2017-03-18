package main

import (
	"github.com/fhs/gompd/mpd"
	"strconv"
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
	ping
	nop
)

type mpcMessage struct {
	request mpcMessageType
	err     chan error
}

/*Dial Connects to mpd server.*/
func Dial(network, addr string) (*Player, error) {
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
	mpc              MpdClient
	watcher          mpd.Watcher
	daemonStop       chan bool
	daemonRequest    chan *mpcMessage
	mutex            *sync.Mutex
	current          Song
	currentModified  time.Time
	status           PlayerStatus
	comments         mpd.Attrs
	commentsModified time.Time
	library          []Song
	libraryModified  time.Time
	playlist         []Song
	playlistModified time.Time
}

/*PlayerStatus represents mpd status.*/
type PlayerStatus struct {
	Volume       int     `json:"volume"`
	Repeat       bool    `json:"repeat"`
	Random       bool    `json:"random"`
	Single       bool    `json:"single"`
	Consume      bool    `json:"consume"`
	State        string  `json:"state"`
	SongPos      int     `json:"song_pos"`
	SongElapsed  float32 `json:"song_elapsed"`
	SongLength   int     `json:"song_length"`
	LastModified int64   `json:"last_modified"`
}

/*Song represents mpd song data.*/
type Song struct {
	AlbumArtist string `json:"albumartist"`
	Album       string `json:"album"`
	Artist      string `json:"artist"`
	Date        string `json:"date"`
	Genre       string `json:"genre"`
	Track       int    `json:"track"`
	TrackNo     string `json:"trackno"`
	Title       string `json:"title"`
	File        string `json:"file"`
}

func convSong(d mpd.Attrs) (s Song) {
	checks := []string{
		"Album",
		"Artist",
		"Date",
		"Genre",
		"Track",
		"Title",
	}
	for i := range checks {
		if _, ok := d[checks[i]]; !ok {
			d[checks[i]] = "[no " + checks[i] + "]"
		}
	}
	s.AlbumArtist = d["AlbumArtist"]
	if s.AlbumArtist == "" {
		s.AlbumArtist = d["Artist"]
	}
	s.Album = d["Album"]
	s.Artist = d["Artist"]
	s.Date = d["Date"]
	s.Genre = d["Genre"]
	track, err := strconv.Atoi(d["Track"])
	if err != nil {
		track = -1
	}
	s.Track = track
	s.TrackNo = d["Track"]
	s.Title = d["Title"]
	return
}

func convSongs(d []mpd.Attrs) []Song {
	ret := make([]Song, len(d), len(d)*12)
	for i := range d {
		ret[i] = convSong(d[i])
	}
	return ret
}

/*MpdClient represents mpd.Client for Player.*/
type MpdClient interface {
	Play(int) error
	Pause(bool) error
	Previous() error
	Next() error
	Ping() error
	Close() error
	ReadComments(string) (mpd.Attrs, error)
	CurrentSong() (mpd.Attrs, error)
	Status() (mpd.Attrs, error)
	ListAllInfo(string) ([]mpd.Attrs, error)
	PlaylistInfo(int, int) ([]mpd.Attrs, error)
}

/*Close mpd connection.*/
func (p *Player) Close() error {
	p.daemonStop <- true
	p.mpc.Close()
	return p.watcher.Close()
}

/*Comments returns mpd current song raw meta data.*/
func (p *Player) Comments() (mpd.Attrs, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.comments, p.commentsModified
}

/*Current returns mpd current song data.*/
func (p *Player) Current() (Song, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.current, p.currentModified
}

/*Status returns mpd current song data.*/
func (p *Player) Status() (PlayerStatus, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.status, p.currentModified
}

/*Library returns mpd library song data list.*/
func (p *Player) Library() ([]Song, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.library, p.libraryModified
}

/*Playlist returns mpd playlist song data list.*/
func (p *Player) Playlist() ([]Song, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.playlist, p.playlistModified
}

/*Pause song.*/
func (p *Player) Pause() error {
	return p.mpc.Pause(true)
}

/*Play or resume song.*/
func (p *Player) Play() error {
	return p.request(play)
}

/*Prev song.*/
func (p *Player) Prev() error {
	return p.request(prev)
}

/*Next song.*/
func (p *Player) Next() error {
	return p.request(next)
}

/*Nop ping daemon goroutine.*/
func (p *Player) Nop() {
	p.request(nop)
}

func (p *Player) start() (err error) {
	err = p.connect()
	if err != nil {
		return err
	}
	err = p.syncLibrary()
	if err != nil {
		p.Close()
		return
	}
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
	go p.ping()
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
			case ping:
				m.err <- p.mpc.Ping()
			case nop:
				m.err <- nil
			}
		}
	}
}

func (p *Player) ping() {
	for {
		time.Sleep(1)
		p.request(ping)
	}
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
	p.mpc = mpc
	watcher, err := mpd.NewWatcher(p.network, p.addr, "")
	if err != nil {
		mpc.Close()
		return err
	}
	p.watcher = *watcher
	return nil
}

func (p *Player) request(req mpcMessageType) error {
	r := new(mpcMessage)
	r.request = req
	r.err = make(chan error)
	p.daemonRequest <- r
	return <-r.err
}

func convStatus(song, status mpd.Attrs, s *PlayerStatus) {
	elapsed, err := strconv.ParseFloat(status["elapsed"], 64)
	if err != nil {
		elapsed = 0.0
	}
	volume, err := strconv.Atoi(status["volume"])
	if err != nil {
		volume = -1
	}
	songpos, err := strconv.Atoi(status["song"])
	if err != nil {
		songpos = 0
	}
	state := status["state"]
	if state == "" {
		state = "stopped"
	}
	songlength, err := strconv.Atoi(song["Time"])
	if err != nil {
		songlength = 0
	}
	s.Volume = volume
	s.Repeat = status["repeat"] == "1"
	s.Random = status["random"] == "1"
	s.Single = status["single"] == "1"
	s.Consume = status["consume"] == "1"
	s.State = state
	s.SongPos = songpos
	s.SongElapsed = float32(elapsed)
	s.SongLength = songlength
	s.LastModified = time.Now().Unix()
}

func (p *Player) syncCurrent() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	song, err := p.mpc.CurrentSong()
	if err != nil {
		return err
	}
	p.currentModified = time.Now()
	status, err := p.mpc.Status()
	if err != nil {
		return err
	}
	convStatus(song, status, &p.status)
	if p.comments == nil || p.current.File != song["file"] {
		comments, err := p.mpc.ReadComments(song["file"])
		if err != nil {
			return err
		}
		p.commentsModified = time.Now()
		p.comments = comments
	}

	p.current = convSong(song)
	return nil
}

func (p *Player) syncLibrary() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	library, err := p.mpc.ListAllInfo("/")
	if err != nil {
		return err
	}
	p.library = convSongs(library)
	p.libraryModified = time.Now()
	return nil
}

func (p *Player) syncPlaylist() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	playlist, err := p.mpc.PlaylistInfo(-1, -1)
	if err != nil {
		return err
	}
	p.playlist = convSongs(playlist)
	p.playlistModified = time.Now()
	return nil
}
