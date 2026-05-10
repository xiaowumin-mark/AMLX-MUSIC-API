// Example: 艺人与专辑 — 获取艺人详情、热门歌曲、专辑信息及曲目列表。
//
// 运行方式:
//
//	go run examples/artist_album/main.go
package main

import (
	"context"
	"fmt"
	"os"

	musicapi "github.com/xiaowumin-mark/AMLX-MUSIC-API"

	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/kugou"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/netease"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/qqmusic"
)

// 各平台已知艺人 ID 和专辑 ID
const (
	// 网易云: 周杰伦 (艺人 ID) / 叶惠美 (专辑 ID)
	neteaseArtistID = "6452"
	neteaseAlbumID  = "18915"
	// QQ 音乐: Taylor Swift (MID)
	qqArtistID = "000qrPik2w6lDr"
	qqAlbumID  = "003ySxAn22cjfI"
	// 酷狗: 周杰伦
	kugouArtistID = "3520"
	kugouAlbumID  = "958706"
)

func main() {
	ctx := context.Background()

	// ── 一、获取艺人详情 ──
	fmt.Println("══════════════════════════════════════════════")
	fmt.Println("  艺人详情")
	fmt.Println("══════════════════════════════════════════════")

	artistConfigs := []struct {
		platform string
		id       string
		label    string
	}{
		{"netease", neteaseArtistID, "网易云 · 周杰伦"},
		{"qq", qqArtistID, "QQ 音乐 · Taylor Swift"},
		{"kugou", kugouArtistID, "酷狗 · 周杰伦"},
	}
	for _, cfg := range artistConfigs {
		demoArtist(ctx, cfg.platform, cfg.id, cfg.label)
	}

	// ── 二、获取专辑详情 ──
	fmt.Println("\n══════════════════════════════════════════════")
	fmt.Println("  专辑详情")
	fmt.Println("══════════════════════════════════════════════")

	albumConfigs := []struct {
		platform string
		id       string
		label    string
	}{
		{"netease", neteaseAlbumID, "网易云 · 叶惠美"},
		{"qq", qqAlbumID, "QQ 音乐 · 1989 (Taylor's Version)"},
		{"kugou", kugouAlbumID, "酷狗 · 专辑"},
	}
	for _, cfg := range albumConfigs {
		demoAlbum(ctx, cfg.platform, cfg.id, cfg.label)
	}
}

func demoArtist(ctx context.Context, platform, id, label string) {
	client, err := musicapi.Get(platform)
	if err != nil {
		fmt.Printf("  %s 获取客户端失败: %v\n", label, err)
		return
	}

	artist, err := client.GetArtist(ctx, id)
	if err != nil {
		fmt.Printf("  %s 获取失败: %v\n", label, err)
		return
	}

	fmt.Printf("\n── %s ──\n", label)
	fmt.Printf("  名称: %s\n", artist.Name)
	fmt.Printf("  ID: %s\n", artist.ID)
	if artist.PicURL != "" {
		fmt.Printf("  头像: %s\n", artist.PicURL)
	}
	if artist.Description != "" {
		fmt.Printf("  简介: %s\n", trunc(artist.Description, 80))
	}
	fmt.Printf("  热门歌曲: %d 首\n", len(artist.HotSongs))
	for i, s := range artist.HotSongs {
		if i >= 5 {
			fmt.Printf("    ... 共 %d 首 ...\n", len(artist.HotSongs))
			break
		}
		fmt.Printf("    %2d. %-25s | %3ds\n", i+1, trunc(s.Name, 25), s.Duration)
	}
}

func demoAlbum(ctx context.Context, platform, id, label string) {
	client, err := musicapi.Get(platform)
	if err != nil {
		fmt.Printf("  %s 获取失败: %v\n", label, err)
		return
	}

	album, err := client.GetAlbum(ctx, id)
	if err != nil {
		fmt.Printf("  %s 获取失败: %v\n", label, err)
		return
	}

	fmt.Printf("\n── %s ──\n", label)
	fmt.Printf("  名称: %s\n", album.Name)
	if album.ReleaseDate != "" {
		fmt.Printf("  发行: %s\n", album.ReleaseDate)
	}
	if album.CoverURL != "" {
		fmt.Printf("  封面: %s\n", album.CoverURL)
	}
	if album.Description != "" {
		fmt.Printf("  简介: %s\n", trunc(album.Description, 80))
	}

	fmt.Printf("  曲目: %d 首\n", len(album.Songs))
	for i, s := range album.Songs {
		if i >= 8 {
			fmt.Printf("    ... 共 %d 首 ...\n", len(album.Songs))
			break
		}
		artists := joinArtists(s.Artists)
		fmt.Printf("    %2d. %-25s | %-12s | %3ds\n",
			i+1, trunc(s.Name, 25), artists, s.Duration)
	}
}

func joinArtists(as []musicapi.ArtistBrief) string {
	if len(as) == 0 {
		return ""
	}
	return as[0].Name
}

func trunc(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n-1]) + "…"
	}
	return s
}

func init() { os.Stdout.WriteString("") }
