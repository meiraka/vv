"use strict";

const playlistLength = 9999;

class App {
    constructor() {
        this.preferences = new Preferences();
        this.ui = new UI();
        this.mpdWatcher = new MPDWatcher();
        this.mpd = new MPDClient(this.mpdWatcher);
        this.audio = new MPDAudio(this.mpd, this.preferences);
        this.library = new Library(this.mpd, this.preferences);
        this.background = new UIBackground(this.ui, this.mpd, this.preferences);
        this.listView = new UIListView(this.ui, this.mpd, this.library, this.preferences);
        this.mainView = new UIMainView(this.ui, this.mpd, this.library, this.preferences);
        this.systemWindow = new UISystemWindow(this.ui, this.mpd, this.preferences);
        UIMediaSession.init(this.ui, this.mpd, this.preferences);
        UIModal.init(this.ui);
        UISubModal.init(this.ui, this.mpd);
        UIHeader.init(this.ui, this.mpd, this.library, this.listView, this.mainView, this.systemWindow);
        UIFooter.init(this.ui, this.mpd, this.preferences);
        UINotification.init(this.ui, this.mpd, this.audio, this.systemWindow);
        UITimeUpdater.init(this.ui, this.mpd);
        KeyboardShortCuts.init(this.ui, this.mpd, this.library, this.listView, this.mainView);
        this.mainView.addEventListener("list", () => { this.listView.show(); });
        this.listView.addEventListener("main", () => { this.mainView.show(); });
    }
    _init() {
        const start = () => {
            this.ui.raiseEvent("load");
            this.listView.show();
            this.mpdWatcher.start();
            this.ui.polling();
        };
        if (this.mpd.loaded) {
            start();
        } else {
            this.mpd.addEventListener("load", start);
        }
    }
    start() {
        if (document.readyState === "loading") {
            document.addEventListener("DOMContentLoaded", () => { this._init(); });
        } else {
            this._init();
        }
    }
};

class PubSub {
    constructor() { this.listeners = {}; }
    addEventListener(e, f, opt) {
        if (!(e in this.listeners)) {
            this.listeners[e] = [];
        }
        if (opt && opt.once === true) {
            const fn = (o) => {
                f(o);
                // setTimeout is needed to avoid breaking the list of event functions in the loop
                setTimeout(() => { this.removeEventListener(e, fn); });
            };
            this.listeners[e].push(fn);
            return;
        }
        this.listeners[e].push(f);
    }
    removeEventListener(e, f) {
        for (let i = 0, imax = this.listeners[e].length; i < imax; i++) {
            if (this.listeners[e][i] === f) {
                this.listeners[e].splice(i, 1);
                return;
            }
        }
    }
    raiseEvent(e, o) {
        if (!(e in this.listeners)) {
            return;
        }
        if (!o) {
            o = { currentTarget: this, name: e };
        }
        for (const f of this.listeners[e]) {
            f(o);
        }
    }
};

class MPDAudio {
    constructor(mpd, preferences) {
        this.mpd = mpd;
        this.preferences = preferences;
        this.src = this.preferences.httpoutput.stream;
        this.audio = new Audio();
        this.audio.preload = "metadata";
        // firefox's load() function does not fire canplaythrough event without autoplay is enabled.
        this.audio.autoplay = true;
        this.stopped = true;
        this._preparePlay = false;
        this.audio.addEventListener("ended", () => { // connection lost
            if (!this.stopped) {
                this.stop();
            }
            UINotification.show("client-output", "networkError", { ttl: Infinity });
        });
        this.audio.addEventListener("loadeddata", () => {
            // prevent autoplay to use play() function in canplaythrough event.
            this.audio.pause();
            this._preparePlay = true;
        });
        this.audio.addEventListener("canplaythrough", () => {
            if (!this._preparePlay) {
                return;
            }
            this._preparePlay = false;
            const p = this.audio.play();
            if (!p) {
                return;
            }
            if (this.preferences.httpoutput.stream === "") {
                return;
            }
            if (document.readyState === "loading") {
                return;
            }
            const err = document.getElementById("httpoutput-error");
            p.then(() => {
                UINotification.hide("client-output");
                err.textContent = "";
            }).catch((e) => {
                if (this.preferences.httpoutput.stream === "") {
                    return;
                }
                if (!this.audio.stopped) {
                    this.stop();
                }
                switch (e.name) {
                    case "NotAllowedError":
                        UINotification.show("client-output", "notAllowed", { ttl: Infinity });
                        err.textContent = err.dataset["notAllowed"];
                        break;
                    case "NotSupportedError":
                        UINotification.show("client-output", "unsupportedSource", { ttl: Infinity });
                        err.textContent = err.dataset["unsupportedSource"];
                        break;
                    default:
                        UINotification.show("client-output", e.message);
                        err.textContent = e.message;
                        break;
                }
            });
        });
        this.audio.addEventListener("error", (e) => {
            if (this.mpd.control.state !== "play" || document.readyState === "loading") {
                return;
            }
            if (this.preferences.httpoutput.stream === "") {
                return;
            }
            if (this.audio.stopped) {
                return;
            }
            this.stop();
            const err = document.getElementById("httpoutput-error");
            err.textContent = "";
            switch (e.target.error.code) {
                case e.target.error.MEDIA_ERR_NETWORK:
                    UINotification.show("client-output", "networkError", { ttl: Infinity });
                    err.textContent = err.dataset["networkError"];
                    break;
                case e.target.error.MEDIA_ERR_DECODE:
                    UINotification.show("client-output", "decodeError", { ttl: Infinity });
                    err.textContent = err.dataset["decodeError"];
                    break;
                case e.target.error.MEDIA_ERR_SRC_NOT_SUPPORTED:
                    UINotification.show("client-output", "unsupportedSource", { ttl: Infinity });
                    err.textContent = err.dataset["unsupportedSource"];
                    break;

            }
        });
        this.mpd.addEventListener("control", () => { this.sync(); });
        this.preferences.addEventListener("httpoutput", () => {
            this.audio.volume = this.preferences.httpoutput.volume;
            this.sync();
        });
        this.audio.volume = this.preferences.httpoutput.volume;
    }
    play() {
        this.audio.pause();
        this.stopped = false;
        // https://bugzilla.mozilla.org/show_bug.cgi?id=1129121
        // add cache-busting query
        this.audio.src = this.preferences.httpoutput.stream + "&cb=" + Math.random();
        this.audio.load();
    }
    stop() {
        this.audio.pause();
        this.stopped = true;
        this.audio.removeAttribute("src");
    }
    sync() {
        if (this.preferences.httpoutput.stream === "") {
            return;
        }
        if (this.mpd.control.state === "play") {
            if (this.src !== this.preferences.httpoutput.stream) {
                this.src = this.preferences.httpoutput.stream;
                if (this.src === "" && !this.stopped) { // disable output
                    this.stop();
                } else if (this.src !== "") { // enable or change output
                    this.play();
                }
            } else if (this.stopped) {
                this.play();
            }
        } else {
            if (!this.stopped) {
                this.stop();
            }
        }
    }
};

class Song {
    static tag(song, keys, other) {
        for (const key of keys) {
            if (key in song) {
                return song[key];
            }
        }
        return other;
    }
    static getTagOrElseMulti(song, key, other) {
        if (key in song) {
            return song[key];
        } else if (key === "AlbumSort") {
            return Song.tag(song, ["Album"], other);
        } else if (key === "ArtistSort") {
            return Song.tag(song, ["Artist"], other);
        } else if (key === "AlbumArtist") {
            return Song.tag(song, ["Artist"], other);
        } else if (key === "AlbumArtistSort") {
            return Song.tag(song, ["AlbumArtist", "Artist"], other);
        } else if (key === "AlbumSort") {
            return Song.tag(song, ["Album"], other);
        } else if (key === "Date") {
            return Song.tag(song, ["OriginalDate"], other);
        } else if (key === "OriginalDate") {
            return Song.tag(song, ["Date"], other);
        }
        return other;
    }
    static getOrElseMulti(song, keys, other) {
        let ret = [];
        for (const key of keys.split("-")) {
            const t = Song.getTagOrElseMulti(song, key, other);
            if (!ret.length) {
                ret = t;
            } else if (t.length !== 0) {
                const newret = [];
                for (const oldV of ret) {
                    for (const newV of t) {
                        newret.push(oldV + "-" + newV);
                    }
                }
                ret = newret;
            }
        }
        return ret;
    }
    static getOrElse(song, key, other) {
        const ret = Song.getOrElseMulti(song, key, null);
        if (!ret) {
            return other;
        }
        return ret.join();
    }
    static getOne(song, key) {
        const other = null;
        if (!song.keys) {
            return Song.getOrElseMulti(song, key, [other])[0];
        }
        for (const kv of song.keys) {
            if (kv[0] === key) {
                return kv[1];
            }
        }
        return Song.getOrElseMulti(song, key, [other])[0];
    }
    static get(song, key) { return Song.getOrElse(song, key, `[no ${key}]`); }
    static sortkeys(song, keys, memo) {
        let songs = [Object.assign({}, song)];
        songs[0].sortkey = "";
        songs[0].keys = [];
        for (const key of keys) {
            const writememo = memo.indexOf(key) !== -1;
            const values = Song.getOrElseMulti(song, key, []);
            if (values.length === 0) {
                for (const song of songs) {
                    song.sortkey += " ";
                    if (writememo) {
                        song.keys.push([key, null]);
                    }
                }
            } else if (values.length === 1) {
                for (const song of songs) {
                    song.sortkey += values[0];
                    if (writememo) {
                        song.keys.push([key, values[0]]);
                    }
                }
            } else {
                let newsongs = [];
                for (const song of songs) {
                    for (const value of values) {
                        const newsong = Object.assign({}, song);
                        newsong.keys = Object.assign([], song.keys);
                        newsong.sortkey += value;
                        if (writememo) {
                            newsong.keys.push([key, value]);
                        }
                        newsongs.push(newsong);
                    }
                }
                songs = newsongs;
            }
        }
        return songs;
    }
};

class Songs {
    static sort(songs, keys, memo) {
        const newsongs = [];
        for (const song of songs) {
            Array.prototype.push.apply(newsongs, Song.sortkeys(song, keys, memo));
        }
        const sorted = newsongs.sort((a, b) => {
            if (a.sortkey < b.sortkey) {
                return -1;
            }
            return 1;
        });
        for (let j = 0, jmax = sorted.length; j < jmax; j++) {
            sorted[j].pos = [j];
        }
        return sorted;
    }
    static uniq(songs, key) {
        return songs.filter((song, i, self) => {
            if (i === 0) {
                return true;
            } else if (Song.getOne(song, key) === Song.getOne(self[i - 1], key)) {
                return false;
            }
            return true;
        });
    }
    static filter(songs, filters) {
        return songs.filter(song => {
            for (const key in filters) {
                if (filters.hasOwnProperty(key)) {
                    if (Song.getOne(song, key) !== filters[key]) {
                        return false;
                    }
                }
            }
            return true;
        });
    }
    static weakFilter(songs, filters, must, max) {
        if (songs.length <= max && must === 0) {
            return songs;
        }
        let i = 0;
        for (const filter of filters) {
            if (songs.length <= max && must <= i) {
                return songs;
            }
            const newsongs = [];
            for (const song of songs) {
                if (Song.getOne(song, filter[0]) === filter[1]) {
                    newsongs.push(song);
                }
            }
            songs = newsongs;
            i++;
        }
        if (songs.length > max) {
            const ret = [];
            for (let k = 0; k < max; k++) {
                ret.push(songs[k]);
            }
            return ret;
        }
        return songs;
    }
};

const _requests = {};
class HTTP {
    static abortAll(options) {
        const opts = options || {};
        for (const key in _requests) {
            if (_requests.hasOwnProperty(key)) {
                if (opts.stop) {
                    _requests[key].onabort = () => { };
                }
                _requests[key].abort();
            }
        }
    }
    static get(path, ifmodified, etag, callback, timeout) {
        const key = "GET " + path;
        if (_requests[key]) {
            _requests[key].onabort = () => { };  // disable retry
            _requests[key].abort();
        }
        const xhr = new XMLHttpRequest();
        _requests[key] = xhr;
        if (!timeout) {
            timeout = 1000;
        }
        xhr.responseType = "json";
        xhr.timeout = timeout;
        xhr.onload = () => {
            if (xhr.status === 200 || xhr.status === 304) {
                if (xhr.status === 200 && callback) {
                    callback(
                        xhr.response, xhr.getResponseHeader("Last-Modified"),
                        xhr.getResponseHeader("Etag"),
                        xhr.getResponseHeader("Date"));
                }
                return;
            }
            // error handling
            if (xhr.status !== 0) {
                UINotification.show("network", xhr.statusText);
            }
        };
        xhr.onabort = () => {
            if (timeout < 50000) {
                setTimeout(() => { HTTP.get(path, ifmodified, etag, callback, timeout * 2); });
            }
        };
        xhr.onerror = () => { UINotification.show("network", "Error"); };
        xhr.ontimeout = () => {
            if (timeout < 50000) {
                UINotification.show("network", "timeoutRetry");
                HTTP.abortAll();
                setTimeout(() => { HTTP.get(path, ifmodified, etag, callback, timeout * 2); });
            } else {
                UINotification.show("network", "timeout");
            }
        };
        xhr.open("GET", path, true);
        if (etag !== "") {
            xhr.setRequestHeader("If-None-Match", etag);
        } else {
            xhr.setRequestHeader("If-Modified-Since", ifmodified);
        }
        xhr.send();
    }
    static post(path, obj, callback) {
        const key = "POST " + path;
        if (_requests[key]) {
            _requests[key].abort();
        }
        const xhr = new XMLHttpRequest();
        _requests[key] = xhr;
        xhr.responseType = "json";
        xhr.timeout = 1000;
        xhr.onload = () => {
            if (callback && xhr.response) {
                callback(xhr.response);
            }
            if (xhr.status !== 200 && xhr.status !== 202) {
                if (xhr.response && xhr.response.error) {
                    UINotification.show("mpd", xhr.response.error);
                } else {
                    UINotification.show("network", xhr.responseText);
                }
            }
        };
        xhr.ontimeout = () => {
            if (callback) {
                callback({ error: "timeout" });
            }
            UINotification.show("network", "timeout");
            HTTP.abortAll();
        };
        xhr.onerror = () => {
            if (callback) {
                callback({ error: "error" });
            }
            UINotification.show("network", "error");
        };
        xhr.open("POST", path, true);
        xhr.setRequestHeader("Content-Type", "application/json");
        xhr.send(JSON.stringify(obj));
    }
};

class MPDWatcher extends PubSub {
    constructor() { super(); }
    start() {
        let lastUpdate = (new Date()).getTime();
        let lastConnection = (new Date()).getTime();
        let connected = false;
        let tryNum = 0;
        let ws = null;
        const listennotify = (cause) => {
            if (cause && tryNum > 1) {  // reduce device wakeup reconnecting message
                UINotification.show("network", cause);
            }
            tryNum++;
            HTTP.abortAll({ stop: true });
            lastConnection = (new Date()).getTime();
            connected = false;
            const wsp = document.location.protocol === "https:" ? "wss:" : "ws:";
            const uri = `${wsp}//${location.host}/api/music`;
            if (ws !== null) {
                ws.onclose = () => { };
                ws.close();
            }
            ws = new WebSocket(uri);
            ws.onopen = () => {
                if (tryNum > 1) {
                    UINotification.hide("network");
                }
                connected = true;
                lastUpdate = (new Date()).getTime();
                tryNum = 0;
                this.raiseEvent("connect");
            };
            ws.onmessage = e => {
                if (e && e.data) {
                    const now = (new Date()).getTime();
                    this.raiseEvent(e.data);
                    if (now - lastUpdate > 10000) {
                        // recover lost notification
                        setTimeout(() => { this.raiseEvent("lost"); });
                    }
                    lastUpdate = now;
                }
            };
            ws.onclose = () => {
                setTimeout(() => { this.raiseEvent("lost", "timeoutRetry"); }, 1000);
            };
        }
        const polling = () => {
            const now = (new Date()).getTime();
            if (connected && now - 10000 > lastUpdate) {
                setTimeout(() => { this.raiseEvent("lost", "doesNotRespond"); });
            } else if (!connected && now - 2000 > lastConnection) {
                setTimeout(() => { this.raiseEvent("lost", "timeoutRetry"); });
            }
            setTimeout(polling, 1000);
        };

        this.addEventListener("lost", listennotify);
        listennotify();
        polling();
    }
};

class Preferences extends PubSub {
    constructor() {
        super();
        this.feature = {
            show_scrollbars_when_scrolling: false,
            client_volume_control: true,
        };
        this.playlist = {
            playback_tracks: "all",
            playback_tracks_custom: {},
        };
        this.httpoutput = {
            volume: "0.2",
            volume_max: "1",
            streams: {},
            stream: "",
        };
        this.outputs = {
            volume_max: "100",
        };
        this.appearance = {
            theme: "prefer-coverart",
            color_threshold: 128,
            background_image: true,
            background_image_blur: "64px",
            circled_image: false,
            crossfading_image: true,
            volume: true,
            playlist_follows_playback: true,
            playlist_gridview_album: true,
        }
        this.load();
    }
    load() {
        const permitted = ["playlist", "httpoutput", "outputs", "appearance"];
        let storedPreferences = null;
        try {
            storedPreferences = localStorage.getItem("preferences");
        } catch (_) { }
        if (storedPreferences !== null) {
            const c = JSON.parse(storedPreferences);
            for (const i in c) {
                if (c.hasOwnProperty(i) && permitted.includes(i)) {
                    for (const j in c[i]) {
                        if (c[i].hasOwnProperty(j)) {
                            if (this[i].hasOwnProperty(j)) {
                                this[i][j] = c[i][j];
                            }
                        }
                    }
                }
            }
            // convert old settings
            if (c.appearance && c.appearance.volume_max) {
                this.preferences.outputs.volume_max = c.appearance.volume_max;
            }
        }
        if (navigator.userAgent.indexOf("Mobile") > 1) {
            this.feature.show_scrollbars_when_scrolling = true;
        } else if (navigator.userAgent.indexOf("Macintosh") > 1) {
            this.feature.show_scrollbars_when_scrolling = true;
        } else {
            document.body.classList.add("scrollbar-styling");
        }
        if (["iPad", "iPod", "iPhone"].includes(navigator.platform)) {
            // https://developer.apple.com/library/archive/documentation/AudioVideo/Conceptual/Using_HTML5_Audio_Video/Device-SpecificConsiderations/Device-SpecificConsiderations.html
            // in 2022: The volume property is settable, but not working.
            this.feature.client_volume_control = false;
        }
        for (const key in this.playlist.playback_tracks_custom) {
            if (!(key in TREE)) {
                delete this.playlist.playback_tracks_custom[key];
            }
        }
        for (const key in TREE) {
            if (!(key in this.playlist.playback_tracks_custom)) {
                this.playlist.playback_tracks_custom[key] = 0;
            } else {
                if (this.playlist.playback_tracks_custom[key] >= TREE[key].tree.length) {
                    this.playlist.playback_tracks_custom[key] = 0;
                }
            }
        }
    }
    save() {
        const json = JSON.stringify({
            playlist: this.playlist,
            httpoutput: this.httpoutput,
            outputs: this.outputs,
            appearance: this.appearance,
        });
        try {
            localStorage.setItem("preferences", json);
        } catch (_) { }
    }
    nocover() {
        return "/assets/nocover.svg";
    }
};

class MPDClient extends PubSub {
    constructor(mpdWatcher) {
        super();
        const that = this;
        this.loaded = false;
        this.current = null;
        this.control = {};
        this.librarySongs = [];
        this.library = {};
        this.images = {};
        this.outputs = [];
        this.storage = {};
        this.neighbors = {};
        this.stats = {};
        this.last_modified = {};
        this.etag = {};
        this.last_modified_ms = {};
        this.version = {};
        this.save = {
            current() {
                const json = JSON.stringify(that.current);
                try {
                    localStorage.setItem("current", json);
                    localStorage.setItem("current_last_modified", that.last_modified.current);
                } catch (_) { }
            },
            playlist() {
                const json = JSON.stringify(that.playlist);
                try {
                    localStorage.setItem("playlist", json);
                    localStorage.setItem("playlist_last_modified", that.last_modified.playlist);
                } catch (_) { }
            },
            librarySongs() {
                that._cacheSave("library", that.librarySongs, that.last_modified.librarySongs);
            }
        };
        this._load();
        mpdWatcher.addEventListener("connect", () => { this._fetchAll(); });
        mpdWatcher.addEventListener("/api/music/library/songs", () => { this._fetch("/api/music/library/songs", "librarySongs"); });
        mpdWatcher.addEventListener("/api/music/library", () => { this._fetch("/api/music/library", "library"); });
        mpdWatcher.addEventListener("/api/music", () => { this._fetch("/api/music", "control"); });
        mpdWatcher.addEventListener("/api/music/playlist/songs/current", () => { this._fetch("/api/music/playlist/songs/current", "current"); });
        mpdWatcher.addEventListener("/api/music/outputs", () => { this._fetch("/api/music/outputs", "outputs"); });
        mpdWatcher.addEventListener("/api/music/stats", () => { this._fetch("/api/music/stats", "stats"); });
        mpdWatcher.addEventListener("/api/music/playlist", () => { this._fetch("/api/music/playlist", "playlist"); });
        mpdWatcher.addEventListener("/api/music/images", () => { this._fetch("/api/music/images", "images"); });
        mpdWatcher.addEventListener("/api/music/storage", () => { this._fetch("/api/music/storage", "storage"); });
        mpdWatcher.addEventListener("/api/music/storage/neighbors", () => { this._fetch("/api/music/storage/neighbors", "neighbors"); });
        mpdWatcher.addEventListener("/api/version", () => { this._fetch("/api/version", "version"); });
    }
    rescanLibrary() {
        for (const path in this.storage) {
            if (path !== "" && this.storage.hasOwnProperty(path)) {
                HTTP.post("/api/music/storage", { [path]: { updating: true } });
            }
        }
        HTTP.post("/api/music/library", { updating: true });
        this.library.updating = true;
        this.raiseEvent("library");
    }
    /*static*/ prev() { HTTP.post("/api/music", { state: "previous" }); }
    /*static*/ play() { HTTP.post("/api/music", { state: "play" }); }
    /*static*/ pause() { HTTP.post("/api/music", { state: "pause" }); }
    /*static*/ stop() { HTTP.post("/api/music", { state: "stopped" }); }
    togglePlay() {
        const state = "state" in this.control ? this.control["state"] : "stopped";
        const action = state === "play" ? "pause" : "play";
        HTTP.post("/api/music", { state: action }, (e) => {
            if (e.error) {
                this.control.state = state;
                this.raiseEvent("control");
            }
        });
        this.control.state = action;
        const now = (new Date()).getTime();
        if (action === "pause") {
            const elapsed = parseInt(this.control.song_elapsed * 1000, 10) + now - this.last_modified_ms.control;
            this.control.song_elapsed = elapsed / 1000;
        }
        this.last_modified_ms.control = now;
        this.raiseEvent("control");
    }
    /*static*/ next() { HTTP.post("/api/music", { state: "next" }); }
    sortPlaylist(sort, filters, must, current) {
        HTTP.post("/api/music/playlist", { sort: sort, filters: filters, must: must, current: current });
    }
    toggleRepeat() {
        if (this.control.single) {
            HTTP.post("/api/music", { repeat: false, single: false });
            this.control.single = false;
            this.control.repeat = false;
        } else if (this.control.repeat) {
            HTTP.post("/api/music", { single: true });
            this.control.single = true;
        } else {
            HTTP.post("/api/music", { repeat: true });
            this.control.repeat = true;
        }
        this.raiseEvent("control");
    }
    toggleRandom() {
        HTTP.post("/api/music", { random: !this.control.random });
        this.control.random = !this.control.random;
        this.raiseEvent("control");
    }
    /*static*/ volume(num) { HTTP.post("/api/music", { volume: num }); }
    /*static*/ output(id, on) { HTTP.post(`/api/music/outputs`, { [id]: { enabled: on } }); }
    /*static*/ seek(pos) { HTTP.post("/api/music", { song_elapsed: pos }); }
    elapsed() {
        const data = this.control;
        if ("state" in data) {
            const elapsed = parseInt(data.song_elapsed * 1000, 10);
            let current = elapsed;
            if (data.state === "play") {
                current += (new Date()).getTime() - this.last_modified_ms.control;
            }
            return parseInt(current / 1000, 10);
        }
        return 0;
    }

    _idbUpdateTables(e) {
        const db = e.target.result;
        const st = db.createObjectStore("cache", { keyPath: "id" });
        st.onsuccess = () => { db.close(); };
        st.onerror = () => { db.close(); };
    }
    _cacheLoad(key, callback) {
        if (!window.indexedDB) {
            callback();
            return;
        }
        const open = window.indexedDB.open("storage", 1);
        open.onerror = () => { callback(); };
        open.onupgradeneeded = (e) => { this._idbUpdateTables(e); };
        open.onsuccess = e => {
            const db = e.target.result;
            const req = db.transaction("cache", "readonly").objectStore("cache").get(key);
            req.onsuccess = e => {
                const ret = e.target.result;
                if (ret && ret.value && ret.date) {
                    callback(e.target.result.value, e.target.result.date);
                } else {
                    callback();
                }
                db.close();
            };
            req.onerror = () => {
                callback();
                db.close();
            };
        };
    }
    _cacheSave(key, value, date) {
        if (!window.indexedDB) {
            return;
        }
        const open = window.indexedDB.open("storage", 1);
        open.onerror = () => { };
        open.onupgradeneeded = (e) => { this._idbUpdateTables(e); };
        open.onsuccess = e => {
            const db = e.target.result;
            const os = db.transaction("cache", "readwrite").objectStore("cache");
            const req = os.get(key);
            req.onerror = () => { db.close(); };
            req.onsuccess = e => {
                const ret = e.target.result;
                if (ret && ret.date && ret.date === date) {
                    return;
                }
                const req = os.put({ id: key, value: value, date: date });
                req.onerror = () => { db.close(); };
                req.onsuccess = () => { db.close(); };
            };
        };
    }
    _fetchAll() {
        this._fetch("/api/music/playlist", "playlist");
        this._fetch("/api/version", "version");
        this._fetch("/api/music/outputs", "outputs");
        this._fetch("/api/music/playlist/songs/current", "current");
        this._fetch("/api/music", "control");
        this._fetch("/api/music/library", "library");
        this._fetch("/api/music/library/songs", "librarySongs");
        this._fetch("/api/music/stats", "stats");
        this._fetch("/api/music/images", "images");
        this._fetch("/api/music/storage", "storage");
        this._fetch("/api/music/storage/neighbors", "neighbors");
    }
    _fetch(target, store) {
        HTTP.get(
            target,
            store in this.last_modified ? this.last_modified[store] : "",
            store in this.etag ? this.etag[store] : "",
            (ret, modified, etag, date) => {
                if (!ret.error) {
                    if (Object.prototype.toString.call(ret.data) === "[object Object]" && Object.keys(ret.data).length === 0) {
                        return;
                    }
                    let diff = 0;
                    try {
                        diff = Date.now() - Date.parse(date);
                    } catch (_) { // use default value;
                    }
                    const old = this[store];
                    this[store] = ret;
                    this.last_modified_ms[store] = Date.parse(modified) + diff;
                    this.last_modified[store] = modified;
                    this.etag[store] = etag;
                    if (this.save[store]) {
                        this.save[store]();
                    }
                    this.raiseEvent(store, { old: old, current: ret });
                }
            });
    }
    _load() {
        let storedCurrent = null;
        let storedCurrent_last_modified = null;
        let storedPlaylist = null;
        let storedPlaylist_last_modified = null;

        try {
            if (localStorage.getItem("version") !== "v3") {
                localStorage.clear();
            }
            localStorage.setItem("version", "v3");
            storedCurrent = localStorage.getItem("current");
            storedCurrent_last_modified = localStorage.getItem("current_last_modified");
            storedPlaylist = localStorage.getItem("playlist");
            storedPlaylist_last_modified = localStorage.getItem("playlist_last_modified");
        } catch (_) { }
        if (storedCurrent !== null && storedCurrent_last_modified !== null) {
            const current = JSON.parse(storedCurrent);
            if (Object.prototype.toString.call(current.file) === "[object Array]") {
                this.current = current;
                this.last_modified.current = storedCurrent_last_modified;
            }
        }
        if (storedPlaylist !== null && storedPlaylist_last_modified !== null) {
            this.playlist = JSON.parse(storedPlaylist);
            this.last_modified.playlist = storedPlaylist_last_modified;
        }
        this._cacheLoad("library", (data, date) => {
            if (data && date) {
                this.librarySongs = data;
                this.last_modified.librarySongs = date;
            }
            this.loaded = true;
            this.raiseEvent("load");
        });
    }
};

class Library extends PubSub {
    constructor(mpd, preferences) {
        super();
        this.mpd = mpd;
        this.preferences = preferences;
        this.tree = [];
        this.focus = {};
        this.child = null;
        this._root = "root";
        this._lastRoot = "root"; // hint for absFallback
        this._lastabs = null;
        this._roots = {
            AlbumArtist: [],
            Album: [],
            Artist: [],
            Genre: [],
            Date: [],
            Composer: [],
            Performer: [],
        };
        this._list_child_cache = [{}, {}, {}, {}, {}, {}];
        this._list_cache = {};

        if (this.mpd.loaded) {
            this.updateData(this.mpd.librarySongs);
            this.load();
        } else {
            this.mpd.addEventListener("load", () => { this.updateData(this.mpd.librarySongs); });
            this.mpd.addEventListener("load", () => { this.load(); });
        }
        this.mpd.addEventListener("librarySongs", () => { this.update(this.mpd.librarySongs); });
        this.mpd.addEventListener("current", () => { this._autoFocus(); });
        this.mpd.addEventListener("playlist", () => { this._autoFocus(); });
    }
    _autoFocus() {
        if (!this.preferences.appearance.playlist_follows_playback) {
            return;
        }
        if (!this.mpd.current || !this.mpd.current.Pos || this.mpd.current.Pos.length === 0 || !this.mpd.playlist || this.mpd.librarySongs.length === 0) {
            return;
        }
        const pos = parseInt(this.mpd.current.Pos[0]);
        if (pos !== this.mpd.playlist.current) {
            return;
        }
        this.abs(this.mpd.current);
    }
    load() {
        try {
            this._lastRoot = localStorage.getItem("root");
            if (!TREE_ORDER.includes(this._lastRoot)) {
                this._lastRoot = "root";
            }
        } catch (_) { }
        this._autoFocus();
    }
    save() {
        try {
            localStorage.setItem("root", this._root);
        } catch (e) {
        }
    }
    static _mkmemo(key) {
        const ret = [];
        for (const leef of TREE[key].tree) {
            ret.push(leef[0]);
        }
        return ret;
    }
    list_child() {
        const root = this.rootname();
        if (this._roots[root].length === 0) {
            this._roots[root] = Songs.sort(this.mpd.librarySongs, TREE[root].sort, Library._mkmemo(root));
        }
        const filters = {};
        for (let i = 0, imax = this.tree.length; i < imax; i++) {
            if (i === 0) {
                continue;
            }
            filters[this.tree[i][0]] = this.tree[i][1];
        }
        const ret = {};
        ret.key = TREE[root].tree[this.tree.length - 1][0];
        ret.songs = this._roots[root];
        ret.songs = Songs.filter(ret.songs, filters);
        ret.songs = Songs.uniq(ret.songs, ret.key);
        ret.style = TREE[root].tree[this.tree.length - 1][1];
        ret.isdir = this.tree.length !== TREE[root].tree.length;
        return ret;
    }
    /*static*/ list_root() {
        const ret = [];
        if (this.mpd.librarySongs.length !== 0) {
            for (let i = 0, imax = TREE_ORDER.length; i < imax; i++) {
                ret.push({ root: [TREE_ORDER[i]] });
            }
        }
        return { key: "root", songs: ret, style: "plain", isdir: true };
    }
    update_list() {
        if (this.rootname() === "root") {
            this._list_cache = this.list_root();
            return true;
        }
        const cache = this._list_child_cache[this.tree.length - 1];
        const pwd = JSON.stringify(this.tree);
        if (cache.pwd === pwd) {
            this._list_cache = cache.data;
            return false;
        }
        this._list_cache = this.list_child();
        if (this._list_cache.songs.length === 0) {
            this.up();
        } else {
            this._list_child_cache[this.tree.length - 1].pwd = pwd;
            this._list_child_cache[this.tree.length - 1].data = this._list_cache;
        }
        return true;
    }
    list() {
        if (!this._list_cache.songs || !this._list_cache.songs.length === 0) {
            this.update_list();
        }
        return this._list_cache;
    }
    updateData(data) {
        for (let i = 0, imax = this._list_child_cache.length; i < imax; i++) {
            this._list_child_cache[i] = {};
        }
        for (const key in TREE) {
            if (TREE.hasOwnProperty(key)) {
                if (key === this._root) {
                    this._roots[key] = Songs.sort(data, TREE[key].sort, Library._mkmemo(key));
                } else {
                    this._roots[key] = [];
                }
            }
        }
    }
    update(data) {
        this.updateData(data);
        this.update_list();
        if (!this._lastabs || Song.get(this._lastabs, "file") !== Song.get(this.mpd.current, "file")) {
            this._autoFocus();
        }
        this.raiseEvent("update");
    }
    rootname() {
        let r = "root";
        if (this.tree.length !== 0) {
            r = this.tree[0][1];
            this._lastRoot = r;
        }
        if (r !== this._root) {
            this._root = r;
            this.save();
        }
        return r;
    }
    filters(pos) { return this._roots[this.rootname()][pos].keys; }
    sortkeys() {
        const r = this.rootname();
        if (r === "root") {
            return [];
        }
        return TREE[r].sort;
    }
    up() {
        const songs = this.list().songs;
        if (songs[0]) {
            this.focus = songs[0];
            if (this.rootname() === "root") {
                this.child = null;
            } else {
                this.child = this.tree[this.tree.length - 1][1];
            }
        }
        if (this.rootname() !== "root") {
            this.tree.pop();
        }
        this.update_list();
        if (this.list().songs.length === 1 && this.tree.length !== 0) {
            this.up();
        } else {
            this.raiseEvent("changed");
        }
    }
    down(value) {
        let r = this.rootname();
        if (r === "root") {
            this.tree.push([r, value]);
            r = value;
        } else {
            const key = TREE[r].tree[this.tree.length - 1][0];
            this.tree.push([key, value]);
        }
        this.focus = {};
        this.child = null;
        this.update_list();
        const songs = this.list().songs;
        if (songs.length === 1 && TREE[r].tree.length !== this.tree.length) {
            this.down(Song.getOne(songs[0], this.list().key));
        } else {
            this.raiseEvent("changed");
        }
    }
    absaddr(first, second) {
        this.tree.splice(0, this.tree.length);
        this.tree.push(["root", first]);
        this.down(second);
        this.raiseEvent("list");
    }
    absFallback(song) {
        const root = this._lastRoot;
        if (root !== "root" && song && song.file) {
            this.tree.length = 0;
            this.tree.splice(0, this.tree.length);
            this.tree.push(["root", root]);
            const selected = TREE[root].tree;
            for (let i = 0, imax = selected.length; i < imax; i++) {
                if (i === selected.length - 1) {
                    break;
                }
                const key = selected[i][0];
                this.tree.push([key, Song.getOne(song, key)]);
            }
            this.update_list();
            for (const candidate of this.list().songs) {
                if (candidate.file && candidate.file[0] === song.file[0]) {
                    this._lastabs = candidate;
                    this.focus = candidate;
                    this.child = null;
                    break;
                }
            }
        } else {
            this.tree.splice(0, this.tree.length);
            this.update_list();
        }
        this.raiseEvent("changed");
    }
    absSorted(song) {
        if (!song || !song.Pos) {
            return;
        }
        let root = "";
        const pos = parseInt(song.Pos[0], 10);
        const keys = this.mpd.playlist.sort.join();
        for (const key in TREE) {
            if (TREE.hasOwnProperty(key)) {
                if (TREE[key].sort.join() === keys) {
                    root = key;
                    break;
                }
            }
        }
        if (!root) {
            UINotification.show("fixme", `modal: unknown sort keys: ${keys}`);
            return;
        }
        let songs = this._roots[root];
        if (!songs || songs.length === 0) {
            this._roots[root] = Songs.sort(this.mpd.librarySongs, TREE[root].sort, Library._mkmemo(root));
            songs = this._roots[root];
            if (songs.length === 0) {
                return;
            }
        }
        if (songs.length > playlistLength || this.mpd.playlist.must) {
            const must = this.mpd.playlist.must ? this.mpd.playlist.must : 0;
            songs = Songs.weakFilter(songs, this.mpd.playlist.filters || [], must, playlistLength);
        }
        if (!songs[pos]) {
            return;
        }
        if (songs[pos].file[0] === song.file[0]) {
            this._lastabs = songs[pos];
            this.focus = songs[pos];
            this.child = null;
            this.tree.length = 0;
            this.tree.push(["root", root]);
            for (let i = 0; i < this.focus.keys.length - 1; i++) {
                this.tree.push(this.focus.keys[i]);
            }
            this.update_list();
            this.raiseEvent("changed");
        } else {
            this.absFallback(song);
        }
    }
    abs(song) {
        if (this.mpd.playlist && this.mpd.playlist.hasOwnProperty("sort") && this.mpd.playlist.sort !== null) {
            this.absSorted(song);
        } else {
            this.absFallback(song);
        }
    }
    parent() {
        const root = this.rootname();
        if (root === "root") {
            return;
        }
        const v = this.list().songs;
        if (this.tree.length > 1) {
            const key = TREE[root].tree[this.tree.length - 2][0];
            const style = TREE[root].tree[this.tree.length - 2][1];
            return { key: key, song: v[0], style: style, isdir: true };
        }
        return { key: "top", song: { top: [root] }, style: "plain", isdir: true };
    }
    grandparent() {
        const root = this.rootname();
        if (root === "root") {
            return;
        }
        const v = this.list().songs;
        if (this.tree.length > 2) {
            const key = TREE[root].tree[this.tree.length - 3][0];
            const style = TREE[root].tree[this.tree.length - 3][1];
            return { key: key, song: v[0], style: style, isdir: true };
        } else if (this.tree.length === 2) {
            return { key: "top", song: { top: [root] }, style: "plain", isdir: true };
        }
        return {
            key: "root",
            song: { root: ["Library"] },
            style: "plain",
            isdir: true,
        };
    }
};

class UI extends PubSub {
    constructor() {
        super();
    }
    polling() {
        this.raiseEvent("poll");
        setTimeout(() => { this.polling(); }, 1000);
    };
    static swipe(element, f, resetFunc, leftElement, conditionFunc) {
        element.swipe_target = f;
        let starttime = 0;
        let now = 0;
        let x = 0;
        let y = 0;
        let diff_x = 0;
        let diff_y = 0;
        let swipe = false;
        const start = e => {
            if ((e.buttons && e.buttons !== 1) || (conditionFunc && !conditionFunc())) {
                return;
            }
            const t = e.touches ? e.touches[0] : e;
            x = t.screenX;
            y = t.screenY;
            starttime = (new Date()).getTime();
            swipe = true;
        };
        const finalize = e => {
            starttime = 0;
            now = 0;
            x = 0;
            y = 0;
            diff_x = 0;
            diff_y = 0;
            swipe = false;
            e.currentTarget.classList.remove("swipe");
            e.currentTarget.classList.add("swiped");
            if (leftElement) {
                leftElement.classList.remove("swipe");
                leftElement.classList.add("swiped");
            }
            if (!resetFunc) {
                e.currentTarget.style.transform = "";
                if (leftElement) {
                    leftElement.style.transform = "";
                }
            }
            setTimeout(() => {
                element.classList.remove("swiped");
                if (leftElement) {
                    leftElement.classList.remove("swiped");
                }
            });
        };
        const cancel = e => {
            if (swipe) {
                finalize(e);
                if (resetFunc) {
                    resetFunc();
                }
            }
        };
        const move = e => {
            if (e.buttons === 0 || (e.buttons && e.buttons !== 1) || !swipe || (conditionFunc && !conditionFunc())) {
                cancel(e);
                return;
            }
            const t = e.touches ? e.touches[0] : e;
            diff_x = x - t.screenX;
            diff_y = y - t.screenY;
            now = (new Date()).getTime();
            if (now - starttime < 200 && Math.abs(diff_y) > Math.abs(diff_x)) {
                cancel(e);
            } else if (Math.abs(diff_x) > 3) {
                e.currentTarget.classList.add("swipe");
                e.currentTarget.style.transform = `translate3d(${diff_x * -1}px,0,0)`;
                if (leftElement) {
                    leftElement.classList.add("swipe");
                    leftElement.style.transform = `translate3d(${diff_x * -1 - e.currentTarget.offsetWidth}px,0,0)`;
                }
            }
        };
        const end = e => {
            if ((e.buttons && e.buttons !== 1) || !swipe || (conditionFunc && !conditionFunc())) {
                cancel(e);
                return;
            }
            const p = e.currentTarget.clientWidth / diff_x;
            if ((p > -4 && p < 0) || (now - starttime < 200 && Math.abs(diff_y) < Math.abs(diff_x) && diff_x < 0)) {
                finalize(e);
                f(e);
            } else {
                cancel(e);
            }
        };
        if ("ontouchend" in element) {
            element.addEventListener("touchstart", start, { passive: true });
            element.addEventListener("touchmove", move, { passive: true });
            element.addEventListener("touchend", end, { passive: true });
        } else {
            element.addEventListener("mousedown", start, { passive: true });
            element.addEventListener("mousemove", move, { passive: true });
            element.addEventListener("mouseup", end, { passive: true });
        }
    }
    static disableSwipe(element) {
        const f = (e) => { e.stopPropagation(); };
        if ("ontouchend" in element) {
            element.addEventListener("touchstart", f, { passive: true });
            element.addEventListener("touchmove", f, { passive: true });
            element.addEventListener("touchend", f, { passive: true });
        } else {
            element.addEventListener("mousedown", f, { passive: true });
            element.addEventListener("mousemove", f, { passive: true });
            element.addEventListener("mouseup", f, { passive: true });
        }
    }
    static click(element, f) {
        element.click_target = f;
        const enter = e => { e.currentTarget.classList.add("hover"); };
        const leave = e => { e.currentTarget.classList.remove("hover"); };
        const start = e => {
            if (e.buttons && e.buttons !== 1) {
                return;
            }
            const t = e.touches ? e.touches[0] : e;
            e.currentTarget.x = t.screenX;
            e.currentTarget.y = t.screenY;
            e.currentTarget.touch = true;
            e.currentTarget.classList.add("active");
        };
        const move = e => {
            if (e.buttons && e.buttons !== 1 || !e.currentTarget.touch) {
                return;
            }
            const t = e.touches ? e.touches[0] : e;
            if (Math.abs(e.currentTarget.x - t.screenX) >= 5 || Math.abs(e.currentTarget.y - t.screenY) >= 5) {
                e.currentTarget.touch = false;
                e.currentTarget.classList.remove("active");
            }
        };
        const end = e => {
            if (e.buttons && e.buttons !== 1) {
                return;
            }
            e.currentTarget.classList.remove("active");
            if (e.currentTarget.touch) {
                f(e);
            }
        };
        if ("ontouchend" in element) {
            element.addEventListener("touchstart", start, { passive: true });
            element.addEventListener("touchmove", move, { passive: true });
            element.addEventListener("touchend", end, { passive: true });
        } else {
            element.addEventListener("mousedown", start, { passive: true });
            element.addEventListener("mousemove", move, { passive: true });
            element.addEventListener("mouseup", end, { passive: true });
            element.addEventListener("mouseenter", enter, { passive: true });
            element.addEventListener("mouseleave", leave, { passive: true });
        }
    }
};

class ImageFader {
    constructor(preferences, e1, e2) {
        this.preferences = preferences;
        this.e1 = e1;
        this.e2 = e2;
        this.path = "";
        if (e1.nodeName.toLowerCase() === "img") {
            this.t1 = e1;
            this.t1.addEventListener("load", (e) => { this._onImages(e.currentTarget) });
        } else {
            this.t1 = new Image();
            this.t1.addEventListener("load", (e) => {
                this._onImages(e1);
                e1.style.backgroundImage = `url("${this.path}")`;
            });
        }
        this.t1.addEventListener("load", (e) => { this._onImages(this.e1) });
        if (e2.nodeName.toLowerCase() === "img") {
            this.t2 = e2;
            this.t2.addEventListener("load", (e) => { this._onImages(e.currentTarget) });
        } else {
            this.t2 = new Image();
            this.t2.addEventListener("load", (e) => {
                this._onImages(e2);
                e2.style.backgroundImage = `url("${this.path}")`;
            });
        }
    }
    _onImages(o) {
        if (o.classList.contains("current") && o.dataset.src === this.path) {
            o.classList.add("show");
            if (o.isEqualNode(this.e1)) {
                this.e2.dataset.src = "";
            } else {
                this.e1.dataset.src = "";
            }
        }
    }
    show(path) {
        this.path = path;
        if (!this.preferences.appearance.crossfading_image) {

            this.e2.classList.remove("current");
            this.e2.classList.remove("show");
            this.e1.classList.add("current");
            this.t1.src = path;
            this.e1.dataset.src = path;
            return;
        }
        if (this.e1.classList.contains("current")) {
            if (this.e1.dataset.src === path) {
                return;
            }
            this.e1.classList.remove("current")
            this.e1.classList.remove("show")
            this.e2.classList.add("current")
            this.e2.classList.remove("show")
            this.t2.src = path;
            this.e2.dataset.src = path;
            return;
        }
        if (this.e2.dataset.src === path) {
            return;
        }
        this.e2.classList.remove("current")
        this.e2.classList.remove("show")
        this.e1.classList.add("current")
        this.e1.classList.remove("show")
        this.t1.src = path;
        this.e1.dataset.src = path;
    }
}

// background
class UIBackground {
    constructor(ui, mpd, preferences) {
        this.mpd = mpd;
        this.preferences = preferences;
        this.rgbg = { r: 128, g: 128, b: 128, gray: 128 };
        this.preferences.addEventListener("appearance", (e) => { this.update_theme(e); });
        ui.addEventListener("load", () => { this.onStart(); });
    }
    onStart() {
        document.body.classList.remove("unload");
        const img = new ImageFader(this.preferences, document.getElementById("background-image"), document.getElementById("background-image2"));
        if (this.mpd.current !== null && this.mpd.current.cover && this.mpd.current.cover[0]) {
            this.update_color(this.mpd.current.cover[0]);
            img.show(this.mpd.current.cover[0]);
        } else {
            img.show(this.preferences.nocover());
        }
        this.mpd.addEventListener("current", () => {
            if (this.mpd.current !== null && this.mpd.current.cover && this.mpd.current.cover[0]) {
                this.update_color(this.mpd.current.cover[0]);
                img.show(this.mpd.current.cover[0]);
            } else {
                img.show(this.preferences.nocover());
            }
        });
        var darkmode = window.matchMedia("(prefers-color-scheme: dark)");
        if (darkmode.addEventListener) {
            darkmode.addEventListener("change", () => { this.update_theme(); });
        } else if (darkmode.addListener) {
            darkmode.addListener(() => { this.update_theme(); });
        }
        this.update_theme();
    };
    static mkcolor(rgb, magic) {
        return "#" + (((1 << 24) + (magic(rgb.r) << 16) + (magic(rgb.g) << 8) + magic(rgb.b)).toString(16).slice(1));
    };
    static darker(c) {
        // Vivaldi does not recognize #000000
        if ((c - 20) < 0) {
            return 1;
        }
        return c - 20;
    };
    static lighter(c) {
        if ((c + 100) > 255) {
            return 255;
        }
        return c + 100;
    };
    update_theme() {
        const color = document.querySelector("meta[name=theme-color]");
        if (this.preferences.appearance.theme === "prefer-system") {
            document.body.classList.add("system-theme-color");
            document.body.classList.remove("dark");
            document.body.classList.remove("light");
            if (window.matchMedia("(prefers-color-scheme: dark)").matches) {
                color.setAttribute("content", UIBackground.mkcolor(this.rgbg, UIBackground.darker));
            } else {
                color.setAttribute("content", UIBackground.mkcolor(this.rgbg, UIBackground.lighter));
            }
        } else {
            document.body.classList.remove("system-theme-color");
            var dark = true;
            if (this.preferences.appearance.theme === "light") {
                dark = false;
            } else if (this.preferences.appearance.theme !== "dark" && this.rgbg.gray >= this.preferences.appearance.color_threshold) {
                dark = false;
            }
            if (dark) {
                document.body.classList.add("dark");
                document.body.classList.remove("light");
                color.setAttribute("content", UIBackground.mkcolor(this.rgbg, UIBackground.darker));
            } else {
                document.body.classList.add("light");
                document.body.classList.remove("dark");
                color.setAttribute("content", UIBackground.mkcolor(this.rgbg, UIBackground.lighter));
            }
        }
        const e1 = document.getElementById("background-image");
        const e2 = document.getElementById("background-image2");
        if (!this.preferences.appearance.background_image) {
            e1.classList.add("hide");
            e2.classList.add("hide");
            return;
        }
        e1.classList.remove("hide");
        e2.classList.remove("hide");
        e1.style.filter = `blur(${this.preferences.appearance.background_image_blur})`;
        e2.style.filter = `blur(${this.preferences.appearance.background_image_blur})`;
    };
    update_color(path) {
        const img = new Image();
        img.onload = () => {
            const canvas = document.createElement("canvas");
            const context = canvas.getContext("2d");
            context.drawImage(img, 0, 0, 5, 5);
            try {
                const d = context.getImageData(0, 0, 5, 5).data;
                const new_rgbg = { r: 128, g: 128, b: 128, gray: 128 };
                for (let i = 0; i < d.length - 3; i += 4) {
                    new_rgbg.r += d[i];
                    new_rgbg.g += d[i + 1];
                    new_rgbg.b += d[i + 2];
                    new_rgbg.gray += (d[i] + d[i + 1] + d[i + 2]);
                }
                new_rgbg.r = parseInt(new_rgbg.r * 3 / d.length, 10);
                new_rgbg.g = parseInt(new_rgbg.g * 3 / d.length, 10);
                new_rgbg.b = parseInt(new_rgbg.b * 3 / d.length, 10);
                new_rgbg.gray /= d.length;
                this.rgbg = new_rgbg;
                this.update_theme();
            } catch (e) {
                // failed to getImageData
            }
        };
        img.src = path;
    };
};

class UIMainView extends PubSub {
    constructor(ui, mpd, library, preferences) {
        super();
        ui.addEventListener("poll", () => { this.onPoll(); });
        ui.addEventListener("load", () => { this.onStart(); });
        this.mpd = mpd;
        this.mpd.addEventListener("playlist", () => { this.onPlaylist(); });
        this.mpd.addEventListener("current", () => { this.onCurrent(); });
        this.mpd.addEventListener("control", () => { this.onControl(); });
        this.library = library;
        this.preferences = preferences;
        this.preferences.addEventListener("appearance", () => { this.onPreferences(); });
    }
    onPreferences() {
        const e = document.getElementById("main-cover");
        if (this.preferences.appearance.circled_image) {
            e.classList.add("circled");
        } else {
            e.classList.remove("circled");
        }
    }
    onControl() {
        const o = document.getElementById("main-cover-overlay");
        if (this.mpd.control.state === "play") {
            o.classList.add("pause");
            if (o.classList.contains("play")) {
                o.classList.remove("play");
                o.classList.add("changed");
                requestAnimationFrame(() => { o.classList.remove("changed"); }, 10);
            }
        } else {
            o.classList.add("play");
            if (o.classList.contains("pause")) {
                o.classList.remove("pause");
                o.classList.add("changed");
                requestAnimationFrame(() => { o.classList.remove("changed"); }, 10);
            }
        }
    }
    onPlaylist() {
        document.getElementById("main-cover-overlay").disabled = !(this.mpd.playlist && this.mpd.playlist.hasOwnProperty("current"));
    }
    show() {
        document.body.classList.add("view-main");
        document.body.classList.remove("view-list");
    }
    hidden() {
        const c = document.body.classList;
        if (window.matchMedia("(orientation: portrait)").matches) {
            return !c.contains("view-main");
        }
        return !(c.contains("view-list") || c.contains("view-main"));
    }
    update() {
        if (this.mpd.current === null) {
            return;
        }
        document.getElementById("main-box-title").textContent = this.mpd.current.Title;
        document.getElementById("main-box-artist").textContent = this.mpd.current.Artist;
        document.getElementById("main-seek-label-total").textContent = this.mpd.current.Length;
    }
    onCurrent() { this.update(); }
    onPoll() {
        if (this.mpd.current === null || !this.mpd.current.Time) {
            return;
        }
        if (this.hidden()) {
            return;
        }
        const c = document.getElementById("main-cover-circle-active");
        let elapsed = parseInt(this.mpd.control.song_elapsed * 1000, 10);
        if (this.mpd.control.state === "play") {
            elapsed += (new Date()).getTime() - this.mpd.last_modified_ms.control;
        }
        const total = parseInt(this.mpd.current.Time[0], 10);
        if (!isNaN(elapsed / total)) {
            document.getElementById("main-seek").value = elapsed / total;
        }

        if (document.getElementById("main-cover-circle").classList.contains("hide")) {
            return;
        }
        const d = (elapsed * 360 / 1000 / total - 90) * (Math.PI / 180);
        if (isNaN(d)) {
            return;
        }
        const x = 100 + 90 * Math.cos(d);
        const y = 100 + 90 * Math.sin(d);
        if (x <= 100) {
            c.setAttribute("d", "M 100,10 L 100,10 A 90,90 0 0,1 100,190 L 100,190 A 90,90 0 0,1 " + `${x},${y}`);
        } else {
            c.setAttribute("d", `M 100,10 L 100,10 A 90,90 0 0,1 ${x},${y}`);
        }
    }
    onStart() {
        document.getElementById("control-volume").addEventListener("change", e => {
            this.mpd.volume(parseInt(e.currentTarget.value, 10));
        });
        const img = new ImageFader(this.preferences, document.getElementById("main-cover-img"), document.getElementById("main-cover-img2"));

        if (this.mpd.current !== null && this.mpd.current.cover && this.mpd.current.cover[0]) {
            img.show(this.mpd.current.cover[0]);
        } else {
            img.show(this.preferences.nocover());
        }
        this.mpd.addEventListener("current", () => {
            if (this.mpd.current !== null && this.mpd.current.cover && this.mpd.current.cover[0]) {
                img.show(this.mpd.current.cover[0]);
            } else {
                img.show(this.preferences.nocover());
            }
        });
        UI.click(document.getElementById("main-cover-overlay"), () => {
            if (window.matchMedia("(max-height: 450px) and (orientation: landscape)").matches) {
                this.mpd.togglePlay();
            }
        });
        UI.click(document.getElementById("main-box-title"), () => {
            if (this.mpd.current !== null) {
                UIModal.song(this.mpd.current, this.library, this.preferences);
            }
        });
        UI.disableSwipe(document.getElementById("main-seek"));
        document.getElementById("main-seek").addEventListener("input", (e) => {
            if (this.mpd.current.Time && this.mpd.current.Time[0]) {
                const target = parseInt(e.currentTarget.value, 10) * parseInt(this.mpd.current.Time[0], 10) / 1000;
                this.mpd.seek(target);
            }
        });
        this.onPreferences();
        UI.swipe(
            document.getElementById("main"), () => { this.raiseEvent("list"); }, null,
            document.getElementById("lists"), () => { return window.innerHeight >= window.innerWidth; });
        this.onCurrent();
        this.onPlaylist();
    }
};

class UIListView extends PubSub {
    constructor(ui, mpd, library, preferences) {
        super();
        this.mpd = mpd;
        this.mpd.addEventListener("current", () => { this.onCurrent(); });
        this.preferences = preferences;
        this.preferences.addEventListener("appearance", () => { this.onPreferences(); });
        this.library = library;
        ui.addEventListener("load", () => { this.onStart(); });
    }
    show() {
        document.body.classList.add("view-list");
        document.body.classList.remove("view-main");
    }
    hidden() {
        const c = document.body.classList;
        if (window.matchMedia("(orientation: portrait)").matches) {
            return !c.contains("view-list");
        }
        return !(c.contains("view-list") || c.contains("view-main"));
    }
    _preferences_update() {
        const index = this.library.tree.length;
        const ul = document.getElementById("list-items" + index);
        if (this.preferences.appearance.playlist_gridview_album) {
            ul.classList.add("grid");
            ul.classList.remove("nogrid");
        } else {
            ul.classList.add("nogrid");
            ul.classList.remove("grid");
        }
    }
    _updatepos() {
        const index = this.library.tree.length;
        const lists = document.getElementsByClassName("list");
        for (let listindex = 0; listindex < lists.length; listindex++) {
            if (listindex < index) {
                lists[listindex].style.transform = "translate3d(-100%,0,0)";
            } else if (listindex === index) {
                lists[listindex].style.transform = "translate3d(0,0,0)";
            } else {
                lists[listindex].style.transform = "translate3d(100%,0,0)";
            }
        }
    }
    _updateFocus() {
        const index = this.library.tree.length;
        const ul = document.getElementById("list-items" + index);
        let focus = null;
        let viewNowPlaying = false;
        const rootname = this.library.rootname();
        const focusSong = this.library.focus;
        const focusParent = this.library.child;
        for (const listitem of Array.from(ul.children)) {
            if (listitem.classList.contains("list-header")) {
                continue;
            }
            if (focusSong && focusSong.file && focusParent) {
                if (focusParent == listitem.dataset.key) { // focusParent is null or string, dataset.key is undefined or string
                    focus = listitem;
                    focus.classList.add("selected");
                } else {
                    listitem.classList.remove("selected");
                }
            } else if (rootname !== "root" && focusSong && focusSong.file && listitem.dataset.file === focusSong.file[0]) {
                focus = listitem;
                focus.classList.add("selected");
            } else {
                listitem.classList.remove("selected");
            }
            let treeFocused = true;
            if (this.mpd.playlist && this.mpd.playlist.hasOwnProperty("sort") && this.mpd.playlist.sort !== null) {
                if (rootname === "root") {
                    treeFocused = false;
                } else if (this.mpd.playlist.sort.join() !== TREE[rootname].sort.join()) {
                    treeFocused = false;
                }
            }
            const elapsed = Array.from(listitem.getElementsByClassName("song-elapsed"));
            const sep = Array.from(listitem.getElementsByClassName("song-lengthseparator"));
            if (treeFocused && elapsed.length !== 0 && this.mpd.current !== null && this.mpd.current.file && this.mpd.current.file[0] === listitem.dataset.file) {
                viewNowPlaying = true;
                if (focusSong && !focusSong.file) {
                    listitem.classList.add("selected");
                }
                if (listitem.classList.contains("playing")) {
                    continue;
                }
                listitem.classList.add("playing");
                for (const e of elapsed) {
                    e.classList.add("elapsed");
                    e.setAttribute("aria-hidden", "false");
                }
                for (const s of sep) {
                    s.setAttribute("aria-hidden", "false");
                }
            } else {
                if (!listitem.classList.contains("playing")) {
                    continue;
                }
                listitem.classList.remove("playing");
                for (const e of elapsed) {
                    e.classList.remove("elapsed");
                    e.setAttribute("aria-hidden", "true");
                }
                for (const s of sep) {
                    s.setAttribute("aria-hidden", "true");
                }
            }
        }

        const scroll = document.getElementById("list" + index);
        if (focus) {
            window.requestAnimationFrame(() => {
                const pos = focus.offsetTop;
                const t = scroll.scrollTop;
                if (t >= pos || pos >= t + scroll.clientHeight) {
                    scroll.scrollTop = pos;
                }
            });
        } else {
            scroll.scrollTop = 0;
        }

        if (viewNowPlaying) {
            document.getElementById("header-main").classList.add("playing");
        } else {
            document.getElementById("header-main").classList.remove("playing");
        }
    }
    _clearAllLists() {
        const lists = document.getElementsByClassName("list");
        for (let treeindex = 0; treeindex < lists.length; treeindex++) {
            const oldul = lists[treeindex].getElementsByClassName("list-items")[0];
            while (oldul.lastChild) {
                oldul.removeChild(oldul.lastChild);
            }
            lists[treeindex].dataset.pwd = "";
        }
    }
    _element(song, key, style, header) {
        const c = document.querySelector(`#list-${style}-template`).content;
        const e = c.querySelector("li");
        const v = Song.getOne(song, key);
        if (v !== null) {
            e.dataset.key = v;
        } else {
            e.removeAttribute("data-key"); // safari does not support delete
        }
        if (header) {
            e.classList.add("list-header");
            e.classList.remove("selectable");
        } else {
            e.classList.add("selectable");
            e.classList.remove("list-header");
        }
        if (song.file) {
            e.dataset.file = song.file[0];
            e.dataset.pos = song.pos;
        } else {
            e.dataset.file = "";
            e.dataset.pos = "";
        }
        for (const n of e.querySelectorAll("span")) {
            if (!n.dataset) {
                continue;
            }
            const target = n.dataset.textContent;
            if (target === "key") {
                n.textContent = v ? v : `[no ${key}]`;
            } else if (target) {
                n.textContent = Song.get(song, target);
            }
        }
        if (style === "song") {
            if (song.file) {
                const tooltip = ["Length", "Artist", "Album", "Track", "Genre", "Performer"].map(key => `${key}: ${Song.get(song, key)}`);
                tooltip.unshift(Song.get(song, "Title"));
                e.setAttribute("title", tooltip.join("\n"));
            } else {
                e.removeAttribute("title");
            }
        } else if (style === "album") {
            const smallCover = c.querySelector(".small-album-cover");
            const mediumCover = c.querySelector(".medium-album-cover");
            if (song.cover && song.cover.length !== 0) {
                const smallImgsize = parseInt(70 * window.devicePixelRatio, 10);
                const mediumImgsize = parseInt(150 * window.devicePixelRatio, 10);
                const s = (song.cover[0].lastIndexOf("?") == -1) ? "?" : "&";
                smallCover.src = `${song.cover[0]}${s}width=${smallImgsize}&height=${smallImgsize}`;
                smallCover.width = smallImgsize;
                smallCover.height = smallImgsize;
                mediumCover.src = `${song.cover[0]}${s}width=${mediumImgsize}&height=${mediumImgsize}`;
                mediumCover.width = mediumImgsize;
                mediumCover.height = mediumImgsize;
            } else {
                smallCover.src = this.preferences.nocover();
                mediumCover.src = this.preferences.nocover();
            }
            smallCover.alt = `Cover art: ${Song.get(song, "Album")} ` + `by ${Song.get(song, "AlbumArtist")}`;
            mediumCover.alt = smallCover.alt;
        }
        return document.importNode(c, true);
    }
    _listHandler(e) {
        if (e.currentTarget.classList.contains("playing")) {
            if (this.mpd.current === null) {
                return;
            }
            this.library.abs(this.mpd.current);
            this.raiseEvent("main");
            return;
        }
        const value = e.currentTarget.dataset.key ? e.currentTarget.dataset.key : null;
        const pos = parseInt(e.currentTarget.dataset.pos, 10);
        if (e.currentTarget.classList.contains("song")) {
            const filters = this.library.filters(pos);
            let must = 0;
            if (this.preferences.playlist.playback_tracks === "list") {
                must = filters.length - 1;
            } else if (this.preferences.playlist.playback_tracks === "custom") {
                const root = this.library.rootname();
                if (root !== "root") {
                    must = this.preferences.playlist.playback_tracks_custom[root];
                }
            }
            this.mpd.sortPlaylist(this.library.sortkeys(), filters, must, pos);
        } else {
            this.library.down(value);
        }
    }
    _update() {
        const index = this.library.tree.length;
        const scroll = document.getElementById("list" + index);
        const pwd = JSON.stringify(this.library.tree);
        if (scroll.dataset.pwd === pwd) {
            this._updatepos();
            this._updateFocus();
            return;
        }
        scroll.dataset.pwd = pwd;
        const ls = this.library.list();
        const key = ls.key;
        const songs = ls.songs;
        const style = ls.style;
        const newul = document.createDocumentFragment();
        const lists = document.getElementsByClassName("list");
        for (let treeindex = 0; treeindex < this.library.tree.length; treeindex++) {
            const currentpwd = JSON.stringify(this.library.tree.slice(0, treeindex + 1));
            const viewpwd = lists[treeindex + 1].dataset.pwd;
            if (currentpwd !== viewpwd) {
                const oldul = lists[treeindex + 1].getElementsByClassName("list-items")[0];
                while (oldul.lastChild) {
                    oldul.removeChild(oldul.lastChild);
                }
                lists[treeindex + 1].dataset.pwd = "";
            }
        }
        this._updatepos();
        const ul = document.getElementById("list-items" + index);
        while (ul.lastChild) {
            ul.removeChild(ul.lastChild);
        }
        ul.classList.remove("songlist");
        ul.classList.remove("albumlist");
        ul.classList.remove("plainlist");
        ul.classList.add(style + "list");
        this._preferences_update();
        const p = this.library.parent();
        for (let i = 0, imax = songs.length; i < imax; i++) {
            if (i === 0 && p) {
                const li = this._element(p.song, p.key, p.style, true);
                newul.appendChild(li);
            }
            const li = this._element(songs[i], key, style, false);
            UI.click(li.querySelector("li"), (e) => { this._listHandler(e); }, false);
            newul.appendChild(li);
        }
        ul.appendChild(newul);
        this._updateFocus();
    }
    _updateForce() {
        this._clearAllLists();
        this._update();
    }
    _select_near_item() {
        const index = this.library.tree.length;
        const scroll = document.getElementById("list" + index);
        let updated = false;
        for (const selectable of document.querySelectorAll(`#list-items${index} .selectable`)) {
            const p = selectable.offsetTop;
            if (scroll.scrollTop < p && p < scroll.scrollTop + scroll.clientHeight && !updated) {
                selectable.classList.add("selected");
                updated = true;
            } else {
                selectable.classList.remove("selected");
            }
        }
    }
    _select_focused_or(target) {
        const style = this.library.list().style;
        const index = this.library.tree.length;
        const scroll = document.getElementById("list" + index);
        const list = document.getElementById("list-items" + index);
        const t = scroll.scrollTop;
        const h = scroll.clientHeight;
        const selected = list.getElementsByClassName("selected");
        const playing = list.getElementsByClassName("playing");
        if (selected.length === 0 && playing.length === 1) {
            const p = playing[0].offsetTop;
            if (t < p && p < t + h) {
                playing[0].classList.add("selected");
                return;
            }
        }
        if (selected.length > 0) {
            const p = selected[0].offsetTop;
            if (p < t || t + h < p + selected[0].offsetHeight) {
                this._select_near_item();
                return;
            }
        }
        if (selected.length === 0 && playing.length === 0) {
            this._select_near_item();
            return;
        }
        if (selected.length > 0) {
            const selectables = list.getElementsByClassName("selectable");
            let itemcount = 1;
            if (this.preferences.appearance.playlist_gridview_album && style === "album") {
                if (selectables.length !== 0) {
                    itemcount = parseInt(scroll.clientWidth / selectables[0].clientWidth, 10);
                }
            }
            if (target === "up" && selectables[0] === selected[0]) {
                return;
            }
            if (target === "down" && selectables[selectables.length - 1] === selected[0]) {
                return;
            }
            for (let i = 0; i < selectables.length; i++) {
                const item = selectables[i];
                if (item === selected[0]) {
                    if (i > 0 && target === "left") {
                        const left = selectables[i - 1];
                        item.classList.remove("selected");
                        left.classList.add("selected");
                        p = left.offsetTop;
                        if (p < t) {
                            scroll.scrollTop = p;
                        }
                        return;
                    }
                    if (i > itemcount - 1 && target === "up") {
                        const up = selectables[i - itemcount];
                        item.classList.remove("selected");
                        up.classList.add("selected");
                        p = up.offsetTop;
                        if (p < t) {
                            scroll.scrollTop = p;
                        }
                        return;
                    }
                    if (i !== (selectables.length - 1) && target === "right") {
                        const right = selectables[i + 1];
                        item.classList.remove("selected");
                        right.classList.add("selected");
                        p = right.offsetTop + right.offsetHeight;
                        if (t + h < p) {
                            scroll.scrollTop = p - h;
                        }
                        return;
                    }
                    if (i < (selectables.length - 1) && target === "down") {
                        let down = null;
                        if (i + itemcount >= selectables.length) {
                            down = selectables[selectables.length - 1];
                        } else {
                            down = selectables[i + itemcount];
                        }
                        item.classList.remove("selected");
                        down.classList.add("selected");
                        p = down.offsetTop + down.offsetHeight;
                        if (t + h < p) {
                            scroll.scrollTop = p - h;
                        }
                        return;
                    }
                }
            }
        }
    }
    up() { this._select_focused_or("up"); }
    left() { this._select_focused_or("left"); }
    right() { this._select_focused_or("right"); }
    down() { this._select_focused_or("down"); }
    activate() {
        const index = this.library.tree.length;
        const es = document.getElementById("list-items" + index).getElementsByClassName("selected");
        if (es.length !== 0) {
            const e = {};
            e.currentTarget = es[0];
            es[0].click_target(e);
            return true;
        }
        return false;
    }
    onCurrent() { this._update(); }
    onPreferences() { this._preferences_update(); }
    onStart() {
        this.library.addEventListener("update", () => { this._updateForce(); });
        this.library.addEventListener("changed", () => { this._update(); });
        this.library.addEventListener("list", () => { this.show(); });
        const list = [
            document.getElementById("list0"),
            document.getElementById("list1"),
            document.getElementById("list2"),
            document.getElementById("list3"),
            document.getElementById("list4"),
            document.getElementById("list5"),
        ];
        UI.swipe(list[1], () => { this.library.up(); }, () => { this._updatepos(); }, list[0]);
        UI.swipe(list[2], () => { this.library.up(); }, () => { this._updatepos(); }, list[1]);
        UI.swipe(list[3], () => { this.library.up(); }, () => { this._updatepos(); }, list[2]);
        UI.swipe(list[4], () => { this.library.up(); }, () => { this._updatepos(); }, list[3]);
        UI.swipe(list[5], () => { this.library.up(); }, () => { this._updatepos(); }, list[4]);
        this.onCurrent();
    }
};

class UISystemWindow {
    constructor(ui, mpd, preferences) {
        ui.addEventListener("load", (e) => { this.onStart(e); });
        this.mpd = mpd;
        this.mpd.addEventListener("version", (e) => { this.onVersion(e); });
        this.mpd.addEventListener("library", (e) => { this.onLibrary(e); });
        this.mpd.addEventListener("images", (e) => { this.onImages(e); });
        this.mpd.addEventListener("stats", (e) => { this.onStats(e); });
        this.mpd.addEventListener("outputs", (e) => { this.onOutputs(e); });
        this.mpd.addEventListener("storage", (e) => { this.onStorage(e); });
        this.preferences = preferences;
        this.preferences.addEventListener("appearance", () => { this.onPreferencesAppearance(); });
        this.preferences.addEventListener("playlist", () => { this.onPreferencesPlaylist(); });
        this.preferences.addEventListener("outputs", () => { this.onPreferencesOutputs(); });
        this.preferences.addEventListener("httpoutput", () => { this.onPreferencesHTTPOutout(); });
    }
    _initconfig(id) {
        const obj = document.getElementById(id);
        const suffix = id.indexOf("_");  // remove _XX suffix for config key
        if (suffix !== -1) {
            id = id.slice(0, suffix);
        }
        const s = id.indexOf("-");
        const mainkey = id.slice(0, s);
        const subkey = id.slice(s + 1).replace(/-/g, "_");
        let getter = null;
        if (obj.type === "checkbox") {
            obj.checked = this.preferences[mainkey][subkey];
            getter = () => { return obj.checked; };
        } else if (obj.tagName.toLowerCase() === "select") {
            obj.value = String(this.preferences[mainkey][subkey]);
            getter = () => { return obj.value; };
        } else if (obj.type === "range") {
            obj.value = String(this.preferences[mainkey][subkey]);
            getter = () => { return parseFloat(obj.value); };
            obj.addEventListener("input", () => {
                this.preferences[mainkey][subkey] = getter();
                this.preferences.raiseEvent(mainkey);
            });
            UI.disableSwipe(obj);
        } else if (obj.type === "radio") {
            if (obj.value === this.preferences[mainkey][subkey]) {
                obj.checked = "checked";
            }
            getter = () => { return obj.value; };
        }
        obj.addEventListener("change", () => {
            this.preferences[mainkey][subkey] = getter();
            this.preferences.save();
            this.preferences.raiseEvent(mainkey);
        });
    }
    onPreferences() {
        this.onPreferencesAppearance();
        this.onPreferencesPlaylist();
        this.onPreferencesOutputs();
    }
    onPreferencesAppearance() {
        if (this.preferences.appearance.theme === "prefer-coverart") {
            document.getElementById("config-appearance-color-threshold").classList.remove("hide");
        } else {
            document.getElementById("config-appearance-color-threshold").classList.add("hide");
        }
    }
    onPreferencesPlaylist() {
        for (const e of document.querySelectorAll(`.playlist-playback-tracks-custom`)) {
            if (this.preferences.playlist.playback_tracks === "custom") {
                e.classList.remove("hide");
            } else {
                e.classList.add("hide");
            }
        }
    }
    onPreferencesOutputs() {
        document.getElementById("outputs-volume").max = this.preferences.outputs.volume_max;
    }
    onPreferencesHTTPOutout() {
        if (this.preferences.httpoutput.stream === "" || !this.preferences.feature.client_volume_control ) {
            document.getElementById("httpoutput-volume-group").classList.add("hide");
        } else {
            document.getElementById("httpoutput-volume-group").classList.remove("hide");
        }
        const httpVolume = document.getElementById("httpoutput-volume");
        // httpVolume.value = this.preferences.httpoutput.volume;
        httpVolume.max = this.preferences.httpoutput.volume_max;
        document.getElementById("httpoutput-volume-max").value = this.preferences.httpoutput.volume_max;
        document.getElementById("httpoutput-volume-string").textContent = (this.preferences.httpoutput.volume * 100).toFixed(1) + "%";
    }
    onStorage() {
        if (Object.keys(this.mpd.storage).length === 0) {
            document.getElementById("storage-header").classList.add("hide");
            document.getElementById("storage").classList.add("hide");
            return;
        }
        document.getElementById("storage-header").classList.remove("hide");
        document.getElementById("storage").classList.remove("hide");
        const ul = document.getElementById("storage-list");
        while (ul.lastChild) {
            ul.removeChild(ul.lastChild);
        }
        const newul = document.createDocumentFragment();
        for (const path in this.mpd.storage) {
            if (path !== "" && this.mpd.storage.hasOwnProperty(path)) {
                const s = this.mpd.storage[path];
                const c = document.querySelector("#storage-template").content;
                const e = c.querySelector("li");
                e.querySelector(".path").textContent = path;
                e.querySelector(".uri").textContent = s.uri;
                e.querySelector(".unmount").dataset.path = path;
                const d = document.importNode(c, true);
                d.querySelector(".unmount").addEventListener("click", (e) => {
                    HTTP.post("/api/music/storage", { [e.currentTarget.dataset.path]: { "uri": null } });
                });
                newul.appendChild(d);
            }
        }
        ul.appendChild(newul);
    }
    onOutputs() {
        const ul = document.getElementById("devices");
        while (ul.lastChild) {
            ul.removeChild(ul.lastChild);
        }
        const newul = document.createDocumentFragment();
        const inputs = document.getElementById("httpoutput-stream");
        const newInputs = document.createDocumentFragment();
        let streamIndex = 1;
        let streamChanged = false;
        let streams = {};
        for (const id in this.mpd.outputs) {
            if (this.mpd.outputs.hasOwnProperty(id)) {
                const o = this.mpd.outputs[id];
                const c = document.querySelector("#device-template").content;
                const e = c.querySelector("li");
                e.querySelector(".name").textContent = o.name;
                if (o.plugin) {
                    e.querySelector(".plugin").textContent = o.plugin;
                    e.querySelector(".plugin").classList.remove("hide");
                } else {
                    e.querySelector(".plugin").classList.add("hide");
                }
                const sw = e.querySelector(".device-switch");
                sw.setAttribute("aria-label", o.name);
                sw.dataset.deviceid = id;
                sw.checked = o.enabled;
                const dop = e.querySelector(".device-dop");
                dop.classList.add("hide");
                const dopSW = e.querySelector(".device-dop-switch");
                dopSW.dataset.deviceid = id;
                const allowedFormats = e.querySelector(".device-allowed-formats");
                allowedFormats.classList.add("hide");
                e.querySelector(".device-allowed-formats").dataset.deviceid = id;
                if (o.attributes) {
                    if (o.attributes.hasOwnProperty("dop") && o.enabled) {
                        dop.classList.remove("hide");
                        dopSW.checked = o.attributes.dop;
                    }
                    if (o.attributes.hasOwnProperty("allowed_formats") && o.enabled) {
                        allowedFormats.classList.remove("hide");
                    }
                }
                const d = document.importNode(c, true);
                d.querySelector(".device-switch").addEventListener("change", e => {
                    this.mpd.output(parseInt(e.currentTarget.dataset.deviceid, 10), e.currentTarget.checked);
                });
                d.querySelector(".device-dop-switch").addEventListener("change", e => {
                    HTTP.post("/api/music/outputs", {
                        [parseInt(e.currentTarget.dataset.deviceid, 10)]: { "attributes": { "dop": e.currentTarget.checked } }
                    });
                });
                d.querySelector(".device-allowed-formats").addEventListener("click", (e) => {
                    UISubModal.showAllowedFormats(parseInt(e.currentTarget.dataset.deviceid, 10), this.mpd);
                });
                newul.appendChild(d);
                if (o.stream && o.enabled) {
                    const on = document.createElement("option");
                    on.value = o.stream;
                    on.textContent = o.name;
                    newInputs.appendChild(on);
                    streams[o.name] = o.stream;
                    const t = inputs.children[streamIndex];
                    if (!t || t.textContent !== o.name || t.value !== o.stream) {
                        streamChanged = true;
                    }
                    streamIndex++;
                }
            }
        }
        for (const input of Array.from(inputs.children)) {
            if (input.value !== "" && !streams[input.textContent]) {
                streamChanged = true;
            }
        }
        if (streamChanged) {
            while (inputs.lastChild) {
                inputs.removeChild(inputs.lastChild);
            }
            const off = document.createElement("option");
            off.value = "";
            off.textContent = inputs.dataset.disabledLabel;
            inputs.appendChild(off);
            inputs.appendChild(newInputs);
            if (streamIndex === 1) {
                document.getElementById("httpoutput").classList.add("hide");
            } else {
                document.getElementById("httpoutput").classList.remove("hide");
            }
            inputs.value = this.preferences.httpoutput.stream;
            inputs.value = inputs.value;
            this.preferences.httpoutput.streams = streams;
            if (this.preferences.httpoutput.stream !== inputs.value) {
                this.preferences.httpoutput.stream = inputs.value;
                this.preferences.save();
            }
            if (inputs.value === "") {
                let e = document.createEvent("HTMLEvents");
                e.initEvent("change", false, true);
                inputs.dispatchEvent(e);
            }
        }
        ul.appendChild(newul);
    }
    onLibrary() {
        const e = document.getElementById("library-rescan");
        if (this.mpd.library.updating && !e.disabled) {
            e.disabled = true;
        } else if (!this.mpd.library.updating && e.disabled) {
            e.disabled = false;
        }
    }
    onImages() {
        const e = document.getElementById("library-rescan-images");
        if (this.mpd.images.updating && !e.disabled) {
            e.disabled = true;
        } else if (!this.mpd.images.updating && e.disabled) {
            e.disabled = false;
        }
    }
    onControl() {
        if (this.mpd.control.hasOwnProperty("volume") && this.mpd.control.volume !== null) {
            document.getElementById("outputs-volume").value = this.mpd.control.volume;
            document.getElementById("outputs-volume-string").textContent = this.mpd.control.volume + "%";
            document.getElementById("outputs-volume-box").classList.remove("hide");
            document.getElementById("appearance-volume-box").classList.remove("hide");
        } else {
            document.getElementById("outputs-volume-box").classList.add("hide");
            document.getElementById("appearance-volume-box").classList.add("hide");
        }
        document.getElementById("outputs-replay-gain").value = this.mpd.control.replay_gain;
        document.getElementById("outputs-crossfade").value = this.mpd.control.crossfade.toString(10);
    }
    onStart() {
        // preferences
        this.onPreferences();

        this.mpd.addEventListener("control", (e) => { this.onControl(e); });

        if (window.matchMedia("(prefers-color-scheme: dark)").matches === window.matchMedia("(prefers-color-scheme: light)").matches) {
            document.getElementById("appearance-theme_prefer-system").disabled = true;
            if (this.preferences.appearance.theme === "prefer-system") {
                this.preferences.appearance.theme = "prefer-coverart";
            }
        }

        this._initconfig("appearance-theme_light");
        this._initconfig("appearance-theme_dark");
        this._initconfig("appearance-theme_prefer-system");
        this._initconfig("appearance-theme_prefer-coverart");
        this._initconfig("appearance-color-threshold");
        this._initconfig("appearance-background-image");
        this._initconfig("appearance-background-image-blur");
        this._initconfig("appearance-circled-image");
        this._initconfig("appearance-crossfading-image");
        this._initconfig("appearance-playlist-gridview-album");
        this._initconfig("appearance-playlist-follows-playback");
        this._initconfig("appearance-volume");
        this._initconfig("playlist-playback-tracks_all");
        this._initconfig("playlist-playback-tracks_list");
        this._initconfig("playlist-playback-tracks_custom");
        this._initconfig("outputs-volume-max");
        document.getElementById("system-reload").addEventListener("click", () => {
            location.reload();
        });
        document.getElementById("storage-mount").addEventListener("click", () => {
            UISubModal.showStorageMount();
        });
        document.getElementById("library-rescan").addEventListener("click", () => {
            this.mpd.rescanLibrary();
        });
        document.getElementById("library-rescan-images").addEventListener("click", () => {
            HTTP.post("/api/music/images", { updating: true });
            this.mpd.images.updating = true;
            this.mpd.raiseEvent("images");
        });

        document.getElementById("outputs-replay-gain").addEventListener("change", (e) => {
            HTTP.post("/api/music", { replay_gain: e.currentTarget.value });
        });
        document.getElementById("outputs-crossfade").addEventListener("input", (e) => {
            HTTP.post("/api/music", { crossfade: parseInt(e.currentTarget.value) });
        });

        // info
        document.getElementById("user-agent").textContent = navigator.userAgent;

        const navs = Array.from(document.getElementsByClassName("system-nav-item"));
        const showChild = e => {
            document.getElementById("system-nav").classList.remove("on");
            document.getElementById("system-box-nav-back").classList.remove("root");
            for (const nav of navs) {
                if (nav === e.currentTarget) {
                    nav.classList.add("on");
                    document.getElementById(nav.dataset.target).classList.add("on");
                } else {
                    nav.classList.remove("on");
                    document.getElementById(nav.dataset.target).classList.remove("on");
                }
                document.getElementById(nav.dataset.target).classList.remove("fallback");
                nav.classList.remove("fallback");
            }
        };
        const showParent = () => {
            document.getElementById("system-nav").classList.add("on");
            document.getElementById("system-box-nav-back").classList.add("root");
            for (const nav of navs) {
                if (nav.classList.contains("on")) {
                    document.getElementById(nav.dataset.target).classList.remove("on");
                    document.getElementById(nav.dataset.target).classList.add("fallback");
                    nav.classList.remove("on");
                    nav.classList.add("fallback");
                }
            }
        };
        for (const nav of navs) {
            nav.addEventListener("click", showChild);
            UI.swipe(
                document.getElementById(nav.dataset.target), showParent, null,
                document.getElementById("system-nav"),
                () => { return window.innerWidth <= 760; });
        }
        document.getElementById("system-box-nav-back").addEventListener("click", showParent);

        const ul = document.getElementById("playlist-playback-range");
        const newul = document.createDocumentFragment();
        for (let i = 0, imax = TREE_ORDER.length; i < imax; i++) {
            const label = TREE_ORDER[i];
            if (this.preferences.playlist.playback_tracks_custom.hasOwnProperty(label)) {
                const c = document.querySelector("#playlist-playback-tracks-custom-template").content;
                const e = c.querySelector("li");
                const l = e.querySelector(".system-setting-desc");
                l.textContent = l.dataset.prefix + label + l.dataset.suffix;
                if (this.preferences.playlist.playback_tracks !== "custom") {
                    e.classList.add("hide");
                }
                const ts = e.querySelector(".tool-select");
                ts.setAttribute("aria-label", label);
                ts.dataset.label = label;
                while (ts.lastChild) {
                    ts.removeChild(ts.lastChild);
                }
                const all = document.createElement("option");
                all.textContent = ts.dataset.allTracks;
                all.value = "0";
                ts.appendChild(all);
                for (let j = 0, jmax = TREE[label].tree.length - 1; j < jmax; j++) {
                    const filter = document.createElement("option");
                    filter.textContent = TREE[label].tree[j][0];
                    filter.value = j + 1;
                    ts.appendChild(filter);
                }
                const d = document.importNode(c, true);
                const tso = d.querySelector(".tool-select");
                tso.value = (this.preferences.playlist.playback_tracks_custom[label]).toString(10);
                tso.addEventListener("change", e => {
                    const v = parseInt(e.currentTarget.value);
                    if (!isNaN(v)) {
                        this.preferences.playlist.playback_tracks_custom[e.currentTarget.dataset.label] = v;
                        this.preferences.save();
                    }
                });
                newul.appendChild(d);
            }
        }
        ul.appendChild(newul);

        const mpdVolume = document.getElementById("outputs-volume");
        mpdVolume.addEventListener("change", e => {
            this.mpd.volume(parseInt(e.currentTarget.value, 10));
        });
        mpdVolume.addEventListener("input", e => {
            document.getElementById("outputs-volume-string").textContent = e.currentTarget.value + "%";
        });
        const inputs = document.getElementById("httpoutput-stream");
        const newInputs = document.createDocumentFragment();
        let streamCnt = 0;
        for (const name in this.preferences.httpoutput.streams) {
            if (this.preferences.httpoutput.streams.hasOwnProperty(name)) {
                const on = document.createElement("option");
                on.value = this.preferences.httpoutput.streams[name];
                on.textContent = name;
                newInputs.appendChild(on);
                streamCnt++;
            }
        }
        if (streamCnt === 0) {
            document.getElementById("httpoutput").classList.add("hide");
        }
        inputs.appendChild(newInputs);
        inputs.value = this.preferences.httpoutput.stream;
        inputs.value = inputs.value;
        if (this.preferences.httpoutput.stream !== inputs.value) {
            this.preferences.httpoutput.stream = inputs.value;
            this.preferences.raiseEvent("httpoutput");
            this.preferences.save();
        }
        this.onPreferencesHTTPOutout();
        this._initconfig("httpoutput-stream");
        this._initconfig("httpoutput-volume-max");
        this._initconfig("httpoutput-volume");
        UI.disableSwipe(document.getElementById("httpoutput-volume"));
    }
    _zfill2(i) {
        if (i < 100) {
            return ("00" + i).slice(-2);
        }
        return i;
    }
    _strtimedelta(i) {
        const zfill2 = this._zfill2;
        const uh = parseInt(i / (60 * 60), 10);
        const um = parseInt((i - uh * 60 * 60) / 60, 10);
        const us = parseInt(i - uh * 60 * 60 - um * 60, 10);
        return `${zfill2(uh)}:${zfill2(um)}:${zfill2(us)}`;
    }
    onStats() {
        document.getElementById("stat-albums").textContent = this.mpd.stats.albums.toString(10);
        document.getElementById("stat-artists").textContent = this.mpd.stats.artists.toString(10);
        document.getElementById("stat-db-playtime").textContent = this._strtimedelta(this.mpd.stats.library_playtime, 10);
        document.getElementById("stat-tracks").textContent = this.mpd.stats.songs;
        const db_update = new Date(this.mpd.stats.library_update * 1000);
        const options = {
            hour: "numeric",
            minute: "numeric",
            second: "numeric",
            year: "numeric",
            month: "short",
            day: "numeric",
            weekday: "short",
        };
        document.getElementById("stat-db-update").textContent = db_update.toLocaleString(document.documentElement.lang, options);
    }
    onVersion() {
        if (this.mpd.version.app) {
            document.getElementById("version").textContent = this.mpd.version.app;
            document.getElementById("mpd-version").textContent = this.mpd.version.mpd;
            document.getElementById("go-version").textContent = this.mpd.version.go;
        }
    }
    show() {
        document.getElementById("modal-background").classList.remove("hide");
        document.getElementById("modal-outer").classList.remove("hide");
        document.getElementById("modal-system").classList.remove("hide");
    }
};

// header
class UIHeader {
    static init(ui, mpd, library, listView, mainView, systemWindow) {
        const update = () => {
            const e = document.getElementById("header-back-label");
            const b = document.getElementById("header-back");
            const m = document.getElementById("header-main");
            if (library.rootname() === "root") {
                b.classList.add("root");
                m.classList.add("root");
            } else {
                b.classList.remove("root");
                m.classList.remove("root");
                const songs = library.list().songs;
                if (songs[0]) {
                    const p = library.grandparent();
                    if (p) {
                        const v = Song.getOne(p.song, p.key);
                        e.textContent = v ? v : `[no ${p.key}]`;
                        b.setAttribute("title", b.dataset.titleFormat.replace("%s", e.textContent));
                        b.setAttribute("aria-label", b.dataset.ariaLabelFormat.replace("%s", e.textContent));
                    }
                }
            }
        };
        ui.addEventListener("load", () => {
            document.getElementById("header-back").addEventListener("click", e => {
                if (listView.hidden()) {
                    if (mpd.current !== null) {
                        library.abs(mpd.current);
                    }
                } else {
                    library.up();
                }
                listView.show();
                e.stopPropagation();
            });
            document.getElementById("header-main").addEventListener("click", e => {
                e.stopPropagation();
                if (mpd.current !== null) {
                    library.abs(mpd.current);
                }
                mainView.show();
                e.stopPropagation();
            });
            document.getElementById("header-system").addEventListener("click", e => {
                systemWindow.show();
                e.stopPropagation();
            });
            update();
            library.addEventListener("changed", update);
            library.addEventListener("update", update);
        });
    }
};

class UIFooter {
    static init(ui, mpd, preferences) {
        ui.addEventListener("load", () => { UIFooter.onStart(mpd, preferences); });
        mpd.addEventListener("control", () => { UIFooter.onControl(mpd); });
        mpd.addEventListener("playlist", () => { UIFooter.onPlaylist(mpd); });
        preferences.addEventListener("appearance", () => { UIFooter.onPreferences(preferences); });
    }
    static onPreferences(preferences) {
        const c = document.getElementById("control-volume");
        c.max = parseInt(preferences.outputs.volume_max, 10);
        if (preferences.appearance.volume) {
            c.classList.remove("hide");
        } else {
            c.classList.add("hide");
        }
    }
    static onStart(mpd, preferences) {
        UIFooter.onPlaylist(mpd);
        UIFooter.onPreferences(preferences);
        document.getElementById("control-prev").addEventListener("click", e => {
            mpd.prev();
            e.stopPropagation();
        });
        document.getElementById("control-toggleplay").addEventListener("click", e => {
            mpd.togglePlay();
            e.stopPropagation();
        });
        document.getElementById("control-next").addEventListener("click", e => {
            mpd.next();
            e.stopPropagation();
        });
        const c = document.getElementById("control-volume");
        if (mpd.control.hasOwnProperty("volume") && mpd.control.volume !== null) {
            c.value = mpd.control.volume;
            c.disabled = false;
        } else {
            c.disabled = true;
        }
        document.getElementById("control-repeat").addEventListener("click", e => {
            mpd.toggleRepeat();
            e.stopPropagation();
        });
        document.getElementById("control-random").addEventListener("click", e => {
            mpd.toggleRandom();
            e.stopPropagation();
        });
    }
    static onPlaylist(mpd) {
        const disabled = !(mpd.playlist && mpd.playlist.hasOwnProperty("current"));
        document.getElementById("control-prev").disabled = disabled;
        document.getElementById("control-toggleplay").disabled = disabled;
        document.getElementById("control-next").disabled = disabled;
    }
    static onControl(mpd) {
        const toggleplay = document.getElementById("control-toggleplay");
        if (mpd.control.state === "play") {
            toggleplay.setAttribute("aria-label", toggleplay.dataset.ariaLabelPause);
            toggleplay.classList.add("pause");
            toggleplay.classList.remove("play");
        } else {
            toggleplay.setAttribute("aria-label", toggleplay.dataset.ariaLabelPlay);
            toggleplay.classList.add("play");
            toggleplay.classList.remove("pause");
        }
        const repeat = document.getElementById("control-repeat");
        if (mpd.control.single) {
            repeat.setAttribute("aria-label", repeat.dataset.ariaLabelOn);
            repeat.classList.add("single-on");
            repeat.classList.remove("single-off");
        } else {
            repeat.classList.add("single-off");
            repeat.classList.remove("single-on");
        }
        if (mpd.control.repeat) {
            if (!mpd.control.single) {
                repeat.setAttribute("aria-label", repeat.dataset.ariaLabelSingleOff);
            }
            repeat.classList.add("on");
            repeat.classList.remove("off");
        } else {
            if (!mpd.control.single) {
                repeat.setAttribute("aria-label", repeat.dataset.ariaLabelOff);
            }
            repeat.classList.add("off");
            repeat.classList.remove("on");
        }
        const random = document.getElementById("control-random");
        if (mpd.control.random) {
            random.setAttribute("aria-label", random.dataset.ariaLabelOn);
            random.classList.add("on");
            random.classList.remove("off");
        } else {
            random.setAttribute("aria-label", random.dataset.ariaLabelOff);
            random.classList.add("off");
            random.classList.remove("on");
        }
    }
};

class UIMediaSession {
    static init(ui, mpd, preferences) {
        ui.addEventListener("load", () => { UIMediaSession.onStart(ui, mpd, preferences); });
    }
    static onStart(ui, mpd, preferences) {
        if ('mediaSession' in navigator) {
            UIMediaSession.update(mpd);
            mpd.addEventListener("current", () => { UIMediaSession.updateCurrent(mpd, preferences); });
            mpd.addEventListener("control", () => { UIMediaSession.updateControl(mpd); });
        }
    }
    static update(mpd) {
        if (!('mediaSession' in navigator)) {
            return;
        }
        UIMediaSession.updateCurrent(mpd);
        UIMediaSession.updateControl(mpd);
        navigator.mediaSession.setActionHandler("nexttrack", () => { mpd.next() });
        navigator.mediaSession.setActionHandler("previoustrack", () => { mpd.prev() });
        navigator.mediaSession.setActionHandler("play", () => { mpd.play() });
        navigator.mediaSession.setActionHandler("pause", () => { mpd.pause() });
        navigator.mediaSession.setActionHandler("stop", () => { mpd.stop() });
        navigator.mediaSession.setActionHandler("seekto", (e) => { mpd.seek(e.seekTime); });
        navigator.mediaSession.setActionHandler("seekbackward", (e) => { mpd.seek(Math.max(mpd.elapsed() - 10, 0)); });
        navigator.mediaSession.setActionHandler("seekforward", (e) => {
            const duration = Number(mpd.current.duration)
            if (isNaN(duration)) {
                return;
            }
            mpd.seek(Math.min(mpd.elapsed() + 10, duration));
        });
    }
    static updateCurrent(mpd) {
        if (!mpd.current) {
            return;
        }
        let covers = [];
        const song = mpd.current;
        if (song.cover && song.cover.length !== 0) {
            const s = (song.cover[0].lastIndexOf("?") == -1) ? "?" : "&";
            const sizes = [96, 128, 192, 256, 384, 512];
            for (let size of sizes) {
                covers.push({
                    src: `${song.cover[0]}${s}width=${size}&height=${size}`,
                    sizes: `${size}x${size}`,
                });
            }
        } else {
            const sizes = [96, 128, 192, 256, 384, 512];
            for (let size of sizes) {
                covers.push({
                    src: nocover(),
                    sizes: `${size}x${size}`,
                });
            }
        }
        if (!navigator.mediaSession.metadata) {
            navigator.mediaSession.metadata = new MediaMetadata({
                title: Song.get(song, "Title"),
                artist: Song.get(song, "Artist"),
                album: Song.get(song, "Album"),
                artwork: covers
            });
        } else {
            navigator.mediaSession.metadata.title = Song.get(song, "Title");
            navigator.mediaSession.metadata.artist = Song.get(song, "Artist");
            navigator.mediaSession.metadata.album = Song.get(song, "Album");
            if (navigator.mediaSession.metadata.artwork[0] !== covers[0]) {
                navigator.mediaSession.metadata.artwork = covers;
            }
        }
    }
    static updateControl(mpd) {
        if (!mpd.current) {
            return;
        }
        if (mpd.control.state === "play") {
            navigator.mediaSession.playbackState = "playing";
        } else {
            navigator.mediaSession.playbackState = "paused";
        }
        const duration = Number(mpd.current.duration)
        if (isNaN(duration)) {
            return;
        }
        navigator.mediaSession.setPositionState({
            duration: duration,
            position: mpd.elapsed()
        })
    }
};


class UINotification {
    static init(ui, mpd, audio, systemWindow) {
        ui.addEventListener("load", () => {
            mpd.addEventListener("version", () => {
                if (mpd.version.mpd) {
                    UINotification.hide("mpd-connection");
                } else {
                    UINotification.show("mpd-connection", "reconnecting", { ttl: Infinity });
                }
            });
            document.getElementById("popup-client-output-button-retry").addEventListener("click", () => {
                UINotification.hide("client-output");
                audio.play();
            });
            const openOutputSettings = () => {
                UINotification.hide("client-output");
                systemWindow.show();
                let e = document.createEvent("HTMLEvents");
                e.initEvent("click", false, true);
                document.getElementById("system-nav-outputs").dispatchEvent(e);
            };
            document.getElementById("popup-client-output-button-settings").addEventListener("click", openOutputSettings);
            mpd.addEventListener("library", (e) => {
                if (!e || !e.old || !e.current) {
                    return;
                }
                if (e.old.updating === e.current.updating) {
                    return;
                }
                if (e.current.updating) {
                    UINotification.show("library", "updating");
                } else if (e.old.hasOwnProperty("updating") && !e.current.updating) {
                    UINotification.show("library", "updated");
                }
            });
            mpd.addEventListener("images", (e) => {
                if (!e || !e.old || !e.current) {
                    return;
                }
                if (e.old.updating === e.current.updating) {
                    return;
                }
                if (e.current.updating) {
                    UINotification.show("coverart", "updating");
                } else if (e.old.hasOwnProperty("updating") && !e.current.updating) {
                    UINotification.show("coverart", "updated");
                }
            });
        });
    }
    static show(target, description, { ttl = 4000 } = {}) {
        const obj = document.getElementById("popup-" + target);
        if (!obj) {
            UINotification.show("fixme", `popup-${target} is not found in html`);
            return;
        }
        if (description) {
            const desc = obj.getElementsByClassName("popup-description")[0];
            const textContent = desc.dataset[description] || description;
            desc.textContent = textContent;
        }
        obj.classList.remove("hide");
        obj.classList.add("show");
        obj.timestamp = (new Date()).getTime();
        obj.ttl = obj.timestamp + ttl;
        if (ttl === Infinity) {
            return;
        }
        setTimeout(() => {
            if (obj.ttl !== 0 && ((new Date()).getTime() - obj.ttl) > 0) {
                obj.classList.remove("show");
                obj.classList.add("hide");
            }
        }, ttl + 100);
    }
    static hide(target) {
        const obj = document.getElementById("popup-" + target);
        if (obj) {
            const now = (new Date()).getTime();
            if (now - obj.timestamp < 600) {
                obj.ttl = obj.timestamp + 500;
                setTimeout(() => {
                    if ((new Date()).getTime() - obj.ttl > 0) {
                        obj.classList.remove("show");
                        obj.classList.add("hide");
                    }
                }, 600);
            } else {
                obj.ttl = now;
                obj.classList.remove("show");
                obj.classList.add("hide");
            }
        }
    }
};

class UITimeUpdater {
    static init(ui, mpd) {
        const update = () => {
            const data = mpd.control;
            if ("state" in data) {
                const current = mpd.elapsed();
                const min = parseInt(current / 60, 10);
                const sec = current % 60;
                const label = min + ":" + ("0" + sec).slice(-2);
                [].forEach.call(document.getElementsByClassName("elapsed"), x => {
                    if (x.textContent !== label) {
                        x.textContent = label;
                    }
                });
            }
        };
        mpd.addEventListener("control", update);
        ui.addEventListener("poll", update);
    }
};

class UIModal {
    static init(ui) {
        ui.addEventListener("load", () => {
            document.getElementById("modal-background").addEventListener("click", () => { UIModal.hide(); });
            document.getElementById("modal-outer").addEventListener("click", () => { UIModal.hide(); });
            for (const w of Array.from(document.getElementsByClassName("modal-window"))) {
                w.addEventListener("click", e => { e.stopPropagation(); });
            }
            for (const w of Array.from(document.getElementsByClassName("modal-window-close"))) {
                w.addEventListener("click", () => { UIModal.hide(); });
            }
        });
    }
    static hide() {
        document.getElementById("modal-background").classList.add("hide");
        document.getElementById("modal-outer").classList.add("hide");
        for (const w of Array.from(document.getElementsByClassName("modal-window"))) {
            w.classList.add("hide");
        }
    }
    static help() {
        const b = document.getElementById("modal-background");
        if (!b.classList.contains("hide")) {
            return;
        }
        b.classList.remove("hide");
        document.getElementById("modal-outer").classList.remove("hide");
        document.getElementById("modal-help").classList.remove("hide");
    }
    static song(song, library, preferences) {
        const mustkeys = ["Title", "Artist", "Album", "Date", "AlbumArtist", "Genre", "Performer", "Disc", "Track", "Composer", "Length"];
        for (const key of mustkeys) {
            const doc = document.getElementById("modal-song-box-" + key);
            while (doc.lastChild) {
                doc.removeChild(doc.lastChild);
            }
            const newdoc = document.createDocumentFragment();
            const values = Song.getOrElseMulti(song, key, []);
            if (values.length === 0) {
                const emptyvalue = document.createElement("span");
                emptyvalue.classList.add("modal-song-box-item-value");
                emptyvalue.classList.add("modal-song-box-item-value-empty");
                newdoc.appendChild(emptyvalue);
            } else {
                const root = TREE[key];
                let targetValues = [];
                if (root && root.tree) {
                    const target = root.tree[0][0];
                    if (target.split("-").indexOf(key) !== -1) {
                        targetValues = Song.getOrElseMulti(song, target, values);
                    }
                }
                for (const value of values) {
                    const obj = document.createElement("span");
                    obj.classList.add("modal-song-box-item-value");
                    obj.textContent = value;
                    if (targetValues.length) {
                        obj.dataset.root = key;
                        for (const targetValue of targetValues) {
                            if (targetValue.includes(value)) {
                                obj.dataset.value = targetValue;
                                obj.classList.add("modal-song-box-item-value-clickable");
                                obj.addEventListener("click", e => {
                                    const d = e.currentTarget.dataset;
                                    library.absaddr(d.root, d.value);
                                });
                                break;
                            }
                        }
                    } else {
                        obj.classList.add("modal-song-box-item-value-unclickable");
                    }
                    newdoc.appendChild(obj);
                }
            }
            doc.appendChild(newdoc);
        }
        const cover = document.getElementById("modal-song-box-cover");
        if (song.cover && song.cover[0]) {
            const imgsize = window.devicePixelRatio * 112;
            const s = (song.cover[0].lastIndexOf("?") == -1) ? "?" : "&";
            cover.src = `${song.cover[0]}${s}width=${imgsize}&height=${imgsize}`;
        } else {
            cover.src = preferences.nocover();
        }
        document.getElementById("modal-background").classList.remove("hide");
        document.getElementById("modal-outer").classList.remove("hide");
        document.getElementById("modal-song").classList.remove("hide");
    }
};

class UISubModal {
    static init(ui, mpd, mpdWatcher) {
        ui.addEventListener("load", () => {
            for (const w of Array.from(document.getElementsByClassName("submodal-window-close"))) {
                w.addEventListener("click", () => { UISubModal.hide(); });
            }
            document.getElementById("submodal-outer").addEventListener("click", () => { UISubModal.hide(); });
            for (const w of Array.from(document.getElementsByClassName("submodal-window"))) {
                w.addEventListener("click", (e) => { e.stopPropagation(); });
            }
            // storage
            document.getElementById("storage-ok").addEventListener("click", () => {
                HTTP.post("/api/music/storage", { [document.getElementById("storage-path").value]: { "uri": document.getElementById("storage-uri").value } }, (e) => {
                    if (e.error) {
                        document.getElementById("storage-error").textContent = e.error;
                        return;
                    }
                    UISubModal.hide();
                });
            });
            document.getElementById("storage-uri").addEventListener("input", (e) => {
                const v = e.currentTarget.value;
                for (const n of document.getElementById("list-neighbors").querySelectorAll(".select-item")) {
                    if (n.querySelector(".uri").textContent === v) {
                        n.classList.add("selected");
                        console.log("sel");
                    } else {
                        n.classList.remove("selected");
                        console.log("no");
                    }
                }
            });
            // neighbors
            mpd.addEventListener("neighbors", () => {
                const v = document.getElementById("storage-uri").value;
                const newul = document.createDocumentFragment();
                for (const path in mpd.neighbors) {
                    if (path !== "" && mpd.neighbors.hasOwnProperty(path)) {
                        const s = mpd.neighbors[path];
                        const c = document.querySelector("#neighbors-template").content;
                        const e = c.querySelector("li");
                        e.querySelector(".path").textContent = path;
                        e.querySelector(".uri").textContent = s.uri;
                        if (s.uri === v) {
                            e.classList.add("selected");
                        } else {
                            e.classList.remove("selected");
                        }
                        const d = document.importNode(c, true);
                        d.querySelector(".select-item").addEventListener("click", (e) => {
                            // use basename as path
                            // https://github.com/MusicPlayerDaemon/MPD/blob/a8c77a6fba49d3f06a1e24a134cfe8db1e3c951c/src/command/StorageCommands.cxx#L180-L188
                            const path = e.currentTarget.querySelector(".path").textContent.split('/').reverse()[0];
                            document.getElementById("storage-path").value = path;
                            document.getElementById("storage-uri").value = e.currentTarget.querySelector(".uri").textContent;
                            for (const n of document.getElementById("list-neighbors").querySelectorAll(".select-item")) {
                                if (e.currentTarget === n) {
                                    n.classList.add("selected");
                                } else {
                                    n.classList.remove("selected");
                                }
                            }
                        });
                        newul.appendChild(d);
                    }
                }
                const ul = document.getElementById("list-neighbors");
                while (ul.lastChild) {
                    ul.removeChild(ul.lastChild);
                }
                ul.appendChild(newul);
            });
            // allowed formats
            const allowedFormats = document.getElementById("submodal-allowed-formats");
            document.getElementById("allowed-formats-auto").addEventListener("change", () => {
                for (const n of allowedFormats.querySelectorAll(".slideswitch")) {
                    n.disabled = true;
                    n.classList.add("disabled");
                    n.checked = false;
                }
                HTTP.post("/api/music/outputs", {
                    [parseInt(document.getElementById("submodal-allowed-formats").dataset.deviceid, 10)]: { "attributes": { "allowed_formats": [] } }
                });
            });
            document.getElementById("allowed-formats-custom").addEventListener("change", () => {
                for (const n of allowedFormats.querySelectorAll(".slideswitch")) {
                    n.disabled = false;
                    n.classList.remove("disabled");
                }
            });
            for (const n of allowedFormats.querySelectorAll(".slideswitch")) {
                const fmt = n.name.split(":");
                if (fmt.length === 2) {
                    n.dataset.sample = fmt[0];
                    n.dataset.bits = "1";
                    n.dataset.dop = fmt[1].substring(1);
                } else if (fmt.length === 3) {
                    n.dataset.sample = fmt[0];
                    n.dataset.bits = fmt[1];
                }
                n.addEventListener("change", (e) => {
                    const t = e.currentTarget;
                    const dsd = allowedFormats.querySelector(".allowed-formats-dsd");
                    const pcm = allowedFormats.querySelector(".allowed-formats-pcm");
                    if (t.checked) {
                        if (t.dataset.bits === "1") {
                            for (const sw of dsd.querySelectorAll(".slideswitch")) {
                                if (t.dataset.sample === sw.dataset.sample && t.dataset.dop !== sw.dataset.dop) {
                                    sw.checked = false; // toggle dop
                                    break;
                                }
                            }
                        } else {
                            const sample = parseInt(n.dataset.sample, 10);
                            const bits = parseInt(t.dataset.bits, 10);
                            for (const sw of pcm.querySelectorAll(".slideswitch")) {
                                const swsample = parseInt(sw.dataset.sample, 10);
                                const swbits = parseInt(sw.dataset.bits, 10);
                                if (t.dataset.sample === sw.dataset.sample && t.dataset.bits == sw.dataset.bits) {
                                    break;
                                }
                                if (swsample <= sample && swbits <= bits) {
                                    sw.checked = true;
                                }
                            }
                        }
                    }
                    const allowedPCM = [];
                    const allowedDSD = [];
                    for (const sw of pcm.querySelectorAll(".slideswitch")) {
                        if (sw.checked) {
                            allowedPCM.push(sw.name);
                        }
                    }
                    for (const sw of dsd.querySelectorAll(".slideswitch")) {
                        if (sw.checked) {
                            allowedDSD.push(sw.name);
                        }
                    }
                    allowedPCM.sort().reverse();
                    allowedDSD.sort().reverse();
                    HTTP.post("/api/music/outputs", {
                        [parseInt(document.getElementById("submodal-allowed-formats").dataset.deviceid, 10)]: { "attributes": { "allowed_formats": allowedPCM.concat(allowedDSD) } }
                    });
                });
            }
        });
    }
    static showStorageMount() {
        document.getElementById("storage-path").value = "";
        document.getElementById("storage-uri").value = "";
        document.getElementById("storage-error").textContent = "";
        document.getElementById("submodal-background").classList.remove("hide");
        document.getElementById("submodal-outer").classList.remove("hide");
        document.getElementById("submodal-storage-mount").classList.remove("hide");
        for (const n of document.getElementById("list-neighbors").querySelectorAll(".select-item")) {
            n.classList.remove("selected");
        }
    }
    static showAllowedFormats(t, mpd) {
        if (!mpd.outputs[t] || !mpd.outputs[t].attributes || !mpd.outputs[t].attributes.allowed_formats) {
            return;
        }
        const lists = mpd.outputs[t].attributes.allowed_formats;
        const w = document.getElementById("submodal-allowed-formats");
        w.dataset.deviceid = t;
        let e = document.createEvent("HTMLEvents");
        e.initEvent("change", false, true);
        if (lists.length === 0) {
            const o = document.getElementById("allowed-formats-auto");
            o.checked = true;
            o.dispatchEvent(e);
        } else {
            const o = document.getElementById("allowed-formats-custom");
            o.checked = true;
            o.dispatchEvent(e);
        }
        for (const n of w.querySelectorAll(".slideswitch")) {
            n.checked = false;
            for (let i = 0, imax = lists.length; i < imax; i++) {
                if (lists[i] === n.name) {
                    n.checked = true;
                }
            }
        }
        document.getElementById("submodal-background").classList.remove("hide");
        document.getElementById("submodal-outer").classList.remove("hide");
        w.classList.remove("hide");
    }
    static hide() {
        document.getElementById("submodal-background").classList.add("hide");
        document.getElementById("submodal-outer").classList.add("hide");
        for (const w of Array.from(document.getElementsByClassName("submodal-window"))) {
            w.classList.add("hide");
        }
    }
};

class KeyboardShortCuts {
    static init(ui, mpd, library, listView, mainView) {
        const shift = 1 << 3;
        // const alt = 1 << 2;
        const ctrl = 1 << 1;
        const meta = 1;
        const none = 0;
        const any = t => {
            return () => {
                t();
                return true;
            };
        };
        const inList = (t) => {
            return () => {
                if (!listView.hidden()) {
                    t();
                    return true;
                }
                return false;
            };
        };
        const back = () => {
            if (listView.hidden()) {
                if (mpd.current !== null) {
                    library.abs(mpd.current);
                }
            } else {
                library.up();
            }
            listView.show();
        };
        const keymap = {
            [none]: {
                Enter: () => { return !listView.hidden() && listView.activate(); },
                Backspace: any(() => { back(); }),
                ArrowLeft: inList(() => { listView.left(); }),
                ArrowUp: inList(() => { listView.up(); }),
                ArrowRight: inList(() => { listView.right(); }),
                ArrowDown: inList(() => { listView.down(); }),
                [" "]: any(() => { mpd.togglePlay(); }),
                ["?"]: any(() => { UIModal.help(); }),
            },
            [shift]: { ["?"]: any(() => { UIModal.help(); }) },
            [meta]: {
                ArrowLeft: any(() => { back(); }),
                ArrowRight: any(() => {
                    if (library.rootname() !== "root") {
                        if (mpd.current !== null) {
                            library.abs(mpd.current);
                        }
                    }
                    mainView.show();
                }),
            },
            [shift | ctrl]:
                { ArrowLeft: any(() => { mpd.prev(); }), ArrowRight: any(() => { mpd.next(); }) }
        };
        ui.addEventListener("load", () => {
            document.addEventListener("keydown", e => {
                if (!document.getElementById("submodal-outer").classList.contains("hide")) {
                    if (e.key === "Escape" || e.key === "Esc") {
                        UISubModal.hide();
                    }
                    return;
                }
                if (!document.getElementById("modal-background").classList.contains("hide")) {
                    if (e.key === "Escape" || e.key === "Esc") {
                        UIModal.hide();
                    }
                    return;
                }
                const mod = e.shiftKey << 3 | e.altKey << 2 | e.ctrlKey << 1 | e.metaKey;
                if (mod in keymap && e.key in keymap[mod]) {
                    if (keymap[mod][e.key]()) {
                        e.stopPropagation();
                        e.preventDefault();
                    }
                }
            });
        });
    }
};

const app = new App();
app.start();
