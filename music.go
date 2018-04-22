package main

import (
	"errors"
	"github.com/meiraka/gompd/mpd"
	"strconv"
	"sync"
	"time"
)

const (
	playlistLength = 9999
)

/*Dial Connects to mpd server.*/
func Dial(network, addr, passwd, musicDirectory string) (*Music, error) {
	p := new(Music)
	p.network = network
	p.addr = addr
	p.passwd = passwd
	p.musicDirectory = musicDirectory
	return p, p.initIfNot()
}

/*Music represents mpd control interface.*/
type Music struct {
	network            string
	addr               string
	passwd             string
	musicDirectory     string
	watcherResponse    chan error
	daemonStop         chan bool
	daemonRequest      chan *musicMessage
	coverCache         map[string]string
	init               sync.Mutex
	mutex              sync.Mutex
	status             statusStorage
	current            songStorage
	stats              mapStorage
	library            songsStorage
	librarySort        songsStorage
	playlist           songsStorage
	playlistSort       songsStorage
	playlistSortLock   sync.Mutex
	playlistSorted     bool
	playlistSortkeys   []string
	playlistFilters    [][]string
	playlistSortUpdate time.Time
	outputs            sliceMapStorage
	notification       PubSub
}

/*Close mpd connection.*/
func (p *Music) Close() error {
	p.daemonStop <- true
	p.notification.EnsureStop()
	return nil
}

/*Current returns mpd current song data.*/
func (p *Music) Current() (Song, time.Time) {
	return p.current.get()
}

/*Library returns mpd library song data list.*/
func (p *Music) Library() ([]Song, time.Time) {
	return p.library.get()
}

/*Next song.*/
func (p *Music) Next() error {
	return p.request(func(mpc mpdClient) error { return mpc.Next() })
}

/*Output enable output if true.*/
func (p *Music) Output(id int, on bool) error {
	if on {
		return p.request(func(mpc mpdClient) error { return mpc.EnableOutput(id) })
	}
	return p.request(func(mpc mpdClient) error { return mpc.DisableOutput(id) })
}

/*Outputs return output device list.*/
func (p *Music) Outputs() ([]map[string]string, time.Time) {
	return p.outputs.get()
}

/*Pause song.*/
func (p *Music) Pause() error {
	return p.request(func(mpc mpdClient) error { return mpc.Pause(true) })
}

/*Play or resume song.*/
func (p *Music) Play() error {
	return p.request(func(mpc mpdClient) error { return mpc.Play(-1) })
}

/*PlayPos play songs pos.*/
func (p *Music) PlayPos(pos int) error {
	return p.request(func(mpc mpdClient) error { return mpc.Play(pos) })
}

/*Playlist returns mpd playlist song data list.*/
func (p *Music) Playlist() ([]Song, time.Time) {
	return p.playlist.get()
}

/*PlaylistIsSorted returns mpd playlist sort keys and filters.*/
func (p *Music) PlaylistIsSorted() (bool, []string, [][]string, time.Time) {
	p.playlistSortLock.Lock()
	defer p.playlistSortLock.Unlock()
	return p.playlistSorted, p.playlistSortkeys, p.playlistFilters, p.playlistSortUpdate
}

/*Prev song.*/
func (p *Music) Prev() error {
	return p.request(func(mpc mpdClient) error { return mpc.Previous() })
}

/*Random enable if true*/
func (p *Music) Random(on bool) error {
	return p.request(func(mpc mpdClient) error { return mpc.Random(on) })
}

/*Repeat enable if true*/
func (p *Music) Repeat(on bool) error {
	return p.request(func(mpc mpdClient) error { return mpc.Repeat(on) })
}

/*Single enable if true*/
func (p *Music) Single(on bool) error {
	return p.request(func(mpc mpdClient) error { return mpc.Single(on) })
}

/*RescanLibrary scans music directory and update library database.*/
func (p *Music) RescanLibrary() error {
	return p.request(func(mpc mpdClient) error {
		_, err := mpc.Update("")
		return err
	})
}

/*SortPlaylist sorts playlist by song tag name.*/
func (p *Music) SortPlaylist(keys []string, filters [][]string, pos int) (err error) {
	return p.request(func(mpc mpdClient) error { return p.sortPlaylist(mpc, keys, filters, pos) })
}

func (p *Music) sortPlaylist(mpc mpdClient, keys []string, filters [][]string, pos int) error {
	return p.librarySort.lock(func(masterLibrary []Song, _ time.Time) error {
		update := false
		library, newpos := SortSongs(masterLibrary, keys, filters, playlistLength, pos)
		p.playlistSort.lock(func(playlist []Song, _ time.Time) error {
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
		p.playlistSortLock.Lock()
		p.playlistSorted = true
		p.playlistSortkeys = keys
		p.playlistFilters = filters
		p.playlistSortUpdate = time.Now().UTC()
		p.playlistSortLock.Unlock()
		if update {
			cl := musicMpdBeginCommandList(mpc)
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
			if i == newpos {
				return mpc.Play(i)
			}
		}
		return nil
	})
}

/*Status returns mpd current song data.*/
func (p *Music) Status() (Status, time.Time) {
	return p.status.get()
}

/*Stats returns mpd statistics.*/
func (p *Music) Stats() (map[string]string, time.Time) {
	return p.stats.get()
}

// Subscribe server events.
func (p *Music) Subscribe(c chan string) {
	p.notification.Subscribe(c)
	p.updateSubscribers()
}

// Unsubscribe server events.
func (p *Music) Unsubscribe(c chan string) {
	p.notification.Unsubscribe(c)
	p.updateSubscribers()
}

/*Volume set music volume.*/
func (p *Music) Volume(v int) error {
	return p.request(func(mpc mpdClient) error { return mpc.SetVolume(v) })
}

func (p *Music) updateSubscribers() {
	stats, modified := p.stats.get()
	newStats := map[string]string{}
	for k, v := range stats {
		newStats[k] = v
	}
	newStats["subscribers"] = strconv.Itoa(p.notification.Count())
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

func (p *Music) notify(n string) error {
	return p.notification.Notify(n)
}

type musicMessage struct {
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
	Single(bool) error
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

type mpdClientCommandList interface {
	Clear()
	Add(string)
	End() error
}

func (p *Music) initIfNot() error {
	p.init.Lock()
	defer p.init.Unlock()
	if p.daemonStop == nil {
		p.daemonStop = make(chan bool)
		p.daemonRequest = make(chan *musicMessage)
		p.coverCache = make(map[string]string)
		mpc, watcher := p.connect()
		go p.run(mpc, watcher)
	}
	return nil
}

func musicRealMpdDial(net, addr, passwd string) (mpdClient, error) {
	return mpd.DialAuthenticated(net, addr, passwd)
}

func musicRealMpdNewWatcher(net, addr, passwd string) (*mpd.Watcher, error) {
	return mpd.NewWatcher(net, addr, passwd)
}

func musicRealMpdWatcherClose(w mpd.Watcher) error {
	return w.Close()
}

func musicRealMpdBeginCommandList(m mpdClient) mpdClientCommandList {
	return m.BeginCommandList()
}

var musicMpdDial = musicRealMpdDial
var musicMpdNewWatcher = musicRealMpdNewWatcher
var musicMpdWatcherClose = musicRealMpdWatcherClose
var musicMpdBeginCommandList = musicRealMpdBeginCommandList

func (p *Music) connect() (mpdClient, *mpd.Watcher) {
	mpc, err := musicMpdDial(p.network, p.addr, p.passwd)
	if err != nil {
		return nil, new(mpd.Watcher)
	}
	watcher, err := musicMpdNewWatcher(p.network, p.addr, p.passwd)
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

func (p *Music) run(mpc mpdClient, watcher *mpd.Watcher) {
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

func (p *Music) request(f func(mpdClient) error) error {
	ec := make(chan error)
	p.requestAsync(f, ec)
	return <-ec
}

func (p *Music) requestAsync(f func(mpdClient) error, ec chan error) {
	r := new(musicMessage)
	r.request = f
	r.err = ec
	p.daemonRequest <- r
}

func (p *Music) updateCurrentSong(mpc mpdClient) error {
	tags, err := mpc.CurrentSongTags()
	if err != nil {
		return err
	}
	if _, found := tags["file"]; !found {
		return nil
	}
	if _, found := tags["Id"]; !found {
		return nil
	}
	current, _ := p.current.get()
	if len(current["file"]) == 0 || current["file"][0] != tags["file"][0] ||
		len(current["Id"]) == 0 || current["Id"][0] != tags["Id"][0] {
		p.mutex.Lock()
		song := MakeSong(tags, p.musicDirectory, "cover.*", p.coverCache)
		p.mutex.Unlock()
		p.current.set(song, time.Now().UTC())
		return p.notify("playlist/current")
	}
	return nil
}

func (p *Music) updateStatus(mpc mpdClient) error {
	status, err := mpc.Status()
	if err != nil {
		return err
	}
	p.status.set(MakeStatus(status), time.Now().UTC())
	return p.notify("status")
}

func (p *Music) updateStats(mpc mpdClient) error {
	stats, err := mpc.Stats()
	if err != nil {
		return err
	}
	stats["subscribers"] = strconv.Itoa(p.notification.Count())
	p.stats.set(stats, time.Now().UTC())
	return p.notify("stats")
}

func (p *Music) updateLibrary(mpc mpdClient) error {
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

	p.updatePlaylistSort()
	return p.notify("library")
}

/*updatePlaylist update playlist data and playlist sorted status.*/
func (p *Music) updatePlaylist(mpc mpdClient) error {
	playlistTags, err := mpc.PlaylistInfoTags(-1, -1)
	if err != nil {
		return err
	}
	p.mutex.Lock()
	playlist := MakeSongs(playlistTags, p.musicDirectory, "cover.*", p.coverCache)
	p.mutex.Unlock()
	playlistSort := make([]Song, len(playlist))
	copy(playlistSort, playlist)
	p.playlist.set(playlist, time.Now().UTC())
	p.playlistSort.set(playlistSort, time.Now().UTC())

	p.updatePlaylistSort()
	return p.notify("playlist")
}

/*updatePlaylistSort compares library and playlist to update sort status.
  send playlist/sort notify if status is updated.
*/
func (p *Music) updatePlaylistSort() {
	isSorted := p.checkPlaylistIsSorted()
	p.playlistSortLock.Lock()
	defer p.playlistSortLock.Unlock()
	if isSorted != p.playlistSorted {
		p.playlistSorted = isSorted
		p.playlistSortUpdate = time.Now().UTC()
		p.notify("playlist/sort")
	}
}

/*checkPlaylistIsSorted returns true if playlist is sorted.*/
func (p *Music) checkPlaylistIsSorted() bool {
	ret := false
	p.playlistSortLock.Lock()
	defer p.playlistSortLock.Unlock()
	if p.playlistSortkeys == nil || p.playlistFilters == nil {
		return ret
	}
	p.playlistSort.lock(func(playlist []Song, _ time.Time) error {
		p.librarySort.lock(func(masterLibrary []Song, _ time.Time) error {
			library, _ := SortSongs(masterLibrary, p.playlistSortkeys, p.playlistFilters, playlistLength, 0)
			ret = true
			if len(library) != len(playlist) {
				ret = false
			} else {
				for i := range library {
					if library[i]["file"][0] != playlist[i]["file"][0] {
						ret = false
						break
					}
				}
			}
			return nil
		})
		return nil
	})
	return ret
}

func (p *Music) updateOutputs(mpc mpdClient) error {
	outputs, err := mpc.ListOutputs()
	if err != nil {
		return err
	}
	n := make([]map[string]string, len(outputs))
	for k, v := range outputs {
		n[k] = map[string]string(v)
	}
	p.outputs.set(n, time.Now().UTC())
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
	storage  []map[string]string
	modified time.Time
}

func (s *sliceMapStorage) set(l []map[string]string, t time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	s.storage = l
	s.modified = t
}

func (s *sliceMapStorage) get() ([]map[string]string, time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.storage == nil {
		s.storage = []map[string]string{}
	}
	return s.storage, s.modified
}

type mapStorage struct {
	m        sync.Mutex
	storage  map[string]string
	modified time.Time
}

func (s *mapStorage) set(l map[string]string, t time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	s.storage = l
	s.modified = t
}

func (s *mapStorage) get() (map[string]string, time.Time) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.storage == nil {
		s.storage = map[string]string{}
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
