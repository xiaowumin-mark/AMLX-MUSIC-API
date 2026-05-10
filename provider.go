package musicapi

import (
	"context"
	"fmt"
	"sync"
)

// MusicProvider defines the core interface that every music platform must
// implement. It provides a unified API for searching, fetching song/album/
// artist/playlist details, and retrieving lyrics.
type MusicProvider interface {
	// Name returns the platform identifier (e.g. "netease", "kugou", "qq").
	Name() string

	// Search performs a search query against the platform's API.
	// keyword is the search text; searchType specifies the category;
	// page and limit control pagination.
	Search(ctx context.Context, keyword string, searchType SearchType, page, limit int) (*SearchResult, error)

	// GetSong fetches detailed information for a single song by its platform ID.
	GetSong(ctx context.Context, songID string) (*Song, error)

	// GetLyric retrieves and decrypts lyrics for a song. If a LyricCleaner is
	// configured, the result is automatically cleaned.
	GetLyric(ctx context.Context, songID string) (*Lyric, error)

	// GetPlaylist fetches playlist details and its track list.
	GetPlaylist(ctx context.Context, playlistID string) (*Playlist, error)

	// GetArtist fetches artist details along with hot songs and albums.
	GetArtist(ctx context.Context, artistID string) (*Artist, error)

	// GetAlbum fetches album details and its track list.
	GetAlbum(ctx context.Context, albumID string) (*Album, error)

	// GetDailyRecommend returns daily song recommendations.
	// This typically requires a logged-in session; anonymous users may
	// receive an error.
	GetDailyRecommend(ctx context.Context) ([]*Song, error)
}

// LyricDecryptor is an optional interface for platforms that encrypt their
// lyrics data. Each provider can implement this separately.
//
// Decrypt receives the raw encrypted lyric string and returns the
// decrypted plaintext.
type LyricDecryptor interface {
	Decrypt(raw string) (string, error)
}

// LyricCleaner is an optional post-processing step applied to decrypted
// lyrics. It can strip metadata lines (composer, arranger, production
// credits, etc.) and copyright notices.
//
// Clean receives the raw decrypted lyric string and returns a cleaned
// version. The default implementation strips metadata based on a
// configurable keyword/regex list.
type LyricCleaner interface {
	Clean(raw string) (string, error)
}

// ProviderFactory is a function that constructs a MusicProvider instance
// with the given options.
type ProviderFactory func(opts ...Option) (MusicProvider, error)

var (
	registryMu sync.RWMutex
	providers  = map[string]ProviderFactory{}
)

// Register associates a factory with a platform name. It must be called
// during package initialisation (typically in an init() function of the
// provider package). Panics if the name is already registered.
func Register(name string, factory ProviderFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := providers[name]; ok {
		panic(fmt.Sprintf("musicapi: provider %q already registered", name))
	}
	providers[name] = factory
}

// Get creates a MusicProvider for the given platform name by calling its
// registered factory. Additional options can be passed to customise the
// resulting client.
func Get(name string, opts ...Option) (MusicProvider, error) {
	registryMu.RLock()
	factory, ok := providers[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("musicapi: unknown provider %q", name)
	}
	return factory(opts...)
}

// ListProviders returns the names of all registered providers.
func ListProviders() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(providers))
	for n := range providers {
		names = append(names, n)
	}
	return names
}
