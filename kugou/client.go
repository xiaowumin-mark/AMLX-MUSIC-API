package kugou

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xiaowumin-mark/AMLX-MUSIC-API"
	"github.com/xiaowumin-mark/AMLX-MUSIC-API/internal/httpclient"
	"github.com/xiaowumin-mark/AMLX-MUSIC-API/internal/util"
)

const (
	name = "kugou"

	apiGateway     = "https://gateway.kugou.com"
	userServiceURL = "https://userservice.kugou.com"
	lyricsAPIURL   = "https://lyrics.kugou.com"
	openapiURL     = "https://openapi.kugou.com"

	searchSongPath    = "/v3/search/song"
	albumDetailPath   = "/kmr/v2/albums"
	albumSongsPath    = "/v1/album_audio/lite"
	singerSongsPath   = "/kmr/v1/audio_group/author"
	playlistInfoPath  = "/v3/get_list_info"
	playlistSongsPath = "/pubsongs/v2/get_other_list_file_nofilt"
	songDetailPath    = "/v2/get_res_privilege/lite"
	registerDevPath   = "/risk/v1/r_register_dev"
	lyricSearchPath   = "/search"
	lyricDownloadPath = "/download"
	everydayRecPath   = "/everyday_song_recommend"

	androidUA = "Android15-1070-11083-46-0-DiscoveryDRADProtocol-wifi"
)

// Client is the KuGou Music API client implementing musicapi.MusicProvider.
type Client struct {
	http *httpclient.Client
	cfg  *musicapi.ClientConfig
	dfid string
	mid  string
	uuid string
}

func init() {
	musicapi.Register(name, func(opts ...musicapi.Option) (musicapi.MusicProvider, error) {
		return New(opts...)
	})
}

// New creates a new KuGou client.
func New(opts ...musicapi.Option) (*Client, error) {
	cfg := musicapi.DefaultClientConfig()
	for _, o := range opts {
		o(cfg)
	}

	c := &Client{
		http: httpclient.New(cfg.HTTPClient),
		cfg:  cfg,
	}
	c.http.SetCommonHeader("User-Agent", androidUA)
	c.http.SetCommonHeader("kg-tid", "255")

	// Try to register device; non-fatal
	c.registerDevice(context.Background())
	if c.dfid == "" {
		c.setDeviceFromDFID("-")
	}

	return c, nil
}

// Name returns the platform identifier.
func (c *Client) Name() string { return name }

func (c *Client) ct() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func (c *Client) commonParams(extra map[string]string) map[string]string {
	p := map[string]string{
		"appid":      appID,
		"clientver":  clientVer,
		"clienttime": c.ct(),
		"dfid":       c.dfid,
		"mid":        c.mid,
		"userid":     "0",
		"uuid":       c.uuid,
	}
	for k, v := range extra {
		p[k] = v
	}
	return p
}

// Search performs a search query.
func (c *Client) Search(ctx context.Context, keyword string, searchType musicapi.SearchType, page, limit int) (*musicapi.SearchResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}

	params := c.commonParams(map[string]string{
		"keyword":      keyword,
		"page":         strconv.Itoa(page),
		"pagesize":     strconv.Itoa(limit),
		"iscorrection": "1",
		"platform":     "AndroidFilter",
		"nocollect":    "0",
		"albumhide":    "0",
	})
	params["signature"] = signatureAndroidParams(params, "", false)

	headers := map[string]string{
		"x-router": "complexsearch.kugou.com",
	}

	raw, err := c.http.Get(apiGateway+searchSongPath, params, headers)
	if err != nil {
		return nil, fmt.Errorf("kugou search: %w", err)
	}

	var resp SearchSongResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("kugou search parse: %w", err)
	}
	if resp.Status != 1 {
		return nil, fmt.Errorf("kugou search: status=%d raw=%s", resp.Status, truncRaw(raw))
	}

	result := &musicapi.SearchResult{
		Total: firstInt(resp.Data.Info.Total, resp.Data.Total, len(resp.Data.Lists)),
		Page:  page,
		Limit: limit,
	}
	for _, s := range resp.Data.Lists {
		result.Songs = append(result.Songs, convertSongFromSearch(s))
	}
	if len(result.Songs) > 0 {
		_ = c.fillSearchSongCovers(ctx, result.Songs)
	}
	return result, nil
}

func (c *Client) fillSearchSongCovers(ctx context.Context, songs []*musicapi.Song) error {
	_ = ctx
	var resources []map[string]any
	for _, song := range songs {
		if song == nil || song.ID == "" || song.CoverURL != "" {
			continue
		}
		resources = append(resources, map[string]any{
			"type":     "audio",
			"page_id":  0,
			"hash":     song.ID,
			"album_id": 0,
		})
	}
	if len(resources) == 0 {
		return nil
	}
	body := map[string]any{
		"appid":            appID,
		"area_code":        1,
		"behavior":         "play",
		"clientver":        clientVer,
		"need_hash_offset": 1,
		"relate":           1,
		"support_verify":   1,
		"resource":         resources,
		"qualities":        []string{"128", "320", "flac", "high", "viper_atmos", "viper_tape", "viper_clear", "super", "multitrack"},
	}
	jsonBody, _ := json.Marshal(body)
	params := c.commonParams(nil)
	params["signature"] = signatureAndroidParams(params, string(jsonBody), false)
	headers := map[string]string{"x-router": "media.store.kugou.com", "Content-Type": "application/json"}

	rawURL := apiGateway + songDetailPath + "?" + buildQuery(params)
	raw, err := c.http.RawPost(rawURL, jsonBody, headers)
	if err != nil {
		return err
	}
	var resp SongDetailResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return err
	}
	covers := make(map[string]string, len(resp.Data))
	for _, s := range resp.Data {
		converted := convertSongPrivilege(s)
		covers[converted.ID] = converted.CoverURL
	}
	for _, song := range songs {
		if cover := covers[song.ID]; cover != "" {
			song.CoverURL = cover
			if song.Album != nil {
				song.Album.CoverURL = cover
			}
		}
	}
	return nil
}

// GetSong fetches song detail.
func (c *Client) GetSong(ctx context.Context, songID string) (*musicapi.Song, error) {
	body := map[string]any{
		"appid":            appID,
		"area_code":        1,
		"behavior":         "play",
		"clientver":        clientVer,
		"need_hash_offset": 1,
		"relate":           1,
		"support_verify":   1,
		"resource":         []map[string]any{{"type": "audio", "page_id": 0, "hash": songID, "album_id": 0}},
		"qualities":        []string{"128", "320", "flac", "high", "viper_atmos", "viper_tape", "viper_clear", "super", "multitrack"},
	}
	jsonBody, _ := json.Marshal(body)

	params := c.commonParams(nil)
	params["signature"] = signatureAndroidParams(params, string(jsonBody), false)

	headers := map[string]string{"x-router": "media.store.kugou.com", "Content-Type": "application/json"}

	rawURL := apiGateway + songDetailPath + "?" + buildQuery(params)
	raw, err := c.http.RawPost(rawURL, jsonBody, headers)
	if err != nil {
		return nil, fmt.Errorf("kugou song detail: %w", err)
	}
	var resp SongDetailResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("kugou song detail parse: %w", err)
	}
	if resp.Status != 1 || len(resp.Data) == 0 {
		return nil, fmt.Errorf("kugou song detail: status=%d raw=%s", resp.Status, truncRaw(raw))
	}
	return convertSongPrivilege(resp.Data[0]), nil
}

// GetLyric retrieves and decrypts KRC lyrics.
func (c *Client) GetLyric(ctx context.Context, songID string) (*musicapi.Lyric, error) {
	// Step 1: Search for lyric candidates
	searchParams := map[string]string{
		"ver":     "1",
		"man":     "yes",
		"client":  "pc",
		"hash":    songID,
		"keyword": "",
	}
	raw, err := c.http.Get(lyricsAPIURL+lyricSearchPath, searchParams, nil)
	if err != nil {
		return nil, fmt.Errorf("kugou lyric search: %w", err)
	}
	var searchResp LyricSearchResponse
	if err := json.Unmarshal([]byte(raw), &searchResp); err != nil {
		return nil, fmt.Errorf("kugou lyric search parse: %w", err)
	}
	if len(searchResp.Candidates) == 0 {
		return nil, fmt.Errorf("kugou lyric not found for hash=%s", songID)
	}
	cand := searchResp.Candidates[0]

	// Step 2: Download and decrypt
	dlParams := map[string]string{
		"ver":       "1",
		"client":    "pc",
		"id":        cand.ID,
		"accesskey": cand.AccessKey,
		"fmt":       "krc",
		"charset":   "utf8",
	}
	raw, err = c.http.Get(lyricsAPIURL+lyricDownloadPath, dlParams, nil)
	if err != nil {
		return nil, fmt.Errorf("kugou lyric download: %w", err)
	}
	var dlResp LyricDownloadResponse
	if err := json.Unmarshal([]byte(raw), &dlResp); err != nil {
		return nil, fmt.Errorf("kugou lyric download parse: %w", err)
	}

	// Decrypt KRC
	decrypted, err := util.KRCDecrypt(dlResp.Content)
	if err != nil {
		return nil, fmt.Errorf("kugou krc decrypt: %w", err)
	}

	lyric := &musicapi.Lyric{Raw: decrypted}

	if c.cfg.EnableLyricClean {
		cleaner := c.getCleaner()
		cleaned, _ := cleaner.Clean(decrypted)
		lyric.Lines = parseKRCLines(cleaned)
	} else {
		lyric.Lines = parseKRCLines(decrypted)
	}

	return lyric, nil
}

// GetPlaylist fetches playlist details.
func (c *Client) GetPlaylist(ctx context.Context, playlistID string) (*musicapi.Playlist, error) {
	body := map[string]any{
		"data":   []map[string]any{{"global_collection_id": playlistID}},
		"userid": 0,
		"token":  "",
	}
	jsonBody, _ := json.Marshal(body)
	params := c.commonParams(nil)
	params["signature"] = signatureAndroidParams(params, string(jsonBody), false)

	headers := map[string]string{"x-router": "pubsongs.kugou.com", "Content-Type": "application/json"}

	// Build URL with query params
	rawURL := apiGateway + playlistInfoPath + "?" + buildQuery(params)
	raw, err := c.http.RawPost(rawURL, jsonBody, headers)
	if err != nil {
		return nil, fmt.Errorf("kugou playlist: %w", err)
	}

	var resp PlaylistDetailResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		var one struct {
			Status int          `json:"status"`
			Data   PlaylistInfo `json:"data"`
		}
		if err2 := json.Unmarshal([]byte(raw), &one); err2 != nil {
			return nil, fmt.Errorf("kugou playlist parse: %w", err)
		}
		resp.Status = one.Status
		if one.Data.GlobalCollectionID != "" || one.Data.Name != "" {
			resp.Data = []PlaylistInfo{one.Data}
		}
	}
	if resp.Status != 1 || len(resp.Data) == 0 {
		return nil, fmt.Errorf("kugou playlist: status=%d raw=%s", resp.Status, truncRaw(raw))
	}

	pl := convertPlaylist(resp.Data[0])
	songsParams := c.commonParams(map[string]string{
		"global_collection_id": playlistID,
		"pagesize":             "100",
		"begin_idx":            "0",
		"area_code":            "1",
		"plat":                 "1",
		"type":                 "1",
		"mode":                 "1",
	})
	songsParams["signature"] = signatureAndroidParams(songsParams, "", false)
	songsRaw, err := c.http.Get(apiGateway+playlistSongsPath, songsParams, nil)
	if err == nil {
		var songsResp PlaylistSongsResponse
		if err := json.Unmarshal([]byte(songsRaw), &songsResp); err == nil {
			songs := songsResp.Data.Songs
			if len(songs) == 0 {
				songs = songsResp.Data.Lists
			}
			for _, s := range songs {
				pl.Songs = append(pl.Songs, convertPlaylistSong(s))
			}
			if pl.TrackCount == 0 {
				pl.TrackCount = len(pl.Songs)
			}
		} else {
			return nil, fmt.Errorf("kugou playlist songs parse: %w raw=%s", err, truncRaw(songsRaw))
		}
	}

	return pl, nil
}

// GetArtist fetches artist songs.
func (c *Client) GetArtist(ctx context.Context, artistID string) (*musicapi.Artist, error) {
	clienttime := c.ct()
	body := map[string]any{
		"appid":      appID,
		"clientver":  clientVer,
		"mid":        c.mid,
		"clienttime": clienttime,
		"key":        signParamsKey(appID, clientVer, clienttime),
		"author_id":  artistID,
		"pagesize":   "30",
		"page":       "1",
		"sort":       "1",
		"area_code":  "all",
	}
	jsonBody, _ := json.Marshal(body)
	params := c.commonParams(nil)
	params["signature"] = signatureAndroidParams(params, string(jsonBody), false)

	headers := map[string]string{"x-router": "openapi.kugou.com", "Content-Type": "application/json"}

	rawURL := openapiURL + singerSongsPath + "?" + buildQuery(params)
	raw, err := c.http.RawPost(rawURL, jsonBody, headers)
	if err != nil {
		return nil, fmt.Errorf("kugou artist songs: %w", err)
	}
	var resp SingerSongsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("kugou artist songs parse: %w raw=%s", err, truncRaw(raw))
	}
	if resp.Status != 1 || resp.ErrorCode != 0 {
		return nil, fmt.Errorf("kugou artist songs: status=%d error_code=%d raw=%s", resp.Status, resp.ErrorCode, truncRaw(raw))
	}

	artist := &musicapi.Artist{ID: artistID}
	for _, s := range resp.Data {
		song := convertAudioInfo(s)
		if artist.Name == "" && len(song.Artists) > 0 {
			artist.Name = song.Artists[0].Name
		}
		artist.HotSongs = append(artist.HotSongs, song)
	}
	return artist, nil
}

// GetAlbum fetches album details and tracks.
func (c *Client) GetAlbum(ctx context.Context, albumID string) (*musicapi.Album, error) {
	body := map[string]any{
		"data":   []map[string]any{{"album_id": albumID}},
		"is_buy": 0,
		"fields": "album_id,album_name,publish_date,sizable_cover,intro,language,is_publish,heat,type,quality,authors,exclusive,author_name,trans_param",
	}
	jsonBody, _ := json.Marshal(body)
	params := c.commonParams(nil)
	params["signature"] = signatureAndroidParams(params, string(jsonBody), false)

	headers := map[string]string{"x-router": "openapi.kugou.com", "Content-Type": "application/json"}

	rawURL := apiGateway + albumDetailPath + "?" + buildQuery(params)
	raw, err := c.http.RawPost(rawURL, jsonBody, headers)
	if err != nil {
		return nil, fmt.Errorf("kugou album detail: %w", err)
	}
	var resp AlbumDetailResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("kugou album detail parse: %w", err)
	}
	if resp.Status != 1 || len(resp.Data) == 0 {
		return nil, fmt.Errorf("kugou album: status=%d raw=%s", resp.Status, truncRaw(raw))
	}

	info := resp.Data[0]
	album := &musicapi.Album{
		ID:          string(info.AlbumID),
		Name:        info.AlbumName,
		CoverURL:    info.SizableCover,
		Description: info.Intro,
		ReleaseDate: info.PublishDate,
		PlatformExtra: map[string]any{
			"language": info.Language,
		},
	}
	for _, ar := range info.Authors {
		album.Artists = append(album.Artists, musicapi.ArtistBrief{
			ID: ar.AuthorID, Name: ar.AuthorName,
		})
	}

	// Fetch album songs
	songsBody := map[string]any{
		"album_id": albumID,
		"is_buy":   "",
		"page":     1,
		"pagesize": 30,
	}
	songsJSON, _ := json.Marshal(songsBody)
	songsParams := c.commonParams(nil)
	songsParams["signature"] = signatureAndroidParams(songsParams, string(songsJSON), false)
	songsRaw, err := c.http.RawPost(apiGateway+albumSongsPath+"?"+buildQuery(songsParams), songsJSON, map[string]string{"x-router": "openapi.kugou.com", "Content-Type": "application/json"})
	if err == nil {
		var songsResp AlbumSongsResponse
		if json.Unmarshal([]byte(songsRaw), &songsResp) == nil {
			for _, s := range songsResp.Data.Songs {
				album.Songs = append(album.Songs, convertAlbumSong(s, string(info.AlbumID), info.AlbumName))
			}
		}
	}

	return album, nil
}

// GetDailyRecommend returns daily song recommendations.
func (c *Client) GetDailyRecommend(ctx context.Context) ([]*musicapi.Song, error) {
	params := c.commonParams(map[string]string{
		"platform": "android",
	})
	params["signature"] = signatureAndroidParams(params, "", false)

	raw, err := c.http.Get(apiGateway+everydayRecPath, params, map[string]string{"x-router": "everydayrec.service.kugou.com"})
	if err != nil {
		return nil, fmt.Errorf("kugou recommend: %w", err)
	}
	var resp RecommendSongsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("kugou recommend parse: %w", err)
	}

	songs := make([]*musicapi.Song, 0, len(resp.Data.SongList))
	for _, s := range resp.Data.SongList {
		songs = append(songs, convertAudioInfo(s))
	}
	return songs, nil
}

// ------- Internal helpers -------

func (c *Client) registerDevice(ctx context.Context) {
	payload := map[string]string{
		"mid":    "",
		"uuid":   "",
		"appid":  "1014",
		"userid": "0",
	}
	payloadJSON, _ := json.Marshal(payload)
	encodedPayload := []byte(base64.StdEncoding.EncodeToString(payloadJSON))

	params := map[string]string{
		"appid":      "1014",
		"clientver":  clientVer,
		"clienttime": c.ct(),
		"dfid":       "-",
		"mid":        "",
		"uuid":       "",
		"userid":     "0",
		"platid":     "4",
		"p.token":    "",
	}
	params["signature"] = signatureRegisterParams(params)

	rawURL := userServiceURL + registerDevPath + "?" + buildQuery(params)
	headerMid := util.MD5Hex("-")
	raw, err := c.http.RawPost(rawURL, encodedPayload, map[string]string{"mid": headerMid})
	if err != nil {
		return
	}
	var resp RegisterDevResponse
	if json.Unmarshal([]byte(raw), &resp) != nil {
		return
	}
	if resp.Status != 1 || resp.Data.Dfid == "" {
		return
	}
	c.setDeviceFromDFID(resp.Data.Dfid)
}

func (c *Client) setDeviceFromDFID(dfid string) {
	c.dfid = dfid
	c.mid = util.MD5Hex(c.dfid)
	c.uuid = util.MD5Hex(c.dfid + c.mid)
}

func (c *Client) getCleaner() musicapi.LyricCleaner {
	if c.cfg.CustomLyricCleaner != nil {
		return c.cfg.CustomLyricCleaner
	}
	return musicapi.NewDefaultLyricCleaner()
}

func buildQuery(params map[string]string) string {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	return q.Encode()
}

func truncRaw(s string) string {
	const n = 300
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ------- Data converters -------

func convertSongFromSearch(s KuGouSong) *musicapi.Song {
	hash := firstNonEmpty(s.Hash, s.FileHash)
	name := firstNonEmpty(s.SongName, s.OriSongName)
	albumID := firstNonEmpty(string(s.AlbumID), s.AlbumID2)
	albumName := firstNonEmpty(s.AlbumName, s.AlbumName2)
	audioID := s.AudioID
	if audioID == 0 {
		audioID = s.AudioID2
	}
	duration := s.Duration
	if duration == 0 {
		duration = s.Duration2
	}
	artistID, artistName := s.SingerID, s.SingerName
	if artistName == "" && len(s.Singers) > 0 {
		artistID = strconv.Itoa(s.Singers[0].ID)
		artistName = s.Singers[0].Name
	}
	return &musicapi.Song{
		ID:       hash,
		Name:     name,
		Duration: duration,
		Artists:  []musicapi.ArtistBrief{{ID: artistID, Name: artistName}},
		Album:    &musicapi.AlbumBrief{ID: albumID, Name: albumName},
		PlatformExtra: map[string]any{
			"audio_id":       audioID,
			"album_audio_id": int(s.AlbumAudioID),
			"fee_type":       s.FeeType,
		},
	}
}

func convertAudioInfo(s KuGouAudioInfo) *musicapi.Song {
	name := firstNonEmpty(s.SongName, s.AudioName)
	artist := firstNonEmpty(s.SingerName, s.AuthorName)
	duration := s.Duration
	if duration == 0 {
		duration = s.TimeLength
	}
	if duration == 0 {
		duration = s.TimeLength2
	}
	if duration > 10000 {
		duration /= 1000
	}
	if artist == "" && len(s.SingerInfo) > 0 {
		artist = s.SingerInfo[0].Name
		if s.SingerID == "" {
			s.SingerID = string(s.SingerInfo[0].ID)
		}
	}
	return &musicapi.Song{
		ID:       s.Hash,
		Name:     name,
		Duration: duration,
		Artists:  []musicapi.ArtistBrief{{ID: s.SingerID, Name: artist}},
		Album:    &musicapi.AlbumBrief{ID: string(s.AlbumID), Name: s.AlbumName},
		PlatformExtra: map[string]any{
			"audio_id": s.AudioID,
			"fee_type": s.FeeType,
		},
	}
}

func convertAlbumSong(s KuGouAlbumSong, albumID, albumName string) *musicapi.Song {
	hash := firstNonEmpty(s.AudioInfo.Hash, s.AudioInfo.Hash320, s.AudioInfo.Hash128, s.AudioInfo.HashFlac)
	return &musicapi.Song{
		ID:       hash,
		Name:     s.Base.SongName,
		Duration: s.AudioInfo.Duration / 1000,
		Artists:  []musicapi.ArtistBrief{{Name: s.Base.SingerName}},
		Album:    &musicapi.AlbumBrief{ID: albumID, Name: albumName},
	}
}

func convertSongPrivilege(s SongPrivilege) *musicapi.Song {
	name := firstNonEmpty(s.SongName, s.AudioName, s.Name)
	artist := s.SingerName
	if parts := strings.SplitN(name, " - ", 2); len(parts) == 2 {
		if artist == "" {
			artist = parts[0]
		}
		name = parts[1]
	}
	duration := s.Duration
	if duration == 0 && s.Info.Duration > 0 {
		duration = s.Info.Duration / 1000
	}
	cover := firstNonEmpty(s.AlbumImg, s.Info.Image)
	if strings.Contains(cover, "{size}") {
		cover = strings.ReplaceAll(cover, "{size}", "400")
	}
	return &musicapi.Song{
		ID:       s.Hash,
		Name:     name,
		Duration: duration,
		CoverURL: cover,
		Artists:  []musicapi.ArtistBrief{{Name: artist}},
		Album:    &musicapi.AlbumBrief{ID: string(s.AlbumID), Name: s.AlbumName, CoverURL: cover},
		PlatformExtra: map[string]any{
			"audio_id": s.AudioID,
			"fee_type": s.FeeType,
		},
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstInt(vals ...int) int {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}

func convertPlaylist(p PlaylistInfo) *musicapi.Playlist {
	cover := firstNonEmpty(p.ImgURL, p.Pic)
	if strings.Contains(cover, "{size}") {
		cover = strings.ReplaceAll(cover, "{size}", "150")
	}
	pl := &musicapi.Playlist{
		ID:          p.GlobalCollectionID,
		Name:        p.Name,
		CoverURL:    cover,
		CreatorName: firstNonEmpty(p.Username, p.ListCreateUsername),
		Description: p.Intro,
		TrackCount:  p.Count,
	}
	for _, t := range p.MusicList {
		pl.Songs = append(pl.Songs, convertPlaylistSong(t))
	}
	return pl
}

func convertPlaylistSong(t KuGouSongItem) *musicapi.Song {
	name := firstNonEmpty(t.SongName, t.Name)
	artistID, artistName := t.SingerID, t.SingerName
	if artistName == "" && len(t.SingerInfo) > 0 {
		artistID = string(t.SingerInfo[0].ID)
		artistName = t.SingerInfo[0].Name
	}
	if parts := strings.SplitN(name, " - ", 2); len(parts) == 2 {
		if artistName == "" {
			artistName = parts[0]
		}
		name = parts[1]
	}
	duration := t.Duration
	if duration == 0 && t.TimeLen > 0 {
		duration = t.TimeLen / 1000
	}
	return &musicapi.Song{
		ID:       t.Hash,
		Name:     name,
		Duration: duration,
		Artists:  []musicapi.ArtistBrief{{ID: artistID, Name: artistName}},
		Album:    &musicapi.AlbumBrief{ID: string(t.AlbumID), Name: t.AlbumName},
	}
}

// parseKRCLines extracts timed lyric lines from decrypted KRC content.
// KRC format: [start_ms,duration_ms]text<offset,duration,0>syllable_text...
func parseKRCLines(content string) []musicapi.LyricLine {
	var lines []musicapi.LyricLine
	// Simple line-based parsing: extract [start_ms,dur_ms] timestamps
	i := 0
	runes := []rune(content)
	for i < len(runes) {
		if runes[i] == '[' {
			// Extract timestamp
			start := i
			for i < len(runes) && runes[i] != ']' {
				i++
			}
			if i < len(runes) {
				i++ // skip ']'
			}
			header := string(runes[start:i])
			// Parse [<start>,<duration>]
			var startMs, durMs int
			_, _ = fmt.Sscanf(header, "[%d,%d]", &startMs, &durMs)

			// Read until next '[' or end
			textStart := i
			for i < len(runes) && runes[i] != '[' {
				i++
			}
			text, syllables := parseKRCSyllables(string(runes[textStart:i]), startMs)
			if text != "" {
				lines = append(lines, musicapi.LyricLine{
					Time:      int64(startMs),
					Duration:  int64(durMs),
					Text:      text,
					Syllables: syllables,
				})
			}
		} else {
			i++
		}
	}
	return lines
}

func parseKRCSyllables(s string, lineStartMs int) (string, []musicapi.LyricSyllable) {
	var text strings.Builder
	var syllables []musicapi.LyricSyllable
	runes := []rune(s)
	for i := 0; i < len(runes); {
		if runes[i] != '<' {
			text.WriteRune(runes[i])
			i++
			continue
		}
		tagStart := i
		for i < len(runes) && runes[i] != '>' {
			i++
		}
		if i >= len(runes) {
			text.WriteString(string(runes[tagStart:]))
			break
		}
		tag := string(runes[tagStart : i+1])
		i++
		var offsetMs, durMs, _unused int
		if _, err := fmt.Sscanf(tag, "<%d,%d,%d>", &offsetMs, &durMs, &_unused); err != nil {
			continue
		}
		wordStart := i
		for i < len(runes) && runes[i] != '<' {
			i++
		}
		word := string(runes[wordStart:i])
		word = strings.Trim(word, "\r\n")
		if word == "" {
			continue
		}
		text.WriteString(word)
		syllables = append(syllables, musicapi.LyricSyllable{
			Time:     int64(lineStartMs + offsetMs),
			Duration: int64(durMs),
			Text:     word,
		})
	}
	return trimSpace(text.String()), syllables
}

func stripSyllableTags(s string) string {
	result := make([]rune, 0, len([]rune(s)))
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result = append(result, r)
		}
	}
	return string(result)
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
