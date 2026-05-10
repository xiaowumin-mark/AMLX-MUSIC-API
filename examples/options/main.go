// Example: 选项模式 — 演示 WithHTTPClient、WithLyricClean、WithCustomLyricCleaner、
// WithCookie、WithProxy 以及 WithCustomLyricCleaner 所有可配置选项的用法。
//
// 运行方式:
//
//	go run examples/options/main.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/xiaowumin-mark/AMLX-MUSIC-API"

	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/netease"
)

func main() {
	ctx := context.Background()
	fmt.Println("══════════════════════════════════════════════")
	fmt.Println("  选项模式演示")
	fmt.Println("══════════════════════════════════════════════")

	// ── 1. WithHTTPClient — 注入自定义 HTTP 客户端 ──
	fmt.Println("\n── 1. WithHTTPClient ──")
	demoWithHTTPClient(ctx)

	// ── 2. WithLyricClean — 开关歌词清洗 ──
	fmt.Println("\n── 2. WithLyricClean ──")
	demoWithLyricClean(ctx)

	// ── 3. WithCustomLyricCleaner — 注入自定义清洗器 ──
	fmt.Println("\n── 3. WithCustomLyricCleaner ──")
	demoWithCustomLyricCleaner(ctx)

	// ── 4. WithCookie — 注入登录态 ──
	fmt.Println("\n── 4. WithCookie ──")
	demoWithCookie(ctx)

	// ── 5. WithProxy — 配置 HTTP 代理 ──
	fmt.Println("\n── 5. WithProxy ──")
	demoWithProxy(ctx)

	// ── 6. 组合多个 Option ──
	fmt.Println("\n── 6. 组合多个 Option ──")
	demoMultipleOptions(ctx)
}

// 1. 自定义 HTTP 客户端（设置超时、重试等）
func demoWithHTTPClient(ctx context.Context) {
	customClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
		},
	}

	client, err := musicapi.Get("netease",
		musicapi.WithHTTPClient(customClient),
	)
	if err != nil {
		fmt.Printf("  错误: %v\n", err)
		return
	}

	result, err := client.Search(ctx, "晴天", musicapi.SearchTypeSong, 1, 3)
	if err != nil {
		fmt.Printf("  搜索失败（可能无网络）: %v\n", err)
		return
	}
	fmt.Printf("  使用自定义 HTTP Client，搜索到 %d 条结果\n", result.Total)
}

// 2. 歌词清洗控制
func demoWithLyricClean(ctx context.Context) {
	// 开启清洗（默认）
	cleanClient, _ := musicapi.Get("netease", musicapi.WithLyricClean(true))
	// 关闭清洗
	rawClient, _ := musicapi.Get("netease", musicapi.WithLyricClean(false))

	lyricClean, err := cleanClient.GetLyric(ctx, "186664")
	if err == nil {
		fmt.Printf("  清洗后: %d 句歌词\n", len(lyricClean.Lines))
	}

	lyricRaw, err := rawClient.GetLyric(ctx, "186664")
	if err == nil {
		// 原始数据可能包含词/曲/编曲等元数据行
		metaCount := countMetaLines(lyricRaw.Raw)
		fmt.Printf("  未清洗: %d 句 + %d 行元数据\n", len(lyricRaw.Lines), metaCount)
	} else {
		fmt.Printf("  歌词获取失败（可能无网络）: %v\n", err)
	}
}

// 3. 自定义歌词清洗器
func demoWithCustomLyricCleaner(ctx context.Context) {
	client, _ := musicapi.Get("netease",
		musicapi.WithCustomLyricCleaner(&uppercaseCleaner{}),
	)

	lyric, err := client.GetLyric(ctx, "186664")
	if err != nil {
		fmt.Printf("  歌词获取失败（可能无网络）: %v\n", err)
		return
	}
	if len(lyric.Lines) > 0 {
		fmt.Printf("  自定义清洗后第一句: %s\n", lyric.Lines[0].Text)
	}
}

type uppercaseCleaner struct{}

func (u *uppercaseCleaner) Clean(raw string) (string, error) {
	// 示例：将所有歌词转为大写（实际场景可做更复杂的处理）
	return strings.ToUpper(raw), nil
}

// 4. Cookie 注入
func demoWithCookie(ctx context.Context) {
	// 不提供 Cookie
	noCookie, _ := musicapi.Get("netease")
	_, err := noCookie.GetDailyRecommend(ctx)
	fmt.Printf("  无 Cookie 推荐: %v\n", err)

	// 提供 Cookie（此处为假 Cookie，实际环境下替换为真实值）
	withCookie, _ := musicapi.Get("netease",
		musicapi.WithCookie("MUSIC_U=your_real_cookie_here; os=pc"),
	)
	_, err = withCookie.GetDailyRecommend(ctx)
	fmt.Printf("  有 Cookie 推荐: %v\n", err)
	fmt.Println("  (提示: 请将 MUSIC_U 替换为真实的 Cookie 值)")
}

// 5. HTTP 代理
func demoWithProxy(ctx context.Context) {
	_ = ctx
	fmt.Println("  代理配置示例（本地环境，不会实际连接）:")

	// HTTP 代理
	proxyURL, _ := url.Parse("http://127.0.0.1:10809")
	customClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
	_, err := musicapi.Get("netease",
		musicapi.WithHTTPClient(customClient),
	)
	fmt.Printf("  HTTP 代理客户端创建: %v\n", err)
}

// 6. 组合多个选项
func demoMultipleOptions(ctx context.Context) {
	customClient := &http.Client{Timeout: 15 * time.Second}

	client, err := musicapi.Get("netease",
		musicapi.WithHTTPClient(customClient),
		musicapi.WithCookie("MUSIC_U=xxx"),
		musicapi.WithLyricClean(false),
	)
	if err != nil {
		fmt.Printf("  创建客户端失败: %v\n", err)
		return
	}

	fmt.Printf("  客户端创建成功: %s\n", client.Name())
	fmt.Println("  已组合 3 个选项: 自定义 Client + Cookie + 关闭歌词清洗")
}

func countMetaLines(raw string) int {
	count := 0
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "作词") ||
			strings.Contains(trimmed, "作曲") ||
			strings.Contains(trimmed, "编曲") ||
			strings.Contains(trimmed, "制作人") ||
			strings.Contains(trimmed, "发行") {
			count++
		}
	}
	return count
}

func init() { os.Stdout.WriteString("") }
