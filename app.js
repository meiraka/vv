var Mpd = (function() {
    TREE = {
        "albumartist": {
            "sort":
                ["albumartist", "date", "album", "disc", "trackno", "title"],
            "tree":
                [["albumartist", "plain"],
                 ["album", "plain"],
                 ["title", "plain"]
                ],
        },
        "genre": {
            "sort":
                ["genre", "album", "disc", "trackno", "title"],
            "tree":
                [["genre", "plain"],
                 ["album", "plain"],
                 ["title", "plain"],
                ]
        }
    }
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
        $("#current .title").text(data["title"])
        $("#current .artist").text(data["artist"])
    };


    p.update_control_data_request = function() {
        $.ajax({
            type: "GET",
            url: "api/control",
            ifModified: true,
			dataType: "json",
            success: function(data, status) {
				if (status == "success" && data["errors"] == null) {
                    p.update_control_data(data["data"])
				}
			},
        })
        p.refreash_control_data();
    };
    p.update_control_data = function(data) {
        sessionStorage.control = JSON.stringify(data)
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
                    sessionStorage.library = JSON.stringify(data["data"])
                    sessionStorage.library_albumartist = JSON.stringify(
                        sortSongs(data["data"], TREE["albumartist"]["sort"]));
                    sessionStorage.library_genre = JSON.stringify(
                        sortSongs(data["data"], TREE["genre"]["sort"]));
				}
			},
        })
    }

    p.show_current = function() {
        $("#list").hide();
        $("#current").show();
    }

    p.show_root = function() {
        if ($("#current").css("display") == "none") {
            var data = JSON.parse(sessionStorage.tree)
            if (data.length > 0) {
                data.pop();
            }
            sessionStorage.tree = JSON.stringify(data)
        }
        p.show_list();
    }

    p.show_list = function() {
        var tree = JSON.parse(sessionStorage.tree)
        $("#current").hide()
        $("#list ol").empty()
        if (tree.length == 0) {
            for (key in TREE) {
                $("#list ol").append("<li key="+key+">" + key + "</li>")
            }
            $("#list ol li").bind("click", function() {
                var key = $(this).attr("key");
                sessionStorage.tree = JSON.stringify([["root", key]]);
                p.show_list();
                return false;
            });
        } else {
            var root = tree[0][1];
            if (tree[0][1] == "albumartist") {
                var library = JSON.parse(sessionStorage.library_albumartist);
            } else {
                var library = JSON.parse(sessionStorage.library_genre);
            }
            var filters = {}
            for (leef in tree) {
                if (leef == 0) { continue; }
                filters[tree[leef][0]] = tree[leef][1];
            }
            library = library.filter(function(e, i, self) {
                for (f in filters) {
                    if (!(f in e && e[f] == filters[f])) {
                        return false;
                    }
                }
                return true;
            });
            var child = TREE[root]["tree"][tree.length - 1];
            var key = child[0];
            var style = child[1];
            library = uniqSongs(library, key);
            for (i in library) {
                var song = library[i];
                $("#list ol").append("<li></li>")
                var added = $("#list ol li:last-child")
                added.text(song[key])
                added.attr("key", song[key])
                added.attr("uri", song["file"])
            }
            $("#list ol li").bind("click", function() {
                var value = $(this).attr("key");
                if (tree.length == TREE[root]["tree"].length) {
                    var uri = $(this).attr("uri");
                    console.log(TREE[root]["sort"]);
                    console.log(uri);

                } else {
                    tree.push([key, value])
                    sessionStorage.tree = JSON.stringify(tree)
                    p.show_list();
                }
                return false;
            });
        }
        $("#list").show()
    }
    return mpd;
})();

function parseSongTime(val) {
    var current = parseInt(val)
    var min = parseInt(current / 60)
    var sec = current % 60
    return min + ':' + ("0" + sec).slice(-2)
}

function sortSongs(data, keys) {
    return data.map(function(song) {
        sortkey = '';
        for (i in keys) {
            sortkey += getOrElse(song, keys[i], 'no ' + keys[i]);
        }
        return [song, sortkey];
    }).sort(function (a, b) {
        if (a[1] < b[1]) { return -1; } else { return 1; };
    }).map(function(s) { return s[0]; });
}
function uniqSongs(data, key) {
    return data.filter(function (e, i , self) {
        if (i == 0) {
            return true;
        } else if (e[key] != self[i - 1][key]) {
            return true;
        } else {
            return false;
        }
    });
}

function getOrElse(m, k, v) {
    return k in m? m[k] : v;
}

$(document).ready(function(){
    mpc = new Mpd()
    $("#menu .up").bind("click", function() {
        mpc.show_root()
        return false;
    });
    $("#menu .back").bind("click", function() {
        mpc.show_current()
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
