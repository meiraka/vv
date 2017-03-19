package main

import (
	"fmt"
	"github.com/fhs/gompd/mpd"
	"strconv"
	"strings"
	"sync"
	"time"
)

type mpcMessageType int

const (
	syncLibrary mpcMessageType = iota
	syncPlaylist
	syncCurrent
	pause
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
	watcherResponse  chan error
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
	Artist          string `json:"artist"`
	ArtistSort      string `json:"artistsort"`
	Album           string `json:"album"`
	AlbumSort       string `json:"albumsort"`
	AlbumArtist     string `json:"albumartist"`
	AlbumArtistSort string `json:"albumartistsort"`
	Title           string `json:"title"`
	Track           int    `json:"track"`
	TrackNumber     string `json:"tracknumber"`
	Genre           string `json:"genre"`
	Date            string `json:"date"`
	Composer        string `json:"composer"`
	Performer       string `json:"performer"`
	Comment         string `json:"comment"`
	Disc            int    `json:"disc"`
	DiscNumber      string `json:"discnumber"`
	Time            int    `json:"time"`
	Length          string `json:"length"`
	File            string `json:"file"`
}

func convSong(d mpd.Attrs) (s Song) {
	checks := []string{
		"Artist",
		"Album",
		"Title",
		"Track",
		"Genre",
		"Date",
		"Composer",
		"Performer",
		"Comment",
	}
	for i := range checks {
		if _, ok := d[checks[i]]; !ok {
			d[checks[i]] = "[no " + checks[i] + "]"
		}
	}
	s.Artist = d["Artist"]
	s.ArtistSort = d["ArtistSort"]
	if s.ArtistSort == "" {
		s.ArtistSort = s.Artist
	}
	s.Album = d["Album"]
	s.AlbumSort = d["AlbumSort"]
	if s.AlbumSort == "" {
		s.AlbumSort = s.Album
	}
	s.AlbumArtist = d["AlbumArtist"]
	if s.AlbumArtist == "" {
		s.AlbumArtist = s.Artist
	}
	s.AlbumArtistSort = d["AlbumArtistSort"]
	if s.AlbumArtistSort == "" {
		s.AlbumArtistSort = s.AlbumArtist
	}
	s.Title = d["Title"]
	track, err := strconv.Atoi(d["Track"])
	if err != nil {
		track = -1
	}
	s.Track = track
	s.TrackNumber = fmt.Sprintf("%04d", track)
	s.Genre = d["Genre"]
	s.Date = d["Date"]
	s.Composer = d["Composer"]
	s.Performer = d["Performer"]
	s.Comment = d["Comment"]
	disc, err := strconv.Atoi(d["Disc"])
	if err != nil {
		disc = 1
	}
	s.Disc = disc
	s.DiscNumber = fmt.Sprintf("%04d", disc)
	time, err := strconv.Atoi(d["Time"])
	if err != nil {
		time = 0
	}
	s.Time = time
	s.Length = fmt.Sprintf("%02d:%02d", time/60, time%60)
	s.File = d["file"]
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
	return p.request(pause)
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
			case pause:
				m.err <- p.mpc.Pause(true)
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
	watchBack := func(err error) {
		if p.watcherResponse != nil {
			p.watcherResponse <- err
		}
	}
	for subsystem := range p.watcher.Event {
		switch subsystem {
		case "database":
			watchBack(p.request(syncLibrary))
		case "playlist":
			watchBack(p.request(syncPlaylist))
		case "player":
			watchBack(p.request(syncCurrent))
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

func songString(s Song, keys []string) string {
	sp := make([]string, len(keys))
	for i := range keys {
		key := keys[i]
		if key == "albumartist" {
			sp = append(sp, s.AlbumArtist)
		} else if key == "album" {
			sp = append(sp, s.Album)
		} else if key == "artist" {
			sp = append(sp, s.Artist)
		} else if key == "date" {
			sp = append(sp, s.Date)
		} else if key == "genre" {
			sp = append(sp, s.Genre)
		} else if key == "tracknumber" {
			sp = append(sp, s.TrackNumber)
		} else if key == "title" {
			sp = append(sp, s.Title)
		} else if key == "file" {
			sp = append(sp, s.File)
		} else if key == "discno" {
			sp = append(sp, s.DiscNumber)
		}
	}
	return strings.Join(sp, "")
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
