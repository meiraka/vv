"use strict";
const vv = {
  env: {},
  consts: {playlistLength: 9999},
  obj: {},
  song: {},
  songs: {},
  storage: {},
  model: {list: {}},
  view:
      {main: {}, list: {}, system: {}, popup: {}, modal: {help: {}, song: {}}},
  control: {}
};
vv.obj = (function() {
  const pub = {};
  pub.getOrElse = function(m, k, v) { return k in m ? m[k] : v; };
  pub.copy = function(t) {
    if (Object.prototype.toString.call(t) === "[object Array]") {
      const ret = [];
      for (let i = 0, imax = t.length; i < imax; i++) {
        ret[i] = t[i];
      }
      return ret;
    }
    const ret = {};
    Object.keys(t).forEach(function(k) { ret[k] = t[k]; });
    return ret;
  };
  return pub;
})();
vv.song = (function() {
  const pub = {};
  const tag = function(song, keys, other) {
    for (const key of keys) {
      if (key in song) {
        return song[key];
      }
    }
    return other;
  };
  const getTagOrElseMulti = function(song, key, other) {
    if (key in song) {
      return song[key];
    } else if (key === "AlbumSort") {
      return tag(song, ["Album"], other);
    } else if (key === "ArtistSort") {
      return tag(song, ["Artist"], other);
    } else if (key === "AlbumArtist") {
      return tag(song, ["Artist"], other);
    } else if (key === "AlbumArtistSort") {
      return tag(song, ["AlbumArtist", "Artist"], other);
    } else if (key === "AlbumSort") {
      return tag(song, ["Album"], other);
    }
    return other;
  };
  pub.getOrElseMulti = function(song, key, other) {
    let ret = [];
    const keys = key.split("-");
    for (const key of keys) {
      const t = getTagOrElseMulti(song, key, other);
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
  };
  const getOrElse = function(song, key, other) {
    const ret = pub.getOrElseMulti(song, key, null);
    if (!ret) {
      return other;
    }
    return ret.join();
  };
  const getOneOrElse = function(song, key, other) {
    if (!song.keys) {
      return pub.getOrElseMulti(song, key, [other])[0];
    }
    for (const kv of song.keys) {
      if (kv[0] === key) {
        return kv[1];
      }
    }
    return pub.getOrElseMulti(song, key, [other])[0];
  };
  pub.getOne = function(song, key) {
    return getOneOrElse(song, key, "[no " + key + "]");
  };
  pub.get = function(song, key) {
    return getOrElse(song, key, "[no " + key + "]");
  };
  pub.sortkeys = function(song, keys, memo) {
    let songs = [vv.obj.copy(song)];
    songs[0].sortkey = "";
    songs[0].keys = [];
    for (const key of keys) {
      const writememo = memo.indexOf(key) !== -1;
      const values = pub.getOrElseMulti(song, key, []);
      if (values.length === 0) {
        for (const song of songs) {
          song.sortkey += " ";
          if (writememo) {
            song.keys.push([key, "[no " + key + "]"]);
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
            const newsong = vv.obj.copy(song);
            newsong.keys = vv.obj.copy(song.keys);
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
  };
  pub.element = function(e, song, key, style, largeImage) {
    e.classList.remove("plain");
    e.classList.remove("song");
    e.classList.remove("album");
    e.classList.remove("playing");
    e.classList.add(style);
    e.classList.add("note-line");
    e.dataset.key = vv.song.getOne(song, key);
    if (song.file) {
      e.dataset.file = song.file[0];
      e.dataset.pos = song.pos;
      e.setAttribute("contextmenu", "conext-" + style + song.file[0]);
      const menu = document.createElement("menu");
      menu.setAttribute("type", "context");
      menu.classList.add("contextmenu");
      menu.id = "conext-" + style + song.file[0];
      const menuitem = document.createElement("menuitem");
      menuitem.setAttribute("label", "Song Infomation");
      menuitem.addEventListener("click", function(e) {
        vv.view.modal.song.show(song);
        e.stopPropagation();
      });
      menu.appendChild(menuitem);
      e.appendChild(menu);
    }
    if (style === "song") {
      if (song.file) {
        let tooltip = vv.song.get(song, "Title") + "\n";
        const keys = [
          "Length", "Artist", "Album", "Track", "Genre", "Performer"];
        for (const key of keys) {
          tooltip += key + ": " + vv.song.get(song, key) + "\n";
        }
        e.setAttribute("title", tooltip);
      }
      const track = document.createElement("span");
      track.classList.add("song-track");
      track.textContent = vv.song.get(song, "TrackNumber");
      e.appendChild(track);
      const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
      svg.classList.add("song-playingicon");
      svg.classList.add("reversible-icon");
      svg.setAttribute("width", "22");
      svg.setAttribute("height", "22");
      svg.setAttribute("viewBox", "0 0 100 100");
      const path = document.createElementNS("http://www.w3.org/2000/svg", "path");
      path.classList.add("fill");
      path.setAttribute("d", "M 25,20 80,50 25,80 z");
      svg.appendChild(path);
      e.appendChild(svg);
      const title = document.createElement("span");
      title.classList.add("song-title");
      title.textContent = vv.song.get(song, "Title");
      e.appendChild(title);
      const artist = document.createElement("span");
      artist.classList.add("song-artist");
      artist.textContent = vv.song.get(song, "Artist");
      if (vv.song.get(song, "Artist") !== vv.song.get(song, "AlbumArtist")) {
        artist.classList.add("low-prio");
      }
      e.appendChild(artist);
      const elapsed = document.createElement("span");
      elapsed.classList.add("song-elapsed");
      elapsed.setAttribute("aria-hidden", "true");
      e.appendChild(elapsed);
      const length_separator = document.createElement("span");
      length_separator.classList.add("song-lengthseparator");
      length_separator.setAttribute("aria-hidden", "true");
      length_separator.textContent = "/";
      e.appendChild(length_separator);
      const length = document.createElement("span");
      length.classList.add("song-length");
      length.textContent = vv.song.get(song, "Length");
      e.appendChild(length);
    } else if (style === "album") {
      const coverbox = document.createElement("div");
      coverbox.classList.add("album-coverbox");
      const p = window.devicePixelRatio;
      const cover = document.createElement("img");
      cover.classList.add("album-cover");
      let imgsize = parseInt(70 * p, 10);
      if (song.cover) {
        if (largeImage) {
          imgsize = 150 * p;
        }
        cover.src = "/api/images/music_directory/" + song.cover + "?width=" +
            imgsize + "&height=" + imgsize;
      } else {
        cover.src = "/assets/nocover.svg";
      }
      cover.alt = 'Cover art: ' + vv.song.get(song, "Album") + ' by ' +
          vv.song.get(song, "AlbumArtist");
      coverbox.appendChild(cover);
      e.appendChild(coverbox);

      const detail = document.createElement("div");
      detail.classList.add("album-detail");
      const date = document.createElement("span");
      date.classList.add("album-detail-date");
      date.textContent = vv.song.get(song, "Date");
      detail.appendChild(date);
      const album = document.createElement("span");
      album.classList.add("album-detail-album");
      album.textContent = vv.song.get(song, "Album");
      detail.appendChild(album);
      const albumartist = document.createElement("span");
      albumartist.classList.add("album-detail-albumartist");
      albumartist.textContent = vv.song.get(song, "AlbumArtist");
      detail.appendChild(albumartist);
      e.appendChild(detail);
    } else {
      const plain = document.createElement("span");
      plain.classList.add("plain-key");
      plain.textContent = vv.song.getOne(song, key);
      e.appendChild(plain);
    }
    return e;
  };

  return pub;
})();
vv.songs = (function() {
  const pub = {};
  pub.sort = function(songs, keys, memo) {
    const newsongs = [];
    for (const song of songs) {
      Array.prototype.push.apply(
          newsongs, vv.song.sortkeys(song, keys, memo));
    }
    const sorted = newsongs.sort(function(a, b) {
      if (a.sortkey < b.sortkey) {
        return -1;
      }
      return 1;
    });
    for (let j = 0, jmax = sorted.length; j < jmax; j++) {
      sorted[j].pos = [j];
    }
    return sorted;
  };
  pub.uniq = function(songs, key) {
    return songs.filter(function(song, i, self) {
      if (i === 0) {
        return true;
      } else if (
          vv.song.getOne(song, key) === vv.song.getOne(self[i - 1], key)) {
        return false;
      }
      return true;
    });
  };
  pub.filter = function(songs, filters) {
    return songs.filter(function(song) {
      for (const key in filters) {
        if (filters.hasOwnProperty(key)) {
          if (vv.song.getOne(song, key) !== filters[key]) {
            return false;
          }
        }
      }
      return true;
    });
  };
  pub.weakFilter = function(songs, filters, max) {
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
  };
  return pub;
})();
vv.storage = (function() {
  const idbUpdateTables = function(e) {
    const db = e.target.result;
    const st = db.createObjectStore("cache", {keyPath: "id"});
    const close = function() { db.close(); };
    st.onsuccess = close;
    st.onerror = close;
  };
  const cacheLoad = function(key, callback) {
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
    req.onerror = function() {};
    req.onupgradeneeded = idbUpdateTables;
    req.onsuccess = function(e) {
      const db = e.target.result;
      const t = db.transaction("cache", "readonly");
      const so = t.objectStore("cache");
      const req = so.get(key);
      req.onsuccess = function(e) {
        const ret = e.target.result;
        if (ret && ret.value && ret.date) {
          callback(e.target.result.value, e.target.result.date);
        } else {
          callback();
        }
        db.close();
      };
      req.onerror = function() {
        callback();
        db.close();
      };
    };
  };

  const cacheSave = function(key, value, date) {
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
    req.onerror = function() {};
    req.onupgradeneeded = idbUpdateTables;
    req.onsuccess = function(e) {
      const db = e.target.result;
      const t = db.transaction("cache", "readwrite");
      const so = t.objectStore("cache");
      const req = so.get(key);
      req.onerror = function() { db.close(); };
      req.onsuccess = function(e) {
        const ret = e.target.result;
        if (ret && ret.date && ret.date === date) {
          return;
        }
        const req = so.put({id: key, value: value, date: date});
        req.onerror = function() { db.close(); };
        req.onsuccess = function() { db.close(); };
      };
    };
  };

  const pub = {
    loaded: false,
    root: "root",
    tree: [],
    current: null,
    control: {},
    library: [],
    outputs: [],
    stats: {},
    last_modified: {},
    last_modified_ms: {},
    version: {}
  };

  const listener = {onload: []};
  pub.addEventListener = function(ev, func) { listener[ev].push(func); };
  const raiseEvent = function(ev) {
    if (!(ev in listener)) {
      return;
    }
    for (const l of listener[ev]) {
      l();
    }
  };
  pub.preferences = {
    volume: {show: true, max: "100"},
    playback: {view_follow: true},
    appearance: {
      color_threshold: 128,
      animation: true,
      background_image: true,
      background_image_blur: 32,
      circled_image: true,
      gridview_album: true,
      auto_hide_scrollbar: true
    }
  };
  pub.save = {};
  pub.save.current = function() {
    try {
      localStorage.current = JSON.stringify(pub.current);
      localStorage.current_last_modified = pub.last_modified.current;
    } catch (e) {
    }
  };
  pub.save.root = function() {
    try {
      localStorage.root = pub.root;
    } catch (e) {
    }
  };
  pub.save.preferences = function() {
    try {
      localStorage.preferences = JSON.stringify(pub.preferences);
    } catch (e) {
    }
  };
  pub.save.sorted = function() {
    try {
      localStorage.sorted = JSON.stringify(pub.sorted);
      localStorage.sorted_last_modified = pub.last_modified.sorted;
    } catch (e) {
    }
  };
  pub.save.library = function() {
    try {
      cacheSave("library", pub.library, pub.last_modified.library);
    } catch (e) {
    }
  };
  pub.load = function() {
    try {
      if (localStorage.root && localStorage.root.length !== 0) {
        pub.root = localStorage.root;
        if (pub.root !== "root") {
          pub.tree.push(["root", pub.root]);
        }
      }
      if (localStorage.preferences) {
        const c = JSON.parse(localStorage.preferences);
        for (const i in c) {
          if (c.hasOwnProperty(i)) {
            for (const j in c[i]) {
              if (c[i].hasOwnProperty(j)) {
                if (pub.preferences[i]) {
                  pub.preferences[i][j] = c[i][j];
                }
              }
            }
          }
        }
      }
      if (localStorage.current && localStorage.current_last_modified) {
        const current = JSON.parse(localStorage.current);
        if (Object.prototype.toString.call(current.file) === "[object Array]") {
          pub.current = current;
          pub.last_modified.current = localStorage.current_last_modified;
        }
      }
      if (localStorage.sorted && localStorage.sorted_last_modified) {
        const sorted = JSON.parse(localStorage.sorted);
        pub.sorted = sorted;
        pub.last_modified.sorted = localStorage.sorted_last_modified;
      }
      cacheLoad("library", function(data, date) {
        if (data && date) {
          pub.library = data;
          pub.last_modified.library = date;
        }
        pub.loaded = true;
        raiseEvent("onload");
      });
    } catch (e) {
      pub.loaded = true;
      raiseEvent("onload");
      // private browsing
    }
    // Mobile
    if (navigator.userAgent.indexOf("Mobile") > 1) {
      pub.preferences.appearance.auto_hide_scrollbar = false;
    }
  };
  pub.load();
  return pub;
})();

vv.model.list = (function() {
  const pub = {};
  const library = {
    AlbumArtist: [],
    Album: [],
    Artist: [],
    Genre: [],
    Date: [],
    Composer: [],
    Performer: []
  };
  pub.TREE = Object.freeze({
    AlbumArtist: {
      sort: [
        "AlbumArtist", "Date", "Album", "DiscNumber", "TrackNumber", "Title",
        "file"
      ],
      tree: [["AlbumArtist", "plain"], ["Album", "album"], ["Title", "song"]]
    },
    Album: {
      sort: [
        "AlbumArtist-Date-Album", "DiscNumber", "TrackNumber", "Title", "file"
      ],
      tree: [["AlbumArtist-Date-Album", "album"], ["Title", "song"]]
    },
    Artist: {
      sort: [
        "Artist", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"
      ],
      tree: [["Artist", "plain"], ["Title", "song"]]
    },
    Genre: {
      sort: ["Genre", "Album", "DiscNumber", "TrackNumber", "Title", "file"],
      tree: [["Genre", "plain"], ["Album", "album"], ["Title", "song"]]
    },
    Date: {
      sort: ["Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"],
      tree: [["Date", "plain"], ["Album", "album"], ["Title", "song"]]
    },
    Composer: {
      sort: [
        "Composer", "Date", "Album", "DiscNumber", "TrackNumber", "Title",
        "file"
      ],
      tree: [["Composer", "plain"], ["Album", "album"], ["Title", "song"]]
    },
    Performer: {
      sort: [
        "Performer", "Date", "Album", "DiscNumber", "TrackNumber", "Title",
        "file"
      ],
      tree: [["Performer", "plain"], ["Album", "album"], ["Title", "song"]]
    }
  });
  let focus = {};
  let child = null;
  let list_cache = {};
  const listener = {changed: [], update: []};
  pub.addEventListener = function(ev, func) { listener[ev].push(func); };
  pub.removeEventListener = function(ev, func) {
    for (let i = 0, imax = listener[ev].length; i < imax; i++) {
      if (listener[ev][i] === func) {
        listener[ev].splice(i, 1);
        return;
      }
    }
  };
  const raiseEvent = function(ev) {
    if (!(ev in listener)) {
      return;
    }
    for (const f of listener[ev]) {
      f();
    }
  };
  const mkmemo = function(key) {
    const ret = [];
    for (const leef of pub.TREE[key].tree) {
      ret.push(leef[0]);
    }
    return ret;
  };
  const list_child_cache = [{}, {}, {}, {}, {}, {}];
  const list_child = function() {
    const root = pub.rootname();
    if (library[root].length === 0) {
      library[root] =
          vv.songs.sort(vv.storage.library, pub.TREE[root].sort, mkmemo(root));
    }
    const filters = {};
    for (let i = 0, imax = vv.storage.tree.length; i < imax; i++) {
      if (i === 0) {
        continue;
      }
      filters[vv.storage.tree[i][0]] = vv.storage.tree[i][1];
    }
    const ret = {};
    ret.key = pub.TREE[root].tree[vv.storage.tree.length - 1][0];
    ret.songs = library[root];
    ret.songs = vv.songs.filter(ret.songs, filters);
    ret.songs = vv.songs.uniq(ret.songs, ret.key);
    ret.style = pub.TREE[root].tree[vv.storage.tree.length - 1][1];
    ret.isdir = vv.storage.tree.length !== pub.TREE[root].tree.length;
    return ret;
  };
  const list_root = function() {
    const ret = [];
    for (const key in pub.TREE) {
      if (pub.TREE.hasOwnProperty(key)) {
        ret.push({root: [key]});
      }
    }
    return {key: "root", songs: ret, style: "plain", isdir: true};
  };
  const update_list = function() {
    if (pub.rootname() === "root") {
      list_cache = list_root();
      return true;
    }
    const cache = list_child_cache[vv.storage.tree.length - 1];
    const pwd = vv.storage.tree.join();
    if (cache.pwd === pwd) {
      list_cache = cache.data;
      return false;
    }
    list_cache = list_child();
    if (list_cache.songs.length === 0) {
      pub.up();
    } else {
      list_child_cache[vv.storage.tree.length - 1].pwd = pwd;
      list_child_cache[vv.storage.tree.length - 1].data = list_cache;
    }
    return true;
  };
  const updateData = function(data) {
    for (let i = 0, imax = list_child_cache.length; i < imax; i++) {
      list_child_cache[i] = {};
    }
    for (const key in pub.TREE) {
      if (pub.TREE.hasOwnProperty(key)) {
        if (key === vv.storage.root) {
          library[key] = vv.songs.sort(data, pub.TREE[key].sort, mkmemo(key));
        } else {
          library[key] = [];
        }
      }
    }
  };

  pub.update = function(data) {
    updateData(data);
    update_list();
    raiseEvent("update");
  };

  pub.rootname = function() {
    let r = "root";
    if (vv.storage.tree.length !== 0) {
      r = vv.storage.tree[0][1];
    }
    if (r !== vv.storage.root) {
      vv.storage.root = r;
      vv.storage.save.root();
    }
    return r;
  };
  pub.filters = function(pos) {
    return library[pub.rootname()][pos].keys;
  };
  pub.focused = function() { return [focus, child]; };
  pub.sortkeys = function() {
    const r = pub.rootname();
    if (r === "root") {
      return [];
    }
    return pub.TREE[r].sort;
  };
  pub.up = function() {
    const songs = pub.list().songs;
    if (songs[0]) {
      focus = songs[0];
      if (pub.rootname() === "root") {
        child = null;
      } else {
        child = vv.storage.tree[vv.storage.tree.length - 1][1];
      }
    }
    if (pub.rootname() !== "root") {
      vv.storage.tree.pop();
    }
    update_list();
    if (pub.list().songs.length === 1 && vv.storage.tree.length !== 0) {
      pub.up();
    } else {
      raiseEvent("changed");
    }
  };
  pub.TREE = pub.TREE;
  pub.down = function(value) {
    const r = pub.rootname();
    let key = "root";
    if (r !== "root") {
      key = pub.TREE[r].tree[vv.storage.tree.length - 1][0];
    }
    vv.storage.tree.push([key, value]);
    focus = {};
    child = null;
    update_list();
    const songs = pub.list().songs;
    if (songs.length === 1 &&
        pub.TREE[r].tree.length !== vv.storage.tree.length) {
      pub.down(vv.song.get(songs[0], pub.list().key));
    } else {
      raiseEvent("changed");
    }
  };
  pub.absaddr = function(first, second) {
    vv.storage.tree.splice(0, vv.storage.tree.length);
    vv.storage.tree.push(["root", first]);
    pub.down(second);
  };
  const absFallback = function(song) {
    if (pub.rootname() !== "root" && song.file) {
      const r = vv.storage.tree[0];
      vv.storage.tree.length = 0;
      vv.storage.tree.splice(0, vv.storage.tree.length);
      vv.storage.tree.push(r);
      const root = vv.storage.tree[0][1];
      const selected = pub.TREE[root].tree;
      for (let i = 0, imax = selected.length; i < imax; i++) {
        if (i === selected.length - 1) {
          break;
        }
        const key = selected[i][0];
        vv.storage.tree.push([key, vv.song.getOne(song, key)]);
      }
      update_list();
      const songs = pub.list().songs;
      for (const candidate of songs) {
        if (candidate.file && candidate.file[0] === song.file[0]) {
          focus = candidate;
          child = null;
          break;
        }
      }
    } else {
      vv.storage.tree.splice(0, vv.storage.tree.length);
      update_list();
    }
    raiseEvent("changed");
  };
  const absSorted = function(song) {
    let root = "";
    const pos = parseInt(song.Pos[0], 10);
    const keys = vv.storage.sorted.keys.join();
    for (const key in pub.TREE) {
      if (pub.TREE.hasOwnProperty(key)) {
        if (pub.TREE[key].sort.join() === keys) {
          root = key;
          break;
        }
      }
    }
    if (!root) {
      vv.view.popup.show("fixme", "modal: unknown sort keys: " + keys);
      return;
    }
    let songs = library[root];
    if (!songs || songs.length === 0) {
      library[root] =
          vv.songs.sort(vv.storage.library, pub.TREE[root].sort, mkmemo(root));
      songs = library[root];
      if (songs.length === 0) {
        return;
      }
    }
    if (songs.length > vv.consts.playlistLength) {
      songs = vv.songs.weakFilter(
          songs, vv.storage.sorted.filters, vv.consts.playlistLength);
    }
    if (!songs[pos]) {
      return;
    }
    if (songs[pos].file[0] === song.file[0]) {
      focus = songs[pos];
      child = null;
      vv.storage.tree.length = 0;
      vv.storage.tree.push(["root", root]);
      for (let i = 0; i < focus.keys.length - 1; i++) {
        vv.storage.tree.push(focus.keys[i]);
      }
      update_list();
      raiseEvent("changed");
    } else {
      absFallback(song);
    }
  };
  pub.abs = function(song) {
    if (vv.storage.sorted && vv.storage.sorted.sorted) {
      absSorted(song);
    } else {
      absFallback(song);
    }
  };
  pub.list = function() {
    if (!list_cache.songs || !list_cache.songs.length === 0) {
      update_list();
    }
    return list_cache;
  };
  pub.parent = function() {
    const root = pub.rootname();
    if (root === "root") {
      return;
    }
    const v = pub.list().songs;
    if (vv.storage.tree.length > 1) {
      const key = pub.TREE[root].tree[vv.storage.tree.length - 2][0];
      const style = pub.TREE[root].tree[vv.storage.tree.length - 2][1];
      return {key: key, song: v[0], style: style, isdir: true};
    }
    return {key: "top", song: {top: [root]}, style: "plain", isdir: true};
  };
  pub.grandparent = function() {
    const root = pub.rootname();
    if (root === "root") {
      return;
    }
    const v = pub.list().songs;
    if (vv.storage.tree.length > 2) {
      const key = pub.TREE[root].tree[vv.storage.tree.length - 3][0];
      const style = pub.TREE[root].tree[vv.storage.tree.length - 3][1];
      return {key: key, song: v[0], style: style, isdir: true};
    } else if (vv.storage.tree.length === 2) {
      return {key: "top", song: {top: [root]}, style: "plain", isdir: true};
    }
    return {
      key: "root",
      song: {root: ["Library"]},
      style: "plain",
      isdir: true
    };
  };
  if (vv.storage.loaded) {
    updateData(vv.storage.library);
  } else {
    vv.storage.addEventListener(
        "onload", function() { updateData(vv.storage.library); });
  }
  return pub;
})();
vv.control = (function() {
  const pub = {};
  const listener = {};
  pub.addEventListener = function(ev, func) {
    if (!(ev in listener)) {
      listener[ev] = [];
    }
    listener[ev].push(func);
  };
  pub.removeEventListener = function(ev, func) {
    for (let i = 0, imax = listener[ev].length; i < imax; i++) {
      if (listener[ev][i] === func) {
        listener[ev].splice(i, 1);
        return;
      }
    }
  };
  pub.raiseEvent = function(ev) {
    if (!(ev in listener)) {
      return;
    }
    for (const f of listener[ev]) {
      f();
    }
  };

  pub.swipe = function(element, f, resetFunc, leftElement) {
    element.swipe_target = f;
    let starttime = 0;
    let now = 0;
    let x = 0;
    let y = 0;
    let diff_x = 0;
    let diff_y = 0;
    let diff_x_l = 0;
    let diff_y_l = 0;
    let swipe = false;
    const start = function(e) {
      if (e.buttons && e.buttons !== 1) {
        return;
      }
      if (e.touches) {
        x = e.touches[0].screenX;
        y = e.touches[0].screenY;
      } else {
        x = e.screenX;
        y = e.screenY;
      }
      starttime = (new Date()).getTime();
      swipe = true;
    };
    const finalize = function(e) {
      starttime = 0;
      now = 0;
      x = 0;
      y = 0;
      diff_x = 0;
      diff_y = 0;
      diff_x_l = 0;
      diff_y_l = 0;
      swipe = false;
      e.currentTarget.classList.remove("swipe");
      e.currentTarget.classList.add("swiped");
      if (leftElement) {
        leftElement.classList.remove("swipe");
        leftElement.classList.add("swiped");
      }
      if (!resetFunc) {
        e.currentTarget.style.transform = "translate3d(0,0,0)";
      }
      setTimeout(function() {
        element.classList.remove("swiped");
        if (leftElement) {
          leftElement.classList.remove("swiped");
        }
      });
    };
    const cancel = function(e) {
      if (swipe) {
        finalize(e);
        if (resetFunc) {
          resetFunc();
        }
      }
    };
    const move = function(e) {
      if (e.buttons === 0 || (e.buttons && e.buttons !== 1)) {
        cancel(e);
        return;
      }
      if (!swipe) {
        cancel(e);
        return;
      }
      if (e.touches) {
        diff_x = x - e.touches[0].screenX;
        diff_y = y - e.touches[0].screenY;
      } else {
        diff_x = x - e.screenX;
        diff_y = y - e.screenY;
      }
      now = (new Date()).getTime();
      diff_x_l = diff_x > 0 ? diff_x : diff_x * -1;
      diff_y_l = diff_y > 0 ? diff_y : diff_y * -1;
      if (now - starttime < 200 && diff_y_l > diff_x_l) {
        cancel(e);
      } else if (diff_x_l > 3) {
        e.currentTarget.classList.add("swipe");
        e.currentTarget.style.transform =
            "translate3d(" + diff_x * -1 + "px,0,0)";
        if (leftElement) {
          leftElement.classList.add("swipe");
          leftElement.style.transform = "translate3d(" +
              (diff_x * -1 - e.currentTarget.offsetWidth) + "px,0,0)";
        }
      }
    };
    const end = function(e) {
      if (e.buttons && e.buttons !== 1) {
        cancel(e);
        return;
      }
      if (!swipe) {
        cancel(e);
        return;
      }
      const p = e.currentTarget.clientWidth / diff_x;
      if (p > -4 && p < 0) {
        finalize(e);
        f(e);
      } else if (now - starttime < 200 && diff_y_l < diff_x_l && diff_x < 0) {
        finalize(e);
        f(e);
      } else {
        cancel(e);
      }
    };
    if ("ontouchend" in element) {
      element.addEventListener("touchstart", start, {passive: true});
      element.addEventListener("touchmove", move, {passive: true});
      element.addEventListener("touchend", end, {passive: true});
    } else {
      element.addEventListener("mousedown", start, {passive: true});
      element.addEventListener("mousemove", move, {passive: true});
      element.addEventListener("mouseup", end, {passive: true});
    }
  };

  pub.click = function(element, f) {
    element.click_target = f;
    const enter = function(e) { e.currentTarget.classList.add("hover"); };
    const leave = function(e) { e.currentTarget.classList.remove("hover"); };
    const start = function(e) {
      if (e.buttons && e.buttons !== 1) {
        return;
      }
      if (e.touches) {
        e.currentTarget.x = e.touches[0].screenX;
        e.currentTarget.y = e.touches[0].screenY;
      } else {
        e.currentTarget.x = e.screenX;
        e.currentTarget.y = e.screenY;
      }
      e.currentTarget.touch = true;
      e.currentTarget.classList.add("active");
    };
    const move = function(e) {
      if (e.buttons && e.buttons !== 1) {
        return;
      }
      if (!e.currentTarget.touch) {
        return;
      }
      let change = false;
      let diff;
      if (e.touches) {
        diff = e.currentTarget.x - e.touches[0].screenX;
        change = diff < -5 || diff > 5;
        if (!change) {
          diff = e.currentTarget.y - e.touches[0].screenY;
          change = diff < -5 || diff > 5;
        }
      } else {
        diff = e.currentTarget.x - e.screenX;
        change = diff < -5 || diff > 5;
        if (!change) {
          diff = e.currentTarget.y - e.screenY;
          change = diff < -5 || diff > 5;
        }
      }
      if (change) {
        e.currentTarget.touch = false;
        e.currentTarget.classList.remove("active");
      }
    };
    const end = function(e) {
      if (e.buttons && e.buttons !== 1) {
        return;
      }
      e.currentTarget.classList.remove("active");
      if (e.currentTarget.touch) {
        f(e);
      }
    };
    if ("ontouchend" in element) {
      element.addEventListener("touchstart", start, {passive: true});
      element.addEventListener("touchmove", move, {passive: true});
      element.addEventListener("touchend", end, {passive: true});
    } else {
      element.addEventListener("mousedown", start, {passive: true});
      element.addEventListener("mousemove", move, {passive: true});
      element.addEventListener("mouseup", end, {passive: true});
      element.addEventListener("mouseenter", enter, {passive: true});
      element.addEventListener("mouseleave", leave, {passive: true});
    }
  };

  const requests = {};
  const abort_all_requests = function(options) {
    options = options || {};
    for (const key in requests) {
      if (requests.hasOwnProperty(key)) {
        if (options.stop) {
          requests[key].onabort = function() {};
        }
        requests[key].abort();
      }
    }
  };
  const get_request = function(path, ifmodified, callback, timeout) {
    const key = "GET " + path;
    if (requests[key]) {
      requests[key].onabort = function() {};  // disable retry
      requests[key].abort();
    }
    const xhr = new XMLHttpRequest();
    requests[key] = xhr;
    if (!timeout) {
      timeout = 1000;
    }
    xhr.responseType = "json";
    xhr.timeout = timeout;
    xhr.onload = function() {
      if (xhr.status === 200 || xhr.status === 304) {
        if (xhr.status === 200 && callback) {
          callback(
              xhr.response, xhr.getResponseHeader("Last-Modified"),
              xhr.getResponseHeader("Date"));
        }
        return;
      }
      // error handling
      if (xhr.status !== 0) {
        vv.view.popup.show("network", xhr.statusText);
      }
    };
    xhr.onabort = function() {
      if (timeout < 50000) {
        setTimeout(function() {
          get_request(path, ifmodified, callback, timeout * 2);
        });
      }
    };
    xhr.onerror = function() { vv.view.popup.show("network", "Error"); };
    xhr.ontimeout = function() {
      if (timeout < 50000) {
        vv.view.popup.show("network", "timeoutRetry");
        abort_all_requests();
        setTimeout(function() {
          get_request(path, ifmodified, callback, timeout * 2);
        });
      } else {
        vv.view.popup.show("network", "timeout");
      }
    };
    xhr.open("GET", path, true);
    xhr.setRequestHeader("If-Modified-Since", ifmodified);
    xhr.send();
  };

  const post_request = function(path, obj) {
    const key = "POST " + path;
    if (requests[key]) {
      requests[key].abort();
    }
    const xhr = new XMLHttpRequest();
    requests[key] = xhr;
    xhr.responseType = "json";
    xhr.timeout = 1000;
    xhr.onload = function() {
      if (xhr.status !== 200) {
        if (xhr.response && xhr.response.error) {
          vv.view.popup.show("network", xhr.response.error);
        } else {
          vv.view.popup.show("network", xhr.responseText);
        }
      }
    };
    xhr.ontimeout = function() {
      vv.view.popup.show("network", "timeout");
      abort_all_requests();
    };
    xhr.onerror = function() { vv.view.popup.show("network", "Error"); };
    xhr.open("POST", path, true);
    xhr.setRequestHeader("Content-Type", "application/json");
    xhr.send(JSON.stringify(obj));
  };

  const fetch = function(target, store) {
    get_request(
        target, vv.obj.getOrElse(vv.storage.last_modified, store, ""),
        function(ret, modified, date) {
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
            vv.storage[store] = ret.data;
            vv.storage.last_modified_ms[store] = Date.parse(modified) + diff;
            vv.storage.last_modified[store] = modified;
            if (store === "library") {
              vv.storage.save.library();
            } else if (store === "sorted") {
              vv.storage.save.sorted();
            }
            pub.raiseEvent(store);
          }
        });
  };

  pub.rescan_library = function() {
    post_request("/api/music/library", {action: "rescan"});
    vv.storage.control.update_library = true;
    pub.raiseEvent("control");
  };

  pub.prev = function() {
    post_request("/api/music/control", {state: "prev"});
  };

  pub.play_pause = function() {
    const state = vv.obj.getOrElse(vv.storage.control, "state", "stopped");
    const action = state === "play" ? "pause" : "play";
    post_request("/api/music/control", {state: action});
    vv.storage.control.state = action;
    pub.raiseEvent("control");
  };

  pub.next = function() {
    post_request("/api/music/control", {state: "next"});
  };

  pub.toggle_repeat = function() {
    if (vv.storage.control.single) {
      post_request("/api/music/control", {repeat: false, single: false});
      vv.storage.control.single = false;
      vv.storage.control.repeat = false;
    } else if (vv.storage.control.repeat) {
      post_request("/api/music/control", {single: true});
      vv.storage.control.single = true;
    } else {
      post_request("/api/music/control", {repeat: true});
      vv.storage.control.repeat = true;
    }
    pub.raiseEvent("control");
  };

  pub.toggle_random = function() {
    post_request("/api/music/control", {random: !vv.storage.control.random});
    vv.storage.control.random = !vv.storage.control.random;
    pub.raiseEvent("control");
  };

  pub.play = function(pos) {
    post_request("/api/music/songs/sort", {
      keys: vv.model.list.sortkeys(),
      filters: vv.model.list.filters(pos),
      play: pos
    });
  };

  pub.volume = function(num) {
    post_request("/api/music/control", {volume: num});
  };

  pub.output = function(id, on) {
    post_request("/api/music/outputs/" + id, {outputenabled: on});
  };

  const update_all = function() {
    fetch("/api/music/songs/sort", "sorted");
    fetch("/api/version", "version");
    fetch("/api/music/outputs", "outputs");
    fetch("/api/music/songs/current", "current");
    fetch("/api/music/control", "control");
    fetch("/api/music/library", "library");
  };

  let notify_last_update = (new Date()).getTime();
  let notify_last_connection = (new Date()).getTime();
  let connected = false;
  let notify_err_cnt = 0;
  let ws = null;
  const listennotify = function(cause) {
    abort_all_requests({stop: true});
    if (cause) {
      vv.view.popup.show("network", cause);
    }
    notify_last_connection = (new Date()).getTime();
    connected = false;
    let uri = "ws://" + location.host + "/api/music/notify";
    if (document.location.protocol === "https:") {
      uri = "wss://" + location.host + "/api/music/notify";
    }
    if (ws !== null) {
      ws.onclose = function() {};
      ws.close();
    }
    ws = new WebSocket(uri);
    ws.onopen = function() {
      if (notify_err_cnt > 0) {
        vv.view.popup.hide("network");
      }
      connected = true;
      notify_last_update = (new Date()).getTime();
      update_all();
    };
    ws.onmessage = function(e) {
      if (e && e.data) {
        if (e.data === "library") {
          fetch("/api/music/library", "library");
        } else if (e.data === "status") {
          fetch("/api/music/control", "control");
        } else if (e.data === "current") {
          fetch("/api/music/songs/current", "current");
        } else if (e.data === "outputs") {
          fetch("/api/music/outputs", "outputs");
        } else if (e.data === "stats") {
          fetch("/api/music/stats", "stats");
        } else if (e.data === "playlist") {
          fetch("/api/music/songs/sort", "sorted");
        }
        const new_notify_last_update = (new Date()).getTime();
        if (new_notify_last_update - notify_last_update > 10000) {
          // recover lost notification
          setTimeout(listennotify);
        }
        notify_last_update = new_notify_last_update;
        notify_err_cnt = 0;
      }
    };
    ws.onclose = function() {
      if (notify_err_cnt > 0) {
        vv.view.popup.show("network", "closed");
      }
      notify_last_update = (new Date()).getTime();
      notify_err_cnt++;
      setTimeout(listennotify, 1000);
    };
  };

  const init = function() {
    const polling = function() {
      const now = (new Date()).getTime();
      if (connected && now - 10000 > notify_last_update) {
        notify_err_cnt++;
        setTimeout(function() { listennotify("doesNotRespond"); });
      } else if (!connected && now - 2000 > notify_last_connection) {
        notify_err_cnt++;
        setTimeout(function() { listennotify("timeoutRetry"); });
      }
      pub.raiseEvent("poll");
      setTimeout(polling, 1000);
    };
    const start = function() {
      pub.raiseEvent("start");
      vv.view.list.show();
      pub.raiseEvent("current");
      listennotify();
      polling();
    };
    if (vv.storage.loaded) {
      start();
    } else {
      vv.storage.addEventListener("onload", start);
    }
  };

  pub.start = function() {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", init);
    } else {
      init();
    }
  };

  const focus = function() {
    vv.storage.save.current();
    if (vv.storage.preferences.playback.view_follow &&
        vv.storage.current !== null) {
      vv.model.list.abs(vv.storage.current);
    }
  };

  let unsorted = !vv.storage.sorted;
  const focusremove = function(key, remove) {
    const n = function() {
      if (unsorted && vv.storage.sorted && vv.storage.current !== null) {
        if (vv.storage.sorted && vv.storage.preferences.playback.view_follow) {
          vv.model.list.abs(vv.storage.current);
        }
        unsorted = false;
      }
      setTimeout(function() { remove(key, n); });
    };
    return n;
  };
  pub.addEventListener("current", focus);
  pub.addEventListener(
      "library", function() { vv.model.list.update(vv.storage.library); });
  if (unsorted) {
    pub.addEventListener(
        "current", focusremove("current", pub.removeEventListener));
    pub.addEventListener(
        "sorted", focusremove("sorted", pub.removeEventListener));
    vv.model.list.addEventListener(
        "update", focusremove("update", vv.model.list.removeEventListener));
  }

  return pub;
})();

// background
(function() {
  let color = 128;
  const update_theme = function() {
    if (color < vv.storage.preferences.appearance.color_threshold) {
      document.body.classList.add("dark");
      document.body.classList.remove("light");
    } else {
      document.body.classList.add("light");
      document.body.classList.remove("dark");
    }
  };
  const calc_color = function(path) {
    const img = new Image();
    img.onload = function() {
      const canvas = document.createElement("canvas");
      const context = canvas.getContext("2d");
      context.drawImage(img, 0, 0, 5, 5);
      try {
        const d = context.getImageData(0, 0, 5, 5).data;
        let newcolor = 0;
        for (const c of d) {
          newcolor += c;
        }
        color = newcolor / d.length;
        update_theme();
      } catch (e) {
        // failed to getImageData
      }
    };
    img.src = path;
  };
  const update = function() {
    const e = document.getElementById("background-image");
    if (vv.storage.preferences.appearance.background_image) {
      e.classList.remove("hide");
      document.getElementById("background-image").classList.remove("hide");
      let cover = "/assets/nocover.svg";
      let coverForCalc = "/assets/nocover.svg";
      if (vv.storage.current !== null && vv.storage.current.cover) {
        cover = "/music_directory/" + vv.storage.current.cover[0];
        const p = window.devicePixelRatio;
        const imgsize = parseInt(70 * p, 10);
        coverForCalc = "/api/images/music_directory/" +
            vv.storage.current.cover[0] + "?width=" + imgsize + "&height=" +
            imgsize;
      }
      const newimage = "url(\"" + cover + "\")";
      if (e.style.backgroundImage !== newimage) {
        calc_color(coverForCalc);
        e.style.backgroundImage = newimage;
      }
      e.style.filter = "blur(" +
          vv.storage.preferences.appearance.background_image_blur + "px)";
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
})();

vv.view.main = (function() {
  const pub = {};
  const load_volume_preferences = function() {
    const c = document.getElementById("control-volume");
    c.max = parseInt(vv.storage.preferences.volume.max, 10);
    if (vv.storage.preferences.volume.show) {
      c.classList.remove("hide");
    } else {
      c.classList.add("hide");
    }
  };
  vv.control.addEventListener("control", function() {
    const c = document.getElementById("control-volume");
    c.value = vv.storage.control.volume;
    if (vv.storage.control.volume < 0) {
      c.classList.add("disabled");
    } else {
      c.classList.remove("disabled");
    }
  });
  vv.control.addEventListener("preferences", load_volume_preferences);
  pub.show = function() {
    document.body.classList.add("view-main");
    document.body.classList.remove("view-list");
  };
  pub.hidden = function() {
    const e = document.body;
    if (window.matchMedia("(orientation: portrait)").matches) {
      return !e.classList.contains("view-main");
    }
    return !(
        e.classList.contains("view-list") || e.classList.contains("view-main"));
  };
  pub.update = function() {
    if (vv.storage.current === null) {
      return;
    }
    document.getElementById("main-box-title").textContent =
        vv.storage.current.Title;
    document.getElementById("main-box-artist").textContent =
        vv.storage.current.Artist;
    if (vv.storage.current.cover) {
      document.getElementById("main-cover-img").style.backgroundImage =
          "url(\"/music_directory/" + vv.storage.current.cover[0] + "\")";
    } else {
      document.getElementById("main-cover-img").style.backgroundImage = "";
    }
  };
  const update_style = function() {
    const e = document.getElementById("main-cover");
    if (vv.storage.preferences.appearance.circled_image) {
      e.classList.add("circled");
    } else {
      e.classList.remove("circled");
    }
    if (vv.storage.preferences.appearance.auto_hide_scrollbar) {
      document.body.classList.add("auto-hide-scrollbar");
    } else {
      document.body.classList.remove("auto-hide-scrollbar");
    }
  };
  vv.control.addEventListener("preferences", update_style);
  const update_elapsed = function() {
    if (vv.storage.current === null) {
      return;
    }
    if (pub.hidden() ||
        document.getElementById("main-cover-circle")
            .classList.contains("hide")) {
      return;
    }
    const c = document.getElementById("main-cover-circle-active");
    let elapsed = parseInt(vv.storage.control.song_elapsed * 1000, 10);
    if (vv.storage.control.state === "play") {
      elapsed += (new Date()).getTime() - vv.storage.last_modified_ms.control;
    }
    const total = parseInt(vv.storage.current.Time[0], 10);
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
              x + "," + y);
    } else {
      c.setAttribute("d", "M 100,10 L 100,10 A 90,90 0 0,1 " + x + "," + y);
    }
  };
  const init = function() {
    document.getElementById("control-volume")
        .addEventListener("change", function() {
          vv.control.volume(parseInt(this.value, 10));
        });
    document.getElementById("main-cover").addEventListener("click", function() {
      if (vv.storage.current !== null) {
        vv.view.modal.song.show(vv.storage.current);
      }
    });
    load_volume_preferences();
    update_style();
    vv.control.swipe(document.getElementById("main"), function() {
      if (vv.storage.current === null) {
        return;
      }
      vv.model.list.abs(vv.storage.current);
      vv.view.list.show();
    });
  };
  vv.control.addEventListener("current", pub.update);
  vv.control.addEventListener("poll", update_elapsed);
  vv.control.addEventListener("start", init);
  return pub;
})();
vv.view.list = (function() {
  const pub = {};
  pub.show = function() {
    document.body.classList.add("view-list");
    document.body.classList.remove("view-main");
  };
  pub.hidden = function() {
    const e = document.body;
    if (window.matchMedia("(orientation: portrait)").matches) {
      return !e.classList.contains("view-list");
    }
    return !(
        e.classList.contains("view-list") || e.classList.contains("view-main"));
  };
  const preferences_update = function() {
    const index = vv.storage.tree.length;
    const ul = document.getElementById("list-items" + index);
    if (vv.storage.preferences.appearance.gridview_album) {
      ul.classList.add("grid");
      ul.classList.remove("nogrid");
    } else {
      ul.classList.add("nogrid");
      ul.classList.remove("grid");
    }
  };
  const updatepos = function() {
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
  };

  const updateFocus = function() {
    const index = vv.storage.tree.length;
    const ul = document.getElementById("list-items" + index);
    let focus = null;
    let viewNowPlaying = false;
    const rootname = vv.model.list.rootname();
    const f = vv.model.list.focused();
    const focusSong = f[0];
    const focusParent = f[1];
    for (const listitem of ul.children) {
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
      if (vv.storage.sorted && vv.storage.sorted.sorted) {
        if (rootname === "root") {
          treeFocused = false;
        } else if (
            vv.storage.sorted.keys.join() !==
            vv.model.list.TREE[rootname].sort.join()) {
          treeFocused = false;
        }
      }
      const elapsed = listitem.getElementsByClassName("song-elapsed");
      const sep = listitem.getElementsByClassName("song-lengthseparator");
      if (treeFocused && elapsed.length !== 0 && vv.storage.current !== null &&
          vv.storage.current.file[0] === listitem.dataset.file) {
        viewNowPlaying = true;
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
      window.requestAnimationFrame(function() {
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
  };
  const clearAllLists = function() {
    const lists = document.getElementsByClassName("list");
    for (let treeindex = 0; treeindex < vv.storage.tree.length; treeindex++) {
      const oldul = lists[treeindex + 1].getElementsByClassName(
        "list-items")[0];
      while (oldul.lastChild) {
        oldul.removeChild(oldul.lastChild);
      }
      lists[treeindex + 1].dataset.pwd = "";
    }
  };
  const update = function() {
    const index = vv.storage.tree.length;
    const scroll = document.getElementById("list" + index);
    const pwd = vv.storage.tree.join();
    if (scroll.dataset.pwd === pwd) {
      updatepos();
      updateFocus();
      return;
    }
    scroll.dataset.pwd = pwd;
    const ls = vv.model.list.list();
    const key = ls.key;
    const songs = ls.songs;
    const isdir = ls.isdir;
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
    updatepos();
    const ul = document.getElementById("list-items" + index);
    while (ul.lastChild) {
      ul.removeChild(ul.lastChild);
    }
    ul.classList.remove("songlist");
    ul.classList.remove("albumlist");
    ul.classList.remove("plainlist");
    ul.classList.add(style + "list");
    preferences_update();
    for (let i = 0, imax = songs.length; i < imax; i++) {
      if (i === 0) {
        const p = vv.model.list.parent();
        if (p) {
          const li = vv.song.element(
            document.createElement("li"), p.song, p.key, p.style);
          li.classList.add("list-header");
          newul.appendChild(li);
        }
      }
      const li = vv.song.element(document.createElement("li"),
          songs[i], key, style, ul.classList.contains("grid"));
      li.classList.add("selectable");
      vv.control.click(li, function(e) {
        if (e.currentTarget.classList.contains("playing")) {
          if (vv.storage.current === null) {
            return;
          }
          vv.model.list.abs(vv.storage.current);
          vv.view.main.show();
          return;
        }
        const value = e.currentTarget.dataset.key;
        const pos = e.currentTarget.dataset.pos;
        if (isdir) {
          vv.model.list.down(value);
        } else {
          vv.control.play(parseInt(pos, 10));
        }
      }, false);
      newul.appendChild(li);
    }
    ul.appendChild(newul);
    updateFocus();
  };
  const updateForce = function() {
    clearAllLists();
    update();
  };
  const select_near_item = function() {
    const index = vv.storage.tree.length;
    const scroll = document.getElementById("list" + index);
    let updated = false;
    for (const selectable of document.getElementById(
      "list-items" + index).getElementsByClassName("selectable")) {
      const p = selectable.offsetTop;
      if (scroll.scrollTop < p && p < scroll.scrollTop + scroll.clientHeight &&
          !updated) {
        selectable.classList.add("selected");
        updated = true;
      } else {
        selectable.classList.remove("selected");
      }
    }
  };
  const select_focused_or = function(target) {
    const style = vv.model.list.list().style;
    const index = vv.storage.tree.length;
    const scroll = document.getElementById("list" + index);
    const l = document.getElementById("list-items" + index);
    let itemcount = parseInt(scroll.clientWidth / 160, 10);
    if (!vv.storage.preferences.appearance.gridview_album) {
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
        select_near_item();
        return;
      }
    }
    if (s.length === 0 && f.length === 0) {
      select_near_item();
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
  };
  pub.up = function() { select_focused_or("up"); };
  pub.left = function() { select_focused_or("left"); };
  pub.right = function() { select_focused_or("right"); };
  pub.down = function() { select_focused_or("down"); };

  pub.activate = function() {
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
  };

  vv.control.addEventListener("current", update);
  vv.control.addEventListener("preferences", preferences_update);
  vv.model.list.addEventListener("update", updateForce);
  vv.model.list.addEventListener("changed", update);
  vv.control.addEventListener("start", function() {
    vv.control.swipe(
        document.getElementById("list1"), vv.model.list.up, updatepos,
        document.getElementById("list0"));
    vv.control.swipe(
        document.getElementById("list2"), vv.model.list.up, updatepos,
        document.getElementById("list1"));
    vv.control.swipe(
        document.getElementById("list3"), vv.model.list.up, updatepos,
        document.getElementById("list2"));
    vv.control.swipe(
        document.getElementById("list4"), vv.model.list.up, updatepos,
        document.getElementById("list3"));
    vv.control.swipe(
        document.getElementById("list5"), vv.model.list.up, updatepos,
        document.getElementById("list4"));
  });
  return pub;
})();
vv.view.system = (function() {
  const pub = {};
  /* const preferences = */ (function() {
    const update_animation = function() {
      if (vv.storage.preferences.appearance.animation) {
        document.body.classList.add("animation");
      } else {
        document.body.classList.remove("animation");
      }
    };
    const initconfig = function(id) {
      const obj = document.getElementById(id);
      const s = id.indexOf("-");
      const mainkey = id.slice(0, s);
      const subkey = id.slice(s + 1).replace(/-/g, "_");
      let getter = null;
      if (obj.type === "checkbox") {
        obj.checked = vv.storage.preferences[mainkey][subkey];
        getter = function() { return obj.checked; };
      } else if (obj.tagName.toLowerCase() === "select") {
        obj.value = String(vv.storage.preferences[mainkey][subkey]);
        getter = function() { return obj.value; };
      } else if (obj.type === "range") {
        obj.value = String(vv.storage.preferences[mainkey][subkey]);
        getter = function() { return parseInt(obj.value, 10); };
        obj.addEventListener("input", function() {
          vv.storage.preferences[mainkey][subkey] = obj.value;
          vv.control.raiseEvent("preferences");
        });
      }
      obj.addEventListener("change", function() {
        vv.storage.preferences[mainkey][subkey] = getter();
        vv.storage.save.preferences();
        vv.control.raiseEvent("preferences");
      });
    };
    const update_devices = function() {
      const ul = document.getElementById("devices");
      while (ul.lastChild) {
        ul.removeChild(ul.lastChild);
      }
      const newul = document.createDocumentFragment();
      for (const o of vv.storage.outputs) {
        const li = document.createElement("li");
        li.classList.add("note-line");
        li.classList.add("system-setting");
        const desc = document.createElement("div");
        desc.classList.add("system-setting-desc");
        desc.textContent = o.outputname;
        const ch = document.createElement("input");
        ch.classList.add("slideswitch");
        ch.setAttribute("aria-label", o.outputname);
        ch.setAttribute("type", "checkbox");
        ch.setAttribute("id", "device_" + o.outputname);
        ch.setAttribute("deviceid", o.outputid);
        ch.checked = o.outputenabled === "1";
        ch.addEventListener("change", function() {
          vv.control.output(
              parseInt(this.getAttribute("deviceid"), 10), this.checked);
        });
        li.appendChild(desc);
        li.appendChild(ch);
        newul.appendChild(li);
      }
      ul.appendChild(newul);
    };
    vv.control.addEventListener("outputs", update_devices);
    vv.control.addEventListener("control", function() {
      const e = document.getElementById("library-rescan");
      if (vv.storage.control.update_library && !e.disabled) {
        e.disabled = true;
      } else if (!vv.storage.control.update_library && e.disabled) {
        e.disabled = false;
      }
    });
    vv.control.addEventListener("start", function() {
      vv.control.addEventListener("preferences", update_animation);
      update_animation();

      // Mobile
      if (navigator.userAgent.indexOf("Mobile") > 1) {
        document.getElementById("config-appearance-auto-hide-scrollbar")
            .classList.add("hide");
      }

      vv.control.addEventListener("control", function() {
        if (vv.storage.control.volume < 0) {
          document.getElementById("volume-header").classList.add("hide");
          document.getElementById("volume-all").classList.add("hide");
        } else {
          document.getElementById("volume-header").classList.remove("hide");
          document.getElementById("volume-all").classList.remove("hide");
        }
      });

      initconfig("appearance-color-threshold");
      initconfig("appearance-animation");
      initconfig("appearance-background-image");
      initconfig("appearance-background-image-blur");
      initconfig("appearance-circled-image");
      initconfig("appearance-gridview-album");
      initconfig("appearance-auto-hide-scrollbar");
      initconfig("playback-view-follow");
      initconfig("volume-show");
      initconfig("volume-max");
      document.getElementById("system-reload")
          .addEventListener("click", function() { location.reload(); });
      document.getElementById("library-rescan")
          .addEventListener(
              "click", function() { vv.control.rescan_library(); });
    });
  })();
  const stats = (function() {
    const pub = {};
    const zfill2 = function(i) {
      if (i < 100) {
        return ("00" + i).slice(-2);
      }
      return i;
    };
    const strtimedelta = function(i) {
      const uh = parseInt(i / (60 * 60), 10);
      const um = parseInt((i - uh * 60 * 60) / 60, 10);
      const us = parseInt(i - uh * 60 * 60 - um * 60, 10);
      return zfill2(uh) + ":" + zfill2(um) + ":" + zfill2(us);
    };

    const update_stats = function() {
      document.getElementById("stat-albums").textContent =
          vv.storage.stats.albums;
      document.getElementById("stat-artists").textContent =
          vv.storage.stats.artists;
      document.getElementById("stat-db-playtime").textContent =
          strtimedelta(parseInt(vv.storage.stats.db_playtime, 10));
      document.getElementById("stat-playtime").textContent =
          strtimedelta(parseInt(vv.storage.stats.playtime, 10));
      document.getElementById("stat-tracks").textContent =
          vv.storage.stats.songs;
      const db_update = new Date(
        parseInt(vv.storage.stats.db_update, 10) * 1000);
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
      document.getElementById("stat-websockets").textContent =
          vv.storage.stats.subscribers;
    };
    const update_time = function() {
      const diff = parseInt(
          ((new Date()).getTime() - vv.storage.last_modified_ms.stats) / 1000,
          10);
      const uptime = parseInt(vv.storage.stats.uptime, 10) + diff;
      if (vv.storage.control.state === "play") {
        const playtime = parseInt(vv.storage.stats.playtime, 10) + diff;
        document.getElementById("stat-playtime").textContent =
            strtimedelta(playtime);
      }
      document.getElementById("stat-uptime").textContent = strtimedelta(uptime);
    };
    vv.control.addEventListener("poll", function() {
      if (document.getElementById("system-stats").classList.contains("on")) {
        update_time();
      }
    });
    vv.control.addEventListener("stats", function() {
      if (document.getElementById("system-stats").classList.contains("on")) {
        update_stats();
      }
    });
    pub.update = function() {
      update_stats();
      update_time();
    };
    return pub;
  })();
  /* const info = */ (function() {
    vv.control.addEventListener("version", function() {
      if (vv.storage.version.vv) {
        document.getElementById("version").textContent = vv.storage.version.vv;
        document.getElementById("go-version").textContent =
            vv.storage.version.go;
      }
    });
    vv.control.addEventListener("start", function() {
      document.getElementById("user-agent").textContent = navigator.userAgent;
    });
  })();
  vv.control.addEventListener("start", function() {
    const navs = document.getElementsByClassName("system-nav-item");
    const showChild = function(e) {
      for (const nav of navs) {
        if (nav === e.currentTarget) {
          if (nav.id === "system-nav-stats") {
            stats.update();
          }
          nav.classList.add("on");
          document.getElementById(nav.dataset.target).classList.add("on");
        } else {
          nav.classList.remove("on");
          document.getElementById(nav.dataset.target)
              .classList.remove("on");
        }
      }
    };
    for (const nav of navs) {
      nav.addEventListener("click", showChild);
    }
    document.getElementById("modal-system-close")
        .addEventListener("click", vv.view.modal.hide);
  });
  pub.show = function() {
    document.getElementById("modal-background").classList.remove("hide");
    document.getElementById("modal-outer").classList.remove("hide");
    document.getElementById("modal-system").classList.remove("hide");
  };
  return pub;
})();

// header
(function() {
  const update = function() {
    const e = document.getElementById("header-back-label");
    const b = document.getElementById("header-back");
    const m = document.getElementById("header-main");
    if (vv.model.list.rootname() === "root") {
      b.classList.add("root");
      m.classList.add("root");
    } else {
      b.classList.remove("root");
      m.classList.remove("root");
      const songs = vv.model.list.list().songs;
      if (songs[0]) {
        const p = vv.model.list.grandparent();
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
  vv.control.addEventListener("start", function() {
    document.getElementById("header-back")
        .addEventListener("click", function(e) {
          if (vv.view.list.hidden()) {
            if (vv.storage.current !== null) {
              vv.model.list.abs(vv.storage.current);
            }
          } else {
            vv.model.list.up();
          }
          vv.view.list.show();
          e.stopPropagation();
        });
    document.getElementById("header-main")
        .addEventListener("click", function(e) {
          e.stopPropagation();
          if (vv.storage.current !== null) {
            vv.model.list.abs(vv.storage.current);
          }
          vv.view.main.show();
          e.stopPropagation();
        });
    document.getElementById("header-system")
        .addEventListener("click", function(e) {
          vv.view.system.show();
          e.stopPropagation();
        });
    update();
    vv.model.list.addEventListener("changed", update);
    vv.model.list.addEventListener("update", update);
  });
})();

// footer
(function() {
  vv.control.addEventListener("start", function() {
    document.getElementById("control-prev")
        .addEventListener("click", function(e) {
          vv.control.prev();
          e.stopPropagation();
        });
    document.getElementById("control-toggleplay")
        .addEventListener("click", function(e) {
          vv.control.play_pause();
          e.stopPropagation();
        });
    document.getElementById("control-next")
        .addEventListener("click", function(e) {
          vv.control.next();
          e.stopPropagation();
        });
    document.getElementById("control-repeat")
        .addEventListener("click", function(e) {
          vv.control.toggle_repeat();
          e.stopPropagation();
        });
    document.getElementById("control-random")
        .addEventListener("click", function(e) {
          vv.control.toggle_random();
          e.stopPropagation();
        });
  });
  vv.control.addEventListener("control", function() {
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
  });
})();

vv.view.popup = (function() {
  const pub = {};
  pub.show = function(target, description) {
    const obj = document.getElementById("popup-" + target);
    if (!obj) {
      vv.view.popup.show("fixme", "popup-" + target + " is not found in html");
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
    setTimeout(function() {
      if ((new Date()).getTime() - obj.ttl > 0) {
        obj.classList.remove("show");
        obj.classList.add("hide");
      }
    }, 5000);
  };
  pub.hide = function(target) {
    const obj = document.getElementById("popup-" + target);
    if (obj) {
      const now = (new Date()).getTime();
      if (now - obj.timestamp < 600) {
        obj.ttl = obj.timestamp + 500;
        setTimeout(function() {
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
  };
  return pub;
})();

// elapsed circle/time updater
(function() {
  const update = function() {
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
      [].forEach.call(document.getElementsByClassName("elapsed"), function(x) {
        if (x.textContent !== label) {
          x.textContent = label;
        }
      });
    }
  };
  vv.control.addEventListener("control", update);
  vv.control.addEventListener("poll", update);
})();

(function() {
  const pub = {};
  pub.hide = function() {
    document.getElementById("modal-background").classList.add("hide");
    document.getElementById("modal-outer").classList.add("hide");
    const ws = document.getElementsByClassName("modal-window");
    for (const w of ws) {
      w.classList.add("hide");
    }
  };
  vv.control.addEventListener("start", function() {
    document.getElementById("modal-background")
        .addEventListener("click", pub.hide);
    document.getElementById("modal-outer").addEventListener("click", pub.hide);
    const ws = document.getElementsByClassName("modal-window");
    for (const w of ws) {
      w.addEventListener("click", function(e) { e.stopPropagation(); });
    }
  });
  vv.view.modal.hide = pub.hide;
})();
vv.view.modal.help = (function() {
  const pub = {};
  pub.show = function() {
    const b = document.getElementById("modal-background");
    if (!b.classList.contains("hide")) {
      return;
    }
    b.classList.remove("hide");
    document.getElementById("modal-outer").classList.remove("hide");
    document.getElementById("modal-help").classList.remove("hide");
  };
  pub.hide = function() {
    document.getElementById("modal-background").classList.add("hide");
    document.getElementById("modal-outer").classList.add("hide");
    document.getElementById("modal-help").classList.add("hide");
  };
  vv.control.addEventListener("start", function() {
    document.getElementById("modal-help-close")
        .addEventListener("click", pub.hide);
  });
  return pub;
})();
vv.view.modal.song = (function() {
  const pub = {};
  pub.show = function(song) {
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
        const root = vv.model.list.TREE[key];
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
                obj.addEventListener("click", function(e) {
                  const d = e.currentTarget.dataset;
                  vv.model.list.absaddr(d.root, d.value);
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
      cover.src = "/api/images/music_directory/" + song.cover + "?width=" +
          imgsize + "&height=" + imgsize;
    } else {
      cover.src = "/assets/nocover.svg";
    }
    document.getElementById("modal-background").classList.remove("hide");
    document.getElementById("modal-outer").classList.remove("hide");
    document.getElementById("modal-song").classList.remove("hide");
  };
  pub.hide = function() {
    document.getElementById("modal-background").classList.add("hide");
    document.getElementById("modal-outer").classList.add("hide");
    document.getElementById("modal-song").classList.add("hide");
  };
  vv.control.addEventListener("start", function() {
    document.getElementById("modal-song-close")
        .addEventListener("click", pub.hide);
  });
  return pub;
})();

// keyboard events
(function() {
  const stopAndPrevent = function(e) {
    e.stopPropagation();
    e.preventDefault();
  };
  vv.control.addEventListener("start", function() {
    document.addEventListener("keydown", function(e) {
      if (!document.getElementById("modal-background")
               .classList.contains("hide")) {
        if (e.key === "Escape" || e.key === "Esc") {
          vv.view.modal.hide();
        }
        return;
      }
      let mod = 0;
      mod |= e.shiftKey << 3;
      mod |= e.altKey << 2;
      mod |= e.ctrlKey << 1;
      mod |= e.metaKey;
      if (mod === 0 && (e.key === " " || e.key === "Spacebar")) {
        vv.control.play_pause();
        stopAndPrevent(e);
      } else if (mod === 10 && e.keyCode === 37) {
        vv.control.prev();
        stopAndPrevent(e);
      } else if (mod === 10 && e.keyCode === 39) {
        vv.control.next();
        stopAndPrevent(e);
      } else if (mod === 0 && e.keyCode === 13) {
        if (!vv.view.list.hidden() && vv.view.list.activate()) {
          stopAndPrevent(e);
        }
      } else if (
          (mod === 0 && e.keyCode === 8) || (mod === 1 && e.keyCode === 37)) {
        if (vv.view.list.hidden()) {
          if (vv.storage.current !== null) {
            vv.model.list.abs(vv.storage.current);
          }
        } else {
          vv.model.list.up();
        }
        vv.view.list.show();
        stopAndPrevent(e);
      } else if (mod === 0 && e.keyCode === 37) {
        if (!vv.view.list.hidden()) {
          vv.view.list.left();
          stopAndPrevent(e);
        }
      } else if (mod === 0 && e.keyCode === 38) {
        if (!vv.view.list.hidden()) {
          vv.view.list.up();
          stopAndPrevent(e);
        }
      } else if (mod === 1 && e.keyCode === 39) {
        if (vv.model.list.rootname() !== "root") {
          if (vv.storage.current !== null) {
            vv.model.list.abs(vv.storage.current);
          }
        }
        vv.view.main.show();
        stopAndPrevent(e);
      } else if (mod === 0 && e.keyCode === 39) {
        if (!vv.view.list.hidden()) {
          vv.view.list.right();
          stopAndPrevent(e);
        }
      } else if (mod === 0 && e.keyCode === 40) {
        if (!vv.view.list.hidden()) {
          vv.view.list.down();
          stopAndPrevent(e);
        }
      } else if ((mod & 7) === 0 && e.key === "?") {
        vv.view.modal.help.show();
        stopAndPrevent(e);
      }
    });
  });
})();

vv.control.start();
