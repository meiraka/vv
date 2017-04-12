var vv = vv || {
    obj: {},
    song: {},
    songs: {},
    storage: {},
    model: {list: {}},
    view: {error: {}, main: {}, list: {}, config: {}, menu: {}, playback: {}, elapsed: {}, dropdown: {}},
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
    return {
        getOrElse: getOrElse,
        get: get,
        str: str,
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
    var config = {"volume": {"show": true, "max": 100}, "playback": {"view_follow": true}}

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
        focus = list()[1][0];
        if (rootname() != "root") {
            vv.storage.tree.pop();
            vv.storage.save();
        }
    };
    var down = function(value) {
        var r = rootname();
        var key = "root";
        if (r != "root") {
            key = TREE[r]["tree"][vv.storage.tree.length - 1][0];
        }
        vv.storage.tree.push([key, value]);
        vv.storage.save();
        focus = list()[1][0];
    };
    var abs = function(song) {
        focus = song;
        var i, root, key, selected;
        if (rootname() != "root") {
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
        }
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
            type = "dir";
        if (vv.storage.tree.length == TREE[root]["tree"].length) {
            type = "file";
        }
        var leef;
        for (leef in vv.storage.tree) {
            if (leef == 0) { continue; }
            filters[vv.storage.tree[leef][0]] = vv.storage.tree[leef][1];
        }
        library = vv.songs.filter(library, filters);
        library = vv.songs.uniq(library, key);
        return [key, library, style, type];
    };
    var list_root = function() {
        var ret = [];
        var rootname = "";
        for (rootname in TREE) {
            ret.push({"root": rootname});
        }
        return ["root", ret, "plain", "dir"];
    }
    return {
        focused: focused,
        update: update,
        rootname: rootname,
        sortkeys: sortkeys,
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
                vv.storage.current = ret["data"];
                vv.storage.current_last_modified = modified;
                if (vv.model.list.rootname() != "root" && vv.storage.config.playback.view_follow) {
                    vv.model.list.abs(ret["data"]);
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

vv.view.error = (function() {
    var hide = function() {
        var e = document.getElementById("error");
        e.children[1].textContent = "";
        e.display = "none";
    }
    var show = function(description) {
        var e = document.getElementById("error");
        e.children[1].textContent = e.children[1].textContent + description;
        e.display = "block";
        setTimeout(5000, hide);
    }
    return {
        hide: hide,
        show: show,
    }
}());

vv.view.main = (function(){
    var load_volume_config = function() {
        var c = document.getElementById("playback_volume").children[0];
        c.max = vv.storage.config["volume"]["max"];
        if (vv.storage.config["volume"]["show"]) {
            c.style.display = "block";
        } else {
            c.style.display = "none";
        }
    };
    vv.control.addEventListener("control", function() {
        document.getElementById("playback_volume").children[0].value=vv.storage.control["volume"]
    });
    vv.control.addEventListener("config", load_volume_config);
    (function() {
        if (document.readyState !== 'loading') {
            init();
        } else {
            document.addEventListener('DOMContentLoaded', init);
        }
    })();
    var show = function() {
        var e = document.getElementById("main");
        e.style.display = "block";
    };
    var hide = function() {
        var e = document.getElementById("main");
        e.style.display = "none";
    }
    var hidden = function() {
        return document.getElementById("main").style.display == "none";
    }
    var update = function() {
        var e = document.getElementById("main");
        e.getElementsByClassName("title")[0].textContent = vv.storage.current["Title"];
        e.getElementsByClassName("artist")[0].textContent = vv.storage.current["Artist"];
        document.getElementById("current_cover").style.backgroundImage = "url(/api/songs/"+vv.storage.current["Pos"]+"?detail=cover)";
    };
    var resize_image = function() {
        var p = window.matchMedia('(orientation: portrait)').matches
        var w = document.body.clientWidth;
        if (!p) {w = parseInt(w / 2);}
        var h = window.innerHeight;
        var e = document.getElementById("current_cover");
        var cs = parseInt((w < h ? w : h) * 0.7);
        e.style.top = (h - cs) / 2 + "px";
        // e.style.left = (p? ((w - cs) / 2) : ((w - cs)/2 + w)) + "px";
        // e.style.width = cs + "px";
        e.style.height = cs + "px";
    };
    var update_elapsed = function() {
        if (hidden()) {
            return;
        }
        var r = document.getElementById("elapsed_right");
        var l = document.getElementById("elapsed_left");
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
            r.setAttribute("d", "M 100,30 L 100,30 A 70,70 0 0,1 100,170");
            l.setAttribute("d", "M 100,170 L 100,170 A 70,70 0 0,1 " + x + "," + y);
        } else {
            r.setAttribute("d", "M 100,30 L 100,30 A 70,70 0 0,1 " + x + "," + y);
            l.setAttribute("d", "M 100,170 L 100,170 A 70,70 0 0,1 100,170");
        }
    }
    var orientation = function() {
        if (!vv.view.config.hidden()) {
            return;
        }
        show();
    }
    var init = function() {
        document.getElementById("playback_volume").children[0].addEventListener("change", function() {
            vv.control.volume(parseInt(this.value));
        });
        load_volume_config();
        window.addEventListener("resize", resize_image, false);
        resize_image();
    };
    window.matchMedia("(orientation: portrait)").addListener(orientation);
    window.matchMedia("(orientation: landscape)").addListener(orientation);
    vv.control.addEventListener("current", update);
    vv.control.addEventListener("poll", update_elapsed);
    vv.control.addEventListener("start", init);
    return {
        show: show,
        hide: hide,
        hidden: hidden,
        update: update,
    };
}());
vv.view.list = (function(){
    var show = function() {
        var e = document.getElementById("list");
        e.style.display = "block";
    };
    var hide = function() {
        var e = document.getElementById("list");
        e.style.display = "none";
    }
    var hidden = function() {
        return document.getElementById("list").style.display == "none";
    }
    var update = function() {
        var ls = vv.model.list.list(),
            songs = ls[1],
            type = ls[3],
            newul = document.createDocumentFragment(),
            ul = document.getElementById("list").children[0],
            li;
        ul.innerHTML = "";
        var i;
        var focus_li = null;
        for (i in songs) {
            li = make_list_item(songs[i], ls[0], ls[2]);
            if (songs[i]['file'] == vv.model.list.focused()['file']) {
                focus_li = li;
            }
            li.addEventListener('click', function() {
                if (!vv.view.dropdown.hidden()) {
                    vv.view.dropdown.hide();
                    return;
                }
                var value = this.getAttribute("key"),
                    uri = this.getAttribute("uri");
                if (type == "dir") {
                    vv.model.list.down(value);
                    vv.view.list.update();
                } else {
                    vv.control.play(uri);
                }
            }, false);
            newul.appendChild(li);
        }
        document.getElementById("list").children[0].appendChild(newul);
        if (focus_li) {
            var pos = focus_li.getBoundingClientRect().top;
            var h = vv.view.menu.height();
            if (h <= pos && pos <= window.innerHeight - vv.view.playback.height()) {
                return;
            }
            window.scrollTo(0, pos + window.pageYOffset - h);
        }
    };
    var make_list_item = function(song, key, style) {
        var li = document.createElement("li");
        var inner = "";
        li.setAttribute("class", style);
        li.setAttribute("key", vv.song.get(song, key));
        li.setAttribute("uri", song["file"]);
        if (style == "song") {
            inner = "<span class=track>"+vv.song.get(song, "TrackNumber")+"</span>"+
                    "<span class=title>"+vv.song.get(song, "Title")+"</span>";
            if (vv.song.get(song, "Artist") != vv.song.get(song, "AlbumArtist")) {
                inner += "<span class=artist>"+vv.song.get(song, "Artist")+"</span>";
            }
            if (vv.song.get(song, "file") == vv.song.get(vv.storage.current, "file")) {
                inner += "<span class=elapsed></span>"+
                         "<span class=length_separator>/</span>";
            }
            inner += "<span class=length>"+vv.song.get(song, "Length")+"</span>";
        } else if (style == "album") {
            inner += '<img class=cover src="/api/library/'+song["Pos"]+'?detail=cover">';
            inner += "<span class=date>"+vv.song.get(song, "Date")+"</span>";
            inner += "<span class=album>"+vv.song.get(song, "Album")+"</span>";
            inner += "<span class=albumartist>"+vv.song.get(song, "AlbumArtist")+"</span>";
        } else {
            inner = "<span class=key>"+vv.song.get(song, key)+"</span>";
        }
        li.innerHTML = inner;
        return li;
    };
    vv.control.addEventListener("library", update);
    var orientation = function() {
        if (!vv.view.config.hidden()) {
            return;
        }
        if (matchMedia("(orientation: portrait)").matches) {
            hide();
        } else {
            show();
        }
    }
    window.matchMedia("(orientation: portrait)").addListener(orientation);
    window.matchMedia("(orientation: landscape)").addListener(orientation);
    vv.control.addEventListener("current", update);
    return {
        show: show,
        hide: hide,
        hidden: hidden,
        update: update,
    };
}());
vv.view.config = (function(){
    var init = function() {
        // TODO: fix loop unrolling
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
        var e = document.getElementById("config");
        e.style.display = "block";
    };
    var hide = function() {
        var e = document.getElementById("config");
        e.style.display = "none";
    }
    var hidden = function() {
        return document.getElementById("config").style.display == "none";
    }
    vv.control.addEventListener("start", init);
    return {
        show: show,
        hide: hide,
        hidden: hidden,
    };
}());
vv.view.menu = (function(){
    var init = function() {
        document.body.addEventListener('click', function() {
            vv.view.menu.hide_sub();
        });
        var menu = document.getElementById("menu");
        menu.getElementsByClassName("back")[0].addEventListener('click', function(e) {
            if (!vv.view.list.hidden()) {
                vv.model.list.up();
            } else {
                vv.model.list.abs(vv.storage.current);
            }
            if (!window.matchMedia('(orientation: landscape)').matches) {
                vv.view.main.hide();
            } else {
                vv.view.main.show();
            }
            vv.view.config.hide();
            vv.view.list.update();
            vv.view.list.show();
            e.stopPropagation();
        });
        menu.getElementsByClassName("main")[0].addEventListener('click', function(e) {
            if (!window.matchMedia('(orientation: landscape)').matches) {
                vv.view.list.hide();
            } else {
                vv.view.list.show();
            }
            vv.model.list.abs(vv.storage.current);
            if (!vv.view.list.hidden()) {
                vv.view.list.update();
            }
            vv.view.config.hide();
            vv.view.main.show();
            e.stopPropagation();
        });
        menu.getElementsByClassName("settings")[0].addEventListener('click', function(e) {
            if (vv.view.menu.hidden_sub()) {
                vv.view.menu.show_sub();
            } else {
                vv.view.menu.hide_sub();
            }
            e.stopPropagation();
        });
        var submenu = document.getElementById("submenu");
        submenu.getElementsByClassName("reload")[0].addEventListener('click', function() {
            location.reload();
        });
        submenu.getElementsByClassName("config")[0].addEventListener('click', function() {
            vv.view.main.hide();
            vv.view.list.hide();
            vv.view.config.show();
        });
    };
    var show_sub = function() {
        var e = document.getElementById("submenu");
        e.style.display = "block";
    };
    var hide_sub = function() {
        var e = document.getElementById("submenu");
        e.style.display = "none";
    };
    var hidden_sub = function() {
        return document.getElementById("submenu").style.display == "none";
    };
    var height = function() {
        return document.getElementsByTagName("header")[0].offsetHeight;
    };
    vv.control.addEventListener("start", init);
    return {
        show_sub: show_sub,
        hide_sub: hide_sub,
        hidden_sub: hidden_sub,
        height: height,
    };
}());
vv.view.playback = (function(){
    var init = function() {
        var playback = document.getElementById("playback");
        playback.getElementsByClassName("prev")[0].addEventListener('click', function(e) {
            vv.control.prev();
            e.stopPropagation();
        });
        playback.getElementsByClassName("play")[0].addEventListener('click', function(e) {
            vv.control.play_pause();
            e.stopPropagation();
        });
        playback.getElementsByClassName("next")[0].addEventListener('click', function(e) {
            vv.control.next();
            e.stopPropagation();
        });
        var playback_list = document.getElementById("playback_list");
        playback_list.getElementsByClassName("repeat")[0].addEventListener('click', function(e) {
            vv.control.toggle_repeat();
            e.stopPropagation();
        });
        playback_list.getElementsByClassName("random")[0].addEventListener('click', function(e) {
            vv.control.toggle_random();
            e.stopPropagation();
        });
    };

    var update = function() {
        if (vv.storage.control["state"] == "play") {
            document.getElementById("playback").getElementsByClassName("play")[0].children[0].src = "/assets/pause.svg";
        } else {
            document.getElementById("playback").getElementsByClassName("play")[0].children[0].src = "/assets/play.svg";
        }
        var current = document.getElementById("playback_list");
        if (vv.storage.control["repeat"]) {
            current.getElementsByClassName("repeat")[0].children[0].style.opacity=1.0;
        } else {
            current.getElementsByClassName("repeat")[0].children[0].style.opacity=0.5;
        }
        if (vv.storage.control["random"]) {
            current.getElementsByClassName("random")[0].children[0].style.opacity=1.0;
        } else {
            current.getElementsByClassName("random")[0].children[0].style.opacity=0.5;
        }
    }
    vv.control.addEventListener("start", init);
    vv.control.addEventListener("control", update);
    var height = function() {
        return document.getElementsByTagName("footer")[0].offsetHeight;
    };
    return {
        update: update,
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
vv.view.dropdown = (function() {
    var hidden = function() {
        return vv.view.menu.hidden_sub();
    }
    var hide = function() {
        vv.view.menu.hide_sub();
    }
    return {
        hidden: hidden,
        hide: hide,
    }
}());
vv.control.start();
