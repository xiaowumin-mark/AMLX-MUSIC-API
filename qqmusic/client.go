package qqmusic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xiaowumin-mark/AMLX-MUSIC-API"
	"github.com/xiaowumin-mark/AMLX-MUSIC-API/internal/httpclient"
)

const (
	name = "qq"

	musicuFCG = "https://u.y.qq.com/cgi-bin/musicu.fcg"
	lrcAPIURL = "https://c.y.qq.com/lyric/fcgi-bin/fcg_query_lyric_new.fcg"
	qqReferer = "https://y.qq.com/portal/player.html"
	qqOrigin  = "https://y.qq.com"

	searchModule      = "music.search.SearchCgiService"
	searchMethod      = "DoSearchForQQMusicMobile"
	songDetailModule  = "music.pf_song_detail_svr"
	songDetailMethod  = "get_song_detail_yqq"
	lyricModule       = "music.musichallSong.PlayLyricInfo"
	lyricMethod       = "GetPlayLyricInfo"
	albumModule       = "music.musichallAlbum.AlbumInfoServer"
	albumMethod       = "GetAlbumDetail"
	albumSongsModule  = "music.musichallAlbum.AlbumSongList"
	albumSongsMethod  = "GetAlbumSongList"
	singerSongsModule = "musichall.song_list_server"
	singerSongsMethod = "GetSingerSongList"
	playlistModule    = "music.srfDissInfo.DissInfo"
	playlistMethod    = "CgiGetDiss"
	recommendModule   = "playlist.HotRecommendServer"
	recommendMethod   = "get_hot_recommend"
)

// Client is the QQ Music API client implementing musicapi.MusicProvider.
type Client struct {
	http  *httpclient.Client
	cfg   *musicapi.ClientConfig
	qimei string
}

func init() {
	musicapi.Register(name, func(opts ...musicapi.Option) (musicapi.MusicProvider, error) {
		return New(opts...)
	})
}

// New creates a new QQ Music client.
func New(opts ...musicapi.Option) (*Client, error) {
	cfg := musicapi.DefaultClientConfig()
	for _, o := range opts {
		o(cfg)
	}

	qimeiResult := GetQimei()

	c := &Client{
		http:  httpclient.New(cfg.HTTPClient),
		cfg:   cfg,
		qimei: qimeiResult.Q36,
	}
	c.http.SetCommonHeader("Referer", qqReferer)
	return c, nil
}

// Name returns the platform identifier.
func (c *Client) Name() string { return name }

// Search performs a search query.
func (c *Client) Search(ctx context.Context, keyword string, searchType musicapi.SearchType, page, limit int) (*musicapi.SearchResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}

	payload := BuildBatchPayload(c.qimei, map[string]BatchMethod{
		"search": {
			Module: searchModule,
			Method: searchMethod,
			Param: map[string]any{
				"searchid":     GetSearchID(),
				"query":        keyword,
				"search_type":  0,
				"num_per_page": limit,
				"page_num":     page,
				"highlight":    1,
				"grp":          1,
			},
		},
	})

	raw, err := c.http.RawPost(musicuFCG, payload, map[string]string{
		"Content-Type": "application/json",
		"Referer":      qqReferer,
	})
	if err != nil {
		return nil, fmt.Errorf("qq search: %w", err)
	}

	var batchResp map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &batchResp); err != nil {
		return nil, fmt.Errorf("qq search batch parse: %w", err)
	}

	searchRaw, ok := batchResp["search"]
	if !ok {
		return nil, fmt.Errorf("qq search: response missing 'search' key")
	}
	var searchResp struct {
		Code int              `json:"code"`
		Data SearchResultData `json:"data"`
	}
	if err := json.Unmarshal(searchRaw, &searchResp); err != nil {
		return nil, fmt.Errorf("qq search parse: %w", err)
	}
	if searchResp.Code != 0 {
		return nil, fmt.Errorf("qq search: code=%d", searchResp.Code)
	}

	result := &musicapi.SearchResult{Page: page, Limit: limit}
	songList := searchResp.Data.Body.ItemSong
	if len(songList) == 0 {
		songList = searchResp.Data.Body.Song.List
	}
	result.Total = -1
	if total := firstInt(
		searchResp.Data.Body.Total,
		searchResp.Data.Body.TotalNum,
		searchResp.Data.Body.Sum,
		searchResp.Data.Body.Song.Total,
		searchResp.Data.Body.Song.TotalNum,
		searchResp.Data.Body.Song.Num,
	); total > 0 {
		result.Total = total
	}
	for _, s := range songList {
		result.Songs = append(result.Songs, convertQQSearchSong(s))
	}
	return result, nil
}

// GetSong fetches song detail.
func (c *Client) GetSong(ctx context.Context, songID string) (*musicapi.Song, error) {
	param := map[string]any{}
	if _, err := strconv.Atoi(songID); err == nil {
		param["song_id"] = songID
	} else {
		param["song_mid"] = songID
	}

	payload := BuildBatchPayload(c.qimei, map[string]BatchMethod{
		"song": {Module: songDetailModule, Method: songDetailMethod, Param: param},
	})

	raw, err := c.http.RawPost(musicuFCG, payload, map[string]string{
		"Content-Type": "application/json",
		"Referer":      qqReferer,
	})
	if err != nil {
		return nil, fmt.Errorf("qq song detail: %w", err)
	}

	var batchResp map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &batchResp); err != nil {
		return nil, fmt.Errorf("qq song detail parse: %w", err)
	}
	songRaw, ok := batchResp["song"]
	if !ok {
		return nil, fmt.Errorf("qq song detail: missing key")
	}
	var resp SongDetailResponse
	if err := json.Unmarshal(songRaw, &resp); err != nil {
		return nil, fmt.Errorf("qq song detail parse: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("qq song detail: code=%d", resp.Code)
	}

	return convertQQTrack(resp.Data.TrackInfo), nil
}

// GetLyric retrieves and decrypts QRC lyrics.
func (c *Client) GetLyric(ctx context.Context, songID string) (*musicapi.Lyric, error) {
	param := map[string]any{
		"qrc":   1,
		"trans": 1,
		"roma":  1,
	}
	if _, err := strconv.Atoi(songID); err == nil {
		param["songId"] = songID
	} else {
		param["songMid"] = songID
	}

	payload := BuildBatchPayload(c.qimei, map[string]BatchMethod{
		"lyric": {Module: lyricModule, Method: lyricMethod, Param: param},
	})

	raw, err := c.http.RawPost(musicuFCG, payload, map[string]string{
		"Content-Type": "application/json",
		"Referer":      qqReferer,
	})
	if err != nil {
		// Fallback to LRC-only endpoint
		return c.getLyricLRCOnly(ctx, songID)
	}

	var batchResp map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &batchResp); err != nil {
		return c.getLyricLRCOnly(ctx, songID)
	}
	lyricRaw, ok := batchResp["lyric"]
	if !ok {
		return c.getLyricLRCOnly(ctx, songID)
	}
	var resp LyricAPIResponse
	if err := json.Unmarshal(lyricRaw, &resp); err != nil {
		return c.getLyricLRCOnly(ctx, songID)
	}

	mainText, _ := QrcDecryptWithFallback(resp.Data.Lyric)
	transText, _ := QrcDecryptWithFallback(resp.Data.Trans)
	romaText, _ := QrcDecryptWithFallback(resp.Data.Roma)

	if mainText == "" {
		return c.getLyricLRCOnly(ctx, songID)
	}

	// Extract from XML wrapper if needed
	if len(mainText) > 0 && mainText[0:5] == "<?xml" {
		mainText = ExtractFromQRcWrapper(mainText)
	}
	mainText = repairUTF8Mojibake(mainText)
	if looksMojibake(mainText) {
		return c.getLyricLRCOnly(ctx, songID)
	}
	transText = repairUTF8Mojibake(ExtractFromQRcWrapper(transText))
	romaText = repairUTF8Mojibake(ExtractFromQRcWrapper(romaText))

	lyric := &musicapi.Lyric{Raw: mainText}

	if c.cfg.EnableLyricClean && mainText != "" {
		cleaner := c.getCleaner()
		cleaned, _ := cleaner.Clean(mainText)
		lyric.Lines = parseQQTimedLines(cleaned)
	} else {
		lyric.Lines = parseQQTimedLines(mainText)
	}

	if transText != "" {
		lyric.Translation = parseQQTimedLines(transText)
	}
	if romaText != "" {
		lyric.Romanization = parseQQTimedLines(romaText)
	}

	return lyric, nil
}

func parseQQTimedLines(content string) []musicapi.LyricLine {
	if !strings.Contains(content, ",") {
		return musicapi.LRCParse(content)
	}
	lines := parseQrcLines(content)
	if len(lines) > 0 {
		return lines
	}
	return musicapi.LRCParse(content)
}

func parseQrcLines(content string) []musicapi.LyricLine {
	var lines []musicapi.LyricLine
	runes := []rune(content)
	for i := 0; i < len(runes); {
		if runes[i] != '[' {
			i++
			continue
		}
		start := i
		for i < len(runes) && runes[i] != ']' {
			i++
		}
		if i >= len(runes) {
			break
		}
		i++
		header := string(runes[start:i])
		var startMs, durMs int
		if _, err := fmt.Sscanf(header, "[%d,%d]", &startMs, &durMs); err != nil {
			continue
		}
		textStart := i
		for i < len(runes) && runes[i] != '[' {
			i++
		}
		text := strings.TrimSpace(stripQrcSyllableTags(string(runes[textStart:i])))
		if text != "" {
			lines = append(lines, musicapi.LyricLine{Time: int64(startMs), Text: text})
		}
	}
	return lines
}

func stripQrcSyllableTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch r {
		case '<', '(':
			inTag = true
		case '>', ')':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

func repairUTF8Mojibake(s string) string {
	best := s
	for i := 0; i < 3 && looksMojibake(best); i++ {
		b := make([]byte, 0, len(best))
		ok := true
		for _, r := range best {
			if r > 255 {
				ok = false
				break
			}
			b = append(b, byte(r))
		}
		if !ok || !utf8.Valid(b) {
			break
		}
		best = string(b)
	}
	return best
}

func looksMojibake(s string) bool {
	if s == "" {
		return false
	}
	return strings.Count(s, "Ã")+strings.Count(s, "Â")+strings.Count(s, "\u0083") > 3
}

func (c *Client) getLyricLRCOnly(ctx context.Context, songID string) (*musicapi.Lyric, error) {
	params := map[string]string{
		"songmid":     songID,
		"pcachetime":  strconv.FormatInt(time.Now().UnixMilli(), 10),
		"g_tk":        "5381",
		"loginUin":    "0",
		"hostUin":     "0",
		"inCharset":   "utf8",
		"outCharset":  "utf-8",
		"notice":      "0",
		"platform":    "yqq",
		"needNewCode": "0",
	}
	headers := map[string]string{"Referer": qqReferer}
	raw, err := c.http.Get(lrcAPIURL, params, headers)
	if err != nil {
		return nil, fmt.Errorf("qq lrc: %w", err)
	}

	mainText, transText, err := NormalizeLRCResponse(raw)
	if err != nil {
		return nil, fmt.Errorf("qq lrc normalize: %w", err)
	}

	lyric := &musicapi.Lyric{Raw: mainText}
	if c.cfg.EnableLyricClean && mainText != "" {
		cleaner := c.getCleaner()
		cleaned, _ := cleaner.Clean(mainText)
		lyric.Lines = musicapi.LRCParse(cleaned)
	} else {
		lyric.Lines = musicapi.LRCParse(mainText)
	}
	if transText != "" {
		lyric.Translation = musicapi.LRCParse(transText)
	}
	return lyric, nil
}

// GetPlaylist fetches playlist detail.
func (c *Client) GetPlaylist(ctx context.Context, playlistID string) (*musicapi.Playlist, error) {
	disstid, err := strconv.Atoi(playlistID)
	if err != nil {
		return nil, fmt.Errorf("qq playlist: invalid id %q", playlistID)
	}

	requestKey := playlistModule + "." + playlistMethod
	payload := BuildBatchPayload(c.qimei, map[string]BatchMethod{
		requestKey: {
			Module: playlistModule,
			Method: playlistMethod,
			Param: map[string]any{
				"disstid":    disstid,
				"song_begin": 0,
				"song_num":   300,
				"userinfo":   true,
				"tag":        true,
			},
		},
	})

	raw, err := c.http.RawPost(musicuFCG, payload, map[string]string{
		"Content-Type": "application/json",
		"Referer":      qqReferer,
	})
	if err != nil {
		return nil, fmt.Errorf("qq playlist: %w", err)
	}

	var batchResp map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &batchResp); err != nil {
		return nil, fmt.Errorf("qq playlist parse: %w", err)
	}
	plRaw, ok := batchResp[requestKey]
	if !ok {
		return nil, fmt.Errorf("qq playlist: missing key")
	}
	var resp PlaylistResponse
	if err := json.Unmarshal(plRaw, &resp); err != nil {
		return nil, fmt.Errorf("qq playlist parse: %w", err)
	}

	info := resp.Data.Info
	if info.ID == 0 && resp.Data.DirInfo.ID != 0 {
		info.ID = resp.Data.DirInfo.ID
		info.Title = resp.Data.DirInfo.Title
		info.CoverURL = resp.Data.DirInfo.CoverURL
		info.HostNick = resp.Data.DirInfo.HostNick
		info.Description = resp.Data.DirInfo.Description
		info.SongCount = resp.Data.DirInfo.SongCount
		if info.HostNick == "" {
			info.HostNick = resp.Data.DirInfo.Creator.Nick
		}
	}
	name := firstText(info.Title, info.Title2)
	cover := firstText(info.CoverURL, info.CoverURL2)
	creator := firstText(info.HostNick, info.HostNick2)
	count := info.SongCount
	if count == 0 {
		count = resp.Data.SongCount
	}
	pl := &musicapi.Playlist{
		ID:          strconv.Itoa(firstInt(info.ID, info.ID2)),
		Name:        name,
		CoverURL:    cover,
		CreatorName: creator,
		Description: info.Description,
		TrackCount:  count,
	}
	for _, s := range resp.Data.SongList {
		pl.Songs = append(pl.Songs, convertQQTrack(s))
	}
	return pl, nil
}

// GetArtist fetches artist songs.
func (c *Client) GetArtist(ctx context.Context, artistID string) (*musicapi.Artist, error) {
	payload := BuildBatchPayload(c.qimei, map[string]BatchMethod{
		"singerSongs": {
			Module: singerSongsModule,
			Method: singerSongsMethod,
			Param: map[string]any{
				"singerMid": artistID,
				"order":     1,
				"number":    50,
				"begin":     0,
			},
		},
	})

	raw, err := c.http.RawPost(musicuFCG, payload, map[string]string{
		"Content-Type": "application/json",
		"Referer":      qqReferer,
	})
	if err != nil {
		return nil, fmt.Errorf("qq artist: %w", err)
	}

	var batchResp map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &batchResp); err != nil {
		return nil, fmt.Errorf("qq artist parse: %w", err)
	}
	singerRaw, ok := batchResp["singerSongs"]
	if !ok {
		return nil, fmt.Errorf("qq artist: missing key")
	}
	var resp SingerSongListResponse
	if err := json.Unmarshal(singerRaw, &resp); err != nil {
		return nil, fmt.Errorf("qq artist parse: %w", err)
	}

	artist := &musicapi.Artist{ID: artistID}
	for _, s := range resp.Data.SongList {
		song := convertQQTrack(s.SongInfo)
		if artist.Name == "" && len(song.Artists) > 0 {
			artist.Name = song.Artists[0].Name
		}
		artist.HotSongs = append(artist.HotSongs, song)
	}
	return artist, nil
}

// GetAlbum fetches album details and tracks.
func (c *Client) GetAlbum(ctx context.Context, albumID string) (*musicapi.Album, error) {
	payload := BuildBatchPayload(c.qimei, map[string]BatchMethod{
		"album":      {Module: albumModule, Method: albumMethod, Param: map[string]any{"albumMId": albumID}},
		"albumSongs": {Module: albumSongsModule, Method: albumSongsMethod, Param: map[string]any{"albumMid": albumID, "albumID": 0, "begin": 0, "num": 100, "order": 2}},
	})

	raw, err := c.http.RawPost(musicuFCG, payload, map[string]string{
		"Content-Type": "application/json",
		"Referer":      qqReferer,
	})
	if err != nil {
		return nil, fmt.Errorf("qq album: %w", err)
	}

	var batchResp map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &batchResp); err != nil {
		return nil, fmt.Errorf("qq album parse: %w", err)
	}

	albumRaw, ok := batchResp["album"]
	if !ok {
		return nil, fmt.Errorf("qq album: missing album key")
	}
	var albumResp AlbumDetailResponse
	if err := json.Unmarshal(albumRaw, &albumResp); err != nil {
		return nil, fmt.Errorf("qq album parse: %w", err)
	}
	if albumResp.Code != 0 {
		return nil, fmt.Errorf("qq album: code=%d", albumResp.Code)
	}

	album := &musicapi.Album{
		ID:          albumResp.Data.BasicInfo.AlbumMID,
		Name:        albumResp.Data.BasicInfo.AlbumName,
		CoverURL:    GetAlbumCoverURL(albumResp.Data.BasicInfo.AlbumMID, 800),
		Description: albumResp.Data.BasicInfo.Desc,
		ReleaseDate: albumResp.Data.BasicInfo.PublishDate,
	}
	for _, s := range albumResp.Data.Singer.SingerList {
		album.Artists = append(album.Artists, musicapi.ArtistBrief{ID: s.Mid, Name: s.Name})
	}

	songsRaw, ok := batchResp["albumSongs"]
	if ok {
		var songsResp AlbumSongListResponse
		if json.Unmarshal(songsRaw, &songsResp) == nil {
			for _, s := range songsResp.Data.SongList {
				album.Songs = append(album.Songs, convertQQTrack(s.SongInfo))
			}
		}
	}

	return album, nil
}

// GetDailyRecommend returns hot recommended playlists.
func (c *Client) GetDailyRecommend(ctx context.Context) ([]*musicapi.Song, error) {
	payload := BuildBatchPayload(c.qimei, map[string]BatchMethod{
		"recommend": {Module: recommendModule, Method: recommendMethod, Param: map[string]any{}},
	})

	raw, err := c.http.RawPost(musicuFCG, payload, map[string]string{
		"Content-Type": "application/json",
		"Referer":      qqReferer,
	})
	if err != nil {
		return nil, fmt.Errorf("qq recommend: %w", err)
	}

	var batchResp map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &batchResp); err != nil {
		return nil, fmt.Errorf("qq recommend parse: %w", err)
	}
	recRaw, ok := batchResp["recommend"]
	if !ok {
		// No recommendations available
		return nil, nil
	}
	var resp RecommendResponse
	if err := json.Unmarshal(recRaw, &resp); err != nil {
		return nil, fmt.Errorf("qq recommend parse: %w", err)
	}

	// Recommendations return playlists, not individual songs.
	// We return empty - callers should use search for song discovery.
	_ = resp
	return nil, nil
}

func (c *Client) getCleaner() musicapi.LyricCleaner {
	if c.cfg.CustomLyricCleaner != nil {
		return c.cfg.CustomLyricCleaner
	}
	return musicapi.NewDefaultLyricCleaner()
}

// ------- Data converters -------

func convertQQSearchSong(s QQSearchSong) *musicapi.Song {
	artists := make([]musicapi.ArtistBrief, len(s.Singer))
	for i, singer := range s.Singer {
		artists[i] = musicapi.ArtistBrief{ID: singer.Mid, Name: singer.Name}
	}
	return &musicapi.Song{
		ID:        s.Mid,
		Name:      s.Title,
		Duration:  s.Interval,
		Artists:   artists,
		Album:     &musicapi.AlbumBrief{ID: s.Album.Mid, Name: s.Album.Name},
		CoverURL:  GetAlbumCoverURL(s.Album.Mid, 300),
		PayStatus: payStatus(s.Pay.PayPlay),
		PlatformExtra: map[string]any{
			"qq_id":   s.ID,
			"qq_mid":  s.Mid,
			"payplay": s.Pay.PayPlay,
		},
	}
}

func convertQQTrack(t QQTrackInfo) *musicapi.Song {
	artists := make([]musicapi.ArtistBrief, len(t.Singer))
	for i, singer := range t.Singer {
		artists[i] = musicapi.ArtistBrief{ID: singer.Mid, Name: singer.Name}
	}
	return &musicapi.Song{
		ID:        t.Mid,
		Name:      t.Name,
		Duration:  t.Interval,
		Artists:   artists,
		Album:     &musicapi.AlbumBrief{ID: t.Album.Mid, Name: t.Album.Name},
		CoverURL:  GetAlbumCoverURL(t.Album.Mid, 300),
		PayStatus: payStatus(t.Pay.PayPlay),
		PlatformExtra: map[string]any{
			"qq_id":   t.ID,
			"qq_mid":  t.Mid,
			"payplay": t.Pay.PayPlay,
		},
	}
}

func payStatus(payPlay int) string {
	switch {
	case payPlay == 0:
		return "free"
	case payPlay <= 3:
		return "vip"
	default:
		return "only"
	}
}

func firstText(vals ...string) string {
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
