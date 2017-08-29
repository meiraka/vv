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
	watcherResponse  chan error
	daemonStop       chan bool
	daemonRequest    chan *playerMessage
	coverCache       map[string]string
	init             sync.Mutex
	mutex            sync.Mutex
	current          mpd.Attrs
	currentModified  time.Time
	status           PlayerStatus
	statusModified   time.Time
	stats            mpd.Attrs
	statsModifiled   time.Time
	library          []mpd.Attrs
	libraryModified  time.Time
	playlist         []mpd.Attrs
	playlistModified time.Time
	outputs          []mpd.Attrs
	outputsModified  time.Time
	notification     pubsub
}

/*Close mpd connection.*/
func (p *Player) Close() error {
	p.daemonStop <- true
	p.notification.ensureStop()
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
	return p.status, p.statusModified
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
	return p.request(func(mpc mpdClient) error {
		_, err := mpc.Update("")
		return err
	})
}

/*Pause song.*/
func (p *Player) Pause() error {
	return p.request(func(mpc mpdClient) error { return mpc.Pause(true) })
}

/*Play or resume song.*/
func (p *Player) Play() error {
	return p.request(func(mpc mpdClient) error { return mpc.Play(-1) })
}

/*Prev song.*/
func (p *Player) Prev() error {
	return p.request(func(mpc mpdClient) error { return mpc.Previous() })
}

/*Next song.*/
func (p *Player) Next() error {
	return p.request(func(mpc mpdClient) error { return mpc.Next() })
}

/*Volume set player volume.*/
func (p *Player) Volume(v int) error {
	return p.request(func(mpc mpdClient) error { return mpc.SetVolume(v) })
}

/*Repeat enable if true*/
func (p *Player) Repeat(on bool) error {
	return p.request(func(mpc mpdClient) error { return mpc.Repeat(on) })
}

/*Random enable if true*/
func (p *Player) Random(on bool) error {
	return p.request(func(mpc mpdClient) error { return mpc.Random(on) })
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
		return p.request(func(mpc mpdClient) error { return mpc.EnableOutput(id) })
	}
	return p.request(func(mpc mpdClient) error { return mpc.DisableOutput(id) })
}

/*SortPlaylist sorts playlist by song tag name.*/
func (p *Player) SortPlaylist(keys []string, uri string) (err error) {
	return p.request(func(mpc mpdClient) error { return p.mpcSortPlaylist(mpc, keys, uri) })
}

func (p *Player) mpcSortPlaylist(mpc mpdClient, keys []string, uri string) (err error) {
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
		cl := mpc.BeginCommandList()
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
			err = mpc.Play(i)
			return
		}
	}
	return
}

// Subscribe server events.
func (p *Player) Subscribe(c chan string) {
	p.notification.subscribe(c)
	p.updateSubscribers()
}

func (p *Player) updateSubscribers() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.stats == nil {
		return
	}
	p.stats["subscribers"] = strconv.Itoa(p.notification.count())
	newTime := time.Now().UTC()
	uptime, err := strconv.Atoi(p.stats["uptime"])
	if err != nil {
		p.statsModifiled = newTime
		p.notify("stats")
		return
	}
	p.stats["uptime"] = strconv.Itoa(uptime + int(newTime.Sub(p.statsModifiled)/time.Second))
	p.statsModifiled = newTime
	p.notify("stats")

}

// Unsubscribe server events.
func (p *Player) Unsubscribe(c chan string) {
	p.notification.unsubscribe(c)
	p.updateSubscribers()
}

func (p *Player) notify(n string) error {
	return p.notification.notify(n)
}

type playerMessage struct {
	request func(mpdClient) error
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
		p.daemonRequest = make(chan *playerMessage)
		p.coverCache = make(map[string]string)
		mpc, watcher := p.connect()
		go p.run(mpc, watcher)
	}
	return nil
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

func (p *Player) connect() (mpdClient, *mpd.Watcher) {
	mpc, err := playerMpdDial(p.network, p.addr, p.passwd)
	if err != nil {
		return nil, new(mpd.Watcher)
	}
	watcher, err := playerMpdNewWatcher(p.network, p.addr, p.passwd)
	if err != nil {
		mpc.Close()
		return nil, new(mpd.Watcher)
	}
	fs := []func(mpdClient) error{p.mpdUpdateLibrary, p.mpdUpdatePlaylist, p.mpdUpdateCurrentSong, p.mpdUpdateStatus, p.mpdUpdateStats, p.mpdUpdateOutputs}
	for i := range fs {
		err := fs[i](mpc)
		if err != nil {
			mpc.Close()
			watcher.Close()
			return nil, new(mpd.Watcher)
		}
	}
	return mpc, watcher
}

func (p *Player) run(mpc mpdClient, watcher *mpd.Watcher) {
	t := time.NewTicker(1 * time.Second)
	sendErr := func(ec chan error, err error) {
		if ec != nil {
			ec <- err
		}
	}
loop:
	for {
		select {
		case <-p.daemonStop:
			t.Stop()
			if mpc != nil {
				mpc.Close()
			}
			watcher.Close()
			break loop
		case m := <-p.daemonRequest:
			if mpc != nil {
				sendErr(m.err, m.request(mpc))
			} else {
				sendErr(m.err, errors.New("no connection"))
			}
		case subsystem := <-watcher.Event:
			switch subsystem {
			case "database":
				sendErr(p.watcherResponse, p.mpdUpdateLibrary(mpc))
				sendErr(p.watcherResponse, p.mpdUpdateStats(mpc))
			case "playlist":
				sendErr(p.watcherResponse, p.mpdUpdatePlaylist(mpc))
			case "player":
				sendErr(p.watcherResponse, p.mpdUpdateCurrentSong(mpc))
				sendErr(p.watcherResponse, p.mpdUpdateStatus(mpc))
				sendErr(p.watcherResponse, p.mpdUpdateStats(mpc))
			case "mixer", "options":
				sendErr(p.watcherResponse, p.mpdUpdateCurrentSong(mpc))
				sendErr(p.watcherResponse, p.mpdUpdateStatus(mpc))
			case "update":
				sendErr(p.watcherResponse, p.mpdUpdateStatus(mpc))
			case "output":
				sendErr(p.watcherResponse, p.mpdUpdateOutputs(mpc))
			}
		case <-t.C:
			if mpc == nil || mpc.Ping() != nil {
				if mpc != nil {
					mpc.Close()
				}
				mpc, watcher = p.connect()
			}
		}
	}
}

func (p *Player) request(f func(mpdClient) error) error {
	ec := make(chan error)
	p.requestAsync(f, ec)
	return <-ec
}

func (p *Player) requestAsync(f func(mpdClient) error, ec chan error) {
	r := new(playerMessage)
	r.request = f
	r.err = ec
	p.daemonRequest <- r
}

func (p *Player) mpdUpdateCurrentSong(mpc mpdClient) error {
	song, err := mpc.CurrentSong()
	if err != nil {
		return err
	}
	if p.current["file"] != song["file"] {
		p.mutex.Lock()
		defer p.mutex.Unlock()
		p.current = songAddReadableData(song)
		p.current = songFindCover(p.current, p.musicDirectory, p.coverCache)
		p.currentModified = time.Now().UTC()
		return p.notify("current")
	}
	return nil
}

func (p *Player) mpdUpdateStatus(mpc mpdClient) error {
	status, err := mpc.Status()
	if err != nil {
		return err
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.statusModified = time.Now().UTC()
	p.status = convStatus(status)
	return p.notify("status")
}

func (p *Player) mpdUpdateStats(mpc mpdClient) error {
	stats, err := mpc.Stats()
	if err != nil {
		return err
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if stats != nil {
		stats["subscribers"] = strconv.Itoa(p.notification.count())
	}
	p.stats = stats
	p.statsModifiled = time.Now().UTC()
	return p.notify("stats")
}

func (p *Player) mpdUpdateLibrary(mpc mpdClient) error {
	library, err := mpc.ListAllInfo("/")
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
	p.libraryModified = time.Now().UTC()
	return p.notify("library")
}

func (p *Player) mpdUpdatePlaylist(mpc mpdClient) error {
	playlist, err := mpc.PlaylistInfo(-1, -1)
	if err != nil {
		return err
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.playlist = songsAddReadableData(playlist)
	p.playlist = songsFindCover(p.playlist, p.musicDirectory, p.coverCache)
	p.playlistModified = time.Now().UTC()
	return p.notify("playlist")
}

func (p *Player) mpdUpdateOutputs(mpc mpdClient) error {
	outputs, err := mpc.ListOutputs()
	if err != nil {
		return err
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.outputs = outputs
	p.outputsModified = time.Now().UTC()
	return p.notify("outputs")
}

type pubsub struct {
	m               sync.Mutex
	subscribeChan   chan chan string
	unsubscribeChan chan chan string
	countChan       chan chan int
	notifyChan      chan pubsubNotify
	stopChan        chan struct{}
}

type pubsubNotify struct {
	message string
	errChan chan error
}

func (p *pubsub) ensureStart() {
	p.m.Lock()
	defer p.m.Unlock()
	if p.subscribeChan == nil {
		p.subscribeChan = make(chan chan string)
		p.unsubscribeChan = make(chan chan string)
		p.countChan = make(chan chan int)
		p.notifyChan = make(chan pubsubNotify)
		p.stopChan = make(chan struct{})
		go p.run()
	}
}

func (p *pubsub) ensureStop() {
	p.ensureStart()
	p.stopChan <- struct{}{}
}

func (p *pubsub) run() {
	subscribers := []chan string{}
loop:
	for {
		select {
		case c := <-p.subscribeChan:
			subscribers = append(subscribers, c)
		case c := <-p.unsubscribeChan:
			newSubscribers := []chan string{}
			for _, o := range subscribers {
				if o != c {
					newSubscribers = append(newSubscribers, o)
				}
			}
			subscribers = newSubscribers
		case pn := <-p.notifyChan:
			errcnt := 0
			for _, c := range subscribers {
				select {
				case c <- pn.message:
				default:
					errcnt++
				}
			}
			if errcnt > 0 {
				pn.errChan <- fmt.Errorf("failed to send %s notify, %d", pn.message, errcnt)
			} else {
				pn.errChan <- nil
			}
		case c := <-p.countChan:
			c <- len(subscribers)
		case <-p.stopChan:
			break loop
		}
	}
}

func (p *pubsub) subscribe(c chan string) {
	p.ensureStart()
	p.subscribeChan <- c
}

func (p *pubsub) unsubscribe(c chan string) {
	p.ensureStart()
	p.unsubscribeChan <- c
}

func (p *pubsub) notify(s string) error {
	p.ensureStart()
	message := pubsubNotify{s, make(chan error)}
	p.notifyChan <- message
	return <-message.errChan
}

func (p *pubsub) count() int {
	p.ensureStart()
	ci := make(chan int)
	p.countChan <- ci
	return <-ci
}
