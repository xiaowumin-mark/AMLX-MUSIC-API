package musicapi

import (
	"net/http"
	"net/url"
)

// Option configures a MusicProvider or the global client behaviour.
type Option func(*ClientConfig)

// ClientConfig holds all configurable settings shared across providers.
type ClientConfig struct {
	// HTTPClient is the *http.Client used for all network requests.
	// It can be configured with a custom Transport for testing.
	HTTPClient *http.Client

	// EnableLyricClean controls whether the default LyricCleaner is
	// applied when fetching lyrics.
	EnableLyricClean bool

	// CustomLyricCleaner allows callers to supply their own LyricCleaner
	// implementation instead of the default one.
	CustomLyricCleaner LyricCleaner

	// Cookie is a raw cookie string sent with authenticated requests
	// (e.g. NetEase MUSIC_U, KuGou token).
	Cookie string

	// ProxyURL sets an HTTP proxy address (e.g. "http://127.0.0.1:10809").
	ProxyURL string
}

// DefaultClientConfig returns a ClientConfig with sensible defaults.
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		HTTPClient:         http.DefaultClient,
		EnableLyricClean:   true,
		CustomLyricCleaner: nil,
	}
}

func defaultClientConfig() *ClientConfig {
	return DefaultClientConfig()
}

// WithHTTPClient sets a custom *http.Client for network requests.
// This is especially useful in tests to inject httptest transports.
func WithHTTPClient(c *http.Client) Option {
	return func(cfg *ClientConfig) {
		if c != nil {
			cfg.HTTPClient = c
		}
	}
}

// WithLyricClean enables or disables the built-in lyrics cleaner.
// When disabled the raw decrypted lyrics are returned as-is.
func WithLyricClean(enable bool) Option {
	return func(cfg *ClientConfig) {
		cfg.EnableLyricClean = enable
	}
}

// WithCustomLyricCleaner injects a custom LyricCleaner implementation.
// Setting this implies EnableLyricClean = true.
func WithCustomLyricCleaner(c LyricCleaner) Option {
	return func(cfg *ClientConfig) {
		cfg.CustomLyricCleaner = c
		cfg.EnableLyricClean = true
	}
}

// WithCookie sets the authentication cookie string.
func WithCookie(cookie string) Option {
	return func(cfg *ClientConfig) {
		cfg.Cookie = cookie
	}
}

// WithProxy sets an HTTP proxy address.
func WithProxy(proxyURL string) Option {
	return func(cfg *ClientConfig) {
		cfg.ProxyURL = proxyURL
		if proxyURL == "" {
			return
		}
		u, err := url.Parse(proxyURL)
		if err != nil {
			return
		}
		cfg.HTTPClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(u),
			},
		}
	}
}

func applyOptions(cfg *ClientConfig, opts ...Option) {
	for _, o := range opts {
		o(cfg)
	}
}
