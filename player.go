package main

import (
	"errors"
	"fmt"
	"github.com/fhs/gompd/mpd"
	"sort"
	"strconv"
	"sync"
	"time"
)

/*Dial Connects to mpd server.*/
func Dial(network, addr, passwd, musicDirectory string) (*Player, error) {
	p := new(Player)
	p.network = network
	p.addr = addr
	p.passwd = passwd
	p.musicDirectory = musicDirectory
	return p, p.initIfNot()
}

/*Player represents mpd control interface.*/
type Player struct {
	network          string
	addr             string
	passwd           string
	musicDirectory   string
	mpc              mpdClient
	watcher          mpd.Watcher
	watcherResponse  chan error
	daemonStop       chan bool
	pingStop         chan bool
	daemonRequest    chan *playerMessage
	coverCache       map[string]string
	init             sync.Mutex
	mutex            sync.Mutex
	current          mpd.Attrs
	currentModified  time.Time
	status           PlayerStatus
	stats            mpd.Attrs
	statsModifiled   time.Time
	library          []mpd.Attrs
	libraryModified  time.Time
	playlist         []mpd.Attrs
	playlistModified time.Time
	outputs          []mpd.Attrs
	outputsModified  time.Time
	subscribers      []chan string
	subscribersMutex sync.Mutex
}

/*Close mpd connection.*/
func (p *Player) Close() error {
	p.pingStop <- true
	p.clearConn()
	p.daemonStop <- true
	return nil
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

/*Stats returns mpd statistics.*/
func (p *Player) Stats() (mpd.Attrs, time.Time) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.stats, p.statsModifiled
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

/*RescanLibrary scans music directory and update library database.*/
func (p *Player) RescanLibrary() error {
	return p.request(func() error {
		_, err := p.mpc.Update("")
		return err
	})
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
	return p.request(func() error { return p.mpc.Previous() })
}

/*Next song.*/
func (p *Player) Next() error {
	return p.request(func() error { return p.mpc.Next() })
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
	} else {
		for i := range l {
			n := l[i]["file"]
			o := p.playlist[i]["file"]
			if n != o {
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

// Subscribe server events.
func (p *Player) Subscribe(c chan string) {
	p.subscribersMutex.Lock()
	defer p.subscribersMutex.Unlock()
	if p.subscribers == nil {
		p.subscribers = []chan string{c}
		return
	}
	p.subscribers = append(p.subscribers, c)
}

// Unsubscribe server events.
func (p *Player) Unsubscribe(c chan string) {
	p.subscribersMutex.Lock()
	defer p.subscribersMutex.Unlock()
	if p.subscribers == nil {
		return
	}
	newSubscribers := []chan string{}
	for _, s := range p.subscribers {
		if s != c {
			newSubscribers = append(newSubscribers, s)
		}
	}
	p.subscribers = newSubscribers
}

func (p *Player) notify(n string) error {
	p.subscribersMutex.Lock()
	defer p.subscribersMutex.Unlock()
	if p.subscribers == nil {
		return nil
	}

	errcnt := 0
	for _, s := range p.subscribers {
		select {
		case s <- n:
		default:
			errcnt++
		}
	}
	if errcnt != 0 {
		return fmt.Errorf("failed to send %s notify, %d", n, errcnt)
	}
	return nil
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
	Stats() (mpd.Attrs, error)
	ListAllInfo(string) ([]mpd.Attrs, error)
	PlaylistInfo(int, int) ([]mpd.Attrs, error)
	BeginCommandList() *mpd.CommandList
	ListOutputs() ([]mpd.Attrs, error)
	DisableOutput(int) error
	EnableOutput(int) error
	Update(string) (int, error)
}

func (p *Player) initIfNot() error {
	p.init.Lock()
	defer p.init.Unlock()
	if p.daemonStop == nil {
		p.daemonStop = make(chan bool)
		p.pingStop = make(chan bool)
		p.daemonRequest = make(chan *playerMessage)
		p.coverCache = make(map[string]string)
		go p.daemon()
		p.clearConn()
		p.initConn()
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
			if p.mpc != nil {
				sendErr(m.err, m.request())
			} else {
				sendErr(m.err, errors.New("no connection"))
			}
		}
	}
}

func (p *Player) ping() {
	last := false
	t := time.NewTicker(1 * time.Second)
loop:
	for {
		select {
		case <-p.pingStop:
			break loop
		case <-t.C:
			err := p.request(func() error { return p.mpc.Ping() })
			if err != nil {
				if last {
					last = false
				}
				p.clearConn()
				p.initConn()
			} else if !last {
				last = true
			}
		}
	}
	t.Stop()
}

func (p *Player) watch() {
	for subsystem := range p.watcher.Event {
		switch subsystem {
		case "database":
			p.requestAsync(p.updateLibrary, p.watcherResponse)
			p.requestAsync(p.updateStats, p.watcherResponse)
		case "playlist":
			p.requestAsync(p.updatePlaylist, p.watcherResponse)
		case "player":
			p.requestAsync(p.updateCurrentSong, p.watcherResponse)
			p.requestAsync(p.updateStatus, p.watcherResponse)
			p.requestAsync(p.updateStats, p.watcherResponse)
		case "mixer", "options":
			p.requestAsync(p.updateCurrentSong, p.watcherResponse)
			p.requestAsync(p.updateStatus, p.watcherResponse)
		case "update":
			p.requestAsync(p.updateStatus, p.watcherResponse)
		case "output":
			p.requestAsync(p.updateOutputs, p.watcherResponse)
		}
	}
}

func playerRealMpdDial(net, addr, passwd string) (mpdClient, error) {
	return mpd.DialAuthenticated(net, addr, passwd)
}

func playerRealMpdNewWatcher(net, addr, passwd string) (*mpd.Watcher, error) {
	return mpd.NewWatcher(net, addr, passwd)
}

func playerRealMpdWatcherClose(w mpd.Watcher) error {
	return w.Close()
}

var playerMpdDial = playerRealMpdDial
var playerMpdNewWatcher = playerRealMpdNewWatcher
var playerMpdWatcherClose = playerRealMpdWatcherClose

func (p *Player) clearConn() {
	if p.mpc != nil {
		p.daemonStop <- true
		p.mpc.Close()
		p.mpc = nil
		go p.daemon()
	}
	if p.watcher.Event != nil {
		playerMpdWatcherClose(p.watcher)
		p.watcher.Event = nil
	}
}

func (p *Player) initConn() error {
	fs := []func() error{p.connect, p.updateLibrary, p.updatePlaylist, p.updateCurrentSong, p.updateStatus, p.updateStats, p.updateOutputs}
	for i := range fs {
		err := fs[i]()
		if err != nil {
			return err
		}
	}
	go p.watch()
	return nil
}

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
		p.current = songFindCover(p.current, p.musicDirectory, p.coverCache)
		p.currentModified = time.Now()
		return p.notify("current")
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
	return p.notify("status")
}

func (p *Player) updateStats() error {
	stats, err := p.mpc.Stats()
	if err != nil {
		return err
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.stats = stats
	p.statsModifiled = time.Now()
	return p.notify("stats")
}

func (p *Player) updateLibrary() error {
	library, err := p.mpc.ListAllInfo("/")
	if err != nil {
		return err
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.library = songsAddReadableData(library)
	p.library = songsFindCover(p.library, p.musicDirectory, p.coverCache)
	for i := range p.library {
		p.library[i]["Pos"] = strconv.Itoa(i)
	}
	p.libraryModified = time.Now()
	return p.notify("library")
}

func (p *Player) updatePlaylist() error {
	playlist, err := p.mpc.PlaylistInfo(-1, -1)
	if err != nil {
		return err
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.playlist = songsAddReadableData(playlist)
	p.playlist = songsFindCover(p.playlist, p.musicDirectory, p.coverCache)
	p.playlistModified = time.Now()
	return p.notify("playlist")
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
	return p.notify("outputs")
}
