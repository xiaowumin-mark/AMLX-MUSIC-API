// Example: 跨平台搜索 — 在 QQ 音乐、网易云音乐、酷狗音乐中搜索歌曲、专辑、艺人、歌单。
//
// 运行方式:
//
//	go run examples/search/main.go
//
// 可修改 searchKeyword 和 searchType 变量来测试不同的搜索场景。
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/xiaowumin-mark/AMLX-MUSIC-API"

	// 导入各平台实现以触发 init() 注册
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/kugou"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/netease"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/qqmusic"
)

func main() {
	ctx := context.Background()

	// ── 修改这些变量来测试不同的搜索 ──
	searchKeyword := "海阔天空"
	searchType := musicapi.SearchTypeSong // 可改为 SearchTypeAlbum / Artist / Playlist
	page := 1
	limit := 5

	typeLabel := searchTypeLabel(searchType)

	fmt.Printf("══════════════════════════════════════════════\n")
	fmt.Printf("  跨平台搜索 \"%s\" (%s)\n", searchKeyword, typeLabel)
	fmt.Printf("══════════════════════════════════════════════\n\n")

	platforms := []string{"netease", "kugou", "qq"}
	for _, p := range platforms {
		fmt.Printf("── %s ──\n", platformLabel(p))
		searchOn(ctx, p, searchKeyword, searchType, page, limit)
		fmt.Println()
	}
}

func searchOn(ctx context.Context, platform string, keyword string, st musicapi.SearchType, page, limit int) {
	client, err := musicapi.Get(platform)
	if err != nil {
		fmt.Printf("  获取客户端失败: %v\n", err)
		return
	}

	result, err := client.Search(ctx, keyword, st, page, limit)
	if err != nil {
		fmt.Printf("  搜索失败: %v\n", err)
		return
	}

	if result.Total >= 0 {
		fmt.Printf("  共 %d 条结果 (第 %d/%d 页)\n", result.Total, page, limit)
	} else {
		fmt.Printf("  本页 %d 条结果 (第 %d 页，每页 %d 条；平台未返回总数)\n", searchResultCount(result), page, limit)
	}
	printSearchResults(result)
}

func searchResultCount(r *musicapi.SearchResult) int {
	return len(r.Songs) + len(r.Albums) + len(r.Artists) + len(r.Playlists)
}

func printSearchResults(r *musicapi.SearchResult) {
	if len(r.Songs) > 0 {
		fmt.Println("  【歌曲】")
		for _, s := range r.Songs {
			fmt.Printf("    %-30s | %s | %s\n", trunc(s.Name, 30), artistsStr(s.Artists), s.ID)
		}
	}
	if len(r.Albums) > 0 {
		fmt.Println("  【专辑】")
		for _, a := range r.Albums {
			fmt.Printf("    %-30s | %s | %s\n", trunc(a.Name, 30), artistsStr(a.Artists), a.ID)
		}
	}
	if len(r.Artists) > 0 {
		fmt.Println("  【艺人】")
		for _, a := range r.Artists {
			fmt.Printf("    %-20s | %s\n", a.Name, a.ID)
		}
	}
	if len(r.Playlists) > 0 {
		fmt.Println("  【歌单】")
		for _, p := range r.Playlists {
			fmt.Printf("    %-30s | %d 首 | %s\n", trunc(p.Name, 30), p.TrackCount, p.ID)
		}
	}
}

func searchTypeLabel(st musicapi.SearchType) string {
	switch st {
	case musicapi.SearchTypeSong:
		return "歌曲"
	case musicapi.SearchTypeAlbum:
		return "专辑"
	case musicapi.SearchTypeArtist:
		return "艺人"
	case musicapi.SearchTypePlaylist:
		return "歌单"
	default:
		return "未知"
	}
}

func platformLabel(p string) string {
	switch p {
	case "netease":
		return "网易云音乐"
	case "qq":
		return "QQ 音乐"
	case "kugou":
		return "酷狗音乐"
	default:
		return p
	}
}

func artistsStr(artists []musicapi.ArtistBrief) string {
	names := make([]string, len(artists))
	for i, a := range artists {
		names[i] = a.Name
	}
	return strings.Join(names, "/")
}

func trunc(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n-1]) + "…"
	}
	return s
}

func init() {
	// 禁止非必要输出
	os.Stdout.WriteString("")
}
