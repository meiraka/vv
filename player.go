package main

import (
	"fmt"
	"github.com/fhs/gompd/mpd"
	"sort"
	"strconv"
	"sync"
	"time"
)

/*Dial Connects to mpd server.*/
func Dial(network, addr, passwd string) (*Player, error) {
	p := new(Player)
	p.mutex = new(sync.Mutex)
	p.daemonStop = make(chan bool)
	p.daemonRequest = make(chan *playerMessage)
	p.network = network
	p.addr = addr
	p.passwd = passwd
	return p, p.start()
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
	mutex            *sync.Mutex
	current          mpd.Attrs
	currentModified  time.Time
	status           PlayerStatus
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
	LastModified int64   `json:"last_modified"`
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

/*Volume set player volume.*/
func (p *Player) Volume(v int) error {
	r := new(playerMessage)
	r.request = volume
	r.i = v
	r.err = make(chan error)
	p.daemonRequest <- r
	return <-r.err
}

/*Repeat enable if true*/
func (p *Player) Repeat(on bool) error {
	return p.requestBool(repeat, on)
}

/*Random enable if true*/
func (p *Player) Random(on bool) error {
	return p.requestBool(random, on)
}

type playerMessageType int

const (
	updateLibrary playerMessageType = iota
	updatePlaylist
	updateCurrent
	sortPlaylist
	pause
	prev
	play
	next
	ping
	volume
	repeat
	random
)

type playerMessage struct {
	request     playerMessageType
	requestData []string
	i           int
	b           bool
	err         chan error
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
}

func (p *Player) start() (err error) {
	err = p.connect()
	if err != nil {
		return err
	}
	err = p.updateLibrary()
	if err != nil {
		p.Close()
		return
	}
	err = p.updatePlaylist()
	if err != nil {
		p.Close()
		return
	}
	err = p.updateCurrent()
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
			switch m.request {
			case prev:
				sendErr(m.err, p.mpc.Previous())
			case pause:
				sendErr(m.err, p.mpc.Pause(true))
			case play:
				sendErr(m.err, p.mpc.Play(-1))
			case next:
				sendErr(m.err, p.mpc.Next())
			case volume:
				sendErr(m.err, p.mpc.SetVolume(m.i))
			case repeat:
				sendErr(m.err, p.mpc.Repeat(m.b))
			case random:
				sendErr(m.err, p.mpc.Random(m.b))
			case updateLibrary:
				sendErr(m.err, p.updateLibrary())
			case updatePlaylist:
				sendErr(m.err, p.updatePlaylist())
			case updateCurrent:
				sendErr(m.err, p.updateCurrent())
			case sortPlaylist:
				sendErr(m.err, p.sortPlaylist(m.requestData))
			case ping:
				sendErr(m.err, p.mpc.Ping())
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
			p.requestAsync(updateLibrary, p.watcherResponse)
		case "playlist":
			p.requestAsync(updatePlaylist, p.watcherResponse)
		case "player", "mixer", "options":
			p.requestAsync(updateCurrent, p.watcherResponse)
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
func (p *Player) request(req playerMessageType) error {
	ec := make(chan error)
	p.requestAsync(req, ec)
	return <-ec
}

func (p *Player) requestBool(req playerMessageType, b bool) error {
	r := new(playerMessage)
	r.request = req
	r.b = b
	r.err = make(chan error)
	p.daemonRequest <- r
	return <-r.err
}

func (p *Player) requestAsync(req playerMessageType, ec chan error) {
	r := new(playerMessage)
	r.request = req
	r.err = ec
	p.daemonRequest <- r
}

func (p *Player) requestData(req playerMessageType, data []string) error {
	r := new(playerMessage)
	r.request = req
	r.requestData = data
	r.err = make(chan error)
	p.daemonRequest <- r
	return <-r.err
}

func convStatus(status mpd.Attrs) PlayerStatus {
	volume, err := strconv.Atoi(status["volume"])
	if err != nil {
		volume = -1
	}
	repeat := status["repeat"] == "1"
	random := status["random"] == "1"
	single := status["single"] == "1"
	consume := status["consume"] == "1"
	state := status["state"]
	if state == "" {
		state = "stopped"
	}
	songpos, err := strconv.Atoi(status["song"])
	if err != nil {
		songpos = 0
	}
	elapsed, err := strconv.ParseFloat(status["elapsed"], 64)
	if err != nil {
		elapsed = 0.0
	}
	lastModified := time.Now().Unix()
	return PlayerStatus{
		volume,
		repeat,
		random,
		single,
		consume,
		state,
		songpos,
		float32(elapsed),
		lastModified,
	}

}

/*SortPlaylist sorts playlist by song tag name.*/
func (p *Player) SortPlaylist(keys []string, uri string) (err error) {
	keys = append(keys, uri)
	return p.requestData(sortPlaylist, keys)
}

func (p *Player) sortPlaylist(keys []string) (err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	uri := keys[len(keys)-1]
	keys = keys[:len(keys)-1]
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
	for i := range p.playlist {
		if p.playlist[i]["file"] == uri {
			err = p.mpc.Play(i)
			return
		}
	}
	return
}

func (p *Player) updateCurrentSong() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	song, err := p.mpc.CurrentSong()
	if err != nil {
		return err
	}
	c := songAddReadableData(song)
	cm := time.Now()
	if p.current["file"] != c["file"] {
		p.current = c
		p.currentModified = cm
	}
	return nil
}

func (p *Player) updateStatus() error {
	status, err := p.mpc.Status()
	if err != nil {
		return err
	}
	p.status = convStatus(status)
	return nil
}

func (p *Player) updateCurrent() error {
	err := p.updateCurrentSong()
	if err != nil {
		return err
	}
	return p.updateStatus()
}

func (p *Player) updateLibrary() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	library, err := p.mpc.ListAllInfo("/")
	if err != nil {
		return err
	}
	p.library = songsAddReadableData(library)
	p.libraryModified = time.Now()
	return nil
}

func (p *Player) updatePlaylist() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	playlist, err := p.mpc.PlaylistInfo(-1, -1)
	if err != nil {
		return err
	}
	p.playlist = songsAddReadableData(playlist)
	p.playlistModified = time.Now()
	return nil
}
