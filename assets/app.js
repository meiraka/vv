var vv = vv || {
    obj: {},
    song: {},
    songs: {},
    storage: {},
    model: {list: {}},
    view: {header: {}, background: {}, main: {}, list: {}, system: {}, footer: {}, elapsed: {}, modal: {help: {}}},
    control : {},
};
vv.obj = (function(){
    function getOrElse(m, k, v) {
        return k in m? m[k] : v;
    }
    return {getOrElse: getOrElse};
})();
vv.song = (function(){
    var tag = function(song, keys, other) {
        var i;
        for (i in keys) {
            var key = keys[i];
            if (key in song) {
                return song[key];
            }
        }
        return other;
    }
    var getOrElse = function(song, key, other) {
       if (key in song) {
           return song[key];
       } else if (key == "AlbumSort") {
           return tag(song, ["Album"], other);
       } else if (key == "ArtistSort") {
           return tag(song, ["Artist"], other);
       } else if (key == "AlbumArtist") {
           return tag(song, ["Artist"], other);
       } else if (key == "AlbumArtistSort") {
           return tag(song, ["AlbumArtist", "Artist"], other);
       } else if (key == "AlbumSort") {
           return tag(song, ["Album"], other);
       } else {
           return other;
       }
    }
    var get = function(song, key) {
        return getOrElse(song, key, '[no ' + key + ']');
    }
    var str = function(song, keys) {
        var sortkey = '';
        var i;
        for (i in keys) {
            sortkey += getOrElse(song, keys[i], ' ')
        }
        return sortkey;
    }
    var element = function(e, song, key, style) {
        var inner = "";
        e.classList.remove("plain");
        e.classList.remove("song");
        e.classList.remove("album");
        e.classList.remove("playing");
        e.classList.add(style);
        e.setAttribute("key", vv.song.get(song, key));
        e.setAttribute("uri", song["file"]);
        if (style == "song") {
            var now_playing = vv.storage.current && vv.storage.current.file && song.file == vv.storage.current.file;
            if (now_playing) {
                e.classList.add("playing");
            }
            inner = "<span class=track>"+vv.song.get(song, "TrackNumber")+"</span>";
            if (now_playing) {
                inner += '<svg width="22" height="22" viewBox="0 0 100 100"><path class="fill" d="M 25,20 80,50 25,80 z"/></svg>';
            }
            inner += "<span class=title>"+vv.song.get(song, "Title")+"</span>";
            if (vv.song.get(song, "Artist") != vv.song.get(song, "AlbumArtist")) {
                inner += "<span class=artist>"+vv.song.get(song, "Artist")+"</span>";
            } else {
                inner += '<span class="artist low-prio">'+vv.song.get(song, "Artist")+"</span>";
            }
            if (now_playing) {
                inner += "<span class=elapsed></span>"+
                         "<span class=length_separator>/</span>";
            }
            inner += "<span class=length>"+vv.song.get(song, "Length")+"</span>";
        } else if (style == "album") {
            var cover = "/assets/nocover.svg";
            if (song.cover) {
                cover = "/music_directory/" + song.cover;
            }
            inner += '<div class=img-sq><img class=cover src="'+cover+'"></div>';
            inner += "<div class=detail>"
            inner += "<span class=date>"+vv.song.get(song, "Date")+"</span>";
            inner += "<span class=album>"+vv.song.get(song, "Album")+"</span>";
            inner += "<span class=albumartist>"+vv.song.get(song, "AlbumArtist")+"</span>";
            inner += "</div>"
        } else {
            inner = "<span class=key>"+vv.song.get(song, key)+"</span>";
        }
        e.innerHTML = inner;
        return e;
    };

    return {
        getOrElse: getOrElse,
        get: get,
        str: str,
        element: element,
    };
}());
vv.songs = (function(){
    var sort = function(songs, keys) {
        return songs.map(function(song) {
            return [song, vv.song.str(song, keys)]
        }).sort(function (a, b) {
            if (a[1] < b[1]) { return -1; } else { return 1; }
        }).map(function(s) { return s[0]; });
    }
    var uniq = function(songs, key) {
        return songs.filter(function (song, i , self) {
            if (i == 0) {
                return true;
            } else if (vv.song.getOrElse(song, key, ' ') != vv.song.getOrElse(self[i - 1], key, ' ')) {
                return true;
            } else {
                return false;
            }
        });
    }
    var filter = function(songs, filters) {
        return songs.filter(function(song) {
            var f;
            for (f in filters) {
                if (vv.song.get(song, f) != filters[f]) {
                    return false;
                }
            }
            return true;
        });
    }
    return {
        sort: sort,
        uniq: uniq,
        filter: filter,
    };
}());
vv.storage = (function(){
    var tree = [];
    var current = [];
    var current_last_modified = "";
    var control = [];
    var control_last_modified = "";
    var library = {
        "AlbumArtist": [],
        "Album": [],
        "Genre": [],
        "Date": [],
    }
    var library_last_modified = "";
    var outputs = [];
    var outputs_last_modified = "";
    var stats = {};
    var preferences = {
        "volume": {"show": true, "max": 100}, "playback": {"view_follow": true},
        "appearance": {"dark": false, "background_image": true, "background_image_blur": 32, "circled_image": true},
    };
    // Presto Opera
    if (navigator.userAgent.indexOf("Presto/2") > 1) {
        preferences.appearance.background_image_blur = 0;
        preferences.appearance.circled_image = false;
    }
    var save = function() {
        try {
            localStorage.tree = JSON.stringify(data.tree);
            localStorage.preferences = JSON.stringify(data.preferences);
            localStorage.last_state = data.last_state;
        } catch (e) {
            // private browsing
        }
    }
    var load = function() {
        try {
            if (localStorage.tree) {
                data.tree = JSON.parse(localStorage.tree);
            }
            if (localStorage.preferences) {
                var c = JSON.parse(localStorage.preferences);
                var i, j;
                for (i in c) {
                    for (j in c[i]) {
                        data.preferences[i][j] = c[i][j];
                    }
                }
            }
            if (localStorage.last_state) {
                data.last_state = localStorage.last_state;
            }
        } catch (e) {
            // private browsing
        }
    }
    var data = {
        tree: tree,
        current: current,
        current_last_modified: current_last_modified,
        control: control,
        control_last_modified: control_last_modified,
        library: library,
        library_last_modified: library_last_modified,
        outputs: outputs,
        outputs_last_modified: outputs_last_modified,
        preferences: preferences,
        stats: stats,
        version: {},
        save: save,
        load: load,
        last_state: "main",
    }
    load();
    return data;
}());

vv.model.list = (function() {
    var TREE = {
        "AlbumArtist": {
            "sort":
                ["AlbumArtist", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"],
            "tree":
                [["AlbumArtist", "plain"],
                 ["Album", "album"],
                 ["Title", "song"]
                ],
        },
        "Album": {
            "sort":
                ["AlbumArtist", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"],
            "tree":
                [["Album", "album"],
                 ["Title", "song"]
                ],
        },
        "Genre": {
            "sort":
                ["Genre", "Album", "DiscNumber", "TrackNumber", "Title", "file"],
            "tree":
                [["Genre", "plain"],
                 ["Album", "album"],
                 ["Title", "song"],
                ]
        },
        "Date": {
            "sort":
                ["Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"],
            "tree":
                [["Date", "plain"],
                 ["Album", "album"],
                 ["Title", "song"],
                ]
        }
    }
    var focus = {};
    var list_cache = {};
    var listener = {"changed": []}
    var addEventListener = function(ev, func) {
        listener[ev].push(func);
    };
    var raiseEvent = function(ev) {
        var i;
        for (i in listener[ev]) {
            listener[ev][i]();
        }
    };
    var update = function(data) {
        var key;
        for (key in TREE) {
            vv.storage.library[key] = vv.songs.sort(data, TREE[key]["sort"]);
        }
        update_list();
    };
    var rootname = function() {
        var r = "root";
        if (vv.storage.tree.length != 0) {
            r = vv.storage.tree[0][1];
        }
        return r;
    };
    var focused = function() {
        return focus;
    };
    var sortkeys = function() {
        var r = rootname();
        if (r == "root") {
            return [];
        }
        return TREE[r]["sort"];
    }
    var up = function() {
        var songs = list().songs;
        if (songs[0]) { focus = songs[0]; }
        if (rootname() != "root") {
            vv.storage.tree.pop();
            vv.storage.save();
        }
        update_list();
        raiseEvent("changed");
    };
    var down = function(value) {
        var r = rootname();
        var key = "root";
        if (r != "root") {
            key = TREE[r]["tree"][vv.storage.tree.length - 1][0];
        }
        vv.storage.tree.push([key, value]);
        vv.storage.save();
        focus = {};
        update_list();
        raiseEvent("changed");
    };
    var abs = function(song) {
        focus = song;
        var i, root, key, selected;
        if (rootname() != "root" && song.file) {
            var r = vv.storage.tree[0];
            vv.storage.tree.length = 0;
            vv.storage.tree.splice(0, vv.storage.tree.length);
            vv.storage.tree.push(r);
            root = vv.storage.tree[0][1];
            selected = TREE[root]["tree"];
            for (i in selected) {
                if (i == selected.length - 1) {
                    break;
                }
                key = selected[i][0];
                vv.storage.tree.push([key, vv.song.get(song, key)]);
            }
            vv.storage.save();
        } else {
            vv.storage.tree.splice(0, vv.storage.tree.length);
            vv.storage.save();
        }
        update_list();
        raiseEvent("changed");
    };
    var list = function() {
        if (!list_cache.songs || !list_cache.songs.length == 0) {
            update_list();
        }
        return list_cache;
    }
    var update_list = function() {
        if (rootname() == "root") {
            list_cache = list_root();
        } else {
            list_cache = list_child();
        }
    };
    var list_child = function() {
        var root = rootname(),
            library = vv.storage.library[root],
            filters = {},
            key = TREE[root]["tree"][vv.storage.tree.length - 1][0],
            style = TREE[root]["tree"][vv.storage.tree.length - 1][1],
            isdir = true;
        if (vv.storage.tree.length == TREE[root]["tree"].length) {
            isdir = false;
        }
        var leef;
        for (leef in vv.storage.tree) {
            if (leef == 0) { continue; }
            filters[vv.storage.tree[leef][0]] = vv.storage.tree[leef][1];
        }
        library = vv.songs.filter(library, filters);
        library = vv.songs.uniq(library, key);
        return {"key": key, "songs": library, "style": style, "isdir": isdir}
    };
    var list_root = function() {
        var ret = [];
        var rootname = "";
        for (rootname in TREE) {
            ret.push({"root": rootname});
        }
        return {"key": "root", "songs": ret, "style": "plain", "isdir": true};
    };
    var parent = function() {
        var v = list().songs;
        var root = rootname();
        if (root == "root") {
            return;
        }
        if (vv.storage.tree.length > 1) {
            var key = TREE[root]["tree"][vv.storage.tree.length - 2][0];
            var style = TREE[root]["tree"][vv.storage.tree.length - 2][1];
            return {"key": key, "song": v[0], "style": style, "isdir": true};
        }
        return {"key": "top", "song": {"top": root}, "style": "plain", "isdir": true};
    };
    var grandparent = function() {
        var v = list().songs;
        var root = rootname();
        if (root == "root") {
            return;
        }
        if (vv.storage.tree.length > 2) {
            var key = TREE[root]["tree"][vv.storage.tree.length - 3][0];
            var style = TREE[root]["tree"][vv.storage.tree.length - 3][1];
            return {"key": key, "song": v[0], "style": style, "isdir": true};
        } else if (vv.storage.tree.length == 2) {
            return {"key": "top", "song": {"top": root}, "style": "plain", "isdir": true};
        } 
        return {"key": "root", "song": {"root": "Library"}, "style": "plain", "isdir": true};
    };
    return {
        addEventListener: addEventListener,
        focused: focused,
        update: update,
        rootname: rootname,
        sortkeys: sortkeys,
        parent: parent,
        grandparent: grandparent,
        up: up,
        down: down,
        abs: abs,
        list: list,
    };
}());
vv.control = (function() {
    var listener = {"control": [], "preferences": [], "library": [], "playlist": [],
                    "current": [], "outputs": [], "stats": [], "version": [], "start": [], "poll": []}
    var addEventListener = function(ev, func) {
        listener[ev].push(func);
    };
    var raiseEvent = function(ev) {
        var i;
        for (i in listener[ev]) {
            listener[ev][i]();
        }
    };

    var click = function(e, f) {
        if ("ontouchend" in e) {
            e.addEventListener("touchstart", function() { this.touch = true; });
            e.addEventListener("touchmove", function() { this.touch = false; });
            e.addEventListener("touchend", function(a) { if (this.touch) {f(a);} });
        } else {
            e.addEventListener("click", f);
        }
    };
    var err_hide = 0;
    var err_timeout = function(description) {
        return function() {
            var e = document.getElementById("error");
            err_hide = (new Date()).getTime();
            e.getElementsByClassName("title")[0].textContent = "timeout";
            e.getElementsByClassName("description")[0].textContent = description;
            setTimeout(function() {
                if (err_hide + 4000 < (new Date()).getTime()) {
                    e.classList.remove("show");
                    e.classList.add("hide");
                }
            }, 5000);
            e.classList.remove("hide");
            e.classList.add("show");
        }
    }

    var get_request = function(path, ifmodified, callback) {
        var xhr = new XMLHttpRequest();
        xhr.timeout = 1000;
        xhr.onreadystatechange = function() {
            if (xhr.readyState == 4) {
                if (xhr.status == 200 && callback) {
                    callback(JSON.parse(xhr.responseText), xhr.getResponseHeader("Last-Modified"));
                }
            }
        };
        xhr.ontimeout = err_timeout("GET "+path);
        xhr.open("GET", path, true);
        if (ifmodified != "") {
            xhr.setRequestHeader("If-Modified-Since", ifmodified);
        }
        xhr.send();
    }

    var post_request = function(path, obj, callback) {
        var xhr = new XMLHttpRequest();
        xhr.timeout = 1000;
        xhr.onreadystatechange = function() {
            if (xhr.readyState == 4) {
                if (xhr.status == 200 && callback) {
                    callback(JSON.parse(xhr.responseText));
                }
            }
        };
        xhr.ontimeout = err_timeout("POST "+path);
        xhr.open("POST", path, true);
        xhr.setRequestHeader("Content-Type", "application/json");
        xhr.send(JSON.stringify(obj));
    }
    var update_stats = function() {
        get_request("api/stats", "", function(ret) {
            if (!ret.error) {
                vv.storage.stats = ret.data;
                ret.data.last_modified_ms = (new Date()).getTime();
                raiseEvent("stats");
            }
        });
    }
    var update_version = function() {
        get_request("api/version", "", function(ret) {
            if (!ret.error) {
                vv.storage.version = ret.data;
                ret.data.last_modified_ms = (new Date()).getTime();
                raiseEvent("version");
            }
        });
    }

    var rescan_library = function() {
        post_request("api/library", {"action": "rescan"});
    }

    var update_song = function() {
        get_request("api/songs/current", vv.storage.current_last_modified, function(ret, modified) {
            if (!ret.error) {
                var song = ret.data? ret.data : {};
                vv.storage.current = song;
                vv.storage.current_last_modified = modified;
                if (vv.model.list.rootname() != "root" && vv.storage.preferences.playback.view_follow && song.file) {
                    vv.model.list.abs(song);
                }
                raiseEvent("current")
            }
        });
    };

    var update_status = function() {
        get_request("api/control", vv.storage.control_last_modified, function(ret, modified) {
            if (!ret.error) {
                vv.storage.control = ret["data"];
                vv.storage.control_last_modified = modified;
                raiseEvent("control");
            }
        });
    };

    var update_library = function() {
        get_request("api/library", vv.storage.library_last_modified, function(ret, modified) {
            if (!ret.error) {
                vv.model.list.update(ret["data"]);
                vv.storage.library_last_modified = modified;
                update_stats();
                raiseEvent("library");
            }
        });
    };

    var update_outputs = function() {
        get_request("api/outputs", vv.storage.outputs_last_modified, function(ret, modified) {
            if (!ret.error) {
                vv.storage.outputs = ret["data"];
                vv.storage.outputs_last_modified = modified;
                raiseEvent("outputs");
            }
        });
    };

    var prev = function() {
        post_request("api/control", {"state": "prev"})
    }

    var play_pause = function() {
        var state = vv.obj.getOrElse(vv.storage.control, "state", "stopped");
        var action = state == "play" ? "pause" : "play";
        post_request("api/control", {"state": action})
    }

    var next = function() {
        post_request("api/control", {"state": "next"})
    }

    var toggle_repeat = function() {
        post_request("api/control", {"repeat": !vv.storage.control["repeat"]})
    }

    var toggle_random = function() {
        post_request("api/control", {"random": !vv.storage.control["random"]})
    }

    var play = function(uri) {
        var xhr = new XMLHttpRequest();
        xhr.onreadystatechange = function() {};
        xhr.open("POST", "api/songs", true);
        xhr.setRequestHeader("Content-Type", "application/json");
        xhr.send(JSON.stringify(
            {"action": "sort",
             "keys": vv.model.list.sortkeys(),
             "uri": uri
            }
        ));
    }

    var volume = function(num) {
        post_request("/api/control", {"volume": num})
    }

    var output = function(id, on) {
        post_request("api/outputs/" + id, {"outputenabled": on})
    }

    var init = function() {
        var polling = function() {
            vv.control.update_version();
            vv.control.update_song();
            vv.control.update_status();
            vv.control.update_library();
            raiseEvent("poll");
            setTimeout(polling, 1000);
        }
        var show = {
            "main": vv.view.main.show,
            "list": vv.view.list.show,
            "system": vv.view.system.show
        };
        show[vv.storage.last_state]();
        raiseEvent("start");
        polling();
    };

    var start = function() {
        if (document.readyState !== 'loading') {
            init();
        } else {
            document.addEventListener('DOMContentLoaded', init);
        }
    };

    return {
        addEventListener: addEventListener,
        raiseEvent: raiseEvent,
        click: click,
        rescan_library: rescan_library,
        update_song: update_song,
        update_status: update_status,
        update_library: update_library,
        update_outputs: update_outputs,
        update_version: update_version,
        prev: prev,
        play_pause: play_pause,
        next: next,
        play: play,
        toggle_repeat: toggle_repeat,
        toggle_random: toggle_random,
        volume: volume,
        output: output,
        start: start,
    };
}());

vv.view.background = (function() {
    var update = function() {
        var e = document.getElementById("background-image");
        if (vv.storage.preferences.appearance.background_image) {
            e.classList.remove("hide");
            document.getElementById("background").classList.remove("hide");
            var cover = "/assets/nocover.svg";
            if (vv.storage.current.cover) {
                cover = "/music_directory/" + vv.storage.current.cover;
            }
            e.style.backgroundImage = 'url("'+cover+'")';
            e.style.filter = "blur(" + vv.storage.preferences.appearance.background_image_blur + "px)";
        } else {
            e.classList.add("hide");
            document.getElementById("background").classList.add("hide");
        }
    };
    vv.control.addEventListener("current", update);
    vv.control.addEventListener("preferences", update);
    vv.control.addEventListener("start", update);
}());

vv.view.main = (function(){
    var load_volume_preferences = function() {
        var c = document.getElementById("control-volume");
        c.max = vv.storage.preferences["volume"]["max"];
        if (vv.storage.preferences["volume"]["show"]) {
            c.style.visibility = "visible";
        } else {
            c.style.visibility = "hidden";
        }
    };
    vv.control.addEventListener("control", function() {
        document.getElementById("control-volume").value=vv.storage.control["volume"]
    });
    vv.control.addEventListener("preferences", load_volume_preferences);
    var show = function() {
        vv.storage.last_state = "main";
        vv.storage.save();
        document.body.classList.add("view-main");
        document.body.classList.remove("view-system");
        document.body.classList.remove("view-list");
    };
    var hidden = function() {
        var e = document.body;
        if (window.matchMedia('(orientation: portrait)').matches) {
            return !e.classList.contains("view-main");
        } else {
            return !(e.classList.contains("view-list") || e.classList.contains("view-main"));
        }
    }
    var update = function() {
        document.getElementById("main-title").textContent = vv.storage.current["Title"];
        document.getElementById("main-artist").textContent = vv.storage.current["Artist"];
        if (vv.storage.current.cover) {
            document.getElementById("main-cover-img").style.backgroundImage = 'url("/music_directory/'+vv.storage.current["cover"]+'")';
        } else {
            document.getElementById("main-cover-img").style.backgroundImage = '';
        }
    };
    var update_style = function() {
        var e = document.getElementById("main-cover-img");
        var c = document.getElementById("main-elapsed-circle");
        if (vv.storage.preferences.appearance.circled_image && !e.classList.contains("circled")) {
            e.classList.add("circled");
        }
        if (vv.storage.preferences.appearance.circled_image && c.classList.contains("hide")) {
            c.classList.remove("hide");
        }
        if (!vv.storage.preferences.appearance.circled_image && e.classList.contains("circled")) {
            e.classList.remove("circled");
            c.classList.add("hide");
        }
        if (!vv.storage.preferences.appearance.circled_image && !c.classList.contains("hide")) {
            c.classList.add("hide");
        }
    }
    vv.control.addEventListener("preferences", update_style);
    var update_elapsed = function() {
        if (hidden() || document.getElementById("main-elapsed-circle").classList.contains("hide")) {
            return;
        }
        var c = document.getElementById("main-elapsed-circle-active");
        var elapsed = parseInt(vv.storage.control["song_elapsed"] * 1000);
        if (vv.storage.control["state"] == "play") {
            var last_modified = parseInt(vv.storage.control["last_modified"] * 1000);
            var date = new Date();
            elapsed += date.getTime() - last_modified
        }
        var total = parseInt(vv.storage.current["Time"]);
        var d = (elapsed * 360 / 1000 / total - 90) * (Math.PI / 180);
        if (isNaN(d)) {
            return;
        }
        var x = 100 + 90 * Math.cos(d);
        var y = 100 + 90 * Math.sin(d);
        if (x <= 100) {
            c.setAttribute("d", "M 100,10 L 100,10 A 90,90 0 0,1 100,190 L 100,190 A 90,90 0 0,1 " + x + "," + y);
        } else {
            c.setAttribute("d", "M 100,10 L 100,10 A 90,90 0 0,1 " + x + "," + y);
        }
    }
    var init = function() {
        document.getElementById("control-volume").addEventListener("change", function() {
            vv.control.volume(parseInt(this.value));
        });
        load_volume_preferences();
        update_style();
    };
    vv.control.addEventListener("current", update);
    vv.control.addEventListener("poll", update_elapsed);
    vv.control.addEventListener("start", init);
    return {
        show: show,
        hidden: hidden,
        update: update,
    };
}());
vv.view.list = (function(){
    var show = function() {
        vv.storage.last_state = "list";
        vv.storage.save();
        document.body.classList.add("view-list");
        document.body.classList.remove("view-main");
        document.body.classList.remove("view-system");
    }
    var hidden = function() {
        var e = document.body;
        if (window.matchMedia('(orientation: portrait)').matches) {
            return !e.classList.contains("view-list");
        } else {
            return !(e.classList.contains("view-list") || e.classList.contains("view-main"));
        }
    }
    var update = function() {
        if (vv.storage.tree.length % 2 == 0) {
            document.getElementById("list").classList.remove("odd");
            document.getElementById("list").classList.add("even");
        } else {
            document.getElementById("list").classList.remove("even");
            document.getElementById("list").classList.add("odd");
        }
        var ls = vv.model.list.list();
        var key = ls.key;
        var songs = ls.songs;
        var isdir = ls.isdir;
        var style = ls.style;
        var newul = document.createDocumentFragment();
        var ul = document.getElementById("list-items");
        while (ul.lastChild) {
            ul.removeChild(ul.lastChild);
        }
        var li;
        var i;
        var focus_li = null;
        for (i in songs) {
            if (i == 0 && vv.model.list.rootname() != "root") {
                li = document.createElement("li");
                var p = vv.model.list.parent();
                li = vv.song.element(li, p.song, p.key, p.style);
                newul.appendChild(li);
            }
            li = document.createElement("li");
            li = vv.song.element(li, songs[i], key, style);
            if (songs[i] && vv.model.list.focused() &&
                songs[i].file == vv.model.list.focused().file) {
                focus_li = li;
            }
            vv.control.click(li, function(e) {
                if (e.currentTarget.classList.contains("playing")) {
                    vv.model.list.abs(vv.storage.current);
                    vv.view.main.show();
                    return;
                }
                var value = e.currentTarget.getAttribute("key");
                var uri = e.currentTarget.getAttribute("uri");
                if (isdir) {
                    vv.model.list.down(value);
                } else {
                    vv.control.play(uri);
                }
            }, false);
            newul.appendChild(li);
        }
        var e = document.getElementById("list").children[0];
        e.appendChild(newul);
        if (focus_li) {
            var pos = focus_li.getBoundingClientRect().top;
            var h = vv.view.header.height();
            if (h <= pos && pos <= window.innerHeight - vv.view.footer.height()) {
                return;
            }
            window.scrollTo(0, pos + window.pageYOffset - h);
        } else {
            window.scrollTo(0, 0);
        }
    };
    vv.control.addEventListener("library", update);
    vv.control.addEventListener("current", update);
    vv.model.list.addEventListener("changed", update);
    return {
        show: show,
        hidden: hidden,
    };
}());
vv.view.system = (function() {
    var mkshow = function(p, e) {
        return function() {
            document.getElementById(p).classList.add("on");
            document.getElementById(e).classList.add("on");
        };
    }
    var mkhide = function(p, e) {
        return function() {
            document.getElementById(p).classList.remove("on");
            document.getElementById(e).classList.remove("on");
        };
    }
    var preferences = (function() {
        vv.control.addEventListener('start', function() {
            // TODO: fix loop unrolling
            var update_theme = function() {
                if (vv.storage.preferences.appearance.dark) {
                    document.body.classList.add("dark");
                } else {
                    document.body.classList.remove("dark");
                }
            };
            update_theme();
            var dark = document.getElementById("appearance-dark");
            dark.checked = vv.storage.preferences.appearance.dark;
            dark.addEventListener("change", function() {
                vv.storage.preferences.appearance.dark = this.checked;
                vv.storage.save();
                update_theme();
                vv.control.raiseEvent("preferences");
            });
            var background_image = document.getElementById("appearance-background-image");
            background_image.checked = vv.storage.preferences.appearance.background_image;
            background_image.addEventListener("change", function() {
                vv.storage.preferences.appearance.background_image = this.checked;
                vv.storage.save();
                vv.control.raiseEvent("preferences");
            });
            var background_image_blur = document.getElementById("appearance-background-image-blur");
            background_image_blur.value = String(vv.storage.preferences.appearance.background_image_blur);
            background_image_blur.addEventListener("change", function() {
                vv.storage.preferences.appearance.background_image_blur = parseInt(this.value);
                vv.storage.save();
                vv.control.raiseEvent("preferences");
            });
            var circled_image = document.getElementById("appearance-circled-image");
            circled_image.checked = vv.storage.preferences.appearance.circled_image;
            circled_image.addEventListener("change", function() {
                vv.storage.preferences.appearance.circled_image = this.checked;
                vv.storage.save();
                vv.control.raiseEvent("preferences");
            });
            var playback_view_follow = document.getElementById("playback_view_follow");
            playback_view_follow.checked = vv.storage.preferences.playback.view_follow;
            playback_view_follow.addEventListener("change", function() {
                vv.storage.preferences.playback.view_follow = this.checked;
                vv.storage.save();
                vv.control.raiseEvent("preferences");
            });
            var show_volume = document.getElementById("show_volume");
            show_volume.checked = vv.storage.preferences.volume.show;
            show_volume.addEventListener("change", function() {
                vv.storage.preferences.volume.show = this.checked;
                vv.storage.save();
                vv.control.raiseEvent("preferences");
            });
            var max_volume = document.getElementById("max_volume");
            max_volume.value = String(vv.storage.preferences.volume.max);
            max_volume.addEventListener("change", function() {
                vv.storage.preferences.volume.max = parseInt(this.value);
                vv.storage.save();
                vv.control.raiseEvent("preferences");
            });
            var rescan = document.getElementById("library-rescan");
            vv.control.click(rescan, function() {
                vv.control.rescan_library();
            });
        });
        var update_devices = function() {
            var ul = document.getElementById("system").getElementsByClassName("devices")[0];
            while (ul.lastChild) {
                ul.removeChild(ul.lastChild);
            }
            var i;
            for (i in vv.storage.outputs) {
                var o = vv.storage.outputs[i];
                var li = document.createElement("li");
                var desc = document.createElement("div");
                desc.setAttribute("class", "description");
                desc.textContent = o["outputname"];
                var sw = document.createElement("div");
                sw.setAttribute("class", "value switch");
                var ch = document.createElement("input");
                ch.setAttribute("type", "checkbox");
                ch.setAttribute("id", "device_"+o["outputname"]);
                ch.setAttribute("deviceid", o["outputid"]);
                ch.checked = o["outputenabled"] == "1";
                ch.addEventListener("change", function() {
                    vv.control.output(
                        parseInt(this.getAttribute("deviceid")),
                        this.checked);
                });
                var la = document.createElement("label");
                la.setAttribute("for", "device_"+o["outputname"]);
                sw.appendChild(ch);
                sw.appendChild(la);
                li.appendChild(desc);
                li.appendChild(sw);
                ul.appendChild(li);
            }
        }
        vv.control.addEventListener("outputs", update_devices);
        var update_control = function() {
            var e = document.getElementById("library-rescan");
            if (vv.storage.control.update_library) {
                e.disabled = true;
                e.textContent = "Rescanning";
            } else {
                e.disabled = false;
                e.textContent = "Rescan";
            }
        }
        vv.control.addEventListener("control", update_control);
        var show = mkshow("system-preferences", "system-tab-preferences");
        var show_update = function() {
            vv.control.update_outputs();
            update_control();
            show();
        };
        return {
            'show': show_update,
            'hide': mkhide("system-preferences", "system-tab-preferences"),
        }
    })();
    var stats = (function() {
        var zfill2 = function(i) {
            return ("00" + i).slice(-2);
        }
        var strtimedelta = function(i) {
            var ud = parseInt(i / (24*60*60));
            var uh = parseInt((i - ud*24*60*60) / (60*60));
            var um = parseInt((i - ud*24*60*60 - uh*60*60) / 60);
            var us = parseInt(i - ud*24*60*60 - uh*60*60 - um*60);
            return ud + " days, " + zfill2(uh) + ":" + zfill2(um) + ":" + zfill2(us);
        }

        var update = function() {
            document.getElementById("stat-albums").textContent = vv.storage.stats.albums;
            document.getElementById("stat-artists").textContent = vv.storage.stats.artists;
            document.getElementById("stat-db-playtime").textContent = strtimedelta(parseInt(vv.storage.stats.db_playtime));
            document.getElementById("stat-playtime").textContent = strtimedelta(parseInt(vv.storage.stats.playtime));
            document.getElementById("stat-tracks").textContent = vv.storage.stats.songs;
            var db_update = new Date(parseInt(vv.storage.stats.db_update) * 1000);
            var db_update_yyyymmdd = db_update.getFullYear()*1000+db_update.getMonth()*100+db_update.getDay;
            var db_update_str = "";
            var now = new Date();
            var now_yyyymmdd = now.getFullYear()*1000+now.getMonth()*100+now.getDay;
            if (db_update_yyyymmdd == now_yyyymmdd) {
                db_update_str += "today, ";
            } else if (db_update_yyyymmdd + 1 == now_yyyymmdd) {
                db_update_str += "yesterday, ";
            } else {
                db_update_str += db_update.getFullYear() + '.' + db_update.getMonth() + '.' + db_update.getDay() + ' ';
            }
            db_update_str += db_update.getHours() + ":" + db_update.getMinutes() + ":" + db_update.getSeconds();
            document.getElementById("stat-db-update").textContent = db_update_str;
        }
        var update_time = function() {
            var diff = parseInt(((new Date()).getTime() - vv.storage.stats.last_modified_ms) / 1000);
            var uptime = parseInt(vv.storage.stats.uptime) + diff;
            document.getElementById("stat-uptime").textContent = strtimedelta(uptime);
        }
        vv.control.addEventListener("poll", function() {
            if (document.getElementById("system-stats").classList.contains("on")) {
                update_time();
            }
        });
        var show = mkshow("system-stats", "system-tab-stats");
        var show_update = function() {
            update();
            update_time();
            show();
        }
        return {
            'show': show_update,
            'hide': mkhide("system-stats", "system-tab-stats"),
        }
    })();
    var info = (function() {
        var update = function() {
            if (vv.storage.version.vv) {
                document.getElementById("version").textContent = vv.storage.version.vv;
            }
        }
        vv.control.addEventListener("version", update);
        var show = mkshow("system-info", "system-tab-info");
        var show_update = function() {
            vv.control.update_version();
            update();
            show();
        }
        return {
            'show': show_update,
            'hide': mkhide("system-info", "system-tab-info"),
        }
    })();
    var init = function() {
        preferences.show();
        vv.control.click(document.getElementById("system-tab-preferences"), function() {
            stats.hide();
            info.hide();
            preferences.show();
        });
        vv.control.click(document.getElementById("system-tab-stats"), function() {
            preferences.hide();
            info.hide();
            stats.show();
        });
        vv.control.click(document.getElementById("system-tab-info"), function() {
            preferences.hide();
            stats.hide();
            info.show();
        });
        vv.control.click(document.getElementById("system-reload"), function() {
            location.reload();
        });
        document.getElementById("user-agent").textContent = navigator.userAgent;

    };
    var show = function() {
        vv.storage.last_state = "system";
        vv.storage.save();
        document.body.classList.add("view-system");
        document.body.classList.remove("view-list");
        document.body.classList.remove("view-main");
    };
    var hidden = function() {
        return !document.body.classList.contains("view-system");
    }
    vv.control.addEventListener("start", init);
    return {
        show: show,
        hidden: hidden,
    };
}());
vv.view.header = (function(){
    var init = function() {
        vv.control.click(document.getElementById("menu-back"), function(e) {
            if (!vv.view.list.hidden()) {
                vv.model.list.up();
            } else {
                vv.model.list.abs(vv.storage.current);
            }
            vv.view.list.show();
            e.stopPropagation();
        });
        vv.control.click(document.getElementById("menu-main"), function(e) {
            e.stopPropagation();
            if (vv.model.list.rootname() != "root") {
                vv.model.list.abs(vv.storage.current);
            }
            vv.view.main.show();
            e.stopPropagation();
        });
        vv.control.click(document.getElementById("menu-system"), function(e) {
            vv.view.system.show();
            e.stopPropagation();
        });
        update();
        vv.model.list.addEventListener("changed", update);
        vv.control.addEventListener("library", update);
    };
    var update = function() {
        var e = document.getElementById("menu-back-content");
        var b = document.getElementById("menu-back");
        var m = document.getElementById("menu-main");
        if (vv.model.list.rootname() != "root") {
            b.classList.remove("root");
            m.classList.remove("root");
            var songs = vv.model.list.list()["songs"];
            if (songs[0]) {
                var p = vv.model.list.grandparent();
                e.textContent = vv.song.get(p.song, p.key);
            }
        } else {
            b.classList.add("root");
            m.classList.add("root");
        }
    }
    var height = function() {
        return document.getElementsByTagName("header")[0].offsetHeight;
    };
    vv.control.addEventListener("start", init);
    return {
        update: update,
        height: height,
    };
}());
vv.view.footer = (function(){
    var init = function() {
        vv.control.click(document.getElementById("control-prev"), function(e) {
            vv.control.prev();
            e.stopPropagation();
        });
        vv.control.click(document.getElementById("control-toggleplay"), function(e) {
            vv.control.play_pause();
            e.stopPropagation();
        });
        vv.control.click(document.getElementById("control-next"), function(e) {
            vv.control.next();
            e.stopPropagation();
        });
        vv.control.click(document.getElementById("control-repeat"), function(e) {
            vv.control.toggle_repeat();
            e.stopPropagation();
        });
        vv.control.click(document.getElementById("control-random"), function(e) {
            vv.control.toggle_random();
            e.stopPropagation();
        });
    };

    var update_control = function() {
        if (vv.storage.control["state"] == "play") {
            document.getElementById("control-toggleplay").classList.add("pause");
            document.getElementById("control-toggleplay").classList.remove("play");
        } else {
            document.getElementById("control-toggleplay").classList.add("play");
            document.getElementById("control-toggleplay").classList.remove("pause");
        }
        if (vv.storage.control["repeat"]) {
            document.getElementById("control-repeat").classList.add("on");
            document.getElementById("control-repeat").classList.remove("off");
        } else {
            document.getElementById("control-repeat").classList.add("off");
            document.getElementById("control-repeat").classList.remove("on");
        }
        if (vv.storage.control["random"]) {
            document.getElementById("control-random").classList.add("on");
            document.getElementById("control-random").classList.remove("off");
        } else {
            document.getElementById("control-random").classList.add("off");
            document.getElementById("control-random").classList.remove("on");
        }
    }

    vv.control.addEventListener("start", init);
    vv.control.addEventListener("control", update_control);
    var height = function() {
        return document.getElementsByTagName("footer")[0].offsetHeight;
    };
    return {
        height: height,
    }
}());
vv.view.elapsed = (function() {
    var update = function() {
        var data = vv.storage.control;
        if ('state' in data) {
            var elapsed = parseInt(data["song_elapsed"] * 1000);
            var current = elapsed;
            var last_modified = parseInt(data["last_modified"] * 1000);
            var date = new Date();
            if (data["state"] == "play") {
                current += date.getTime() - last_modified
            }
            current = parseInt(current / 1000);
            var min = parseInt(current / 60)
            var sec = current % 60
            var label = min + ':' + ("0" + sec).slice(-2)
            var texts = document.getElementsByClassName("elapsed");
            var i;
            for (i in texts) {
                if (texts[i].textContent != label) {
                    texts[i].textContent = label;
                }
            }
        }
    }
    vv.control.addEventListener("control", update);
    vv.control.addEventListener("poll", update);
    return {update: update};
}());
vv.view.modal.hide = function() {
    document.getElementById("modal-background").classList.remove("show");
    document.getElementById("modal-outer").classList.remove("show");
    var ws = document.getElementsByClassName("modal-window");
    var i;
    for (i in ws) {
        if (ws[i].classList) {
            ws[i].classList.remove("show");
        }
    }
}
vv.view.modal.help = (function() {
    var show = function() {
        var b = document.getElementById("modal-background");
        if (b.classList.contains("show")) {
            return;
        }
        b.classList.add("show");
        document.getElementById("modal-outer").classList.add("show");
        document.getElementById("modal-help").classList.add("show");
    }
    var hide = function() {
        document.getElementById("modal-background").classList.remove("show");
        document.getElementById("modal-outer").classList.remove("show");
        document.getElementById("modal-help").classList.remove("show");
    }
    vv.control.addEventListener("start", function() {
        vv.control.click(document.getElementById("modal-help-close"), hide);
        vv.control.click(document.getElementById("modal-outer"), vv.view.modal.hide);
        vv.control.click(document.getElementById("modal-background"), vv.view.modal.hide);

        var ws = document.getElementsByClassName("modal-window");
        var i;
        for (i in ws) {
            if (ws[i].addEventListener) {
                vv.control.click(ws[i], function(e) {
                    e.stopPropagation();
                });
            }
        }
    });
    return {
        "show": show,
        "hide": hide,
    }
}());
(function() {
    vv.control.addEventListener("start", function() {
        document.addEventListener("keydown", function(e) {
            if (document.getElementById("modal-background").classList.contains("show")) {
                if (e.key == "Escape" || e.key == "Esc") {
                    vv.view.modal.hide();
                }
                return;
            }
            var buble = false;
            var single = !e.altKey && !e.ctrlKey && !e.metaKey;
            if (single && e.keyCode == 37) {
                if (!vv.view.list.hidden()) {
                    vv.model.list.up();
                } else {
                    vv.model.list.abs(vv.storage.current);
                }
                vv.view.list.show();
            } else if (single && e.keyCode == 39) {
                if (vv.model.list.rootname() != "root") {
                    vv.model.list.abs(vv.storage.current);
                }
                vv.view.main.show();
                e.stopPropagation();
            } else if (single && e.key == "?") {
                vv.view.modal.help.show();
            } else {
                buble = true;
            }
            if (!buble) {
                e.stopPropagation();
            }
        });
    });
}());

vv.control.start();
