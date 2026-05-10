package netease

// SearchResponse wraps the NetEase cloudsearch/pc response.
type SearchResponse struct {
	Code   int              `json:"code"`
	Result SearchResultData `json:"result"`
}

// SearchResultData holds the search result from cloudsearch/pc.
type SearchResultData struct {
	SongCount     int               `json:"songCount"`
	Songs         []NeteaseSong     `json:"songs"`
	AlbumCount    int               `json:"albumCount"`
	Albums        []NeteaseAlbum    `json:"albums"`
	ArtistCount   int               `json:"artistCount"`
	Artists       []NeteaseArtist   `json:"artists"`
	PlaylistCount int               `json:"playlistCount"`
	Playlists     []NeteasePlaylist `json:"playlists"`
}

// NeteaseSong represents a song from NetEase API responses.
type NeteaseSong struct {
	ID        int               `json:"id"`
	Name      string            `json:"name"`
	Artists   []NeteaseArtist   `json:"ar"`
	Artists2  []NeteaseArtist   `json:"artists"`
	Album     NeteaseAlbumBasic `json:"al"`
	Album2    NeteaseAlbumBasic `json:"album"`
	Duration  int               `json:"dt"`       // milliseconds
	Duration2 int               `json:"duration"` // milliseconds
	Fee       int               `json:"fee"`      // 0=free, 1=VIP, 4/8=only
	Privilege *PrivilegeInfo    `json:"privilege,omitempty"`
}

// NeteaseAlbumBasic contains minimal album info embedded in song objects.
type NeteaseAlbumBasic struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	PicURL  string `json:"picUrl"`
	PicURL2 string `json:"picUrl,omitempty"`
}

// NeteaseArtist represents an artist from NetEase responses.
type NeteaseArtist struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	PicURL string `json:"picUrl,omitempty"`
}

// NeteaseAlbum holds full album data from /album endpoint.
type NeteaseAlbum struct {
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	PicURL      string          `json:"picUrl"`
	Company     string          `json:"company,omitempty"`
	PublishTime int64           `json:"publishTime"` // milliseconds
	Description string          `json:"description,omitempty"`
	Artists     []NeteaseArtist `json:"artists"`
	Songs       []NeteaseSong   `json:"songs"`
	Size        int             `json:"size"`
}

// NeteasePlaylist holds playlist data.
type NeteasePlaylist struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	CoverImgURL string `json:"coverImgUrl"`
	Creator     struct {
		Nickname string `json:"nickname"`
	} `json:"creator"`
	Description string        `json:"description"`
	TrackCount  int           `json:"trackCount"`
	Tracks      []NeteaseSong `json:"tracks"`
}

// PrivilegeInfo contains song privilege (pay status, playability) info.
type PrivilegeInfo struct {
	ID  int `json:"id"`
	Fee int `json:"fee"`
	St  int `json:"st"` // -200 = unavailable
	Pl  int `json:"pl"` // play level
}

// SongDetailResponse wraps responses from /song/detail.
type SongDetailResponse struct {
	Code       int             `json:"code"`
	Songs      []NeteaseSong   `json:"songs"`
	Privileges []PrivilegeInfo `json:"privileges"`
}

// LyricResponse wraps responses from /song/lyric/v1.
type LyricResponse struct {
	Code    int       `json:"code"`
	Lrc     LyricText `json:"lrc"`
	TLyric  LyricText `json:"tlyric"`
	Klyric  LyricText `json:"klyric"`
	Romalrc LyricText `json:"romalrc"`
	Yrc     LyricText `json:"yrc"`
}

// LyricText holds a single lyric text with version.
type LyricText struct {
	Version int    `json:"version"`
	Lyric   string `json:"lyric"`
}

// PlaylistDetailResponse wraps /playlist/detail.
type PlaylistDetailResponse struct {
	Code     int             `json:"code"`
	Playlist NeteasePlaylist `json:"playlist"`
}

// ArtistResponse wraps /artist/head/info/get.
type ArtistResponse struct {
	Code int                 `json:"code"`
	Data NeteaseArtistDetail `json:"data"`
}

// NeteaseArtistDetail holds detailed artist info.
type NeteaseArtistDetail struct {
	Artist struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		PicURL    string `json:"picUrl"`
		BriefDesc string `json:"briefDesc"`
	} `json:"artist"`
}

// ArtistSongsResponse wraps artist top songs.
type ArtistSongsResponse struct {
	Code     int           `json:"code"`
	HotSongs []NeteaseSong `json:"songs"`
	Artist   NeteaseArtist `json:"artist"`
	Total    int           `json:"total"`
}

// RecommendSongsResponse wraps /recommend/songs.
type RecommendSongsResponse struct {
	Code int `json:"code"`
	Data struct {
		DailySongs []NeteaseSong `json:"dailySongs"`
	} `json:"data"`
}

// RegisterAnonResponse wraps anonymous registration.
type RegisterAnonResponse struct {
	Code   int    `json:"code"`
	UserID int    `json:"userId"`
	Cookie string `json:"cookie"`
}
