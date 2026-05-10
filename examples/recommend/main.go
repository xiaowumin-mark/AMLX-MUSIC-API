// Example: 每日推荐 — 获取每日推荐歌曲，含 Cookie 登录说明。
//
// 运行方式:
//
//	go run examples/recommend/main.go
//
// 注意: 每日推荐接口通常需要登录态 (Cookie)。
// 网易云: 登录后从浏览器 devtools → Application → Cookies → 复制 MUSIC_U 字段
// QQ 音乐: 需要完整的 Cookie 串
// 酷狗: 需要 token
//
// 使用方式:
//
//	go run examples/recommend/main.go --netease-cookie="MUSIC_U=xxx" --qq-cookie="xxx"
//
// 或在代码中直接修改常量。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/xiaowumin-mark/AMLX-MUSIC-API"

	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/kugou"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/netease"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/qqmusic"
)

var (
	neteaseCookie = flag.String("netease-cookie", "", "网易云音乐 MUSIC_U cookie")
	qqCookie      = flag.String("qq-cookie", "", "QQ 音乐 cookie")
	kugouCookie   = flag.String("kugou-cookie", "", "酷狗音乐 cookie")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	// ── 演示 1: 网易云 每日推荐（无需 Cookie 理论也能匿名获取） ──
	fmt.Println("══════════════════════════════════════════════")
	fmt.Println("  每日推荐")
	fmt.Println("══════════════════════════════════════════════")

	// 网易云
	fmt.Println("\n── 网易云音乐 ──")
	tryRecommend(ctx, "netease", *neteaseCookie, "网易云")

	// QQ 音乐
	fmt.Println("\n── QQ 音乐 ──")
	tryRecommend(ctx, "qq", *qqCookie, "QQ 音乐")

	// 酷狗
	fmt.Println("\n── 酷狗音乐 ──")
	tryRecommend(ctx, "kugou", *kugouCookie, "酷狗")

	// ── 说明 ──
	fmt.Println("\n──────────────────────────────────────────────")
	fmt.Println("  使用说明:")
	fmt.Println("  1. 网易云登录: 浏览器打开 music.163.com → F12 → Application →")
	fmt.Println("     Cookies → 复制 MUSIC_U 值")
	fmt.Println("  2. 运行命令:")
	fmt.Println("     go run examples/recommend/main.go \\")
	fmt.Println("       --netease-cookie=\"MUSIC_U=xxxxxxxxx\"")
	fmt.Println("  3. 未提供 Cookie 时部分 API 可能返回 code=-462（需要登录）")
	fmt.Println("──────────────────────────────────────────────")
}

func tryRecommend(ctx context.Context, platform, cookie, label string) {
	opts := []musicapi.Option{}
	if cookie != "" {
		opts = append(opts, musicapi.WithCookie(cookie))
		fmt.Printf("  已加载 %s Cookie\n", label)
	} else {
		fmt.Printf("  未提供 Cookie，尝试匿名访问 %s ...\n", label)
	}

	client, err := musicapi.Get(platform, opts...)
	if err != nil {
		fmt.Printf("  获取客户端失败: %v\n", err)
		return
	}

	songs, err := client.GetDailyRecommend(ctx)
	if err != nil {
		fmt.Printf("  获取推荐失败: %v\n", err)
		fmt.Printf("  (提示: 该平台每日推荐可能需要登录 Cookie)\n")
		return
	}

	if len(songs) == 0 {
		fmt.Println("  暂无推荐歌曲")
		return
	}

	fmt.Printf("  共 %d 首推荐歌曲:\n", len(songs))
	for i, s := range songs {
		if i >= 10 {
			fmt.Printf("  ... 共 %d 首 ...\n", len(songs))
			break
		}
		artists := ""
		if len(s.Artists) > 0 {
			artists = s.Artists[0].Name
		}
		fmt.Printf("    %2d. %-30s | %-12s | %ds\n",
			i+1, trunc(s.Name, 30), artists, s.Duration)
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
