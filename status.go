package main

import (
	"github.com/fhs/gompd/mpd"
	"strconv"
)

/*PlayerStatus represents mpd status.*/
type PlayerStatus struct {
	Volume        int     `json:"volume"`
	Repeat        bool    `json:"repeat"`
	Random        bool    `json:"random"`
	Single        bool    `json:"single"`
	Consume       bool    `json:"consume"`
	State         string  `json:"state"`
	SongPos       int     `json:"song_pos"`
	SongElapsed   float32 `json:"song_elapsed"`
	LastModified  int64   `json:"last_modified"`
	UpdateLibrary bool    `json:"update_library"`
}

func convStatus(status mpd.Attrs, modified int64) PlayerStatus {
	volume, err := strconv.Atoi(status["volume"])
	if err != nil {
		volume = -1
	}
	repeat := status["repeat"] == "1"
	random := status["random"] == "1"
	single := status["single"] == "1"
	consume := status["consume"] == "1"
	state := status["state"]
	if state == "" {
		state = "stopped"
	}
	songpos, err := strconv.Atoi(status["song"])
	if err != nil {
		songpos = 0
	}
	elapsed, err := strconv.ParseFloat(status["elapsed"], 64)
	if err != nil {
		elapsed = 0.0
	}
	_, found := status["updating_db"]
	updateLibrary := found
	return PlayerStatus{
		volume,
		repeat,
		random,
		single,
		consume,
		state,
		songpos,
		float32(elapsed),
		modified,
		updateLibrary,
	}

}
