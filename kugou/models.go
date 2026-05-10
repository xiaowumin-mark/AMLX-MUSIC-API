package kugou

import (
	"encoding/json"
	"strconv"
)

type flexibleString string

func (s *flexibleString) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = flexibleString(str)
		return nil
	}
	var num json.Number
	if err := json.Unmarshal(data, &num); err == nil {
		*s = flexibleString(num.String())
		return nil
	}
	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		*s = flexibleString(strconv.FormatInt(int64(f), 10))
		return nil
	}
	return nil
}

type flexibleInt int

func (i *flexibleInt) UnmarshalJSON(data []byte) error {
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*i = flexibleInt(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		v, _ := strconv.Atoi(s)
		*i = flexibleInt(v)
		return nil
	}
	return nil
}

// SearchSongResponse wraps the /v3/search/song response.
type SearchSongResponse struct {
	Status int            `json:"status"`
	Data   SearchSongData `json:"data"`
}

// SearchSongData contains search result details.
type SearchSongData struct {
	Info  SearchInfo  `json:"info"`
	Lists []KuGouSong `json:"lists"`
	Total int         `json:"total"`
}

// SearchInfo holds pagination info.
type SearchInfo struct {
	Total    int `json:"total"`
	Page     int `json:"page"`
	Pagesize int `json:"pagesize"`
}

// KuGouSong is a song entry in search results.
type KuGouSong struct {
	Hash         string         `json:"hash"`
	FileHash     string         `json:"FileHash"`
	AudioID      int            `json:"audio_id"`
	AudioID2     int            `json:"Audioid"`
	SongName     string         `json:"songname"`
	OriSongName  string         `json:"OriSongName"`
	AlbumID      flexibleString `json:"album_id"`
	AlbumID2     string         `json:"AlbumID"`
	AlbumName    string         `json:"album_name"`
	AlbumName2   string         `json:"AlbumName"`
	AlbumAudioID flexibleInt    `json:"album_audio_id"`
	SingerName   string         `json:"singername"`
	Singers      []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"Singers"`
	SingerID  string `json:"singer_id"`
	Duration  int    `json:"duration"`
	Duration2 int    `json:"Duration"`
	FeeType   int    `json:"fee_type"`
}

// SongDetailResponse wraps /v2/get_res_privilege/lite.
type SongDetailResponse struct {
	Status int             `json:"status"`
	Data   []SongPrivilege `json:"data"`
}

// SongPrivilege contains song privilege info.
type SongPrivilege struct {
	Hash       string         `json:"hash"`
	AudioID    int            `json:"audio_id"`
	AudioName  string         `json:"audio_name"`
	Name       string         `json:"name"`
	SongName   string         `json:"songname"`
	SingerName string         `json:"singername"`
	AlbumID    flexibleString `json:"album_id"`
	AlbumName  string         `json:"album_name"`
	FeeType    int            `json:"fee_type"`
	AlbumImg   string         `json:"album_img"`
	Duration   int            `json:"duration"`
	Info       struct {
		Duration int    `json:"duration"`
		Image    string `json:"image"`
	} `json:"info"`
}

// LyricSearchResponse wraps lyric search results.
type LyricSearchResponse struct {
	Status     int              `json:"status"`
	Candidates []LyricCandidate `json:"candidates"`
}

// LyricCandidate is a matching lyric entry.
type LyricCandidate struct {
	ID        string `json:"id"`
	AccessKey string `json:"accesskey"`
	Score     int    `json:"score"`
}

// LyricDownloadResponse wraps the lyric download.
type LyricDownloadResponse struct {
	Status  int    `json:"status"`
	Content string `json:"content"`
	Format  string `json:"fmt"`
}

// AlbumDetailResponse wraps /kmr/v2/albums.
type AlbumDetailResponse struct {
	Status int              `json:"status"`
	Data   []KuGouAlbumInfo `json:"data"`
}

// KuGouAlbumInfo contains album detail.
type KuGouAlbumInfo struct {
	AlbumID      flexibleString `json:"album_id"`
	AlbumName    string         `json:"album_name"`
	PublishDate  string         `json:"publish_date"`
	Intro        string         `json:"intro"`
	Language     string         `json:"language"`
	SizableCover string         `json:"sizable_cover"`
	AuthorName   string         `json:"author_name"`
	Authors      []struct {
		AuthorID   string `json:"author_id"`
		AuthorName string `json:"author_name"`
	} `json:"authors"`
}

// AlbumSongsResponse wraps /v1/album_audio/lite.
type AlbumSongsResponse struct {
	Status    int            `json:"status"`
	ErrorCode int            `json:"error_code"`
	Data      AlbumSongsData `json:"data"`
}

type AlbumSongsData struct {
	Songs []KuGouAlbumSong `json:"songs"`
}

type KuGouAlbumSong struct {
	Base struct {
		SongName   string `json:"audio_name"`
		SingerName string `json:"author_name"`
	} `json:"base"`
	AudioInfo struct {
		Hash     string `json:"hash"`
		Hash128  string `json:"hash_128"`
		Hash320  string `json:"hash_320"`
		HashFlac string `json:"hash_flac"`
		Duration int    `json:"duration"`
	} `json:"audio_info"`
}

// KuGouAudioInfo contains audio file info.
type KuGouAudioInfo struct {
	Hash         string         `json:"hash"`
	AudioID      int            `json:"audio_id"`
	AudioName    string         `json:"audio_name"`
	SongName     string         `json:"songname"`
	AuthorName   string         `json:"author_name"`
	AlbumID      flexibleString `json:"album_id"`
	AlbumName    string         `json:"album_name"`
	AlbumAudioID flexibleInt    `json:"album_audio_id"`
	SingerName   string         `json:"singername"`
	SingerID     string         `json:"singer_id"`
	Duration     int            `json:"duration"`
	TimeLength   int            `json:"timelength"`
	TimeLength2  int            `json:"time_length"`
	FeeType      int            `json:"fee_type"`
	SingerInfo   []struct {
		ID   flexibleString `json:"id"`
		Name string         `json:"name"`
	} `json:"singerinfo"`
}

// SingerSongsResponse wraps /kmr/v1/audio_group/author.
type SingerSongsResponse struct {
	Status    int              `json:"status"`
	ErrorCode int              `json:"error_code"`
	Data      []KuGouAudioInfo `json:"data"`
}

// PlaylistDetailResponse wraps /v3/get_list_info.
type PlaylistDetailResponse struct {
	Status int            `json:"status"`
	Data   []PlaylistInfo `json:"data"`
}

// PlaylistInfo contains playlist metadata and tracks.
type PlaylistInfo struct {
	GlobalCollectionID string          `json:"global_collection_id"`
	Name               string          `json:"name"`
	Intro              string          `json:"intro"`
	ImgURL             string          `json:"img_url"`
	Pic                string          `json:"pic"`
	Username           string          `json:"username"`
	ListCreateUsername string          `json:"list_create_username"`
	Count              int             `json:"count"`
	MusicList          []KuGouSongItem `json:"music_list"`
}

// KuGouSongItem is a song in a playlist.
type KuGouSongItem struct {
	Hash         string         `json:"hash"`
	AudioID      int            `json:"audio_id"`
	SongName     string         `json:"songname"`
	AlbumID      flexibleString `json:"album_id"`
	AlbumName    string         `json:"album_name"`
	AlbumAudioID flexibleInt    `json:"album_audio_id"`
	SingerName   string         `json:"singername"`
	SingerID     string         `json:"singer_id"`
	Duration     int            `json:"duration"`
	FeeType      int            `json:"fee_type"`
	TimeLen      int            `json:"timelen"`
	Name         string         `json:"name"`
	SingerInfo   []struct {
		ID   flexibleString `json:"id"`
		Name string         `json:"name"`
	} `json:"singerinfo"`
}

// PlaylistSongsResponse wraps /pubsongs/v2/get_other_list_file_nofilt.
type PlaylistSongsResponse struct {
	Status int               `json:"status"`
	Data   PlaylistSongsData `json:"data"`
}

// PlaylistSongsData holds playlist song results.
type PlaylistSongsData struct {
	Info  SearchInfo      `json:"info"`
	Lists []KuGouSongItem `json:"lists"`
	Songs []KuGouSongItem `json:"songs"`
}

// RegisterDevResponse wraps device registration.
type RegisterDevResponse struct {
	Status int `json:"status"`
	Data   struct {
		Dfid string `json:"dfid"`
	} `json:"data"`
}

// RecommendSongsResponse wraps /everyday_song_recommend.
type RecommendSongsResponse struct {
	Status int `json:"status"`
	Data   struct {
		SongList []KuGouAudioInfo `json:"song_list"`
	} `json:"data"`
}
