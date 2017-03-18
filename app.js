var Mpd = (function() {
    var mpd = function() {
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
                    p.update_control_data(data["data"])
				}
			},
        })
        p.refreash_control_data(this.control_data);
    };
    p.update_control_data = function(data) {
        $('#current .elapsed').attr("state", data["state"])
        $('#current .elapsed').attr("elapsed", data["song_elapsed"])
        $('#current .elapsed').attr("last_modified", data["last_modified"])
        p.refreash_control_data();
    };
    p.refreash_control_data = function() {
        if ($('#current .elapsed').attr("last_modified")) {
            var state = $('#current .elapsed').attr("state")
            var elapsed = parseInt($('#current .elapsed').attr("elapsed") * 1000)
            var current = elapsed
            var last_modified = parseInt($('#current .elapsed').attr("last_modified") * 1000)
            var date = new Date()
            if (state == "play") {
                current += date.getTime() - last_modified
            }
            $("#current .elapsed").text(parseSongTime(current / 1000));
        } else {
        }
    };
    return mpd;
})();

function parseSongTime(val) {
    var current = parseInt(val)
    var min = parseInt(current / 60)
    var sec = current % 60
    return min + ':' + ("0" + sec).slice(-2)
}


$(document).ready(function(){
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
        var state = $('#current .elapsed').attr("state")
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

    mpc = new Mpd()
    function polling() {
        mpc.update_song_request()
        mpc.update_control_data_request()
		setTimeout(polling, 1000);
    }
	polling();
});
