package main

import (
	"errors"
	"fmt"
	"github.com/meiraka/gompd/mpd"
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
	network         string
	addr            string
	passwd          string
	musicDirectory  string
	watcherResponse chan error
	daemonStop      chan bool
	daemonRequest   chan *playerMessage
	coverCache      map[string]string
	init            sync.Mutex
	mutex           sync.Mutex
	status          statusStorage
	current         songStorage
	stats           mapStorage
	library         songsStorage
	librarySort     songsStorage
	playlist        songsStorage
	outputs         sliceMapStorage
	notification    pubsub
}

/*Close mpd connection.*/
func (p *Player) Close() error {
	p.daemonStop <- true
	p.notification.ensureStop()
	return nil
}

/*Current returns mpd current song data.*/
func (p *Player) Current() (Song, time.Time) {
	return p.current.get()
}

/*Library returns mpd library song data list.*/
func (p *Player) Library() ([]Song, time.Time) {
	return p.library.get()
}

/*Next song.*/
func (p *Player) Next() error {
	return p.request(func(mpc mpdClient) error { return mpc.Next() })
}

/*Output enable output if true.*/
func (p *Player) Output(id int, on bool) error {
	if on {
		return p.request(func(mpc mpdClient) error { return mpc.EnableOutput(id) })
	}
	return p.request(func(mpc mpdClient) error { return mpc.DisableOutput(id) })
}

/*Outputs return output device list.*/
func (p *Player) Outputs() ([]mpd.Attrs, time.Time) {
	return p.outputs.get()
}

/*Pause song.*/
func (p *Player) Pause() error {
	return p.request(func(mpc mpdClient) error { return mpc.Pause(true) })
}

/*Play or resume song.*/
func (p *Player) Play() error {
	return p.request(func(mpc mpdClient) error { return mpc.Play(-1) })
}

/*Playlist returns mpd playlist song data list.*/
func (p *Player) Playlist() ([]Song, time.Time) {
	return p.playlist.get()
}

/*Prev song.*/
func (p *Player) Prev() error {
	return p.request(func(mpc mpdClient) error { return mpc.Previous() })
}

/*Random enable if true*/
func (p *Player) Random(on bool) error {
	return p.request(func(mpc mpdClient) error { return mpc.Random(on) })
}

/*Repeat enable if true*/
func (p *Player) Repeat(on bool) error {
	return p.request(func(mpc mpdClient) error { return mpc.Repeat(on) })
}

/*RescanLibrary scans music directory and update library database.*/
func (p *Player) RescanLibrary() error {
	return p.request(func(mpc mpdClient) error {
		_, err := mpc.Update("")
		return err
	})
}

/*SortPlaylist sorts playlist by song tag name.*/
func (p *Player) SortPlaylist(keys []string, uri string, filters [][]string) (err error) {
	return p.request(func(mpc mpdClient) error { return p.sortPlaylist(mpc, keys, uri, filters) })
}

func (p *Player) sortPlaylist(mpc mpdClient, keys []string, uri string, filters [][]string) error {
	return p.librarySort.lock(func(masterLibrary []Song, _ time.Time) error {
		update := false
		library := SortSongsUniq(masterLibrary, keys)
		library = WeakFilterSongs(library, filters, 9999)
		p.playlist.lock(func(playlist []Song, _ time.Time) error {
			if len(library) != len(playlist) {
				update = true
				return nil
			}
			for i := range library {
				n := library[i]["file"][0]
				o := playlist[i]["file"][0]
				if n != o {
					update = true
					break
				}
			}
			return nil
		})
		if update {
			cl := mpc.BeginCommandList()
			cl.Clear()
			for i := range library {
				cl.Add(library[i]["file"][0])
			}
			err := cl.End()
			if err != nil {
				return err
			}
		}
		for i := range library {
			if library[i]["file"][0] == uri {
				return mpc.Play(i)
			}
		}
		return nil
	})
}

/*Status returns mpd current song data.*/
func (p *Player) Status() (Status, time.Time) {
	return p.status.get()
}

/*Stats returns mpd statistics.*/
func (p *Player) Stats() (mpd.Attrs, time.Time) {
	return p.stats.get()
}

// Subscribe server events.
func (p *Player) Subscribe(c chan string) {
	p.notification.subscribe(c)
	p.updateSubscribers()
}

// Unsubscribe server events.
func (p *Player) Unsubscribe(c chan string) {
	p.notification.unsubscribe(c)
	p.updateSubscribers()
}

/*Volume set player volume.*/
func (p *Player) Volume(v int) error {
	return p.request(func(mpc mpdClient) error { return mpc.SetVolume(v) })
}

func (p *Player) updateSubscribers() {
	stats, modified := p.stats.get()
	newStats := mpd.Attrs{}
	for k, v := range stats {
		newStats[k] = v
	}
	newStats["subscribers"] = strconv.Itoa(p.notification.count())
	newTime := time.Now().UTC()
	uptime, err := strconv.Atoi(newStats["uptime"])
	if err != nil {
		p.stats.set(newStats, newTime)
		p.notify("stats")
		return
	}
	newStats["uptime"] = strconv.Itoa(uptime + int(newTime.Sub(modified)/time.Second))
	p.stats.set(newStats, newTime)
	p.notify("stats")

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
	CurrentSongTags() (mpd.Tags, error)
	Status() (mpd.Attrs, error)
	Stats() (mpd.Attrs, error)
	ListAllInfoTags(string) ([]mpd.Tags, error)
	PlaylistInfoTags(int, int) ([]mpd.Tags, error)
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
	fs := []func(mpdClient) error{p.updateLibrary, p.updatePlaylist, p.updateCurrentSong, p.updateStatus, p.updateStats, p.updateOutputs}
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
				sendErr(p.watcherResponse, p.updateLibrary(mpc))
				sendErr(p.watcherResponse, p.updateStats(mpc))
			case "playlist":
				sendErr(p.watcherResponse, p.updatePlaylist(mpc))
			case "player":
				sendErr(p.watcherResponse, p.updateCurrentSong(mpc))
				sendErr(p.watcherResponse, p.updateStatus(mpc))
				sendErr(p.watcherResponse, p.updateStats(mpc))
			case "mixer", "options":
				sendErr(p.watcherResponse, p.updateCurrentSong(mpc))
				sendErr(p.watcherResponse, p.updateStatus(mpc))
			case "update":
				sendErr(p.watcherResponse, p.updateStatus(mpc))
			case "output":
				sendErr(p.watcherResponse, p.updateOutputs(mpc))
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

func (p *Player) updateCurrentSong(mpc mpdClient) error {
	tags, err := mpc.CurrentSongTags()
	if err != nil {
		return err
	}
	if _, found := tags["file"]; !found {
		return nil
	}
	current, _ := p.current.get()
	if len(current["file"]) == 0 || current["file"][0] != tags["file"][0] {
		p.mutex.Lock()
		song := MakeSong(tags, p.musicDirectory, "cover.*", p.coverCache)
		p.mutex.Unlock()
		p.current.set(song, time.Now().UTC())
		return p.notify("current")
	}
	return nil
}

func (p *Player) updateStatus(mpc mpdClient) error {
	status, err := mpc.Status()
	if err != nil {
		return err
	}
	p.status.set(MakeStatus(status), time.Now().UTC())
	return p.notify("status")
}

func (p *Player) updateStats(mpc mpdClient) error {
	stats, err := mpc.Stats()
	if err != nil {
		return err
	}
	stats["subscribers"] = strconv.Itoa(p.notification.count())
	p.stats.set(stats, time.Now().UTC())
	return p.notify("stats")
}

func (p *Player) updateLibrary(mpc mpdClient) error {
	libraryTags, err := mpc.ListAllInfoTags("/")
	if err != nil {
		return err
	}
	p.mutex.Lock()
	library := MakeSongs(libraryTags, p.musicDirectory, "cover.*", p.coverCache)
	p.mutex.Unlock()
	for i := range library {
		library[i]["Pos"] = []string{strconv.Itoa(i)}
	}
	librarySort := make([]Song, len(library))
	copy(librarySort, library)
	p.library.set(library, time.Now().UTC())
	p.librarySort.set(librarySort, time.Now().UTC())
	return p.notify("library")
}

func (p *Player) updatePlaylist(mpc mpdClient) error {
	playlistTags, err := mpc.PlaylistInfoTags(-1, -1)
	if err != nil {
		return err
	}
	p.mutex.Lock()
	playlist := MakeSongs(playlistTags, p.musicDirectory, "cover.*", p.coverCache)
	p.mutex.Unlock()
	p.playlist.set(playlist, time.Now().UTC())
	return p.notify("playlist")
}

func (p *Player) updateOutputs(mpc mpdClient) error {
	outputs, err := mpc.ListOutputs()
	if err != nil {
		return err
	}
	p.outputs.set(outputs, time.Now().UTC())
	return p.notify("outputs")
}

type songsStorage struct {
	m        sync.Mutex
	storage  []Song
	modified time.Time
}

func (s *songsStorage) set(l []Song, t time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	s.storage = l
	s.modified = t
}

func (s *songsStorage) get() ([]Song, time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.storage == nil {
		s.storage = []Song{}
	}
	return s.storage, s.modified
}

func (s *songsStorage) lock(f func([]Song, time.Time) error) error {
	s.m.Lock()
	defer s.m.Unlock()
	if s.storage == nil {
		s.storage = []Song{}
	}
	return f(s.storage, s.modified)
}

type songStorage struct {
	m        sync.Mutex
	storage  Song
	modified time.Time
}

func (s *songStorage) set(l Song, t time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	s.storage = l
	s.modified = t
}

func (s *songStorage) get() (Song, time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.storage == nil {
		s.storage = Song{}
	}
	return s.storage, s.modified
}

func (s *songStorage) lock(f func(Song, time.Time) error) error {
	s.m.Lock()
	defer s.m.Unlock()
	if s.storage == nil {
		s.storage = Song{}
	}
	return f(s.storage, s.modified)
}

type sliceMapStorage struct {
	m        sync.Mutex
	storage  []mpd.Attrs
	modified time.Time
}

func (s *sliceMapStorage) set(l []mpd.Attrs, t time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	s.storage = l
	s.modified = t
}

func (s *sliceMapStorage) get() ([]mpd.Attrs, time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.storage == nil {
		s.storage = []mpd.Attrs{}
	}
	return s.storage, s.modified
}

func (s *sliceMapStorage) lock(f func([]mpd.Attrs, time.Time) error) error {
	s.m.Lock()
	defer s.m.Unlock()
	if s.storage == nil {
		s.storage = []mpd.Attrs{}
	}
	return f(s.storage, s.modified)
}

type mapStorage struct {
	m        sync.Mutex
	storage  mpd.Attrs
	modified time.Time
}

func (s *mapStorage) set(l mpd.Attrs, t time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	s.storage = l
	s.modified = t
}

func (s *mapStorage) get() (mpd.Attrs, time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.storage == nil {
		s.storage = mpd.Attrs{}
	}
	return s.storage, s.modified
}

type statusStorage struct {
	m        sync.Mutex
	storage  Status
	modified time.Time
}

func (s *statusStorage) set(l Status, t time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	s.storage = l
	s.modified = t
}

func (s *statusStorage) get() (Status, time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.storage, s.modified
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
