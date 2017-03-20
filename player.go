package main

import (
	"fmt"
	"github.com/fhs/gompd/mpd"
	"sort"
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
	sortPlaylist
	pause
	prev
	play
	next
	ping
	nop
)

type mpcMessage struct {
	request     mpcMessageType
	requestData []string
	err         chan error
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
	current          mpd.Attrs
	currentModified  time.Time
	status           PlayerStatus
	comments         mpd.Attrs
	commentsModified time.Time
	library          []mpd.Attrs
	libraryModified  time.Time
	playlist         []mpd.Attrs
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
	BeginCommandList() *mpd.CommandList
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
func (p *Player) Current() (mpd.Attrs, time.Time) {
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
			case sortPlaylist:
				m.err <- p.sortPlaylist(m.requestData)
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

func (p *Player) requestData(req mpcMessageType, data []string) error {
	r := new(mpcMessage)
	r.request = req
	r.requestData = data
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

/*SortPlaylist sorts playlist by song tag name.*/
func (p *Player) SortPlaylist(keys []string, uri string) (err error) {
	keys = append(keys, uri)
	return p.requestData(sortPlaylist, keys)
}

func (p *Player) sortPlaylist(keys []string) (err error) {
	uri := keys[len(keys)-1]
	keys = keys[:len(keys)-1]
	p.mutex.Lock()
	defer p.mutex.Unlock()
	err = nil
	sort.Slice(p.library, func(i, j int) bool {
		return songString(p.library[i], keys) < songString(p.library[j], keys)
	})
	update := false
	if len(p.library) != len(p.playlist) {
		update = true
		fmt.Printf("length not match")
	} else {
		for i := range p.library {
			n := p.library[i]["file"]
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
		for i := range p.library {
			cl.Add(p.library[i]["file"])
		}
		err = cl.End()
	}
	if err != nil {
		return
	}
	for i := range p.playlist {
		if p.playlist[i]["file"] == uri {
			err = p.mpc.Play(i)
			return
		}
	}
	return
}

func songTag(s mpd.Attrs, keys []string) string {
	for i := range keys {
		key := keys[i]
		if _, ok := s[key]; ok {
			return s[key]
		}
	}
	return " "
}

func songString(s mpd.Attrs, keys []string) string {
	sp := make([]string, len(keys))
	for i := range keys {
		key := keys[i]
		if _, ok := s[key]; ok {
			sp = append(sp, s[key])
		} else if key == "AlbumSort" {
			sp = append(sp, songTag(s, []string{"Album"}))
		} else if key == "ArtistSort" {
			sp = append(sp, songTag(s, []string{"Artist"}))
		} else if key == "AlbumArtist" {
			sp = append(sp, songTag(s, []string{"Artist"}))
		} else if key == "AlbumArtistSort" {
			sp = append(sp, songTag(s, []string{"AlbumArtist", "Artist"}))
		} else {
			sp = append(sp, " ")
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
	if song["file"] != "" && p.comments == nil || p.current["file"] != song["file"] {
		comments, err := p.mpc.ReadComments(song["file"])
		if err != nil {
			return err
		}
		p.commentsModified = time.Now()
		p.comments = comments
	}

	p.current = makeAdditiveSongData(song)
	return nil
}

func (p *Player) syncLibrary() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	library, err := p.mpc.ListAllInfo("/")
	if err != nil {
		return err
	}
	p.library = makeAdditiveSongsData(library)
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
	p.playlist = makeAdditiveSongsData(playlist)
	p.playlistModified = time.Now()
	return nil
}

func makeAdditiveSongData(p mpd.Attrs) mpd.Attrs {
	track, err := strconv.Atoi(p["Track"])
	if err != nil {
		track = 0
	}
	p["TrackNumber"] = fmt.Sprintf("%04d", track)
	disc, err := strconv.Atoi(p["Disc"])
	if err != nil {
		disc = 1
	}
	p["DiscNumber"] = fmt.Sprintf("%04d", disc)

	t, err := strconv.Atoi(p["Time"])
	if err != nil {
		t = 0
	}
	p["Length"] = fmt.Sprintf("%02d:%02d", t/60, t%60)
	return p
}

func makeAdditiveSongsData(ps []mpd.Attrs) []mpd.Attrs {
	for i := range ps {
		ps[i] = makeAdditiveSongData(ps[i])
	}
	return ps
}
