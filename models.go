// Package musicapi provides a unified Go client library for multiple Chinese
// music streaming platforms including QQ Music, NetEase Cloud Music, and KuGou Music.
package musicapi

// ImageSize represents a requested cover image size.
type ImageSize int

const (
	ImageSizeThumb ImageSize = 150
	ImageSizeSmall ImageSize = 300
	ImageSizeMid   ImageSize = 500
	ImageSizeLarge ImageSize = 800
)

// SearchType specifies the category of search results.
type SearchType int

const (
	SearchTypeSong     SearchType = 1
	SearchTypeAlbum    SearchType = 2
	SearchTypeArtist   SearchType = 3
	SearchTypePlaylist SearchType = 4
)

// ArtistBrief contains minimal artist identification.
type ArtistBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AlbumBrief contains minimal album identification.
type AlbumBrief struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	CoverURL string `json:"cover_url,omitempty"`
}

// Song represents a unified song model across all platforms.
type Song struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Artists       []ArtistBrief  `json:"artists"`
	Album         *AlbumBrief    `json:"album,omitempty"`
	Duration      int            `json:"duration"`
	CoverURL      string         `json:"cover_url"`
	PayStatus     string         `json:"pay_status,omitempty"`
	PlatformExtra map[string]any `json:"platform_extra,omitempty"`
}

// Album represents a unified album model across all platforms.
type Album struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Artists       []ArtistBrief  `json:"artists,omitempty"`
	Songs         []*Song        `json:"songs,omitempty"`
	Description   string         `json:"description,omitempty"`
	ReleaseDate   string         `json:"release_date,omitempty"`
	CoverURL      string         `json:"cover_url"`
	PlatformExtra map[string]any `json:"platform_extra,omitempty"`
}

// Artist represents a unified artist model across all platforms.
type Artist struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	PicURL        string         `json:"pic_url"`
	Description   string         `json:"description,omitempty"`
	HotSongs      []*Song        `json:"hot_songs,omitempty"`
	Albums        []*Album       `json:"albums,omitempty"`
	PlatformExtra map[string]any `json:"platform_extra,omitempty"`
}

// Playlist represents a unified playlist model across all platforms.
type Playlist struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	CoverURL      string         `json:"cover_url"`
	CreatorName   string         `json:"creator_name,omitempty"`
	Description   string         `json:"description,omitempty"`
	TrackCount    int            `json:"track_count"`
	Songs         []*Song        `json:"songs,omitempty"`
	PlatformExtra map[string]any `json:"platform_extra,omitempty"`
}

// LyricLine represents a single timed lyric line.
type LyricLine struct {
	Time      int64           `json:"time"` // milliseconds
	Duration  int64           `json:"duration,omitempty"`
	Text      string          `json:"text"`
	Syllables []LyricSyllable `json:"syllables,omitempty"`
}

// LyricSyllable represents a word or syllable-level lyric segment.
// Time is absolute in milliseconds. Duration is in milliseconds.
type LyricSyllable struct {
	Time     int64  `json:"time"`
	Duration int64  `json:"duration,omitempty"`
	Text     string `json:"text"`
}

// Lyric represents a unified lyric model across all platforms.
type Lyric struct {
	Raw           string         `json:"raw"` // raw decrypted lyric content
	Lines         []LyricLine    `json:"lines"`
	Translation   []LyricLine    `json:"translation,omitempty"`
	Romanization  []LyricLine    `json:"romanization,omitempty"`
	PlatformExtra map[string]any `json:"platform_extra,omitempty"`
}

// SearchResult holds paginated search results.
type SearchResult struct {
	Total     int         `json:"total"`
	Page      int         `json:"page"`
	Limit     int         `json:"limit"`
	Songs     []*Song     `json:"songs,omitempty"`
	Albums    []*Album    `json:"albums,omitempty"`
	Artists   []*Artist   `json:"artists,omitempty"`
	Playlists []*Playlist `json:"playlists,omitempty"`
}
