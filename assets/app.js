"use strict";
const vv = {
    consts: { playlistLength: 9999 },
    pubsub: {},
    song: {},
    songs: {},
    storage: {},
    library: {},
    view: { main: {}, list: {}, system: {}, popup: {}, modal: {}, footer: {} },
    control: {},
    ui: {},
    request: {}
};
vv.pubsub = {
    add(listener, ev, func) {
        if (!(ev in listener)) {
            listener[ev] = [];
        }
        listener[ev].push(func);
    },
    rm(listener, ev, func) {
        for (let i = 0, imax = listener[ev].length; i < imax; i++) {
            if (listener[ev][i] === func) {
                listener[ev].splice(i, 1);
                return;
            }
        }
    },
    raise(listener, ev) {
        if (!(ev in listener)) {
            return;
        }
        for (const f of listener[ev]) {
            f();
        }
    }
};
vv.song = {
    tag(song, keys, other) {
        for (const key of keys) {
            if (key in song) {
                return song[key];
            }
        }
        return other;
    },
    getTagOrElseMulti(song, key, other) {
        if (key in song) {
            return song[key];
        } else if (key === "AlbumSort") {
            return vv.song.tag(song, ["Album"], other);
        } else if (key === "ArtistSort") {
            return vv.song.tag(song, ["Artist"], other);
        } else if (key === "AlbumArtist") {
            return vv.song.tag(song, ["Artist"], other);
        } else if (key === "AlbumArtistSort") {
            return vv.song.tag(song, ["AlbumArtist", "Artist"], other);
        } else if (key === "AlbumSort") {
            return vv.song.tag(song, ["Album"], other);
        }
        return other;
    },
    getOrElseMulti(song, keys, other) {
        let ret = [];
        for (const key of keys.split("-")) {
            const t = vv.song.getTagOrElseMulti(song, key, other);
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
    },
    getOrElse(song, key, other) {
        const ret = vv.song.getOrElseMulti(song, key, null);
        if (!ret) {
            return other;
        }
        return ret.join();
    },
    getOne(song, key) {
        const other = `[no ${key}]`;
        if (!song.keys) {
            return vv.song.getOrElseMulti(song, key, [other])[0];
        }
        for (const kv of song.keys) {
            if (kv[0] === key) {
                return kv[1];
            }
        }
        return vv.song.getOrElseMulti(song, key, [other])[0];
    },
    get(song, key) { return vv.song.getOrElse(song, key, `[no ${key}]`); },
    sortkeys(song, keys, memo) {
        let songs = [Object.assign({}, song)];
        songs[0].sortkey = "";
        songs[0].keys = [];
        for (const key of keys) {
            const writememo = memo.indexOf(key) !== -1;
            const values = vv.song.getOrElseMulti(song, key, []);
            if (values.length === 0) {
                for (const song of songs) {
                    song.sortkey += " ";
                    if (writememo) {
                        song.keys.push([key, `[no ${key}]`]);
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
vv.songs = {
    sort(songs, keys, memo) {
        const newsongs = [];
        for (const song of songs) {
            Array.prototype.push.apply(newsongs, vv.song.sortkeys(song, keys, memo));
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
    },
    uniq(songs, key) {
        return songs.filter((song, i, self) => {
            if (i === 0) {
                return true;
            } else if (
                vv.song.getOne(song, key) === vv.song.getOne(self[i - 1], key)) {
                return false;
            }
            return true;
        });
    },
    filter(songs, filters) {
        return songs.filter(song => {
            for (const key in filters) {
                if (filters.hasOwnProperty(key)) {
                    if (vv.song.getOne(song, key) !== filters[key]) {
                        return false;
                    }
                }
            }
            return true;
        });
    },
    weakFilter(songs, filters, max) {
        if (songs.length <= max) {
            return songs;
        }
        for (const filter of filters) {
            const newsongs = [];
            for (const song of songs) {
                if (vv.song.getOne(song, filter[0]) === filter[1]) {
                    newsongs.push(song);
                }
            }
            if (newsongs.length <= max) {
                return newsongs;
            }
            songs = newsongs;
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
vv.storage = {
    _listener: {},
    loaded: false,
    root: "root",
    tree: [],
    current: null,
    control: {},
    library: [],
    outputs: [],
    stats: {},
    last_modified: {},
    etag: {},
    last_modified_ms: {},
    version: {},
    preferences: {
        feature: {
            show_scrollbars_when_scrolling: false,
        },
        playlist: {
            playback_tracks: "all",
        },
        appearance: {
            theme: "prefer-coverart",
            color_threshold: 128,
            background_image: true,
            background_image_blur: 32,
            circled_image: false,
            volume: true,
            volume_max: "100",
            playlist_follows_playback: true,
            playlist_gridview_album: true,
        }
    },
    _idbUpdateTables(e) {
        const db = e.target.result;
        const st = db.createObjectStore("cache", { keyPath: "id" });
        const close = () => { db.close(); };
        st.onsuccess = close;
        st.onerror = close;
    },
    _cacheLoad(key, callback) {
        if (!window.indexedDB) {
            const ls = localStorage[key + "_last_modified"];
            const data = localStorage[key];
            if (ls && data) {
                callback(JSON.parse(data), ls);
                return;
            }
            callback();
            return;
        }
        const req = window.indexedDB.open("storage", 1);
        req.onerror = () => { };
        req.onupgradeneeded = vv.storage._idbUpdateTables;
        req.onsuccess = e => {
            const db = e.target.result;
            const t = db.transaction("cache", "readonly");
            const so = t.objectStore("cache");
            const req = so.get(key);
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
    },
    _cacheSave(key, value, date) {
        if (!window.indexedDB) {
            const ls = localStorage[key + "_last_modified"];
            if (ls && ls === date) {
                return;
            }
            localStorage[key] = JSON.stringify(value);
            localStorage[key + "_last_modified"] = date;
            return;
        }
        const req = window.indexedDB.open("storage", 1);
        req.onerror = () => { };
        req.onupgradeneeded = vv.storage._idbUpdateTables;
        req.onsuccess = e => {
            const db = e.target.result;
            const t = db.transaction("cache", "readwrite");
            const so = t.objectStore("cache");
            const req = so.get(key);
            req.onerror = () => { db.close(); };
            req.onsuccess = e => {
                const ret = e.target.result;
                if (ret && ret.date && ret.date === date) {
                    return;
                }
                const req = so.put({ id: key, value: value, date: date });
                req.onerror = () => { db.close(); };
                req.onsuccess = () => { db.close(); };
            };
        };
    },
    addEventListener(e, f) { vv.pubsub.add(vv.storage._listener, e, f); },
    save: {
        current() {
            try {
                localStorage.current = JSON.stringify(vv.storage.current);
                localStorage.current_last_modified = vv.storage.last_modified.current;
            } catch (e) {
            }
        },
        root() {
            try {
                localStorage.root = vv.storage.root;
            } catch (e) {
            }
        },
        preferences() {
            try {
                localStorage.preferences = JSON.stringify(vv.storage.preferences);
            } catch (e) {
            }
        },
        sorted() {
            try {
                localStorage.sorted = JSON.stringify(vv.storage.sorted);
                localStorage.sorted_last_modified = vv.storage.last_modified.sorted;
            } catch (e) {
            }
        },
        library() {
            try {
                vv.storage._cacheSave(
                    "library", vv.storage.library, vv.storage.last_modified.library);
            } catch (e) {
            }
        }
    },
    load() {
        try {
            if (localStorage.version !== "v2") {
                localStorage.clear();
            }
            localStorage.version = "v2";
            if (localStorage.root && localStorage.root.length !== 0) {
                vv.storage.root = localStorage.root;
                if (vv.storage.root !== "root") {
                    vv.storage.tree.push(["root", vv.storage.root]);
                }
            }
            if (localStorage.preferences) {
                const c = JSON.parse(localStorage.preferences);
                for (const i in c) {
                    if (c.hasOwnProperty(i)) {
                        for (const j in c[i]) {
                            if (c[i].hasOwnProperty(j)) {
                                if (vv.storage.preferences[i]) {
                                    vv.storage.preferences[i][j] = c[i][j];
                                }
                            }
                        }
                    }
                }
            }
            if (localStorage.current && localStorage.current_last_modified) {
                const current = JSON.parse(localStorage.current);
                if (Object.prototype.toString.call(current.file) === "[object Array]") {
                    vv.storage.current = current;
                    vv.storage.last_modified.current = localStorage.current_last_modified;
                }
            }
            if (localStorage.sorted && localStorage.sorted_last_modified) {
                const sorted = JSON.parse(localStorage.sorted);
                vv.storage.sorted = sorted;
                vv.storage.last_modified.sorted = localStorage.sorted_last_modified;
            }
            vv.storage._cacheLoad("library", (data, date) => {
                if (data && date) {
                    vv.storage.library = data;
                    vv.storage.last_modified.library = date;
                }
                vv.storage.loaded = true;
                vv.pubsub.raise(vv.storage._listener, "onload");
            });
        } catch (e) {
            vv.storage.loaded = true;
            vv.pubsub.raise(vv.storage._listener, "onload");
            // private browsing
        }
        if (navigator.userAgent.indexOf("Mobile") > 1) {
            vv.storage.preferences.feature.show_scrollbars_when_scrolling = true;
        } else if (navigator.userAgent.indexOf("Macintosh") > 1) {
            vv.storage.preferences.feature.show_scrollbars_when_scrolling = true;
        } else {
            document.body.classList.add("scrollbar-styling");
        }
    }
};
vv.storage.load();

vv.library = {
    focus: {},
    child: null,
    _roots: {
        AlbumArtist: [],
        Album: [],
        Artist: [],
        Genre: [],
        Date: [],
        Composer: [],
        Performer: []
    },
    _listener: {},
    addEventListener(e, f) { vv.pubsub.add(vv.library._listener, e, f); },
    removeEventListener(e, f) { vv.pubsub.rm(vv.library._listener, e, f); },
    _mkmemo(key) {
        const ret = [];
        for (const leef of TREE[key].tree) {
            ret.push(leef[0]);
        }
        return ret;
    },
    _list_child_cache: [{}, {}, {}, {}, {}, {}],
    list_child() {
        const root = vv.library.rootname();
        if (vv.library._roots[root].length === 0) {
            vv.library._roots[root] = vv.songs.sort(
                vv.storage.library, TREE[root].sort,
                vv.library._mkmemo(root));
        }
        const filters = {};
        for (let i = 0, imax = vv.storage.tree.length; i < imax; i++) {
            if (i === 0) {
                continue;
            }
            filters[vv.storage.tree[i][0]] = vv.storage.tree[i][1];
        }
        const ret = {};
        ret.key = TREE[root].tree[vv.storage.tree.length - 1][0];
        ret.songs = vv.library._roots[root];
        ret.songs = vv.songs.filter(ret.songs, filters);
        ret.songs = vv.songs.uniq(ret.songs, ret.key);
        ret.style = TREE[root].tree[vv.storage.tree.length - 1][1];
        ret.isdir = vv.storage.tree.length !== TREE[root].tree.length;
        return ret;
    },
    list_root() {
        const ret = [];
        for (let i = 0, imax = TREE_ORDER.length; i < imax; i++) {
            ret.push({ root: [TREE_ORDER[i]] });
        }
        return { key: "root", songs: ret, style: "plain", isdir: true };
    },
    _list_cache: {},
    update_list() {
        if (vv.library.rootname() === "root") {
            vv.library._list_cache = vv.library.list_root();
            return true;
        }
        const cache = vv.library._list_child_cache[vv.storage.tree.length - 1];
        const pwd = vv.storage.tree.join();
        if (cache.pwd === pwd) {
            vv.library._list_cache = cache.data;
            return false;
        }
        vv.library._list_cache = vv.library.list_child();
        if (vv.library._list_cache.songs.length === 0) {
            vv.library.up();
        } else {
            vv.library._list_child_cache[vv.storage.tree.length - 1].pwd = pwd;
            vv.library._list_child_cache[vv.storage.tree.length - 1].data =
                vv.library._list_cache;
        }
        return true;
    },
    list() {
        if (!vv.library._list_cache.songs ||
            !vv.library._list_cache.songs.length === 0) {
            vv.library.update_list();
        }
        return vv.library._list_cache;
    },
    updateData(data) {
        for (let i = 0, imax = vv.library._list_child_cache.length; i < imax; i++) {
            vv.library._list_child_cache[i] = {};
        }
        for (const key in TREE) {
            if (TREE.hasOwnProperty(key)) {
                if (key === vv.storage.root) {
                    vv.library._roots[key] = vv.songs.sort(
                        data, TREE[key].sort, vv.library._mkmemo(key));
                } else {
                    vv.library._roots[key] = [];
                }
            }
        }
    },
    update(data) {
        vv.library.updateData(data);
        vv.library.update_list();
        vv.pubsub.raise(vv.library._listener, "update");
    },
    rootname() {
        let r = "root";
        if (vv.storage.tree.length !== 0) {
            r = vv.storage.tree[0][1];
        }
        if (r !== vv.storage.root) {
            vv.storage.root = r;
            vv.storage.save.root();
        }
        return r;
    },
    filters(pos) { return vv.library._roots[vv.library.rootname()][pos].keys; },
    sortkeys() {
        const r = vv.library.rootname();
        if (r === "root") {
            return [];
        }
        return TREE[r].sort;
    },
    up() {
        const songs = vv.library.list().songs;
        if (songs[0]) {
            vv.library.focus = songs[0];
            if (vv.library.rootname() === "root") {
                vv.library.child = null;
            } else {
                vv.library.child = vv.storage.tree[vv.storage.tree.length - 1][1];
            }
        }
        if (vv.library.rootname() !== "root") {
            vv.storage.tree.pop();
        }
        vv.library.update_list();
        if (vv.library.list().songs.length === 1 && vv.storage.tree.length !== 0) {
            vv.library.up();
        } else {
            vv.pubsub.raise(vv.library._listener, "changed");
        }
    },
    down(value) {
        let r = vv.library.rootname();
        if (r === "root") {
            vv.storage.tree.push([r, value]);
            r = value;
        } else {
            const key = TREE[r].tree[vv.storage.tree.length - 1][0];
            vv.storage.tree.push([key, value]);
        }
        vv.library.focus = {};
        vv.library.child = null;
        vv.library.update_list();
        const songs = vv.library.list().songs;
        if (songs.length === 1 &&
            TREE[r].tree.length !== vv.storage.tree.length) {
            vv.library.down(vv.song.get(songs[0], vv.library.list().key));
        } else {
            vv.pubsub.raise(vv.library._listener, "changed");
        }
    },
    absaddr(first, second) {
        vv.storage.tree.splice(0, vv.storage.tree.length);
        vv.storage.tree.push(["root", first]);
        vv.library.down(second);
    },
    absFallback(song) {
        if (vv.library.rootname() !== "root" && song.file) {
            const r = vv.storage.tree[0];
            vv.storage.tree.length = 0;
            vv.storage.tree.splice(0, vv.storage.tree.length);
            vv.storage.tree.push(r);
            const root = vv.storage.tree[0][1];
            const selected = TREE[root].tree;
            for (let i = 0, imax = selected.length; i < imax; i++) {
                if (i === selected.length - 1) {
                    break;
                }
                const key = selected[i][0];
                vv.storage.tree.push([key, vv.song.getOne(song, key)]);
            }
            vv.library.update_list();
            for (const candidate of vv.library.list().songs) {
                if (candidate.file && candidate.file[0] === song.file[0]) {
                    vv.library.focus = candidate;
                    vv.library.child = null;
                    break;
                }
            }
        } else {
            vv.storage.tree.splice(0, vv.storage.tree.length);
            vv.library.update_list();
        }
        vv.pubsub.raise(vv.library._listener, "changed");
    },
    absSorted(song) {
        let root = "";
        const pos = parseInt(song.Pos[0], 10);
        const keys = vv.storage.sorted.sort.join();
        for (const key in TREE) {
            if (TREE.hasOwnProperty(key)) {
                if (TREE[key].sort.join() === keys) {
                    root = key;
                    break;
                }
            }
        }
        if (!root) {
            vv.view.popup.show("fixme", `modal: unknown sort keys: ${keys}`);
            return;
        }
        let songs = vv.library._roots[root];
        if (!songs || songs.length === 0) {
            vv.library._roots[root] = vv.songs.sort(
                vv.storage.library, TREE[root].sort,
                vv.library._mkmemo(root));
            songs = vv.library._roots[root];
            if (songs.length === 0) {
                return;
            }
        }
        if (songs.length > vv.consts.playlistLength) {
            songs = vv.songs.weakFilter(
                songs, vv.storage.sorted.filters || [], vv.consts.playlistLength);
        }
        if (!songs[pos]) {
            return;
        }
        if (songs[pos].file[0] === song.file[0]) {
            vv.library.focus = songs[pos];
            vv.library.child = null;
            vv.storage.tree.length = 0;
            vv.storage.tree.push(["root", root]);
            for (let i = 0; i < vv.library.focus.keys.length - 1; i++) {
                vv.storage.tree.push(vv.library.focus.keys[i]);
            }
            vv.library.update_list();
            vv.pubsub.raise(vv.library._listener, "changed");
        } else {
            vv.library.absFallback(song);
        }
    },
    abs(song) {
        if (vv.storage.sorted && vv.storage.sorted.hasOwnProperty("sort") && vv.storage.sorted.sort !== null) {
            vv.library.absSorted(song);
        } else {
            vv.library.absFallback(song);
        }
    },
    parent() {
        const root = vv.library.rootname();
        if (root === "root") {
            return;
        }
        const v = vv.library.list().songs;
        if (vv.storage.tree.length > 1) {
            const key = TREE[root].tree[vv.storage.tree.length - 2][0];
            const style = TREE[root].tree[vv.storage.tree.length - 2][1];
            return { key: key, song: v[0], style: style, isdir: true };
        }
        return { key: "top", song: { top: [root] }, style: "plain", isdir: true };
    },
    grandparent() {
        const root = vv.library.rootname();
        if (root === "root") {
            return;
        }
        const v = vv.library.list().songs;
        if (vv.storage.tree.length > 2) {
            const key = TREE[root].tree[vv.storage.tree.length - 3][0];
            const style = TREE[root].tree[vv.storage.tree.length - 3][1];
            return { key: key, song: v[0], style: style, isdir: true };
        } else if (vv.storage.tree.length === 2) {
            return { key: "top", song: { top: [root] }, style: "plain", isdir: true };
        }
        return {
            key: "root",
            song: { root: ["Library"] },
            style: "plain",
            isdir: true
        };
    },
    load() {
        if (vv.storage.loaded) {
            vv.library.updateData(vv.storage.library);
        } else {
            vv.storage.addEventListener(
                "onload", () => { vv.library.updateData(vv.storage.library); });
        }
    }
};
vv.library.load();
vv.request = {
    _requests: {},
    abortAll(options) {
        options = options || {};
        for (const key in vv.request._requests) {
            if (vv.request._requests.hasOwnProperty(key)) {
                if (options.stop) {
                    vv.request._requests[key].onabort = () => { };
                }
                vv.request._requests[key].abort();
            }
        }
    },
    get(path, ifmodified, etag, callback, timeout) {
        const key = "GET " + path;
        if (vv.request._requests[key]) {
            vv.request._requests[key].onabort = () => { };  // disable retry
            vv.request._requests[key].abort();
        }
        const xhr = new XMLHttpRequest();
        vv.request._requests[key] = xhr;
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
                vv.view.popup.show("network", xhr.statusText);
            }
        };
        xhr.onabort = () => {
            if (timeout < 50000) {
                setTimeout(
                    () => { vv.request.get(path, ifmodified, etag, callback, timeout * 2); });
            }
        };
        xhr.onerror = () => { vv.view.popup.show("network", "Error"); };
        xhr.ontimeout = () => {
            if (timeout < 50000) {
                vv.view.popup.show("network", "timeoutRetry");
                vv.request.abortAll();
                setTimeout(
                    () => { vv.request.get(path, ifmodified, etag, callback, timeout * 2); });
            } else {
                vv.view.popup.show("network", "timeout");
            }
        };
        xhr.open("GET", path, true);
        if (etag !== "") {
            xhr.setRequestHeader("If-None-Match", etag);
        } else {
            xhr.setRequestHeader("If-Modified-Since", ifmodified);
        }
        xhr.send();
    },
    post(path, obj) {
        const key = "POST " + path;
        if (vv.request._requests[key]) {
            vv.request._requests[key].abort();
        }
        const xhr = new XMLHttpRequest();
        vv.request._requests[key] = xhr;
        xhr.responseType = "json";
        xhr.timeout = 1000;
        xhr.onload = () => {
            if (xhr.status !== 200 && xhr.status !== 202) {
                if (xhr.response && xhr.response.error) {
                    vv.view.popup.show("network", xhr.response.error);
                } else {
                    vv.view.popup.show("network", xhr.responseText);
                }
            }
        };
        xhr.ontimeout = () => {
            vv.view.popup.show("network", "timeout");
            vv.request.abortAll();
        };
        xhr.onerror = () => { vv.view.popup.show("network", "Error"); };
        xhr.open("POST", path, true);
        xhr.setRequestHeader("Content-Type", "application/json");
        xhr.send(JSON.stringify(obj));
    }
};
vv.control = {
    _getOrElse(m, k, v) { return k in m ? m[k] : v; },
    _listener: {},
    addEventListener(e, f) { vv.pubsub.add(vv.control._listener, e, f); },
    removeEventListener(e, f) { vv.pubsub.rm(vv.control._listener, e, f); },
    raiseEvent(e) { vv.pubsub.raise(vv.control._listener, e); },

    rescan_library() {
        vv.request.post("/api/music/library", { updating: true });
        vv.storage.control.update_library = true;
        vv.control.raiseEvent("control");
    },
    prev() { vv.request.post("/api/music", { state: "previous" }); },
    play_pause() {
        const state = vv.control._getOrElse(vv.storage.control, "state", "stopped");
        const action = state === "play" ? "pause" : "play";
        vv.request.post("/api/music", { state: action });
        vv.storage.control.state = action;
        vv.control.raiseEvent("control");
    },
    next() { vv.request.post("/api/music", { state: "next" }); },
    toggle_repeat() {
        if (vv.storage.control.single) {
            vv.request.post("/api/music", { repeat: false, single: false });
            vv.storage.control.single = false;
            vv.storage.control.repeat = false;
        } else if (vv.storage.control.repeat) {
            vv.request.post("/api/music", { single: true });
            vv.storage.control.single = true;
        } else {
            vv.request.post("/api/music", { repeat: true });
            vv.storage.control.repeat = true;
        }
        vv.control.raiseEvent("control");
    },
    toggle_random() {
        vv.request.post("/api/music", { random: !vv.storage.control.random });
        vv.storage.control.random = !vv.storage.control.random;
        vv.control.raiseEvent("control");
    },
    play(pos) {
        const filters = vv.library.filters(pos);
        const must = (vv.storage.preferences.playlist.playback_tracks === "all") ? 0 : filters.length - 1;
        vv.request.post("/api/music/playlist", {
            sort: vv.library.sortkeys(),
            filters: filters,
            must: must,
            current: pos
        });
    },
    volume(num) { vv.request.post("/api/music", { volume: num }); },
    output(id, on) {
        vv.request.post(`/api/music/outputs`, { [id]: { enabled: on } });
    },
    seek(pos) { vv.request.post("/api/music", { song_elapsed: pos }); },
    _fetch(target, store) {
        vv.request.get(
            target,
            vv.control._getOrElse(vv.storage.last_modified, store, ""),
            vv.control._getOrElse(vv.storage.etag, store, ""),
            (ret, modified, etag, date) => {
                if (!ret.error) {
                    if (Object.prototype.toString.call(ret.data) ===
                        "[object Object]" &&
                        Object.keys(ret.data).length === 0) {
                        return;
                    }
                    let diff = 0;
                    try {
                        diff = Date.now() - Date.parse(date);
                    } catch (e) {
                        // use default value;
                    }
                    vv.storage[store] = ret;
                    vv.storage.last_modified_ms[store] = Date.parse(modified) + diff;
                    vv.storage.last_modified[store] = modified;
                    vv.storage.etag[store] = etag;
                    if (store === "library") {
                        vv.storage.save.library();
                    } else if (store === "sorted") {
                        vv.storage.save.sorted();
                    }
                    vv.control.raiseEvent(store);
                }
            });
    },
    _fetchAll() {
        vv.control._fetch("/api/music/playlist", "sorted");
        vv.control._fetch("/api/version", "version");
        vv.control._fetch("/api/music/outputs", "outputs");
        vv.control._fetch("/api/music/playlist/songs/current", "current");
        vv.control._fetch("/api/music", "control");
        vv.control._fetch("/api/music/library/songs", "library");
        vv.control._fetch("/api/music/stats", "stats");
    },
    _notify_last_update: (new Date()).getTime(),
    _notify_last_connection: (new Date()).getTime(),
    _connected: false,
    _notify_try_num: 0,
    _ws: null,
    _listennotify(cause) {
        if (cause && vv.control._notify_try_num > 1) {  // reduce device wakeup reconnecting message
            vv.view.popup.show("network", cause);
        }
        vv.control._notify_try_num++;
        vv.request.abortAll({ stop: true });
        vv.control._notify_last_connection = (new Date()).getTime();
        vv.control._connected = false;
        const wsp = document.location.protocol === "https:" ? "wss:" : "ws:";
        const uri = `${wsp}//${location.host}/api/music`;
        if (vv.control._ws !== null) {
            vv.control._ws.onclose = () => { };
            vv.control._ws.close();
        }
        vv.control._ws = new WebSocket(uri);
        vv.control._ws.onopen = () => {
            if (vv.control._notify_try_num > 1) {
                vv.view.popup.hide("network");
            }
            vv.control._connected = true;
            vv.control._notify_last_update = (new Date()).getTime();
            vv.control._notify_try_num = 0;
            vv.control._fetchAll();
        };
        vv.control._ws.onmessage = e => {
            if (e && e.data) {
                if (e.data === "/api/music/library/songs") {
                    vv.control._fetch("/api/music/library/songs", "library");
                } else if (e.data === "/api/music") {
                    vv.control._fetch("/api/music", "control");
                } else if (e.data === "/api/music/playlist/songs/current") {
                    vv.control._fetch("/api/music/playlist/songs/current", "current");
                } else if (e.data === "/api/music/outputs") {
                    vv.control._fetch("/api/music/outputs", "outputs");
                } else if (e.data === "/api/music/stats") {
                    vv.control._fetch("/api/music/stats", "stats");
                } else if (e.data === "/api/music/playlist") {
                    vv.control._fetch("/api/music/playlist", "sorted");
                }
                const now = (new Date()).getTime();
                if (now - vv.control._notify_last_update > 10000) {
                    // recover lost notification
                    setTimeout(vv.control._listennotify);
                }
                vv.control._notify_last_update = now;
            }
        };
        vv.control._ws.onclose = () => {
            setTimeout(() => { vv.control._listennotify("timeoutRetry"); }, 1000);
        };
    },
    _init() {
        const polling = () => {
            const now = (new Date()).getTime();
            if (vv.control._connected &&
                now - 10000 > vv.control._notify_last_update) {
                setTimeout(() => { vv.control._listennotify("doesNotRespond"); });
            } else if (
                !vv.control._connected &&
                now - 2000 > vv.control._notify_last_connection) {
                setTimeout(() => { vv.control._listennotify("timeoutRetry"); });
            }
            vv.control.raiseEvent("poll");
            setTimeout(polling, 1000);
        };
        const start = () => {
            vv.control.raiseEvent("start");
            vv.view.list.show();
            vv.control.raiseEvent("current");
            vv.control._listennotify();
            polling();
        };
        if (vv.storage.loaded) {
            start();
        } else {
            vv.storage.addEventListener("onload", start);
        }
    },
    start() {
        if (document.readyState === "loading") {
            document.addEventListener("DOMContentLoaded", vv.control._init);
        } else {
            vv.control._init();
        }
    },
    load() {
        const focus = () => {
            vv.storage.save.current();
            if (vv.storage.preferences.appearance.playlist_follows_playback &&
                vv.storage.current !== null) {
                vv.library.abs(vv.storage.current);
            }
        };

        let unsorted = (!vv.storage.sorted || !vv.storage.sorted.hasOwnProperty("sort") || vv.storage.sorted.sort === null);
        const focusremove = (key, remove) => {
            const n = () => {
                if (unsorted && vv.storage.sorted && vv.storage.current !== null) {
                    if (vv.storage.sorted &&
                        vv.storage.preferences.appearance.playlist_follows_playback.view_follow) {
                        vv.library.abs(vv.storage.current);
                    }
                    unsorted = false;
                }
                setTimeout(() => { remove(key, n); });
            };
            return n;
        };
        vv.control.addEventListener("current", focus);
        vv.control.addEventListener(
            "library", () => { vv.library.update(vv.storage.library); });
        if (unsorted) {
            vv.control.addEventListener(
                "current", focusremove("current", vv.control.removeEventListener));
            vv.control.addEventListener(
                "sorted", focusremove("sorted", vv.control.removeEventListener));
            vv.library.addEventListener(
                "update", focusremove("update", vv.library.removeEventListener));
        }
    }
};
vv.control.load();

vv.ui = {
    swipe(element, f, resetFunc, leftElement, conditionFunc) {
        element.swipe_target = f;
        let starttime = 0;
        let now = 0;
        let x = 0;
        let y = 0;
        let diff_x = 0;
        let diff_y = 0;
        let swipe = false;
        const start = e => {
            if ((e.buttons && e.buttons !== 1) ||
                (conditionFunc && !conditionFunc())) {
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
            if (e.buttons === 0 || (e.buttons && e.buttons !== 1) || !swipe ||
                (conditionFunc && !conditionFunc())) {
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
                    leftElement.style.transform =
                        `translate3d(${diff_x * -1 - e.currentTarget.offsetWidth}px,0,0)`;
                }
            }
        };
        const end = e => {
            if ((e.buttons && e.buttons !== 1) || !swipe ||
                (conditionFunc && !conditionFunc())) {
                cancel(e);
                return;
            }
            const p = e.currentTarget.clientWidth / diff_x;
            if ((p > -4 && p < 0) ||
                (now - starttime < 200 && Math.abs(diff_y) < Math.abs(diff_x) &&
                    diff_x < 0)) {
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
    },
    disableSwipe(element) {
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
    },
    click(element, f) {
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
            if (Math.abs(e.currentTarget.x - t.screenX) >= 5 ||
                Math.abs(e.currentTarget.y - t.screenY) >= 5) {
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
}

// background
{
    let rgbg = { r: 128, g: 128, b: 128, gray: 128 };
    const mkcolor = (rgb, magic) => {
        return "#" +
            (((1 << 24) + (magic(rgb.r) << 16) + (magic(rgb.g) << 8) + magic(rgb.b))
                .toString(16)
                .slice(1));
    };
    const darker = c => {
        // Vivaldi does not recognize #000000
        if ((c - 20) < 0) {
            return 1;
        }
        return c - 20;
    };
    const lighter = c => {
        if ((c + 100) > 255) {
            return 255;
        }
        return c + 100;
    };
    const update_theme = () => {
        const color = document.querySelector("meta[name=theme-color]");
        if (vv.storage.preferences.appearance.theme === "prefer-system") {
            document.body.classList.add("system-theme-color");
            document.body.classList.remove("dark");
            document.body.classList.remove("light");
            if (window.matchMedia("(prefers-color-scheme: dark)").matches) {
                color.setAttribute("content", mkcolor(rgbg, darker));
            } else {
                color.setAttribute("content", mkcolor(rgbg, lighter));
            }
        } else {
            document.body.classList.remove("system-theme-color");
            var dark = true;
            if (vv.storage.preferences.appearance.theme === "light") {
                dark = false;
            } else if (vv.storage.preferences.appearance.theme !== "dark" && rgbg.gray >= vv.storage.preferences.appearance.color_threshold) {
                dark = false;
            }
            if (dark) {
                document.body.classList.add("dark");
                document.body.classList.remove("light");
                color.setAttribute("content", mkcolor(rgbg, darker));
            } else {
                document.body.classList.add("light");
                document.body.classList.remove("dark");
                color.setAttribute("content", mkcolor(rgbg, lighter));
            }
        }
    };
    const calc_color = path => {
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
                rgbg = new_rgbg;
                update_theme();
            } catch (e) {
                // failed to getImageData
            }
        };
        img.src = path;
    };
    const update = () => {
        const e = document.getElementById("background-image");
        if (vv.storage.preferences.appearance.background_image) {
            e.classList.remove("hide");
            document.getElementById("background-image").classList.remove("hide");
            let cover = "/assets/nocover.svg";
            let coverForCalc = "/assets/nocover.svg";
            if (vv.storage.current !== null && vv.storage.current.cover) {
                cover = vv.storage.current.cover[0];
                const imgsize = parseInt(70 * window.devicePixelRatio, 10);
                coverForCalc =
                    `${cover}?width=${imgsize}&height=${imgsize}`;
            }
            const newimage = `url("${cover}")`;
            if (e.style.backgroundImage !== newimage) {
                calc_color(coverForCalc);
                e.style.backgroundImage = newimage;
            }
            e.style.filter =
                `blur(${vv.storage.preferences.appearance.background_image_blur}px)`;
        } else {
            e.classList.add("hide");
            document.getElementById("background-image").classList.add("hide");
        }
        document.body.classList.remove("unload");
    };
    vv.control.addEventListener("current", update);
    vv.control.addEventListener("preferences", update);
    vv.control.addEventListener("preferences", update_theme);
    vv.control.addEventListener("start", update);
    vv.control.addEventListener("start", () => {
        var darkmode = window.matchMedia("(prefers-color-scheme: dark)");
        if (darkmode.addEventListener) {
            darkmode.addEventListener("change", update_theme);
        } else if (darkmode.addListener) {
            darkmode.addListener(update_theme);
        }
    });
}

vv.view.main = {
    onPreferences() {
        const e = document.getElementById("main-cover");
        if (vv.storage.preferences.appearance.circled_image) {
            e.classList.add("circled");
        } else {
            e.classList.remove("circled");
        }
    },
    onControl() {
        const c = document.getElementById("control-volume");
        if (vv.storage.control.hasOwnProperty("volume") && vv.storage.control.volume !== null) {
            c.value = vv.storage.control.volume;
            c.classList.remove("disabled");
        } else {
            c.classList.add("disabled");
        }
    },
    show() {
        document.body.classList.add("view-main");
        document.body.classList.remove("view-list");
    },
    hidden() {
        const c = document.body.classList;
        if (window.matchMedia("(orientation: portrait)").matches) {
            return !c.contains("view-main");
        }
        return !(c.contains("view-list") || c.contains("view-main"));
    },
    update() {
        if (vv.storage.current === null) {
            return;
        }
        document.getElementById("main-box-title").textContent =
            vv.storage.current.Title;
        document.getElementById("main-box-artist").textContent =
            vv.storage.current.Artist;
        document.getElementById("main-seek-label-total").textContent =
            vv.storage.current.Length;
        if (vv.storage.current.cover) {
            document.getElementById("main-cover-img").style.backgroundImage =
                `url("${vv.storage.current.cover[0]}")`;
        } else {
            document.getElementById("main-cover-img").style.backgroundImage = "";
        }
    },
    onCurrent() { vv.view.main.update(); },
    onPoll() {
        if (vv.storage.current === null || !vv.storage.current.Time) {
            return;
        }
        if (vv.view.main.hidden()) {
            return;
        }
        const c = document.getElementById("main-cover-circle-active");
        let elapsed = parseInt(vv.storage.control.song_elapsed * 1000, 10);
        if (vv.storage.control.state === "play") {
            elapsed += (new Date()).getTime() - vv.storage.last_modified_ms.control;
        }
        const total = parseInt(vv.storage.current.Time[0], 10);
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
            c.setAttribute(
                "d",
                "M 100,10 L 100,10 A 90,90 0 0,1 100,190 L 100,190 A 90,90 0 0,1 " +
                `${x},${y}`);
        } else {
            c.setAttribute("d", `M 100,10 L 100,10 A 90,90 0 0,1 ${x},${y}`);
        }
    },
    onStart() {
        document.getElementById("control-volume").addEventListener("change", e => {
            vv.control.volume(parseInt(e.currentTarget.value, 10));
        });
        vv.ui.click(document.getElementById("main-cover"), () => {
            if (vv.storage.current !== null) {
                vv.view.modal.song(vv.storage.current);
            }
        });
        vv.ui.disableSwipe(document.getElementById("main-seek"));
        document.getElementById("main-seek").addEventListener("input", (e) => {
            const target = parseInt(e.currentTarget.value, 10) * parseInt(vv.storage.current.Time[0], 10) / 1000;
            vv.control.seek(target);
        });
        vv.view.main.onPreferences();
        vv.ui.swipe(
            document.getElementById("main"), vv.view.list.show, null,
            document.getElementById("lists"), () => { return window.innerHeight >= window.innerWidth; });
    }
};
vv.control.addEventListener("poll", vv.view.main.onPoll);
vv.control.addEventListener("start", vv.view.main.onStart);
vv.control.addEventListener("current", vv.view.main.onCurrent);
vv.control.addEventListener("control", vv.view.main.onControl);
vv.control.addEventListener("preferences", vv.view.main.onPreferences);

vv.view.list = {
    show() {
        document.body.classList.add("view-list");
        document.body.classList.remove("view-main");
    },
    hidden() {
        const c = document.body.classList;
        if (window.matchMedia("(orientation: portrait)").matches) {
            return !c.contains("view-list");
        }
        return !(c.contains("view-list") || c.contains("view-main"));
    },
    _preferences_update() {
        const index = vv.storage.tree.length;
        const ul = document.getElementById("list-items" + index);
        if (vv.storage.preferences.appearance.playlist_gridview_album) {
            ul.classList.add("grid");
            ul.classList.remove("nogrid");
        } else {
            ul.classList.add("nogrid");
            ul.classList.remove("grid");
        }
    },
    _updatepos() {
        const index = vv.storage.tree.length;
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
    },
    _updateFocus() {
        const index = vv.storage.tree.length;
        const ul = document.getElementById("list-items" + index);
        let focus = null;
        let viewNowPlaying = false;
        const rootname = vv.library.rootname();
        const focusSong = vv.library.focus;
        const focusParent = vv.library.child;
        for (const listitem of Array.from(ul.children)) {
            if (listitem.classList.contains("list-header")) {
                continue;
            }
            if (focusSong && focusSong.file && focusParent) {
                if (focusParent === listitem.dataset.key) {
                    focus = listitem;
                    focus.classList.add("selected");
                } else {
                    listitem.classList.remove("selected");
                }
            } else if (
                rootname !== "root" && focusSong && focusSong.file &&
                listitem.dataset.file === focusSong.file[0]) {
                focus = listitem;
                focus.classList.add("selected");
            } else {
                listitem.classList.remove("selected");
            }
            let treeFocused = true;
            if (vv.storage.sorted && vv.storage.sorted.hasOwnProperty("sort") && vv.storage.sorted.sort !== null) {
                if (rootname === "root") {
                    treeFocused = false;
                } else if (
                    vv.storage.sorted.sort.join() !==
                    TREE[rootname].sort.join()) {
                    treeFocused = false;
                }
            }
            const elapsed = Array.from(listitem.getElementsByClassName("song-elapsed"));
            const sep = Array.from(listitem.getElementsByClassName("song-lengthseparator"));
            if (treeFocused && elapsed.length !== 0 && vv.storage.current !== null &&
                vv.storage.current.file &&
                vv.storage.current.file[0] === listitem.dataset.file) {
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
    },
    _clearAllLists() {
        const lists = document.getElementsByClassName("list");
        for (let treeindex = 0; treeindex < vv.storage.tree.length; treeindex++) {
            const oldul =
                lists[treeindex + 1].getElementsByClassName("list-items")[0];
            while (oldul.lastChild) {
                oldul.removeChild(oldul.lastChild);
            }
            lists[treeindex + 1].dataset.pwd = "";
        }
    },
    _element(song, key, style, largeImage, header) {
        const c = document.querySelector(`#list-${style}-template`).content;
        const e = c.querySelector("li");
        e.dataset.key = vv.song.getOne(song, key);
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
                n.textContent = vv.song.getOne(song, key);
            } else if (target) {
                n.textContent = vv.song.get(song, target);
            }
        }
        if (style === "song") {
            if (song.file) {
                const tooltip = [
                    "Length", "Artist", "Album", "Track", "Genre", "Performer"
                ].map(key => `${key}: ${vv.song.get(song, key)}`);
                tooltip.unshift(vv.song.get(song, "Title"));
                e.setAttribute("title", tooltip.join("\n"));
            } else {
                e.removeAttribute("title");
            }
        } else if (style === "album") {
            const cover = c.querySelector(".album-cover");
            if (song.cover) {
                const base = largeImage ? 150 : 70;
                const imgsize = parseInt(base * window.devicePixelRatio, 10);
                cover.src = `${song.cover}?width=${imgsize}&height=${imgsize}`;
            } else {
                cover.src = "/assets/nocover.svg";
            }
            cover.alt = `Cover art: ${vv.song.get(song, "Album")} ` +
                `by ${vv.song.get(song, "AlbumArtist")}`;
        }
        return document.importNode(c, true);
    },
    _listHandler(e) {
        if (e.currentTarget.classList.contains("playing")) {
            if (vv.storage.current === null) {
                return;
            }
            vv.library.abs(vv.storage.current);
            vv.view.main.show();
            return;
        }
        const value = e.currentTarget.dataset.key;
        const pos = e.currentTarget.dataset.pos;
        if (e.currentTarget.classList.contains("song")) {
            vv.control.play(parseInt(pos, 10));
        } else {
            vv.library.down(value);
        }
    },
    _update() {
        const index = vv.storage.tree.length;
        const scroll = document.getElementById("list" + index);
        const pwd = vv.storage.tree.join();
        if (scroll.dataset.pwd === pwd) {
            vv.view.list._updatepos();
            vv.view.list._updateFocus();
            return;
        }
        scroll.dataset.pwd = pwd;
        const ls = vv.library.list();
        const key = ls.key;
        const songs = ls.songs;
        const style = ls.style;
        const newul = document.createDocumentFragment();
        const lists = document.getElementsByClassName("list");
        for (let treeindex = 0; treeindex < vv.storage.tree.length; treeindex++) {
            const currentpwd = vv.storage.tree.slice(0, treeindex + 1).join();
            const viewpwd = lists[treeindex + 1].dataset.pwd;
            if (currentpwd !== viewpwd) {
                const oldul =
                    lists[treeindex + 1].getElementsByClassName("list-items")[0];
                while (oldul.lastChild) {
                    oldul.removeChild(oldul.lastChild);
                }
                lists[treeindex + 1].dataset.pwd = "";
            }
        }
        vv.view.list._updatepos();
        const ul = document.getElementById("list-items" + index);
        while (ul.lastChild) {
            ul.removeChild(ul.lastChild);
        }
        ul.classList.remove("songlist");
        ul.classList.remove("albumlist");
        ul.classList.remove("plainlist");
        ul.classList.add(style + "list");
        vv.view.list._preferences_update();
        const p = vv.library.parent();
        for (let i = 0, imax = songs.length; i < imax; i++) {
            if (i === 0 && p) {
                const li = vv.view.list._element(p.song, p.key, p.style, false, true);
                newul.appendChild(li);
            }
            const li = vv.view.list._element(
                songs[i], key, style, ul.classList.contains("grid"), false);
            vv.ui.click(
                li.querySelector("li"), vv.view.list._listHandler, false);
            newul.appendChild(li);
        }
        ul.appendChild(newul);
        vv.view.list._updateFocus();
    },
    _updateForce() {
        vv.view.list._clearAllLists();
        vv.view.list._update();
    },
    _select_near_item() {
        const index = vv.storage.tree.length;
        const scroll = document.getElementById("list" + index);
        let updated = false;
        for (const selectable of document.querySelectorAll(`#list-items${index} .selectable`)) {
            const p = selectable.offsetTop;
            if (scroll.scrollTop < p && p < scroll.scrollTop + scroll.clientHeight &&
                !updated) {
                selectable.classList.add("selected");
                updated = true;
            } else {
                selectable.classList.remove("selected");
            }
        }
    },
    _select_focused_or(target) {
        const style = vv.library.list().style;
        const index = vv.storage.tree.length;
        const scroll = document.getElementById("list" + index);
        const l = document.getElementById("list-items" + index);
        let itemcount = parseInt(scroll.clientWidth / 160, 10);
        if (!vv.storage.preferences.appearance.playlist_gridview_album) {
            itemcount = 1;
        }
        const t = scroll.scrollTop;
        const h = scroll.clientHeight;
        const s = l.getElementsByClassName("selected");
        const f = l.getElementsByClassName("playing");
        let p = 0;
        let c = null;
        let n = null;
        if (s.length === 0 && f.length === 1) {
            p = f[0].offsetTop;
            if (t < p && p < t + h) {
                f[0].classList.add("selected");
                return;
            }
        }
        if (s.length > 0) {
            p = s[0].offsetTop;
            if (p < t || t + h < p + s[0].offsetHeight) {
                vv.view.list._select_near_item();
                return;
            }
        }
        if (s.length === 0 && f.length === 0) {
            vv.view.list._select_near_item();
            return;
        }
        if (s.length > 0) {
            let selectables = l.getElementsByClassName("selectable");
            if (target === "up" && selectables[0] === s[0]) {
                return;
            }
            if (target === "down" && selectables[selectables.length - 1] === s[0]) {
                return;
            }
            for (let i = 0; i < selectables.length; i++) {
                c = selectables[i];
                if (c === s[0]) {
                    if ((i > 0 && target === "up" && style !== "album") ||
                        (i > 0 && target === "left")) {
                        n = selectables[i - 1];
                        c.classList.remove("selected");
                        n.classList.add("selected");
                        p = n.offsetTop;
                        if (p < t) {
                            scroll.scrollTop = p;
                        }
                        return;
                    }
                    if (i > itemcount - 1 && target === "up" && style === "album") {
                        n = selectables[i - itemcount];
                        c.classList.remove("selected");
                        n.classList.add("selected");
                        p = n.offsetTop;
                        if (p < t) {
                            scroll.scrollTop = p;
                        }
                        return;
                    }
                    if ((i !== (selectables.length - 1) && target === "down" &&
                        style !== "album") ||
                        (i !== (selectables.length - 1) && target === "right")) {
                        n = selectables[i + 1];
                        c.classList.remove("selected");
                        n.classList.add("selected");
                        p = n.offsetTop + n.offsetHeight;
                        if (t + h < p) {
                            scroll.scrollTop = p - h;
                        }
                        return;
                    }
                    if ((i < (selectables.length - 1) && target === "down" &&
                        style === "album") ||
                        (i !== (selectables.length - 1) && target === "right")) {
                        if (i + itemcount >= selectables.length) {
                            n = selectables[selectables.length - 1];
                        } else {
                            n = selectables[i + itemcount];
                        }
                        c.classList.remove("selected");
                        n.classList.add("selected");
                        p = n.offsetTop + n.offsetHeight;
                        if (t + h < p) {
                            scroll.scrollTop = p - h;
                        }
                        return;
                    }
                }
            }
        }
    },
    up() { vv.view.list._select_focused_or("up"); },
    left() { vv.view.list._select_focused_or("left"); },
    right() { vv.view.list._select_focused_or("right"); },
    down() { vv.view.list._select_focused_or("down"); },
    activate() {
        const index = vv.storage.tree.length;
        const es = document.getElementById("list-items" + index)
            .getElementsByClassName("selected");
        if (es.length !== 0) {
            const e = {};
            e.currentTarget = es[0];
            es[0].click_target(e);
            return true;
        }
        return false;
    },
    onCurrent() { vv.view.list._update(); },
    onPreferences() { vv.view.list._preferences_update(); },
    onStart() {
        vv.library.addEventListener("update", vv.view.list._updateForce);
        vv.library.addEventListener("changed", vv.view.list._update);
        vv.ui.swipe(
            document.getElementById("list1"), vv.library.up,
            vv.view.list._updatepos, document.getElementById("list0"));
        vv.ui.swipe(
            document.getElementById("list2"), vv.library.up,
            vv.view.list._updatepos, document.getElementById("list1"));
        vv.ui.swipe(
            document.getElementById("list3"), vv.library.up,
            vv.view.list._updatepos, document.getElementById("list2"));
        vv.ui.swipe(
            document.getElementById("list4"), vv.library.up,
            vv.view.list._updatepos, document.getElementById("list3"));
        vv.ui.swipe(
            document.getElementById("list5"), vv.library.up,
            vv.view.list._updatepos, document.getElementById("list4"));
    }
};
vv.control.addEventListener("current", vv.view.list.onCurrent);
vv.control.addEventListener("preferences", vv.view.list.onPreferences);
vv.control.addEventListener("start", vv.view.list.onStart);

vv.view.system = {
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
            obj.checked = vv.storage.preferences[mainkey][subkey];
            getter = () => { return obj.checked; };
        } else if (obj.tagName.toLowerCase() === "select") {
            obj.value = String(vv.storage.preferences[mainkey][subkey]);
            getter = () => { return obj.value; };
        } else if (obj.type === "range") {
            obj.value = String(vv.storage.preferences[mainkey][subkey]);
            getter = () => { return parseInt(obj.value, 10); };
            obj.addEventListener("input", () => {
                vv.storage.preferences[mainkey][subkey] = obj.value;
                vv.control.raiseEvent("preferences");
            });
            vv.ui.disableSwipe(obj);
        } else if (obj.type === "radio") {
            if (obj.value === vv.storage.preferences[mainkey][subkey]) {
                obj.checked = "checked";
            }
            getter = () => { return obj.value; };
        }
        obj.addEventListener("change", () => {
            vv.storage.preferences[mainkey][subkey] = getter();
            vv.storage.save.preferences();
            vv.control.raiseEvent("preferences");
        });
    },
    onPreferences() {
        if (vv.storage.preferences.appearance.theme === "prefer-coverart") {
            document.getElementById("config-appearance-color-threshold").classList.remove("hide");
        } else {
            document.getElementById("config-appearance-color-threshold").classList.add("hide");
        }
    },
    onOutputs() {
        const ul = document.getElementById("devices");
        while (ul.lastChild) {
            ul.removeChild(ul.lastChild);
        }
        const newul = document.createDocumentFragment();
        for (const id in vv.storage.outputs) {
            if (vv.storage.outputs.hasOwnProperty(id)) {
                const o = vv.storage.outputs[id];
                const c = document.querySelector("#device-template").content;
                const e = c.querySelector("li");
                e.querySelector(".system-setting-desc").textContent = o.name;
                if (o.plugin) {
                    e.querySelector(".plugin").textContent = o.plugin;
                } else {
                    e.querySelector(".plugin").classList.add("disabled");
                }
                const ch = e.querySelector(".slideswitch");
                ch.setAttribute("aria-label", o.name);
                ch.dataset.deviceid = id;
                ch.checked = o.enabled;
                const d = document.importNode(c, true);
                d.querySelector(".slideswitch").addEventListener("change", e => {
                    vv.control.output(
                        parseInt(e.currentTarget.dataset.deviceid, 10),
                        e.currentTarget.checked);
                });
                newul.appendChild(d);
            }
        }
        ul.appendChild(newul);
    },
    onControl() {
        const e = document.getElementById("library-rescan");
        if (vv.storage.control.update_library && !e.disabled) {
            e.disabled = true;
        } else if (!vv.storage.control.update_library && e.disabled) {
            e.disabled = false;
        }
    },
    onStart() {
        // preferences
        vv.view.system.onPreferences();

        vv.control.addEventListener("control", () => {
            if (vv.storage.control.hasOwnProperty("volume") && vv.storage.control.volume !== null) {
                document.getElementById("volume-header").classList.remove("hide");
                document.getElementById("volume-all").classList.remove("hide");
            } else {
                document.getElementById("volume-header").classList.add("hide");
                document.getElementById("volume-all").classList.add("hide");
            }
        });

        if (window.matchMedia("(prefers-color-scheme: dark)").matches === window.matchMedia("(prefers-color-scheme: light)").matches) {
            document.getElementById("appearance-theme-prefer-system").disabled = true;
            if (vv.storage.preferences.appearance.theme === "prefer-system") {
                vv.storage.preferences.appearance.theme = "prefer-coverart";
            }
        }

        vv.view.system._initconfig("appearance-theme_light");
        vv.view.system._initconfig("appearance-theme_dark");
        vv.view.system._initconfig("appearance-theme_prefer-system");
        vv.view.system._initconfig("appearance-theme_prefer-coverart");
        vv.view.system._initconfig("appearance-color-threshold");
        vv.view.system._initconfig("appearance-background-image");
        vv.view.system._initconfig("appearance-background-image-blur");
        vv.view.system._initconfig("appearance-circled-image");
        vv.view.system._initconfig("appearance-playlist-gridview-album");
        vv.view.system._initconfig("appearance-playlist-follows-playback");
        vv.view.system._initconfig("appearance-volume");
        vv.view.system._initconfig("appearance-volume-max");
        vv.view.system._initconfig("playlist-playback-tracks_all");
        vv.view.system._initconfig("playlist-playback-tracks_list");
        document.getElementById("system-reload").addEventListener("click", () => {
            location.reload();
        });
        document.getElementById("library-rescan").addEventListener("click", () => {
            vv.control.rescan_library();
        });
        // info
        document.getElementById("user-agent").textContent = navigator.userAgent;

        const navs = Array.from(document.getElementsByClassName("system-nav-item"));
        const showChild = e => {
            document.getElementById("system-nav").classList.remove("on");
            document.getElementById("system-box-nav-back").classList.remove("root");
            for (const nav of navs) {
                if (nav === e.currentTarget) {
                    if (nav.id === "system-nav-database") {
                        vv.view.system._update_stats();
                    }
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
            vv.ui.swipe(
                document.getElementById(nav.dataset.target), showParent, null,
                document.getElementById("system-nav"),
                () => { return window.innerWidth <= 760; });
        }
        document.getElementById("system-box-nav-back").addEventListener("click", showParent);
    },
    _zfill2(i) {
        if (i < 100) {
            return ("00" + i).slice(-2);
        }
        return i;
    },
    _strtimedelta(i) {
        const zfill2 = vv.view.system._zfill2;
        const uh = parseInt(i / (60 * 60), 10);
        const um = parseInt((i - uh * 60 * 60) / 60, 10);
        const us = parseInt(i - uh * 60 * 60 - um * 60, 10);
        return `${zfill2(uh)}:${zfill2(um)}:${zfill2(us)}`;
    },
    _update_stats() {
        document.getElementById("stat-albums").textContent =
            vv.storage.stats.albums.toString(10);
        document.getElementById("stat-artists").textContent =
            vv.storage.stats.artists.toString(10);
        document.getElementById("stat-db-playtime").textContent =
            vv.view.system._strtimedelta(vv.storage.stats.library_playtime, 10);
        document.getElementById("stat-tracks").textContent = vv.storage.stats.songs;
        const db_update = new Date(vv.storage.stats.library_update * 1000);
        const options = {
            hour: "numeric",
            minute: "numeric",
            second: "numeric",
            year: "numeric",
            month: "short",
            day: "numeric",
            weekday: "short"
        };
        document.getElementById("stat-db-update").textContent =
            db_update.toLocaleString(document.documentElement.lang, options);
    },
    onStats() {
        if (document.getElementById("system-database").classList.contains("on")) {
            vv.view.system._update_stats();
        }
    },
    onVersion() {
        if (vv.storage.version.app) {
            document.getElementById("version").textContent = vv.storage.version.app;
            document.getElementById("mpd-version").textContent = vv.storage.version.mpd;
            document.getElementById("go-version").textContent = vv.storage.version.go;
        }
    },
    show() {
        document.getElementById("modal-background").classList.remove("hide");
        document.getElementById("modal-outer").classList.remove("hide");
        document.getElementById("modal-system").classList.remove("hide");
    }
};
vv.control.addEventListener("start", vv.view.system.onStart);
vv.control.addEventListener("version", vv.view.system.onVersion);
vv.control.addEventListener("control", vv.view.system.onControl);
vv.control.addEventListener("status", vv.view.system.onStats);
vv.control.addEventListener("preferences", vv.view.system.onPreferences);
vv.control.addEventListener("outputs", vv.view.system.onOutputs);

// header
{
    const update = () => {
        const e = document.getElementById("header-back-label");
        const b = document.getElementById("header-back");
        const m = document.getElementById("header-main");
        if (vv.library.rootname() === "root") {
            b.classList.add("root");
            m.classList.add("root");
        } else {
            b.classList.remove("root");
            m.classList.remove("root");
            const songs = vv.library.list().songs;
            if (songs[0]) {
                const p = vv.library.grandparent();
                if (p) {
                    e.textContent = vv.song.getOne(p.song, p.key);
                    if (p.song.keys) {
                        for (const kv of p.song.keys) {
                            if (kv[0] === p.key) {
                                e.textContent = kv[1];
                                break;
                            }
                        }
                    }
                    b.setAttribute(
                        "title", b.dataset.titleFormat.replace("%s", e.textContent));
                    b.setAttribute(
                        "aria-label",
                        b.dataset.ariaLabelFormat.replace("%s", e.textContent));
                }
            }
        }
    };
    vv.control.addEventListener("start", () => {
        document.getElementById("header-back").addEventListener("click", e => {
            if (vv.view.list.hidden()) {
                if (vv.storage.current !== null) {
                    vv.library.abs(vv.storage.current);
                }
            } else {
                vv.library.up();
            }
            vv.view.list.show();
            e.stopPropagation();
        });
        document.getElementById("header-main").addEventListener("click", e => {
            e.stopPropagation();
            if (vv.storage.current !== null) {
                vv.library.abs(vv.storage.current);
            }
            vv.view.main.show();
            e.stopPropagation();
        });
        document.getElementById("header-system").addEventListener("click", e => {
            vv.view.system.show();
            e.stopPropagation();
        });
        update();
        vv.library.addEventListener("changed", update);
        vv.library.addEventListener("update", update);
    });
}

vv.view.footer = {
    onPreferences() {
        const c = document.getElementById("control-volume");
        c.max = parseInt(vv.storage.preferences.appearance.volume_max, 10);
        if (vv.storage.preferences.appearance.volume) {
            c.classList.remove("hide");
        } else {
            c.classList.add("hide");
        }
    },
    onStart() {
        vv.view.footer.onPreferences();
        document.getElementById("control-prev").addEventListener("click", e => {
            vv.control.prev();
            e.stopPropagation();
        });
        document.getElementById("control-toggleplay")
            .addEventListener("click", e => {
                vv.control.play_pause();
                e.stopPropagation();
            });
        document.getElementById("control-next").addEventListener("click", e => {
            vv.control.next();
            e.stopPropagation();
        });
        document.getElementById("control-repeat").addEventListener("click", e => {
            vv.control.toggle_repeat();
            e.stopPropagation();
        });
        document.getElementById("control-random").addEventListener("click", e => {
            vv.control.toggle_random();
            e.stopPropagation();
        });
    },
    onControl() {
        const toggleplay = document.getElementById("control-toggleplay");
        if (vv.storage.control.state === "play") {
            toggleplay.setAttribute("aria-label", toggleplay.dataset.ariaLabelPause);
            toggleplay.classList.add("pause");
            toggleplay.classList.remove("play");
        } else {
            toggleplay.setAttribute("aria-label", toggleplay.dataset.ariaLabelPlay);
            toggleplay.classList.add("play");
            toggleplay.classList.remove("pause");
        }
        const repeat = document.getElementById("control-repeat");
        if (vv.storage.control.single) {
            repeat.setAttribute("aria-label", repeat.dataset.ariaLabelOn);
            repeat.classList.add("single-on");
            repeat.classList.remove("single-off");
        } else {
            repeat.classList.add("single-off");
            repeat.classList.remove("single-on");
        }
        if (vv.storage.control.repeat) {
            if (!vv.storage.control.single) {
                repeat.setAttribute("aria-label", repeat.dataset.ariaLabelSingleOff);
            }
            repeat.classList.add("on");
            repeat.classList.remove("off");
        } else {
            if (!vv.storage.control.single) {
                repeat.setAttribute("aria-label", repeat.dataset.ariaLabelOff);
            }
            repeat.classList.add("off");
            repeat.classList.remove("on");
        }
        const random = document.getElementById("control-random");
        if (vv.storage.control.random) {
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
vv.control.addEventListener("start", vv.view.footer.onStart);
vv.control.addEventListener("control", vv.view.footer.onControl);
vv.control.addEventListener("preferences", vv.view.footer.onPreferences);

vv.view.popup = {
    show(target, description) {
        const obj = document.getElementById("popup-" + target);
        if (!obj) {
            vv.view.popup.show("fixme", `popup-${target} is not found in html`);
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
        obj.ttl = obj.timestamp + 4000;
        setTimeout(() => {
            if ((new Date()).getTime() - obj.ttl > 0) {
                obj.classList.remove("show");
                obj.classList.add("hide");
            }
        }, 5000);
    },
    hide(target) {
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

// elapsed circle/time updater
{
    const update = () => {
        const data = vv.storage.control;
        if ("state" in data) {
            const elapsed = parseInt(data.song_elapsed * 1000, 10);
            let current = elapsed;
            if (data.state === "play") {
                current += (new Date()).getTime() - vv.storage.last_modified_ms.control;
            }
            current = parseInt(current / 1000, 10);
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
    vv.control.addEventListener("control", update);
    vv.control.addEventListener("poll", update);
}

vv.view.modal = {
    hide() {
        document.getElementById("modal-background").classList.add("hide");
        document.getElementById("modal-outer").classList.add("hide");
        for (const w of Array.from(document.getElementsByClassName("modal-window"))) {
            w.classList.add("hide");
        }
    },
    onStart() {
        document.getElementById("modal-background")
            .addEventListener("click", vv.view.modal.hide);
        document.getElementById("modal-outer")
            .addEventListener("click", vv.view.modal.hide);
        for (const w of Array.from(document.getElementsByClassName("modal-window"))) {
            w.addEventListener("click", e => { e.stopPropagation(); });
        }
        for (const w of Array.from(document.getElementsByClassName("modal-window-close"))) {
            w.addEventListener("click", vv.view.modal.hide);
        }
    },
    help() {
        const b = document.getElementById("modal-background");
        if (!b.classList.contains("hide")) {
            return;
        }
        b.classList.remove("hide");
        document.getElementById("modal-outer").classList.remove("hide");
        document.getElementById("modal-help").classList.remove("hide");
    },
    song(song) {
        const mustkeys = [
            "Title", "Artist", "Album", "Date", "AlbumArtist", "Genre", "Performer",
            "Disc", "Track", "Composer", "Length"
        ];
        for (const key of mustkeys) {
            const doc = document.getElementById("modal-song-box-" + key);
            while (doc.lastChild) {
                doc.removeChild(doc.lastChild);
            }
            const newdoc = document.createDocumentFragment();
            const values = vv.song.getOrElseMulti(song, key, []);
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
                        targetValues = vv.song.getOrElseMulti(song, target, values);
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
                                    vv.library.absaddr(d.root, d.value);
                                    vv.view.list.show();
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
        if (song.cover) {
            const imgsize = window.devicePixelRatio * 112;
            cover.src = `${song.cover}?width=${imgsize}&height=${imgsize}`;
        } else {
            cover.src = "/assets/nocover.svg";
        }
        document.getElementById("modal-background").classList.remove("hide");
        document.getElementById("modal-outer").classList.remove("hide");
        document.getElementById("modal-song").classList.remove("hide");
    }
};
vv.control.addEventListener("start", vv.view.modal.onStart);

// keyboard events
{
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
    const inList = t => {
        return () => {
            if (!vv.view.list.hidden()) {
                t();
                return true;
            }
            return false;
        };
    };
    const back = () => {
        if (vv.view.list.hidden()) {
            if (vv.storage.current !== null) {
                vv.library.abs(vv.storage.current);
            }
        } else {
            vv.library.up();
        }
        vv.view.list.show();
    };
    const keymap = {
        [none]: {
            Enter() { return !vv.view.list.hidden() && vv.view.list.activate(); },
            Backspace: any(back),
            ArrowLeft: inList(vv.view.list.left),
            ArrowUp: inList(vv.view.list.up),
            ArrowRight: inList(vv.view.list.right),
            ArrowDown: inList(vv.view.list.down),
            [" "]: any(vv.control.play_pause),
            ["?"]: any(vv.view.modal.help)
        },
        [shift]: { ["?"]: any(vv.view.modal.help) },
        [meta]: {
            ArrowLeft: any(back),
            ArrowRight: any(() => {
                if (vv.library.rootname() !== "root") {
                    if (vv.storage.current !== null) {
                        vv.library.abs(vv.storage.current);
                    }
                }
                vv.view.main.show();
            })
        },
        [shift | ctrl]:
            { ArrowLeft: any(vv.control.prev), ArrowRight: any(vv.control.next) }
    };
    vv.control.addEventListener("start", () => {
        document.addEventListener("keydown", e => {
            if (!document.getElementById("modal-background")
                .classList.contains("hide")) {
                if (e.key === "Escape" || e.key === "Esc") {
                    vv.view.modal.hide();
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

vv.control.start();
