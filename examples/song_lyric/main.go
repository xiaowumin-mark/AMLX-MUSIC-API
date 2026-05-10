// Example: 歌曲详情与歌词 — 获取歌曲信息、解密与清洗歌词，并对比清洗前后的效果。
//
// 运行方式:
//
//	go run examples/song_lyric/main.go
//
// 可按需修改 neteaseSongID / kugouSongID / qqSongID 来测试不同平台歌曲。
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	musicapi "github.com/xiaowumin-mark/AMLX-MUSIC-API"

	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/kugou"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/netease"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/qqmusic"
)

// 各平台已知歌曲 ID 示例（可从 Search 示例中获取）
const (
	// 网易云: 周杰伦 - 安静 (数字 ID)
	neteaseSongID = "186664"
	// QQ 音乐: 周杰伦 - 夜曲 (MID 格式)
	qqSongID = "002B2EAA3brD5b"
	// 酷狗: 周杰伦 - 简单爱 (Hash 格式)
	kugouSongID = "6E44CE19BE5872384FA184B995CCE23B"
)

func main() {
	ctx := context.Background()

	// ── 方案 A: 启用歌词清洗（默认） ──
	fmt.Println("══════════════════════════════════════════════")
	fmt.Println("  歌曲详情 + 歌词（清洗已启用 - 默认）")
	fmt.Println("══════════════════════════════════════════════")
	nClient, _ := musicapi.Get("netease")
	demoSongLyric(ctx, nClient, neteaseSongID, "网易云音乐")

	// ── 方案 B: 关闭歌词清洗（保留原始元数据） ──
	fmt.Println("\n══════════════════════════════════════════════")
	fmt.Println("  歌曲详情 + 歌词（清洗已关闭 - 对比原始数据）")
	fmt.Println("══════════════════════════════════════════════")
	nRaw, _ := musicapi.Get("netease", musicapi.WithLyricClean(false))
	demoSongLyric(ctx, nRaw, neteaseSongID, "网易云音乐(原始)")

	// ── 方案 C: 跨平台对比同一首歌 ──
	fmt.Println("\n══════════════════════════════════════════════")
	fmt.Println("  跨平台歌词 对比")
	fmt.Println("══════════════════════════════════════════════")

	for _, cfg := range []struct {
		name  string
		id    string
		label string
	}{
		{"netease", neteaseSongID, "网易云音乐"},
		{"qq", qqSongID, "QQ 音乐"},
		{"kugou", kugouSongID, "酷狗音乐"},
	} {
		client, err := musicapi.Get(cfg.name)
		if err != nil {
			fmt.Printf("  %s 获取客户端失败: %v\n", cfg.label, err)
			continue
		}
		fmt.Printf("\n── %s ──\n", cfg.label)
		demoSongLyric(ctx, client, cfg.id, cfg.label)
	}
}

func demoSongLyric(ctx context.Context, client musicapi.MusicProvider, songID, label string) {
	// 获取歌曲详情
	song, err := client.GetSong(ctx, songID)
	if err != nil {
		fmt.Printf("  获取歌曲详情失败: %v\n", err)
		return
	}
	printSong(song)

	// 获取歌词
	lyric, err := client.GetLyric(ctx, songID)
	if err != nil {
		fmt.Printf("  获取歌词失败: %v\n", err)
		return
	}
	printLyric(lyric)
}

func printSong(s *musicapi.Song) {
	fmt.Printf("  歌曲: %s\n", s.Name)
	fmt.Printf("  艺人: %s\n", artistsJoin(s.Artists))
	fmt.Printf("  专辑: %s\n", s.Album.Name)
	fmt.Printf("  时长: %d 秒\n", s.Duration)
	if s.CoverURL != "" {
		fmt.Printf("  封面: %s\n", s.CoverURL)
	}
	if s.PayStatus != "" {
		fmt.Printf("  付费: %s\n", s.PayStatus)
	}
	if s.PlatformExtra != nil {
		for k, v := range s.PlatformExtra {
			fmt.Printf("  平台属性 [%s]: %v\n", k, v)
		}
	}
}

func printLyric(l *musicapi.Lyric) {
	fmt.Printf("\n  【歌词】共 %d 句 + %d 翻译 + %d 罗马音\n",
		len(l.Lines), len(l.Translation), len(l.Romanization))

	// 打印前 8 句歌词
	showCount := 8
	if len(l.Lines) < showCount {
		showCount = len(l.Lines)
	}
	for i := range showCount {
		line := l.Lines[i]
		ts := fmtTS(line.Time)
		fmt.Printf("    %s %s\n", ts, line.Text)

		// 有对应翻译时一并显示
		if i < len(l.Translation) && l.Translation[i].Time == line.Time {
			fmt.Printf("        (译) %s\n", l.Translation[i].Text)
		}
	}
	if len(l.Lines) > showCount {
		fmt.Printf("    ... 共 %d 句 ...\n", len(l.Lines))
	}

	// 显示原始歌词的前几行（不含时间戳的行元数据已在清洗后移除）
	if strings.Count(l.Raw, "\n") > 3 {
		fmt.Printf("\n  【原始歌词前 5 行】\n")
		for i, line := range strings.SplitN(l.Raw, "\n", 6) {
			if i == 5 {
				break
			}
			fmt.Printf("    %s\n", trunc(line, 80))
		}
	}
}

func artistsJoin(artists []musicapi.ArtistBrief) string {
	names := make([]string, len(artists))
	for i, a := range artists {
		names[i] = a.Name
	}
	return strings.Join(names, " / ")
}

func fmtTS(ms int64) string {
	m := ms / 60000
	s := (ms % 60000) / 1000
	return fmt.Sprintf("[%02d:%02d]", m, s)
}

func trunc(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n-1]) + "…"
	}
	return s
}

func init() { os.Stdout.WriteString("") }
