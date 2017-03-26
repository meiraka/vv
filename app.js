var vv = vv || {
    obj: {},
    song: {},
    songs: {},
    storage: {},
    model: {list: {}},
    view: {main: {}, list: {}, menu: {}, elapsed: {}, dropdown: {}},
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
            if (a[1] < b[1]) { return -1; } else { return 1; };
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
        return songs.filter(function(song, i, self) {
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

    var save = function() {
        try {
            localStorage.tree = JSON.stringify(tree);
        } catch (e) {}
    }
    var load = function() {
        try {
            if (localStorage.tree) {
                tree = JSON.parse(localStorage.tree);
            }
        } catch (e) {}
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
    var sortkeys = function() {
        var r = rootname();
        if (r == "root") {
            return [];
        }
        return TREE[r]["sort"];
    }
    var up = function() {
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
    };
    var abs = function(song) {
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
        var ls = [];
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
            song = {};
            type = "dir";
        if (vv.storage.tree.length == TREE[root]["tree"].length) {
            type = "file";
        }
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
        update: update,
        rootname: rootname,
        sortkeys: sortkeys,
        up: up,
        down: down,
        abs: abs,
        list: list,
    };
}());
vv.view.main = (function(){
    var show = function() {
        var e = document.getElementById("current");
        e.style.display = "block";
    };
    var hide = function() {
        var e = document.getElementById("current");
        e.style.display = "none";
    }
    var hidden = function() {
        return document.getElementById("current").style.display == "none";
    }
    var update = function() {
        var e = document.getElementById("current");
        var key;
        var ul = e.getElementsByClassName("detail")[0];
        var newul = document.createDocumentFragment();
        var li;
        e.getElementsByClassName("title")[0].textContent = vv.storage.current["Title"];
        e.getElementsByClassName("artist")[0].textContent = vv.storage.current["Artist"];
        ul.innerHTML = "";
        for (key in vv.storage.current) {
            if (key == "Title" || key == "Artist") {
                continue;
            }
            li = document.createElement("li");
            li.textContent = key + ": " + vv.storage.current[key];
            newul.appendChild(li);
        }
        ul.appendChild(newul);
    };
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
            song = {},
            newul = document.createDocumentFragment(),
            ul = document.getElementById("list").children[0],
            li;
        ul.innerHTML = "";
        for (i in songs) {
            song = songs[i];
            li = make_list_item(songs[i], ls[0], ls[2], type);
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
    };
    var make_list_item = function(song, key, style, type) {
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
                inner += "<span class=elapsed>"+vv.song.get(song, "Length")+"</span>"+
                         "<span class=length_separator>/</span>";
            }
            inner += "<span class=length>"+vv.song.get(song, "Length")+"</span>";
        } else if (style == "album") {
            inner += "<span class=date>"+vv.song.get(song, "Date")+"</span>";
            inner += "<span class=album>"+vv.song.get(song, "Album")+"</span>";
            inner += "<span class=albumartist>"+vv.song.get(song, "AlbumArtist")+"</span>";
        } else {
            inner = vv.song.get(song, key);
        }
        li.innerHTML = inner;
        return li;
    };
    return {
        show: show,
        hide: hide,
        hidden: hidden,
        update: update,
    };
}());
vv.view.menu = (function(){
    var show_sub = function() {
        var e = document.getElementById("submenu");
        e.style.display = "block";
    }
    var hide_sub = function() {
        var e = document.getElementById("submenu");
        e.style.display = "none";
    }
    var hidden_sub = function() {
        return document.getElementById("submenu").style.display == "none";
    }
    var update = function() {
        var up = document.getElementById("menu").getElementsByClassName("up")[0];
        var label = vv.view.list.hidden()? "list" : "up";
        if (up.textContent != label) {
            up.textContent = label;
        }
    }
    return {
        show_sub: show_sub,
        hide_sub: hide_sub,
        hidden_sub: hidden_sub,
        update: update,
    };
}());
vv.view.elapsed = (function() {
    var update = function() {
        data = vv.storage.control;
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
vv.control = (function() {
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
    var update_song = function() {
        get_request("api/songs/current", vv.storage.current_last_modified, function(ret, modified) {
            if (ret["errors"] == null) {
                vv.storage.current = ret["data"];
                vv.storage.current_last_modified = modified;
                vv.view.main.update();
                if (vv.model.list.rootname() != "root") {
                    vv.model.list.abs(ret["data"]);
                }
                // update elapsed tag
                vv.view.list.update();
            }
        });
    };

    var update_status = function() {
        get_request("api/control", vv.storage.control_last_modified, function(ret, modified) {
            if (ret["errors"] == null) {
                vv.storage.control = ret["data"];
                vv.storage.control_last_modified = modified;
                vv.view.elapsed.update();
            }
        });
        vv.view.elapsed.update();
    };

    var update_library = function() {
        get_request("api/library", vv.storage.library_last_modified, function(ret, modified) {
            if (ret["errors"] == null) {
                vv.model.list.update(ret["data"]);
                vv.storage.library_last_modified = modified;
            }
        });
    };

    var prev = function() {
        get_request("api/control?action=prev", "");
    }

    var play_pause = function() {
        var state = vv.obj.getOrElse(vv.storage.control, "state", "stopped");
        var action = state == "play" ? "pause" : "play";
        get_request("api/control?action="+action, "");
    }

    var next = function() {
        get_request("api/control?action=next", "");
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

    var init = function() {
        document.body.addEventListener('click', function(e) {
            vv.view.menu.hide_sub();
        });
        menu.getElementsByClassName("up")[0].addEventListener('click', function(e) {
            if (vv.view.main.hidden()) {
                vv.model.list.up();
            } else {
                vv.model.list.abs(vv.storage.current);
            }
            vv.view.main.hide();
            vv.view.list.update();
            vv.view.list.show();
            vv.view.menu.update();
            e.stopPropagation();
        });
        menu.getElementsByClassName("back")[0].addEventListener('click', function(e) {
            vv.view.list.hide();
            vv.view.main.show();
            vv.view.menu.update();
            e.stopPropagation();
        });
        menu.getElementsByClassName("menu")[0].addEventListener('click', function(e) {
            vv.view.menu.show_sub();
            e.stopPropagation();
        });
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

        var polling = function() {
            vv.control.update_song();
            vv.control.update_status();
            vv.control.update_library();
	    	setTimeout(polling, 1000);
        }
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
        update_song: update_song,
        update_status: update_status,
        update_library: update_library,
        prev: prev,
        play_pause: play_pause,
        next: next,
        play: play,
        start: start,
    };
}());

vv.control.start();
