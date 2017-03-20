var MainView = function() {
    TREE = {
        "AlbumArtist": {
            "sort":
                ["AlbumArtist", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"],
            "tree":
                [["AlbumArtist", "plain"],
                 ["Album", "plain"],
                 ["Title", "plain"]
                ],
        },
        "Genre": {
            "sort":
                ["Genre", "Album", "DiscNumber", "TrackNumber", "Title", "file"],
            "tree":
                [["Genre", "plain"],
                 ["Album", "plain"],
                 ["Title", "plain"],
                ]
        },
        "date": {
            "sort":
                ["Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"],
            "tree":
                [["Date", "plain"],
                 ["Album", "plain"],
                 ["Title", "plain"],
                ]
        }
    }
    var mainview = function() {};
    var p = mainview.prototype;
    p.show_current = function() {
        $("#current").show();
    }
    p.hide_current = function() {
        $("#current").hide();
    }
    p.show_list = function() {
        $("#list ol").empty();
        var tree = JSON.parse(sessionStorage.tree);
        if (tree.length == 0) {
            p.update_root(tree);
        } else {
            p.update_child(tree);
        }
        $("#list").show();
    }
    p.hide_list = function() {
        $("#list").hide();
    }
    p.up_list = function() {
        if ($("#current").css("display") == "none") {
            var data = JSON.parse(sessionStorage.tree)
            if (data.length > 0) {
                data.pop();
            }
            sessionStorage.tree = JSON.stringify(data)
        }
    }
    p.update_root = function(tree) {
        var rootname = "";
        var value = "";
        var ol = $("#list ol");
        for (rootname in TREE) {
            ol.append("<li key="+rootname+">" + rootname + "</li>");
        }
        $("#list ol li").bind("click", function() {
            var rootname = $(this).attr("key");
            sessionStorage.tree = JSON.stringify([["root", rootname]]);
            p.show_list();
            return false;
        });
    }
    p.update_child = function(tree) {
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
        for (i in library) {
            song = library[i];
            $("#list ol").append("<li></li>");
            var added = $("#list ol li:last-child");
            added.text(songGet(song, key));
            added.attr("key", songGet(song, key));
            added.attr("uri", song["file"]);
        }
        $("#list ol li").bind("click", function() {
            var value = $(this).attr("key"),
                uri = $(this).attr("uri");
            if (tree.length == TREE[root]["tree"].length) {
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
                            p.hide_list();
                            p.show_current();
				        }
			        },
                });
            } else {
                tree.push([key, value]);
                sessionStorage.tree = JSON.stringify(tree);
                p.show_list();
            }
            return false;
        });
    };
    return mainview;
}();


var Mpd = (function() {
    var mpd = function() {
        sessionStorage.tree = JSON.stringify([])
    };

    var p = mpd.prototype;

    p.update_song_request = function() {
        $.ajax({
            type: "GET",
            url: "api/songs/current",
            ifModified: true,
			dataType: "json",
            success: function(data, status) {
				if (status == "success" && data["errors"] == null) {
                    p.update_song(data["data"])
				}
			},
        })
    };
    p.update_song = function(data) {
        sessionStorage.current = JSON.stringify(data)
        $("#current .title").text(data["Title"])
        $("#current .artist").text(data["Artist"])
    };


    p.update_control_data_request = function() {
        $.ajax({
            type: "GET",
            url: "api/control",
            ifModified: true,
			dataType: "json",
            success: function(data, status) {
				if (status == "success" && data["errors"] == null) {
                    sessionStorage.control = JSON.stringify(data["data"])
                    p.refreash_control_data();
				}
			},
        })
        p.refreash_control_data();
    };

    p.refreash_control_data = function() {
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
            if ($("#current .elapsed").text() != label) {
                $("#current .elapsed").text(label);
            }
        }
    };

    p.update_library_request = function() {
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
    return mpd;
})();

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
    mpc = new Mpd();
    $("#menu .up").bind("click", function() {
        mainview.hide_current();
        if ($("#current").css("display") == "none") {
            mainview.up_list();
        }
        mainview.show_list();
        return false;
    });
    $("#menu .back").bind("click", function() {
        mainview.hide_list();
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
        mpc.update_song_request()
        mpc.update_control_data_request()
        mpc.update_library_request()
		setTimeout(polling, 1000);
    }
	polling();
});
