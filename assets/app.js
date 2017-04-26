var vv = vv || {
    obj: {},
    song: {},
    songs: {},
    storage: {},
    model: {list: {}},
    view: {background: {}, main: {}, list: {}, config: {}, menu: {}, playback: {}, elapsed: {}},
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
                inner += '<svg width="100" height="100" viewBox="0 0 100 100"><path class="fill" d="M 25,20 80,50 25,80 z"/></svg>';
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
            inner += '<div class=img-sq><img class=cover src="/music_directory/'+song["cover"]+'"></div>';
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
         "Genre": [],
    }
    var library_last_modified = "";
    var outputs = [];
    var outputs_last_modified = "";
    var config = {
        "volume": {"show": true, "max": 100}, "playback": {"view_follow": true},
        "appearance": {"dark": false, "background_image": false, "background_image_blur": 0},
    };
    var save = function() {
        try {
            localStorage.tree = JSON.stringify(tree);
            localStorage.config = JSON.stringify(config);
        } catch (e) {
            // private browsing
        }
    }
    var load = function() {
        try {
            if (localStorage.tree) {
                tree = JSON.parse(localStorage.tree);
            }
            if (localStorage.config) {
                var c = JSON.parse(localStorage.config);
                var i, j;
                for (i in c) {
                    for (j in c[i]) {
                        config[i][j] = c[i][j];
                    }
                }
            }
        } catch (e) {
            // private browsing
        }
    }
    load();
    return {
        tree: tree,
        current: current,
        current_last_modified: current_last_modified,
        control: control,
        control_last_modified: control_last_modified,
        library: library,
        library_last_modified: library_last_modified,
        outputs: outputs,
        outputs_last_modified: outputs_last_modified,
        config: config,
        save: save,
        load: load,
    };
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
        raiseEvent("changed");
    };
    var abs = function(song) {
        focus = song;
        var i, root, key, selected;
        if (rootname() != "root" && song.file) {
            vv.storage.tree = [vv.storage.tree[0]];
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
            vv.storage.tree = [];
            vv.storage.save();
        }
        raiseEvent("changed");
    };
    var list = function() {
        if (rootname() == "root") {
            return list_root();
        } else {
            return list_child();
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
    var parent = function(v) {
        if (!v) {
            v = list().songs;
        }
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
    var grandparent = function(v) {
        if (!v) {
            v = list().songs;
        }
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
    var listener = {"control": [], "config": [], "library": [], "playlist": [], "current": [], "outputs": [], "start": [], "poll": []}
    var addEventListener = function(ev, func) {
        listener[ev].push(func);
    };
    var raiseEvent = function(ev) {
        var i;
        for (i in listener[ev]) {
            listener[ev][i]();
        }
    };

    var get_request = function(path, ifmodified, callback) {
        var xhr = new XMLHttpRequest();
        xhr.onreadystatechange = function() {
            if (xhr.readyState == 4) {
                if (xhr.status == 200 && callback) {
                    callback(JSON.parse(xhr.responseText), xhr.getResponseHeader("Last-Modified"));
                }
            }
        };
        xhr.open("GET", path, true);
        if (ifmodified != "") {
            xhr.setRequestHeader("If-Modified-Since", ifmodified);
        }
        xhr.send();
    }

    var post_request = function(path, obj, callback) {
        var xhr = new XMLHttpRequest();
        xhr.onreadystatechange = function() {
            if (xhr.readyState == 4) {
                if (xhr.status == 200 && callback) {
                    callback(JSON.parse(xhr.responseText));
                }
            }
        };
        xhr.open("POST", path, true);
        xhr.setRequestHeader("Content-Type", "application/json");
        xhr.send(JSON.stringify(obj));
    }

    var update_song = function() {
        get_request("api/songs/current", vv.storage.current_last_modified, function(ret, modified) {
            if (!ret.error) {
                var song = ret.data? ret.data : {};
                vv.storage.current = song;
                vv.storage.current_last_modified = modified;
                if (vv.model.list.rootname() != "root" && vv.storage.config.playback.view_follow && song.file) {
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
            vv.control.update_song();
            vv.control.update_status();
            vv.control.update_library();
            vv.control.update_outputs();
            raiseEvent("poll");
            setTimeout(polling, 1000);
        }
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
        update_song: update_song,
        update_status: update_status,
        update_library: update_library,
        update_outputs: update_outputs,
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
        if (vv.storage.config.appearance.background_image) {
            e.classList.remove("hide");
            document.getElementById("background").classList.remove("hide");
            e.style.backgroundImage = 'url("/music_directory/'+vv.storage.current["cover"]+'")';
            e.style.filter = "blur(" + vv.storage.config.appearance.background_image_blur + "px)";
        } else {
            e.classList.add("hide");
            document.getElementById("background").classList.add("hide");
        }
    };
    vv.control.addEventListener("current", update);
    vv.control.addEventListener("config", update);
    vv.control.addEventListener("start", update);
}());

vv.view.main = (function(){
    var load_volume_config = function() {
        var c = document.getElementById("control-volume");
        c.max = vv.storage.config["volume"]["max"];
        if (vv.storage.config["volume"]["show"]) {
            c.style.visibility = "visible";
        } else {
            c.style.visibility = "hidden";
        }
    };
    vv.control.addEventListener("control", function() {
        document.getElementById("control-volume").value=vv.storage.control["volume"]
    });
    vv.control.addEventListener("config", load_volume_config);
    var show = function() {
        document.body.classList.add("view-main");
        document.body.classList.remove("view-config");
        document.body.classList.remove("view-list");
    };
    var hidden = function() {
        return !document.body.classList.contains("view-main");
    }
    var update = function() {
        document.getElementById("main-title").textContent = vv.storage.current["Title"];
        document.getElementById("main-artist").textContent = vv.storage.current["Artist"];
        document.getElementById("main-cover").style.backgroundImage = 'url("/music_directory/'+vv.storage.current["cover"]+'")';
    };
    var update_elapsed = function() {
        if (hidden()) {
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
        var x = 100 + 70 * Math.cos(d);
        var y = 100 + 70 * Math.sin(d);
        if (x <= 100) {
            c.setAttribute("d", "M 100,30 L 100,30 A 70,70 0 0,1 100,170 L 100,170 A 70,70 0 0,1 " + x + "," + y);
        } else {
            c.setAttribute("d", "M 100,30 L 100,30 A 70,70 0 0,1 " + x + "," + y);
        }
    }
    var init = function() {
        show();
        document.getElementById("control-volume").addEventListener("change", function() {
            vv.control.volume(parseInt(this.value));
        });
        load_volume_config();
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
        document.body.classList.add("view-list");
        document.body.classList.remove("view-main");
        document.body.classList.remove("view-config");
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
        var ls = vv.model.list.list(),
            key = ls["key"],
            songs = ls["songs"],
            isdir = ls["isdir"],
            style = ls["style"],
            newul = document.createDocumentFragment(),
            ul = document.getElementById("list").children[0],
            li;
        ul.innerHTML = "";
        var i;
        var focus_li = null;
        for (i in songs) {
            if (i == 0 && vv.model.list.rootname() != "root") {
                li = document.createElement("li");
                var p = vv.model.list.parent(songs);
                li = vv.song.element(li, p.song, p.key, p.style);
                newul.appendChild(li);
            }
            li = document.createElement("li");
            li = vv.song.element(li, songs[i], key, style);
            if (songs[i] && vv.model.list.focused() &&
                songs[i].file == vv.model.list.focused().file) {
                focus_li = li;
            }
            li.addEventListener('click', function() {
                if (this.classList.contains("playing")) {
                    vv.model.list.abs(vv.storage.current);
                    vv.view.main.show();
                    return;
                }
                if (!vv.view.menu.hidden_sub()) {
                    vv.view.menu.hide_sub();
                    return;
                }
                var value = this.getAttribute("key");
                var uri = this.getAttribute("uri");
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
            var h = vv.view.menu.height();
            if (h <= pos && pos <= window.innerHeight - vv.view.playback.height()) {
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
vv.view.config = (function(){
    var init = function() {
        // TODO: fix loop unrolling
        var update_theme = function() {
            if (vv.storage.config.appearance.dark) {
                document.body.classList.add("dark");
            } else {
                document.body.classList.remove("dark");
            }
        };
        update_theme();
        var dark = document.getElementById("appearance-dark");
        dark.checked = vv.storage.config.appearance.dark;
        dark.addEventListener("change", function() {
            vv.storage.config.appearance.dark = this.checked;
            vv.storage.save();
            update_theme();
            vv.control.raiseEvent("config");
        });
        var background_image = document.getElementById("appearance-background-image");
        background_image.checked = vv.storage.config.appearance.background_image;
        background_image.addEventListener("change", function() {
            vv.storage.config.appearance.background_image = this.checked;
            vv.storage.save();
            vv.control.raiseEvent("config");
        });
        var background_image_blur = document.getElementById("appearance-background-image-blur");
        background_image_blur.value = String(vv.storage.config.appearance.background_image_blur);
        background_image_blur.addEventListener("change", function() {
            vv.storage.config.appearance.background_image_blur = parseInt(this.value);
            vv.storage.save();
            vv.control.raiseEvent("config");
        });
        var playback_view_follow = document.getElementById("playback_view_follow");
        playback_view_follow.checked = vv.storage.config.playback.view_follow;
        playback_view_follow.addEventListener("change", function() {
            vv.storage.config.playback.view_follow = this.checked;
            vv.storage.save();
            vv.control.raiseEvent("config");
        });
        var show_volume = document.getElementById("show_volume");
        show_volume.checked = vv.storage.config.volume.show;
        show_volume.addEventListener("change", function() {
            vv.storage.config.volume.show = this.checked;
            vv.storage.save();
            vv.control.raiseEvent("config");
        });
        var max_volume = document.getElementById("max_volume");
        max_volume.value = String(vv.storage.config.volume.max);
        max_volume.addEventListener("change", function() {
            vv.storage.config.volume.max = parseInt(this.value);
            vv.storage.save();
            vv.control.raiseEvent("config");
        });
    };
    var update_devices = function() {
        var ul = document.getElementById("config").getElementsByClassName("devices")[0];
        ul.innerHTML = "";
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
    var show = function() {
        document.body.classList.add("view-config");
        document.body.classList.remove("view-list");
        document.body.classList.remove("view-main");
    };
    var hidden = function() {
        return !document.body.classList.contains("view-config");
    }
    vv.control.addEventListener("start", init);
    return {
        show: show,
        hidden: hidden,
    };
}());
vv.view.menu = (function(){
    var init = function() {
        document.body.addEventListener('click', function() {
            vv.view.menu.hide_sub();
        });
        document.getElementById("menu-back").addEventListener('click', function(e) {
            if (!vv.view.list.hidden()) {
                vv.model.list.up();
            } else {
                vv.model.list.abs(vv.storage.current);
            }
            vv.view.list.show();
            e.stopPropagation();
        });
        document.getElementById("menu-main").addEventListener('click', function(e) {
            e.stopPropagation();
            if (vv.model.list.rootname() == "root") {
                return;
            }
            vv.model.list.abs(vv.storage.current);
            vv.view.main.show();
            e.stopPropagation();
        });
        document.getElementById("menu-settings").addEventListener('click', function(e) {
            if (vv.view.menu.hidden_sub()) {
                vv.view.menu.show_sub();
            } else {
                vv.view.menu.hide_sub();
            }
            e.stopPropagation();
        });
        document.getElementById("menu-settings-list-reload").addEventListener('click', function() {
            location.reload();
        });
        document.getElementById("menu-settings-list-config").addEventListener('click', function() {
            vv.view.config.show();
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
                var p = vv.model.list.grandparent(songs);
                vv.song.element(e, p["song"], p["key"], p["style"]);
            }
        } else {
            b.classList.add("root");
            m.classList.add("root");
        }
    }
    var show_sub = function() {
        var e = document.getElementById("menu-settings-list");
        e.classList.add("show");
    };
    var hide_sub = function() {
        var e = document.getElementById("menu-settings-list");
        e.classList.remove("show");
    };
    var hidden_sub = function() {
        return !document.getElementById("menu-settings-list").classList.contains("show");
    };
    var height = function() {
        return document.getElementsByTagName("header")[0].offsetHeight;
    };
    vv.control.addEventListener("start", init);
    return {
        update: update,
        show_sub: show_sub,
        hide_sub: hide_sub,
        hidden_sub: hidden_sub,
        height: height,
    };
}());
vv.view.playback = (function(){
    var init = function() {
        document.getElementById("control-prev").addEventListener('click', function(e) {
            vv.control.prev();
            e.stopPropagation();
        });
        document.getElementById("control-toggleplay").addEventListener('click', function(e) {
            vv.control.play_pause();
            e.stopPropagation();
        });
        document.getElementById("control-next").addEventListener('click', function(e) {
            vv.control.next();
            e.stopPropagation();
        });
        document.getElementById("control-repeat").addEventListener('click', function(e) {
            vv.control.toggle_repeat();
            e.stopPropagation();
        });
        document.getElementById("control-random").addEventListener('click', function(e) {
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
vv.control.start();
