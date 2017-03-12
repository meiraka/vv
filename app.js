$(document).ready(function(){
    $("#playback .prev").bind("click", function() {
        $.ajax({
            type: "GET",
            url: "/api/prev",
            cache: false
        })
        return false;
    })
});
