package vv

// Tree is a vv playlist view definition.
type Tree map[string]*TreeNode

// TreeNode represents one of smart playlist node.
type TreeNode struct {
	Sort []string    `json:"sort"`
	Tree [][2]string `json:"tree"`
}

var (
	// DefaultTree is a default Tree for HTMLConfig.
	DefaultTree = Tree{
		"AlbumArtist": {
			Sort: []string{"AlbumArtist", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"AlbumArtist", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
		"Album": {
			Sort: []string{"AlbumArtist-Date-Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"AlbumArtist-Date-Album", "album"}, {"Title", "song"}},
		},
		"Artist": {
			Sort: []string{"Artist", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Artist", "plain"}, {"Title", "song"}},
		},
		"Genre": {
			Sort: []string{"Genre", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Genre", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
		"Date": {
			Sort: []string{"Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Date", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
		"Composer": {
			Sort: []string{"Composer", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Composer", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
		"Performer": {
			Sort: []string{"Performer", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Performer", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
	}
	// DefaultTreeOrder is a default TreeOrder for HTMLConfig.
	DefaultTreeOrder = []string{"AlbumArtist", "Album", "Artist", "Genre", "Date", "Composer", "Performer"}
)
