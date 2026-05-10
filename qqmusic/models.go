package qqmusic

// SearchAPIResponse wraps the QQ Music search response.
type SearchAPIResponse struct {
	Code int              `json:"code"`
	Data SearchResultData `json:"data"`
}

// SearchResultData contains the search result body.
type SearchResultData struct {
	Body SearchBody `json:"body"`
}

// SearchBody holds the actual search content.
type SearchBody struct {
	Song struct {
		List     []QQSearchSong `json:"list"`
		Total    int            `json:"total"`
		TotalNum int            `json:"totalnum"`
		Num      int            `json:"num"`
	} `json:"song"`
	ItemSong []QQSearchSong `json:"item_song"`
	Total    int            `json:"total"`
	TotalNum int            `json:"totalnum"`
	Sum      int            `json:"sum"`
}

// QQSearchSong is a song entry in QQ Music search results.
type QQSearchSong struct {
	Mid      string        `json:"mid"`
	ID       int           `json:"id"`
	Name     string        `json:"name"`
	Title    string        `json:"title"`
	Singer   []QQSinger    `json:"singer"`
	Album    QQSearchAlbum `json:"album"`
	Interval int           `json:"interval"` // seconds
	Pay      struct {
		PayPlay int `json:"payplay"`
	} `json:"pay"`
}

// QQSinger is a singer from QQ Music responses.
type QQSinger struct {
	Mid  string `json:"mid"`
	Name string `json:"name"`
}

// QQSearchAlbum is an album entry in search results.
type QQSearchAlbum struct {
	Mid  string `json:"mid"`
	Name string `json:"name"`
}

// SongDetailResponse wraps song detail from the batch API.
type SongDetailResponse struct {
	Code int            `json:"code"`
	Data SongDetailData `json:"data"`
}

// SongDetailData contains track info.
type SongDetailData struct {
	TrackInfo QQTrackInfo `json:"track_info"`
}

// QQTrackInfo holds full track metadata.
type QQTrackInfo struct {
	ID       int         `json:"id"`
	Mid      string      `json:"mid"`
	Name     string      `json:"name"`
	Singer   []QQSinger  `json:"singer"`
	Album    QQAlbumInfo `json:"album"`
	Interval int         `json:"interval"`
	Pay      struct {
		PayPlay int `json:"payplay"`
	} `json:"pay"`
}

// QQAlbumInfo holds album metadata.
type QQAlbumInfo struct {
	ID   int    `json:"id"`
	Mid  string `json:"mid"`
	Name string `json:"name"`
}

// AlbumDetailResponse wraps album detail from the batch API.
type AlbumDetailResponse struct {
	Code int           `json:"code"`
	Data QQAlbumDetail `json:"data"`
}

// QQAlbumDetail contains full album metadata.
type QQAlbumDetail struct {
	BasicInfo struct {
		AlbumMID    string `json:"albumMid"`
		AlbumName   string `json:"albumName"`
		PublishDate string `json:"publishDate"`
		Desc        string `json:"desc"`
	} `json:"basicInfo"`
	Singer struct {
		SingerList []QQSinger `json:"singerList"`
	} `json:"singer"`
}

// AlbumSongListResponse wraps album songs.
type AlbumSongListResponse struct {
	Code int               `json:"code"`
	Data AlbumSongListData `json:"data"`
}

// AlbumSongListData contains album track list.
type AlbumSongListData struct {
	SongList []AlbumSongItem `json:"songList"`
}

// AlbumSongItem holds a song in an album list.
type AlbumSongItem struct {
	SongInfo QQTrackInfo `json:"songInfo"`
}

// SingerSongListResponse wraps singer songs.
type SingerSongListResponse struct {
	Code int                `json:"code"`
	Data SingerSongListData `json:"data"`
}

// SingerSongListData contains singer track list.
type SingerSongListData struct {
	SongList []AlbumSongItem `json:"songList"`
}

// PlaylistResponse wraps playlist detail.
type PlaylistResponse struct {
	Code int          `json:"code"`
	Data PlaylistData `json:"data"`
}

// PlaylistData contains playlist metadata and tracks.
type PlaylistData struct {
	Info struct {
		ID          int    `json:"dissid"`
		ID2         int    `json:"id"`
		Title       string `json:"dissname"`
		Title2      string `json:"title"`
		CoverURL    string `json:"logo"`
		CoverURL2   string `json:"picurl"`
		HostNick    string `json:"nickname"`
		HostNick2   string `json:"host_nick"`
		Description string `json:"desc"`
		SongCount   int    `json:"songnum"`
	} `json:"dissinfo"`
	DirInfo struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		CoverURL    string `json:"picurl"`
		HostNick    string `json:"host_nick"`
		Description string `json:"desc"`
		SongCount   int    `json:"songnum"`
		Creator     struct {
			Nick string `json:"nick"`
		} `json:"creator"`
	} `json:"dirinfo"`
	SongCount int           `json:"songlist_size"`
	SongList  []QQTrackInfo `json:"songlist"`
}

// LyricAPIResponse wraps the primary lyric endpoint.
type LyricAPIResponse struct {
	Code int       `json:"code"`
	Data LyricData `json:"data"`
}

// LyricData contains lyric strings from the API.
type LyricData struct {
	Lyric string `json:"lyric"` // encrypted
	Trans string `json:"trans"` // encrypted translation
	Roma  string `json:"roma"`  // encrypted romanization
}

// RecommendResponse wraps the recommendation response.
type RecommendResponse struct {
	Code          int           `json:"code"`
	RecomPlaylist RecomPlaylist `json:"recomPlaylist"`
}

// RecomPlaylist contains recommended playlist data.
type RecomPlaylist struct {
	Code int `json:"code"`
	Data struct {
		VHot []QQPlaylistItem `json:"v_hot"`
	} `json:"data"`
}

// QQPlaylistItem is a playlist entry.
type QQPlaylistItem struct {
	ContentID int    `json:"content_id"`
	Title     string `json:"title"`
	Cover     string `json:"cover"`
	Username  string `json:"username"`
	SongCount int    `json:"song_num"`
}
