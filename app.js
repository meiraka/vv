
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

var song_tree_list = function(tree) {
    if (!tree) {
        tree = JSON.parse(sessionStorage.tree);
    }
    var ls = [];
    if (tree.length == 0) {
        return _song_tree_list_root();
    } else {
        return _song_tree_list_child(tree);
    }
};
var song_tree_down = function(key, value) {
    var tree = JSON.parse(sessionStorage.tree);
    tree.push([key, value]);
    sessionStorage.tree = JSON.stringify(tree);
};
var song_tree_abs = function(song) {
    var tree = JSON.parse(sessionStorage.tree);
    var i, root, key, selected;
    if (tree.length != 0) {
        tree = [tree[0]];
        root = tree[0][1];
        selected = TREE[root]["tree"];
        for (i in selected) {
            if (i == selected.length - 1) {
                break;
            }
            key = selected[i][0];
            tree.push([key, songGet(song, key)]);
        }
        sessionStorage.tree = JSON.stringify(tree);
    }
};
var song_tree_up = function() {
    var tree = JSON.parse(sessionStorage.tree)
    if (tree.length > 0) {
        tree.pop();
    }
    sessionStorage.tree = JSON.stringify(tree)
};
var _song_tree_list_root = function() {
    var ret = [];
    var rootname = "";
    for (rootname in TREE) {
        ret.push({"root": rootname});
    }
    return ["root", ret, "plain"];
}
var _song_tree_list_child = function(tree) {
    var root = tree[0][1],
        library = JSON.parse(sessionStorage["library_" + root]),
        filters = {},
        key = TREE[root]["tree"][tree.length - 1][0],
        style = TREE[root]["tree"][tree.length - 1][1],
        song = {};
    for (leef in tree) {
        if (leef == 0) { continue; }
        filters[tree[leef][0]] = tree[leef][1];
    }
    library = filterSongs(library, filters);
    library = uniqSongs(library, key);
    return [key, library, style];
};

var MainView = function() {
    var mainview = function() {};
    var p = mainview.prototype;
    p.show_current = function() {
        $("#list").hide();
        $("#current").show();
    }
    p.show_list = function() {
        $("#current").hide();
        p.update_tree();
        $("#list").show();
    }
    p.up_list = function() {
        if ($("#current").css("display") == "none") {
            song_tree_up();
        }
        p.update_tree();
    }

    p.update_tree = function() {
        var current = {};
        var control = {};
        var tree = JSON.parse(sessionStorage.tree);
        var key, songs, style;
        var song = {};
        var root = "";
        // [key, songs, style] = song_tree_list(tree);
        var keysongs = song_tree_list(tree);
        key = keysongs[0];
        songs = keysongs[1];
        style = keysongs[2];
        if (style == "song") {
            current = JSON.parse(sessionStorage.current);
            control = JSON.parse(sessionStorage.control);
        }
        if (tree.length != 0) {
            root = tree[0][1];
        }
        $("#list ol").empty();
        for (i in songs) {
            song = songs[i];
            $("#list ol").append("<li class="+style+"></li>");
            var added = $("#list ol li:last-child");
            added.attr("key", songGet(song, key));
            added.attr("uri", song["file"]);
            if (style == "song") {
                added.append("<span class=track>"+songGet(song, "TrackNumber")+"</span>");
                added.append("<span class=title>"+songGet(song, "Title")+"</span>");
                if (songGet(song, "Artist") != songGet(song, "AlbumArtist")) {
                    added.append("<span class=artist>"+songGet(song, "Artist")+"</span>");
                }
                if (songGet(song, "file") == songGet(current, "file")) {
                    added.append("<span class=elapsed>"+songGet(song, "Length")+"</span>");
                    added.append("<span class=length_separator>/</span>");
                }
                added.append("<span class=length>"+songGet(song, "Length")+"</span>");
            } else if (style == "album") {
                added.append("<span class=album>"+songGet(song, "Album")+"</span>");
                added.append("<span class=albumartist>"+songGet(song, "AlbumArtist")+"</span>");
            } else {
                added.text(songGet(song, key));
            }
        }
        $("#list ol li").bind("click", function() {
            var value = $(this).attr("key"),
                uri = $(this).attr("uri");
            if (tree.length == 0 || tree.length != TREE[root]["tree"].length) {
                song_tree_down(key, value);
                p.show_list();
            } else {
                $.ajax({
                    type: "POST",
                    url: "/api/songs",
                    contentType: 'application/json',
                    data: JSON.stringify(
                            {"action": "sort",
                             "keys": TREE[root]["sort"],
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
    return mainview;
}();


var update_song_request = function() {
    $.ajax({
        type: "GET",
        url: "api/songs/current",
        ifModified: true,
		dataType: "json",
        success: function(data, status) {
			if (status == "success" && data["errors"] == null) {
                sessionStorage.current = JSON.stringify(data["data"])
                $("#current .title").text(data["data"]["Title"])
                $("#current .artist").text(data["data"]["Artist"])
                song_tree_abs(data["data"]);
                mainview.update_tree();
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
                sessionStorage.control = JSON.stringify(data["data"])
                refreash_control_data();
			}
		},
    })
    refreash_control_data();
};

var refreash_control_data = function() {
    data = JSON.parse(sessionStorage.control)
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
                sessionStorage["library_AlbumArtist"] = JSON.stringify(
                    sortSongs(data["data"], TREE["AlbumArtist"]["sort"]));
                sessionStorage["library_Genre"] = JSON.stringify(
                    sortSongs(data["data"], TREE["Genre"]["sort"]));
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

function songTag(song, keys, other) {
    for (i in keys) {
        var key = keys[i];
        if (key in song) {
            return song[key];
        }
    }
    return other;
}

function songGet(song, key) {
    return songGetOrElse(song, key, '[no ' + key + ']');
}

function songGetOrElse(song, key, other) {
   if (key in song) {
       return song[key];
   } else if (key == "AlbumSort") {
       return songTag(song, ["Album"], other);
   } else if (key == "ArtistSort") {
       return songTag(song, ["Artist"], other);
   } else if (key == "AlbumArtist") {
       return songTag(song, ["Artist"], other);
   } else if (key == "AlbumArtistSort") {
       return songTag(song, ["AlbumArtist", "Artist"], other);
   } else if (key == "AlbumSort") {
       return songTag(song, ["Album"], other);
   } else {
       return other;
   }
}
function songString(song, keys) {
    var sortkey = '';
    for (i in keys) {
        sortkey += songGetOrElse(song, keys[i], ' ')
    }
    return sortkey;
}

function sortSongs(songs, keys) {
    return songs.map(function(song) {
        return [song, songString(song, keys)]
    }).sort(function (a, b) {
        if (a[1] < b[1]) { return -1; } else { return 1; };
    }).map(function(s) { return s[0]; });
}
function uniqSongs(songs, key) {
    return songs.filter(function (song, i , self) {
        if (i == 0) {
            return true;
        } else if (songGetOrElse(song, key, ' ') != songGetOrElse(self[i - 1], key, ' ')) {
            return true;
        } else {
            return false;
        }
    });
}
function filterSongs(songs, filters) {
    return songs.filter(function(song, i, self) {
        for (f in filters) {
            if (songGet(song, f) != filters[f]) {
                return false;
            }
        }
        return true;
    });
}

function getOrElse(m, k, v) {
    return k in m? m[k] : v;
}

$(document).ready(function(){
    mainview = new MainView();
    $("#menu .up").bind("click", function() {
        mainview.up_list();
        mainview.show_list();
        return false;
    });
    $("#menu .back").bind("click", function() {
        mainview.show_current();
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
        var state = getOrElse(JSON.parse(sessionStorage.control), "state", "stopped");
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
