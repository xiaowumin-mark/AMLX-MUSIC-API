// Example: 歌单信息 — 获取各平台热门歌单的详情和歌曲列表。
//
// 运行方式:
//
//	go run examples/playlist/main.go
//
// ID 来源：可从搜索示例中获取歌单 ID，也可以直接从各平台官网上复制。
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/xiaowumin-mark/AMLX-MUSIC-API"

	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/kugou"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/netease"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/qqmusic"
)

// 各平台已知歌单 ID
const (
	// 网易云: 官方榜 - 热歌榜 (数字 ID)
	neteasePlaylistID = "3778678"
	// QQ 音乐: 官方歌单 - 欧美 | 流行节奏控
	qqPlaylistID = "7256912512"
	// 酷狗: 推荐歌单 (Collection ID)
	kugouPlaylistID = "collection_3_2132040296_8_0"
)

func main() {
	ctx := context.Background()

	configs := []struct {
		platform string
		id       string
		label    string
	}{
		{"netease", neteasePlaylistID, "网易云音乐 · 热歌榜"},
		{"qq", qqPlaylistID, "QQ 音乐 · 欧美流行节奏控"},
		{"kugou", kugouPlaylistID, "酷狗音乐 · 推荐歌单"},
	}

	for _, cfg := range configs {
		fmt.Printf("\n══════════════════════════════════════════════\n")
		fmt.Printf("  %s\n", cfg.label)
		fmt.Printf("══════════════════════════════════════════════\n")

		client, err := musicapi.Get(cfg.platform)
		if err != nil {
			fmt.Printf("  获取客户端失败: %v\n", err)
			continue
		}

		playlist, err := client.GetPlaylist(ctx, cfg.id)
		if err != nil {
			fmt.Printf("  获取歌单失败: %v\n", err)
			continue
		}

		fmt.Printf("  名称: %s\n", playlist.Name)
		fmt.Printf("  创建者: %s\n", playlist.CreatorName)
		fmt.Printf("  歌曲数: %d\n", playlist.TrackCount)
		if playlist.CoverURL != "" {
			fmt.Printf("  封面: %s\n", playlist.CoverURL)
		}
		if playlist.Description != "" {
			fmt.Printf("  描述: %s\n", trunc(playlist.Description, 100))
		}

		// 展示前 10 首歌曲
		fmt.Printf("\n  【歌曲列表】\n")
		showCount := 10
		if len(playlist.Songs) < showCount {
			showCount = len(playlist.Songs)
		}
		for i := range showCount {
			s := playlist.Songs[i]
			artist := ""
			if len(s.Artists) > 0 {
				artist = s.Artists[0].Name
			}
			fmt.Printf("    %2d. %-25s | %-15s | %3ds\n",
				i+1, trunc(s.Name, 25), artist, s.Duration)
		}
		if len(playlist.Songs) > showCount {
			fmt.Printf("    ... 共 %d 首 ...\n", len(playlist.Songs))
		}
	}
}

func trunc(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n-1]) + "…"
	}
	return s
}

func init() { os.Stdout.WriteString("") }
