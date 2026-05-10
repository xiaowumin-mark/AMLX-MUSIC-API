package netease

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xiaowumin-mark/AMLX-MUSIC-API"
	"github.com/xiaowumin-mark/AMLX-MUSIC-API/internal/httpclient"
	"github.com/xiaowumin-mark/AMLX-MUSIC-API/internal/util"
)

const (
	name = "netease"

	apiBaseURL       = "https://music.163.com"
	apiInterfaceURL  = "https://interface.music.163.com"
	apiInterface3URL = "https://interface3.music.163.com"

	registerAnonPath   = "/api/register/anonimous"
	searchPath         = "/api/cloudsearch/pc"
	lyricPath          = "/api/song/lyric/v1"
	albumPath          = "/v1/album/"
	artistSongsPath    = "/v1/artist/songs"
	playlistDetailPath = "/v6/playlist/detail"
	songDetailPath     = "/v3/song/detail"
	artistDetailPath   = "/api/artist/head/info/get"
	recommendSongsPath = "/v3/discovery/recommend/songs"

	pcUserAgent = "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Safari/537.36 Chrome/91.0.4472.164 NeteaseMusicDesktop/3.1.17.204416"
	referer     = "https://music.163.com"
)

// Client is the NetEase Cloud Music API client implementing musicapi.MusicProvider.
type Client struct {
	http     *httpclient.Client
	cfg      *musicapi.ClientConfig
	deviceID string
}

func init() {
	musicapi.Register(name, func(opts ...musicapi.Option) (musicapi.MusicProvider, error) {
		return New(opts...)
	})
}

// New creates a new NetEase client with the given options.
func New(opts ...musicapi.Option) (*Client, error) {
	cfg := musicapi.DefaultClientConfig()
	for _, o := range opts {
		o(cfg)
	}

	c := &Client{
		http: httpclient.New(cfg.HTTPClient),
		cfg:  cfg,
	}
	c.http.SetCommonHeader("User-Agent", pcUserAgent)
	c.http.SetCommonHeader("Referer", referer)
	if cfg.Cookie != "" {
		c.http.SetCommonHeader("Cookie", cfg.Cookie)
	}

	devID, err := util.RandomHex(26)
	if err != nil {
		return nil, fmt.Errorf("netease: generate device id: %w", err)
	}
	c.deviceID = devID

	// Attempt anonymous registration; non-fatal if it fails.
	_ = c.registerAnonymous(context.Background())

	return c, nil
}

// Name returns the platform identifier.
func (c *Client) Name() string { return name }

// Search performs a search across songs, albums, artists, and playlists.
func (c *Client) Search(ctx context.Context, keyword string, searchType musicapi.SearchType, page, limit int) (*musicapi.SearchResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}

	typeMap := map[musicapi.SearchType]int{
		musicapi.SearchTypeSong:     1,
		musicapi.SearchTypeAlbum:    10,
		musicapi.SearchTypeArtist:   100,
		musicapi.SearchTypePlaylist: 1000,
	}
	t, ok := typeMap[searchType]
	if !ok {
		t = 1
	}

	body := map[string]any{
		"s":      keyword,
		"type":   t,
		"limit":  limit,
		"offset": (page - 1) * limit,
		"total":  false,
	}
	jsonBody, _ := json.Marshal(body)

	eapiURL := apiInterfaceURL + "/eapi" + searchPath
	encrypted, err := EapiEncrypt(searchPath, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("netease search eapi encrypt: %w", err)
	}

	raw, err := c.http.PostForm(eapiURL, map[string]string{"params": encrypted}, nil)
	if err != nil {
		return nil, fmt.Errorf("netease search request: %w", err)
	}

	var resp SearchResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("netease search parse: %w", err)
	}

	result := &musicapi.SearchResult{
		Total: resp.Result.SongCount + resp.Result.AlbumCount + resp.Result.ArtistCount + resp.Result.PlaylistCount,
		Page:  page,
		Limit: limit,
	}
	for _, s := range resp.Result.Songs {
		result.Songs = append(result.Songs, convertSong(s))
	}
	if len(result.Songs) > 0 {
		_ = c.fillSearchSongCovers(ctx, result.Songs)
	}
	for _, a := range resp.Result.Albums {
		result.Albums = append(result.Albums, convertAlbumFromSearch(a))
	}
	for _, ar := range resp.Result.Artists {
		result.Artists = append(result.Artists, convertArtistSimple(ar))
	}
	for _, p := range resp.Result.Playlists {
		result.Playlists = append(result.Playlists, convertPlaylistSimple(p))
	}

	return result, nil
}

func (c *Client) fillSearchSongCovers(ctx context.Context, songs []*musicapi.Song) error {
	var ids []string
	for _, song := range songs {
		if song == nil || song.CoverURL != "" {
			continue
		}
		ids = append(ids, song.ID)
	}
	if len(ids) == 0 {
		return nil
	}
	items := make([]string, 0, len(ids))
	for _, id := range ids {
		items = append(items, fmt.Sprintf(`{"id":%s}`, id))
	}
	body := map[string]any{"c": "[" + strings.Join(items, ",") + "]"}
	jsonBody, _ := json.Marshal(body)

	raw, err := c.weapiRequest(ctx, apiBaseURL+"/weapi"+songDetailPath, string(jsonBody))
	if err != nil {
		return err
	}
	var resp SongDetailResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return err
	}
	covers := make(map[string]string, len(resp.Songs))
	for _, s := range resp.Songs {
		converted := convertSong(s)
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
	albumCovers := map[string]string{}
	for _, song := range songs {
		if song == nil || song.CoverURL != "" || song.Album == nil || song.Album.ID == "" || song.Album.ID == "0" {
			continue
		}
		if _, ok := albumCovers[song.Album.ID]; ok {
			continue
		}
		album, err := c.GetAlbum(ctx, song.Album.ID)
		if err != nil {
			albumCovers[song.Album.ID] = ""
			continue
		}
		albumCovers[song.Album.ID] = album.CoverURL
	}
	for _, song := range songs {
		if song == nil || song.CoverURL != "" || song.Album == nil {
			continue
		}
		if cover := albumCovers[song.Album.ID]; cover != "" {
			song.CoverURL = cover
			song.Album.CoverURL = cover
		}
	}
	return nil
}

// GetSong fetches detailed song information.
func (c *Client) GetSong(ctx context.Context, songID string) (*musicapi.Song, error) {
	body := map[string]any{
		"c": fmt.Sprintf(`[{"id":%s}]`, songID),
	}
	jsonBody, _ := json.Marshal(body)

	raw, err := c.weapiRequest(ctx, apiBaseURL+"/weapi"+songDetailPath, string(jsonBody))
	if err != nil {
		return nil, err
	}
	var resp SongDetailResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("netease song detail parse: %w", err)
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("netease song detail: code=%d", resp.Code)
	}
	if len(resp.Songs) == 0 {
		return nil, fmt.Errorf("netease song not found: %s", songID)
	}
	song := convertSong(resp.Songs[0])
	if song.CoverURL == "" && song.Album != nil && song.Album.ID != "" && song.Album.ID != "0" {
		if album, err := c.GetAlbum(ctx, song.Album.ID); err == nil && album.CoverURL != "" {
			song.CoverURL = album.CoverURL
			song.Album.CoverURL = album.CoverURL
		}
	}
	return song, nil
}

// GetLyric retrieves and decrypts lyrics for a song.
func (c *Client) GetLyric(ctx context.Context, songID string) (*musicapi.Lyric, error) {
	id, _ := strconv.Atoi(songID)
	body := map[string]any{
		"id": id,
		"cp": "false",
		"lv": "0", "kv": "0", "tv": "0", "rv": "0",
		"yv": "0", "ytv": "0", "yrv": "0",
		"csrf_token": "",
	}
	body["header"] = c.eapiHeader()
	jsonBody, _ := json.Marshal(body)

	eapiURL := apiInterface3URL + "/eapi" + lyricPath
	encrypted, err := EapiEncrypt(lyricPath, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("netease lyric eapi encrypt: %w", err)
	}

	raw, err := c.http.PostForm(eapiURL, map[string]string{"params": encrypted}, nil)
	if err != nil {
		return nil, fmt.Errorf("netease lyric request: %w", err)
	}

	var resp LyricResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("netease lyric parse: %w", err)
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("netease lyric: code=%d", resp.Code)
	}

	lyric := &musicapi.Lyric{}

	mainText := resp.Yrc.Lyric
	if mainText == "" {
		mainText = resp.Lrc.Lyric
	}
	lyric.Raw = mainText

	if c.cfg.EnableLyricClean && mainText != "" {
		cleaner := c.getCleaner()
		cleaned, _ := cleaner.Clean(mainText)
		lyric.Lines = parseNeteaseTimedLines(cleaned)
	} else {
		lyric.Lines = parseNeteaseTimedLines(mainText)
	}

	if resp.TLyric.Lyric != "" {
		lyric.Translation = musicapi.LRCParse(resp.TLyric.Lyric)
	}
	if resp.Romalrc.Lyric != "" {
		lyric.Romanization = musicapi.LRCParse(resp.Romalrc.Lyric)
	}

	return lyric, nil
}

func (c *Client) eapiHeader() map[string]any {
	return map[string]any{
		"os":          "pc",
		"appver":      "3.1.17.204416",
		"versioncode": "140",
		"osver":       "Microsoft-Windows-10-Professional-build-22631-64bit",
		"deviceId":    c.deviceID,
		"mobilename":  "",
		"buildver":    strconv.FormatInt(time.Now().Unix(), 10),
		"resolution":  "1920x1080",
		"channel":     "netease",
		"requestId":   fmt.Sprintf("%d_%04d", time.Now().UnixMilli(), time.Now().UnixNano()%1000),
		"__csrf":      "",
	}
}

func parseNeteaseTimedLines(content string) []musicapi.LyricLine {
	if strings.Contains(content, ",") {
		if lines := parseNeteaseYRC(content); len(lines) > 0 {
			return lines
		}
	}
	return musicapi.LRCParse(content)
}

func parseNeteaseYRC(content string) []musicapi.LyricLine {
	var lines []musicapi.LyricLine
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "[") {
			continue
		}
		end := strings.IndexByte(line, ']')
		if end < 0 {
			continue
		}
		var startMs, durMs int
		if _, err := fmt.Sscanf(line[:end+1], "[%d,%d]", &startMs, &durMs); err != nil {
			continue
		}
		text, syllables := parseNeteaseYrcWords(line[end+1:])
		if text != "" {
			lines = append(lines, musicapi.LyricLine{
				Time:      int64(startMs),
				Duration:  int64(durMs),
				Text:      text,
				Syllables: syllables,
			})
		}
	}
	return lines
}

func parseNeteaseYrcWords(s string) (string, []musicapi.LyricSyllable) {
	var text strings.Builder
	var syllables []musicapi.LyricSyllable
	runes := []rune(s)
	for i := 0; i < len(runes); {
		if runes[i] != '(' {
			text.WriteRune(runes[i])
			i++
			continue
		}
		tagStart := i
		for i < len(runes) && runes[i] != ')' {
			i++
		}
		if i >= len(runes) {
			text.WriteString(string(runes[tagStart:]))
			break
		}
		tag := string(runes[tagStart : i+1])
		i++
		var startMs, durMs, _unused int
		if _, err := fmt.Sscanf(tag, "(%d,%d,%d)", &startMs, &durMs, &_unused); err != nil {
			continue
		}
		wordStart := i
		for i < len(runes) && runes[i] != '(' {
			i++
		}
		word := string(runes[wordStart:i])
		if word == "" {
			continue
		}
		text.WriteString(word)
		syllables = append(syllables, musicapi.LyricSyllable{
			Time:     int64(startMs),
			Duration: int64(durMs),
			Text:     word,
		})
	}
	return strings.TrimSpace(text.String()), syllables
}

func stripNeteaseYrcWords(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); {
		if runes[i] == '(' {
			for i < len(runes) && runes[i] != ')' {
				i++
			}
			if i < len(runes) {
				i++
			}
			continue
		}
		b.WriteRune(runes[i])
		i++
	}
	return strings.TrimSpace(b.String())
}

// GetPlaylist fetches playlist details and track list.
func (c *Client) GetPlaylist(ctx context.Context, playlistID string) (*musicapi.Playlist, error) {
	id, _ := strconv.Atoi(playlistID)
	body := map[string]any{"id": id, "s": 8, "n": 100000}
	jsonBody, _ := json.Marshal(body)

	raw, err := c.weapiRequest(ctx, apiBaseURL+"/weapi"+playlistDetailPath, string(jsonBody))
	if err != nil {
		return nil, err
	}
	var resp PlaylistDetailResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("netease playlist parse: %w", err)
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("netease playlist: code=%d", resp.Code)
	}
	return convertPlaylist(resp.Playlist), nil
}

// GetArtist fetches artist details with hot songs.
func (c *Client) GetArtist(ctx context.Context, artistID string) (*musicapi.Artist, error) {
	id, _ := strconv.Atoi(artistID)

	body := map[string]any{"id": id}
	jsonBody, _ := json.Marshal(body)
	encrypted, err := EapiEncrypt(artistDetailPath, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("netease artist detail eapi encrypt: %w", err)
	}
	raw, err := c.http.PostForm(apiInterfaceURL+"/eapi/artist/head/info/get", map[string]string{"params": encrypted}, nil)
	if err != nil {
		return nil, fmt.Errorf("netease artist detail request: %w", err)
	}
	var artistResp ArtistResponse
	if err := json.Unmarshal([]byte(raw), &artistResp); err != nil {
		return nil, fmt.Errorf("netease artist detail parse: %w", err)
	}

	artist := &musicapi.Artist{
		ID:          strconv.Itoa(artistResp.Data.Artist.ID),
		Name:        artistResp.Data.Artist.Name,
		PicURL:      artistResp.Data.Artist.PicURL,
		Description: artistResp.Data.Artist.BriefDesc,
	}

	songsBody := map[string]any{"id": id, "limit": 50, "offset": 0, "order": "hot"}
	songsJSON, _ := json.Marshal(songsBody)
	songsRaw, err := c.weapiRequest(ctx, apiBaseURL+"/weapi"+artistSongsPath, string(songsJSON))
	if err == nil {
		var songsResp ArtistSongsResponse
		if json.Unmarshal([]byte(songsRaw), &songsResp) == nil && songsResp.Code == 200 {
			for _, s := range songsResp.HotSongs {
				artist.HotSongs = append(artist.HotSongs, convertSong(s))
			}
		}
	}

	return artist, nil
}

// GetAlbum fetches album details and track list.
func (c *Client) GetAlbum(ctx context.Context, albumID string) (*musicapi.Album, error) {
	body := map[string]any{}
	jsonBody, _ := json.Marshal(body)

	url := apiBaseURL + "/weapi" + albumPath + albumID
	raw, err := c.weapiRequest(ctx, url, string(jsonBody))
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code  int           `json:"code"`
		Album NeteaseAlbum  `json:"album"`
		Songs []NeteaseSong `json:"songs"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("netease album parse: %w", err)
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("netease album: code=%d", resp.Code)
	}
	if len(resp.Album.Songs) == 0 && len(resp.Songs) > 0 {
		resp.Album.Songs = resp.Songs
	}
	return convertAlbum(resp.Album), nil
}

// GetDailyRecommend returns daily recommended songs. Requires login.
func (c *Client) GetDailyRecommend(ctx context.Context) ([]*musicapi.Song, error) {
	body := map[string]any{}
	jsonBody, _ := json.Marshal(body)

	raw, err := c.weapiRequest(ctx, apiBaseURL+"/weapi"+recommendSongsPath, string(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("netease recommend: %w", err)
	}
	var resp RecommendSongsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("netease recommend parse: %w", err)
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("netease recommend: code=%d (login required)", resp.Code)
	}

	songs := make([]*musicapi.Song, 0, len(resp.Data.DailySongs))
	for _, s := range resp.Data.DailySongs {
		songs = append(songs, convertSong(s))
	}
	return songs, nil
}

// ------- Internal helpers -------

func (c *Client) registerAnonymous(ctx context.Context) error {
	username := CloudMusicDLLEncodeId(c.deviceID)
	body := map[string]any{"username": username}
	jsonBody, _ := json.Marshal(body)
	_, err := c.weapiRequest(ctx, apiBaseURL+"/weapi"+registerAnonPath, string(jsonBody))
	return err
}

func (c *Client) getCleaner() musicapi.LyricCleaner {
	if c.cfg.CustomLyricCleaner != nil {
		return c.cfg.CustomLyricCleaner
	}
	return musicapi.NewDefaultLyricCleaner()
}

func (c *Client) weapiRequest(ctx context.Context, url string, jsonBody string) (string, error) {
	var body map[string]any
	if err := json.Unmarshal([]byte(jsonBody), &body); err == nil {
		if _, ok := body["csrf_token"]; !ok {
			body["csrf_token"] = ""
			b, _ := json.Marshal(body)
			jsonBody = string(b)
		}
	}
	params, encSecKey, err := WeapiEncrypt(jsonBody)
	if err != nil {
		return "", err
	}
	form := map[string]string{
		"params":    params,
		"encSecKey": encSecKey,
	}
	return c.http.PostForm(url, form, nil)
}

// ------- Data converters -------

func convertSong(s NeteaseSong) *musicapi.Song {
	duration := s.Duration
	if duration == 0 {
		duration = s.Duration2
	}
	albumInfo := s.Album
	if albumInfo.ID == 0 && albumInfo.Name == "" {
		albumInfo = s.Album2
	}
	artists := s.Artists
	if len(artists) == 0 {
		artists = s.Artists2
	}
	song := &musicapi.Song{
		ID:       strconv.Itoa(s.ID),
		Name:     s.Name,
		Duration: duration / 1000,
		CoverURL: albumInfo.PicURL,
		Album: &musicapi.AlbumBrief{
			ID:       strconv.Itoa(albumInfo.ID),
			Name:     albumInfo.Name,
			CoverURL: albumInfo.PicURL,
		},
		PlatformExtra: map[string]any{
			"netease_id": s.ID,
			"fee":        s.Fee,
		},
	}
	for _, ar := range artists {
		song.Artists = append(song.Artists, musicapi.ArtistBrief{
			ID:   strconv.Itoa(ar.ID),
			Name: ar.Name,
		})
	}
	return song
}

func convertAlbum(a NeteaseAlbum) *musicapi.Album {
	album := &musicapi.Album{
		ID:          strconv.Itoa(a.ID),
		Name:        a.Name,
		CoverURL:    a.PicURL,
		Description: a.Description,
		ReleaseDate: msToDateStr(a.PublishTime),
		PlatformExtra: map[string]any{
			"netease_id": a.ID,
			"company":    a.Company,
		},
	}
	for _, ar := range a.Artists {
		album.Artists = append(album.Artists, musicapi.ArtistBrief{
			ID:   strconv.Itoa(ar.ID),
			Name: ar.Name,
		})
	}
	for _, s := range a.Songs {
		album.Songs = append(album.Songs, convertSong(s))
	}
	return album
}

func convertAlbumFromSearch(a NeteaseAlbum) *musicapi.Album {
	return &musicapi.Album{
		ID:       strconv.Itoa(a.ID),
		Name:     a.Name,
		CoverURL: a.PicURL,
		Artists:  []musicapi.ArtistBrief{{Name: a.Company}},
	}
}

func convertArtistSimple(ar NeteaseArtist) *musicapi.Artist {
	return &musicapi.Artist{
		ID:     strconv.Itoa(ar.ID),
		Name:   ar.Name,
		PicURL: ar.PicURL,
	}
}

func convertPlaylist(p NeteasePlaylist) *musicapi.Playlist {
	pl := &musicapi.Playlist{
		ID:          strconv.Itoa(p.ID),
		Name:        p.Name,
		CoverURL:    p.CoverImgURL,
		CreatorName: p.Creator.Nickname,
		Description: p.Description,
		TrackCount:  p.TrackCount,
		PlatformExtra: map[string]any{
			"netease_id": p.ID,
		},
	}
	for _, t := range p.Tracks {
		pl.Songs = append(pl.Songs, convertSong(t))
	}
	return pl
}

func convertPlaylistSimple(p NeteasePlaylist) *musicapi.Playlist {
	return &musicapi.Playlist{
		ID:          strconv.Itoa(p.ID),
		Name:        p.Name,
		CoverURL:    p.CoverImgURL,
		CreatorName: p.Creator.Nickname,
		Description: p.Description,
		TrackCount:  p.TrackCount,
	}
}

func msToDateStr(ms int64) string {
	if ms == 0 {
		return ""
	}
	totalDays := int(ms / 86400000)
	year := 1970
	months := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	for totalDays >= 365 {
		daysInYear := 365
		if isLeap(year) {
			daysInYear = 366
		}
		if totalDays >= daysInYear {
			totalDays -= daysInYear
			year++
		} else {
			break
		}
	}
	if isLeap(year) {
		months[1] = 29
	}
	month := 1
	for _, m := range months {
		if totalDays >= m {
			totalDays -= m
			month++
		} else {
			break
		}
	}
	return fmt.Sprintf("%d-%02d-%02d", year, month, totalDays+1)
}

func isLeap(y int) bool {
	return (y%4 == 0 && y%100 != 0) || y%400 == 0
}
