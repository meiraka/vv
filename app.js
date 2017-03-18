var Mpd = (function() {
    TREE = [
        [["albumartist", ["albumartist"]],
         ["album", ["date", "album"]],
         ["title", ["track", "title"]]
        ],
        [["genre", ["genre"]],
         ["album", ["album"]],
         ["title", ["track", "title"]]
        ]
    ]
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
				}
			},
        })
    }
    p.show_root = function() {
        var data = JSON.parse(sessionStorage.tree)
        if (data.length > 0) {
            data.pop();
        }
        sessionStorage.tree = JSON.stringify(data)
        p.show_list();
    }
    p.show_list = function() {
        tree = JSON.parse(sessionStorage.tree)
        $("#current").hide()
        $("#list ol").empty()
        if (tree.length == 0) {
            for (i in TREE) {
                $("#list ol").append("<li index="+i+">" + TREE[i][0][0] + "</li>")
            }
            $("#list ol li").bind("click", function() {
                var index = $(this).attr("index");
                sessionStorage.tree = JSON.stringify([index]);
                p.show_list();
                return false;
            });
        } else {
            library = JSON.parse(sessionStorage.library);
            filters = {}
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
            child = TREE[tree[0]][tree.length - 1];
            labels = sortUniq(library, child[0], child[1]);
            for (i in labels) {
                $("#list ol").append("<li>" + labels[i][0] + "</li>")
            }
            $("#list ol li").bind("click", function() {
                var key = $(this).text();
                if (tree.length == TREE[tree[0]].length) {
                    console.log(TREE[tree[0]]);
                    console.log(key);
                } else {
                    tree.push([child[0], key])
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

function sortUniq(data, key, keys) {
    var labels = data.map(function(song) {
        sortkey = '';
        for (i in keys) {
            sortkey += getOrElse(song, keys[i], 'no ' + keys[i]);
        }
        return [getOrElse(song, key, 'no '+key), sortkey];
    }).sort(function (a, b) {
        if (a[1] < b[1]) { return -1; } else { return 1; };
    })
    return labels.filter(function (e, i , self) {
        if (i == 0) {
            return true;
        } else if (e[1] != self[i - 1][1]) {
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
    $("#menu .left").bind("click", function() {
        mpc.show_root()
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
