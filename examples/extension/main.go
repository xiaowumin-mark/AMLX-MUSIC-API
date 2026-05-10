// Example: 扩展新平台 — 演示如何实现 MusicProvider 接口并注册自定义音乐平台。
//
// 运行方式:
//
//	go run examples/extension/main.go
//
// 核心流程:
//  1. 定义一个结构体实现 musicapi.MusicProvider 接口
//  2. 在 init() 中调用 musicapi.Register() 注册
//  3. 调用方通过 musicapi.Get("myplatform") 透明获取实例
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/xiaowumin-mark/AMLX-MUSIC-API"
)

// ═══════════════════════════════════════════════════════════════════════
// 第一步: 定义你的平台客户端
// ═══════════════════════════════════════════════════════════════════════

// MyPlatformClient 实现 musicapi.MusicProvider 接口。
type MyPlatformClient struct {
	cfg *musicapi.ClientConfig
}

// ═══════════════════════════════════════════════════════════════════════
// 第二步: 在 init() 中注册
// ═══════════════════════════════════════════════════════════════════════

func init() {
	musicapi.Register("myplatform", func(opts ...musicapi.Option) (musicapi.MusicProvider, error) {
		cfg := musicapi.DefaultClientConfig()
		for _, o := range opts {
			o(cfg)
		}
		return &MyPlatformClient{cfg: cfg}, nil
	})
}

// ═══════════════════════════════════════════════════════════════════════
// 第三步: 实现 MusicProvider 全部方法
// ═══════════════════════════════════════════════════════════════════════

func (c *MyPlatformClient) Name() string {
	return "myplatform"
}

func (c *MyPlatformClient) Search(ctx context.Context, keyword string, st musicapi.SearchType, page, limit int) (*musicapi.SearchResult, error) {
	// 这里应实现真实的 API 请求逻辑，示例中返回模拟数据。
	fmt.Printf("  [MyPlatform] 搜索: keyword=%q type=%d page=%d\n", keyword, st, page)

	result := &musicapi.SearchResult{
		Total: 2,
		Page:  page,
		Limit: limit,
		Songs: []*musicapi.Song{
			{
				ID:   "my_001",
				Name: "示例歌曲 A - " + keyword,
				Artists: []musicapi.ArtistBrief{
					{ID: "art_001", Name: "示例艺人"},
				},
				Album:    &musicapi.AlbumBrief{ID: "alb_001", Name: "示例专辑"},
				Duration: 240,
				CoverURL: "https://example.com/cover.jpg",
				PlatformExtra: map[string]any{
					"quality": "HQ",
				},
			},
			{
				ID:   "my_002",
				Name: "示例歌曲 B - " + keyword,
				Artists: []musicapi.ArtistBrief{
					{ID: "art_002", Name: "另一位艺人"},
				},
				Album:    &musicapi.AlbumBrief{ID: "alb_002", Name: "精选集"},
				Duration: 300,
			},
		},
	}
	return result, nil
}

func (c *MyPlatformClient) GetSong(ctx context.Context, songID string) (*musicapi.Song, error) {
	fmt.Printf("  [MyPlatform] 获取歌曲: %s\n", songID)
	return &musicapi.Song{
		ID:       songID,
		Name:     "示例歌曲",
		Duration: 250,
		Artists:  []musicapi.ArtistBrief{{ID: "art_001", Name: "示例艺人"}},
		Album:    &musicapi.AlbumBrief{ID: "alb_001", Name: "示例专辑"},
		CoverURL: "https://example.com/cover_big.jpg",
	}, nil
}

func (c *MyPlatformClient) GetLyric(ctx context.Context, songID string) (*musicapi.Lyric, error) {
	fmt.Printf("  [MyPlatform] 获取歌词: %s\n", songID)

	raw := `[00:00.00]这是自定义平台的歌词
[00:05.00]第二句歌词文本
[00:10.00]第三句歌词内容`

	lyric := &musicapi.Lyric{
		Raw:   raw,
		Lines: musicapi.LRCParse(raw),
	}

	// 如果启用了歌词清洗，手动调用（实际实现中应通过 Option 控制）
	if c.cfg.EnableLyricClean && c.cfg.CustomLyricCleaner != nil {
		cleaned, _ := c.cfg.CustomLyricCleaner.Clean(raw)
		lyric.Lines = musicapi.LRCParse(cleaned)
	}

	return lyric, nil
}

func (c *MyPlatformClient) GetPlaylist(ctx context.Context, playlistID string) (*musicapi.Playlist, error) {
	fmt.Printf("  [MyPlatform] 获取歌单: %s\n", playlistID)
	return &musicapi.Playlist{
		ID:          playlistID,
		Name:        "示例歌单",
		CoverURL:    "https://example.com/playlist.jpg",
		CreatorName: "示例用户",
		Description: "这是一个自定义平台的示例歌单",
		TrackCount:  1,
		Songs: []*musicapi.Song{
			{ID: "my_001", Name: "歌单中的歌曲", Duration: 200,
				Artists: []musicapi.ArtistBrief{{ID: "art_001", Name: "示例艺人"}}},
		},
	}, nil
}

func (c *MyPlatformClient) GetArtist(ctx context.Context, artistID string) (*musicapi.Artist, error) {
	fmt.Printf("  [MyPlatform] 获取艺人: %s\n", artistID)
	return &musicapi.Artist{
		ID:     artistID,
		Name:   "示例艺人",
		PicURL: "https://example.com/artist.jpg",
		HotSongs: []*musicapi.Song{
			{ID: "my_001", Name: "热门歌曲 1", Duration: 240,
				Artists: []musicapi.ArtistBrief{{Name: "示例艺人"}}},
			{ID: "my_002", Name: "热门歌曲 2", Duration: 300,
				Artists: []musicapi.ArtistBrief{{Name: "示例艺人"}}},
		},
	}, nil
}

func (c *MyPlatformClient) GetAlbum(ctx context.Context, albumID string) (*musicapi.Album, error) {
	fmt.Printf("  [MyPlatform] 获取专辑: %s\n", albumID)
	return &musicapi.Album{
		ID:          albumID,
		Name:        "示例专辑",
		CoverURL:    "https://example.com/album_cover.jpg",
		Description: "示例专辑描述",
		ReleaseDate: "2025-01-01",
		Artists:     []musicapi.ArtistBrief{{ID: "art_001", Name: "示例艺人"}},
		Songs: []*musicapi.Song{
			{ID: "my_001", Name: "专辑曲目 1", Duration: 240,
				Artists: []musicapi.ArtistBrief{{Name: "示例艺人"}}},
			{ID: "my_002", Name: "专辑曲目 2", Duration: 300,
				Artists: []musicapi.ArtistBrief{{Name: "示例艺人"}}},
		},
	}, nil
}

func (c *MyPlatformClient) GetDailyRecommend(ctx context.Context) ([]*musicapi.Song, error) {
	fmt.Println("  [MyPlatform] 获取每日推荐")
	return []*musicapi.Song{
		{ID: "rec_001", Name: "今日推荐 1", Duration: 200,
			Artists: []musicapi.ArtistBrief{{Name: "推荐艺人"}}},
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════
// 第四步: 使用自定义平台（与内置平台完全相同的调用方式）
// ═══════════════════════════════════════════════════════════════════════

func main() {
	ctx := context.Background()

	// 查看所有已注册的平台
	fmt.Println("══════════════════════════════════════════════")
	fmt.Println("  已注册平台:", musicapi.ListProviders())
	fmt.Println("══════════════════════════════════════════════")

	// 获取自定义平台客户端（与内置平台相同的调用方式）
	client, err := musicapi.Get("myplatform")
	if err != nil {
		fmt.Printf("获取自定义平台失败: %v\n", err)
		return
	}
	fmt.Printf("\n已获取平台: %s\n\n", client.Name())

	// ── 调用全部 API ──
	fmt.Println("── 1. Search ──")
	result, _ := client.Search(ctx, "关键词", musicapi.SearchTypeSong, 1, 5)
	fmt.Printf("  结果: %d 条\n", result.Total)
	fmt.Printf("  第一首歌: %s\n", result.Songs[0].Name)

	fmt.Println("\n── 2. GetSong ──")
	song, _ := client.GetSong(ctx, "my_001")
	fmt.Printf("  歌曲: %s (%ds)\n", song.Name, song.Duration)

	fmt.Println("\n── 3. GetLyric ──")
	lyric, _ := client.GetLyric(ctx, "my_001")
	fmt.Printf("  歌词行数: %d\n", len(lyric.Lines))
	for _, line := range lyric.Lines {
		fmt.Printf("    [%02d:%02d] %s\n", line.Time/60000, (line.Time%60000)/1000, line.Text)
	}

	fmt.Println("\n── 4. GetPlaylist ──")
	pl, _ := client.GetPlaylist(ctx, "pl_001")
	fmt.Printf("  歌单: %s (%d 首)\n", pl.Name, pl.TrackCount)

	fmt.Println("\n── 5. GetArtist ──")
	artist, _ := client.GetArtist(ctx, "art_001")
	fmt.Printf("  艺人: %s (%d 首热门)\n", artist.Name, len(artist.HotSongs))

	fmt.Println("\n── 6. GetAlbum ──")
	album, _ := client.GetAlbum(ctx, "alb_001")
	fmt.Printf("  专辑: %s (%d 曲目, %s)\n", album.Name, len(album.Songs), album.ReleaseDate)

	fmt.Println("\n── 7. GetDailyRecommend ──")
	rec, _ := client.GetDailyRecommend(ctx)
	fmt.Printf("  推荐: %d 首\n", len(rec))

	// ── 可以使用 Option 配置自定义平台 ──
	fmt.Println("\n── 8. 使用 Option 配置 ──")
	customClient, _ := musicapi.Get("myplatform",
		musicapi.WithCookie("my_token=abc123"),
		musicapi.WithLyricClean(false),
	)
	fmt.Printf("  已创建带 Option 的自定义客户端: %s\n", customClient.Name())

	fmt.Println("\n──────────────────────────────────────────────")
	fmt.Println("  扩展新平台的步骤:")
	fmt.Println("  1. 定义结构体实现 MusicProvider 所有方法")
	fmt.Println("  2. 在 init() 中调用 musicapi.Register()")
	fmt.Println("  3. 调用方 import 你的包即可透明使用")
	fmt.Println("──────────────────────────────────────────────")
}

func init() { os.Stdout.WriteString("") }
