var vv = vv || {
    song: {},
    songs: {},
    storage: {},
    model: {list: {}},
    view: {main: {}, list: {}},
};
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
    var control = [];
    var library = {
        "AlbumArtist": [],
         "Genre": [],
    }

    var save = function() {
        localStorage.tree = JSON.stringify(tree);
    }
    var load = function() {
        if (localStorage.tree) {
            tree = JSON.parse(localStorage.tree);
        }
    }
    load();
    return {
        tree: tree,
        current: current,
        control: control,
        library: library,
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
        "date": {
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
        vv.storage.library["AlbumArtist"] = vv.songs.sort(data, TREE["AlbumArtist"]["sort"]);
        vv.storage.library["Genre"] = vv.songs.sort(data, TREE["Genre"]["sort"]);
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
        sortkeys, sortkeys,
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
    return {
        show: show,
        hide: hide,
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
    return {
        show: show,
        hide: hide,
    };
}());


var update_tree = function() {
    var key, songs, style, type;
    var song = {};
    var root = "";
    var keysongs = vv.model.list.list();
    key = keysongs[0];
    songs = keysongs[1];
    style = keysongs[2];
    type = keysongs[3];
    if (vv.storage.tree.length != 0) {
        root = vv.storage.tree[0][1];
    }
    $("#list ol").empty();
    for (i in songs) {
        song = songs[i];
        $("#list ol").append("<li class="+style+"></li>");
        var added = $("#list ol li:last-child");
        added.attr("key", vv.song.get(song, key));
        added.attr("uri", song["file"]);
        if (style == "song") {
            added.append("<span class=track>"+vv.song.get(song, "TrackNumber")+"</span>");
            added.append("<span class=title>"+vv.song.get(song, "Title")+"</span>");
            if (vv.song.get(song, "Artist") != vv.song.get(song, "AlbumArtist")) {
                added.append("<span class=artist>"+vv.song.get(song, "Artist")+"</span>");
            }
            if (vv.song.get(song, "file") == vv.song.get(vv.storage.current, "file")) {
                added.append("<span class=elapsed>"+vv.song.get(song, "Length")+"</span>");
                added.append("<span class=length_separator>/</span>");
            }
            added.append("<span class=length>"+vv.song.get(song, "Length")+"</span>");
        } else if (style == "album") {
            added.append("<span class=date>"+vv.song.get(song, "Date")+"</span>");
            added.append("<span class=album>"+vv.song.get(song, "Album")+"</span>");
            added.append("<span class=albumartist>"+vv.song.get(song, "AlbumArtist")+"</span>");
        } else {
            added.text(vv.song.get(song, key));
        }
    }
    $("#list ol li").bind("click", function() {
        var value = $(this).attr("key"),
            uri = $(this).attr("uri");
        if (type == "dir") {
            vv.model.list.down(value);
            update_tree();
        } else {
            $.ajax({
                type: "POST",
                url: "/api/songs",
                contentType: 'application/json',
                data: JSON.stringify(
                        {"action": "sort",
                         "keys": vv.model.list.sortkeys(),
                         "uri": uri
                        }),
                cache: false,
                success: function(data, status) {
			        if (status == "success" && data["errors"] == null) {
			        }
		        },
            });
        }
        return false;
    });
};


var update_song_request = function() {
    $.ajax({
        type: "GET",
        url: "api/songs/current",
        ifModified: true,
		dataType: "json",
        success: function(data, status) {
			if (status == "success" && data["errors"] == null) {
                vv.storage.current = data["data"];
                $("#current .title").text(data["data"]["Title"])
                $("#current .artist").text(data["data"]["Artist"])
                var key;
                $("#current .detail").empty();
                for (key in data["data"]) {
                    if (key == "Title" || key == "Artist") {
                        continue;
                    }
                    $("#current .detail").append("<li>" + key + ": " + data["data"][key] + "</li>");
                }
                if (vv.model.list.rootname() != "root") {
                    vv.model.list.abs(data["data"]);
                }
                // update elapsed tag
                update_tree();
			}
		},
    })
};

var update_control_data_request = function() {
    $.ajax({
        type: "GET",
        url: "api/control",
        ifModified: true,
		dataType: "json",
        success: function(data, status) {
			if (status == "success" && data["errors"] == null) {
                vv.storage.control = data["data"];
                refreash_control_data();
			}
		},
    })
    refreash_control_data();
};

var refreash_control_data = function() {
    data = vv.storage.control
    if ('state' in data) {
        var elapsed = parseInt(data["song_elapsed"] * 1000)
        var current = elapsed
        var last_modified = parseInt(data["last_modified"] * 1000)
        var date = new Date()
        if (data["state"] == "play") {
            current += date.getTime() - last_modified
        }
        var label = parseSongTime(current / 1000);
        if ($(".elapsed").text() != label) {
            $(".elapsed").text(label);
        }
    }
};

var update_library_request = function() {
    $.ajax({
        type: "GET",
        url: "api/library",
        ifModified: true,
		dataType: "json",
        success: function(data, status) {
			if (status == "success" && data["errors"] == null) {
                vv.model.list.update(data["data"]);
			}
		},
    })
}

function parseSongTime(val) {
    var current = parseInt(val)
    var min = parseInt(current / 60)
    var sec = current % 60
    return min + ':' + ("0" + sec).slice(-2)
}


function getOrElse(m, k, v) {
    return k in m? m[k] : v;
}


$(document).ready(function(){
    $("#menu .up").bind("click", function() {
        if ($("#current").css("display") == "none") {
            vv.model.list.up();
        } else {
            vv.model.list.abs(vv.storage.current);
        }
        vv.view.main.hide();
        update_tree();
        vv.view.list.show();
        return false;
    });
    $("#menu .back").bind("click", function() {
        vv.view.list.hide();
        vv.view.main.show();
        return false;
    });
    $("#menu .reset").bind("click", function() {
        sessionStorage.tree = "[]";
        sessionStorage.current = "{}";
        sessionStorage.control = "{}";
        sessionStorage.playlist = "[]";
        sessionStorage.library = "[]";
        sessionStorage.library_AlbumArtist = "[]";
        sessionStorage.library_Genre = "[]";
        return false;
    });
    $("#playback .prev").bind("click", function() {
        $.ajax({
            type: "GET",
            url: "/api/songs/current",
            data: {"action": "prev"},
            cache: false
        })
        return false;
    });
    $("#playback .play").bind("click", function() {
        var state = getOrElse(vv.storage.control, "state", "stopped");
        var action = state == "play" ? "pause" : "play"
        $.ajax({
            type: "GET",
            url: "/api/songs/current",
            data: {"action": action},
            cache: false
        })
        return false;
    });
    $("#playback .next").bind("click", function() {
        $.ajax({
            type: "GET",
            url: "/api/songs/current",
            data: {"action": "next"},
            cache: false
        })
        return false;
    });

    function polling() {
        update_song_request()
        update_control_data_request()
        update_library_request()
		setTimeout(polling, 1000);
    }
	polling();
});
