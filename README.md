# AMLX-MUSIC-API

一个用 Go 编写的多平台音乐 API 聚合库，统一封装网易云音乐、QQ 音乐、酷狗音乐的搜索、歌曲、歌词、歌单、艺人、专辑和推荐接口。

项目重点处理了各平台实际接口里的请求参数、签名、鉴权、设备信息、响应结构差异，以及歌词解密和清洗逻辑。

## 文档

- [完整使用文档](docs/USER_GUIDE.md)
- [示例程序](examples/)

## 安装

```bash
go get github.com/xiaowumin-mark/AMLX-MUSIC-API
```

当前 `go.mod` 使用 Go `1.25.1`。

## 快速开始

```go
package main

import (
	"context"
	"fmt"

	musicapi "github.com/xiaowumin-mark/AMLX-MUSIC-API"

	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/kugou"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/netease"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/qqmusic"
)

func main() {
	ctx := context.Background()

	client, err := musicapi.Get("netease")
	if err != nil {
		panic(err)
	}

	result, err := client.Search(ctx, "海阔天空", musicapi.SearchTypeSong, 1, 5)
	if err != nil {
		panic(err)
	}

	for _, song := range result.Songs {
		artist := ""
		if len(song.Artists) > 0 {
			artist = song.Artists[0].Name
		}
		fmt.Printf("%s - %s (%s)\n", song.Name, artist, song.ID)
	}

	if len(result.Songs) == 0 {
		return
	}

	lyric, err := client.GetLyric(ctx, result.Songs[0].ID)
	if err != nil {
		panic(err)
	}

	for _, line := range lyric.Lines {
		fmt.Printf("[%dms] %s\n", line.Time, line.Text)
	}
}
```

平台名：

| 平台 | Provider 名 |
| --- | --- |
| 网易云音乐 | `netease` |
| QQ 音乐 | `qq` |
| 酷狗音乐 | `kugou` |

使用某个平台前，需要匿名导入对应包，让它在 `init()` 中完成注册。

## 统一接口

所有平台都实现同一个 `MusicProvider` 接口：

```go
type MusicProvider interface {
	Name() string
	Search(ctx context.Context, keyword string, searchType SearchType, page, limit int) (*SearchResult, error)
	GetSong(ctx context.Context, songID string) (*Song, error)
	GetLyric(ctx context.Context, songID string) (*Lyric, error)
	GetPlaylist(ctx context.Context, playlistID string) (*Playlist, error)
	GetArtist(ctx context.Context, artistID string) (*Artist, error)
	GetAlbum(ctx context.Context, albumID string) (*Album, error)
	GetDailyRecommend(ctx context.Context) ([]*Song, error)
}
```

支持的搜索类型：

| 常量 | 说明 |
| --- | --- |
| `SearchTypeSong` | 歌曲 |
| `SearchTypeAlbum` | 专辑 |
| `SearchTypeArtist` | 艺人 |
| `SearchTypePlaylist` | 歌单 |

返回模型统一为 `Song`、`Album`、`Artist`、`Playlist`、`Lyric`。平台特有字段会放在 `PlatformExtra` 中。

## 功能支持

| 功能 | 网易云音乐 | QQ 音乐 | 酷狗音乐 |
| --- | --- | --- | --- |
| 歌曲搜索 | 支持 | 支持 | 支持 |
| 歌曲详情 | 支持 | 支持 | 支持 |
| 歌词获取 | 支持 LRC/YRC | 支持 QRC/LRC fallback | 支持 KRC/LRC |
| 歌单详情 | 支持 | 支持 | 支持 |
| 艺人热门歌曲 | 支持 | 支持 | 支持 |
| 专辑详情和曲目 | 支持 | 支持 | 支持 |
| 每日推荐 | 匿名可尝试，登录更稳定 | 当前返回推荐歌单，库中不转成歌曲 | 支持匿名推荐 |

说明：

- QQ 音乐搜索接口当前不稳定返回总数，库会在无法取得总数时将 `SearchResult.Total` 设为 `-1`。示例会显示“本页 N 条结果”。
- 推荐接口依赖平台策略和登录态，可能随时间变化。生产使用时建议对空结果和登录要求做兼容。
- 部分歌曲、歌词、歌单可能因版权、地区、会员权限返回空或受限，这是平台行为。

## 配置选项

```go
client, err := musicapi.Get("netease",
	musicapi.WithHTTPClient(customHTTPClient),
	musicapi.WithLyricClean(true),
	musicapi.WithCookie("MUSIC_U=xxxx;"),
	musicapi.WithProxy("http://127.0.0.1:7890"),
)
```

| Option | 说明 |
| --- | --- |
| `WithHTTPClient(*http.Client)` | 使用自定义 HTTP Client，适合测试、超时、Transport 定制 |
| `WithLyricClean(bool)` | 开启或关闭歌词清洗，默认开启 |
| `WithCustomLyricCleaner(LyricCleaner)` | 注入自定义歌词清洗器 |
| `WithCookie(string)` | 设置登录 Cookie |
| `WithProxy(string)` | 设置 HTTP 代理地址；复杂代理配置建议使用 `WithHTTPClient` |

自定义歌词清洗器：

```go
type MyCleaner struct{}

func (c *MyCleaner) Clean(raw string) (string, error) {
	return raw, nil
}

client, _ := musicapi.Get("netease",
	musicapi.WithCustomLyricCleaner(&MyCleaner{}),
)
```

## ID 格式

| 平台 | 歌曲 ID | 专辑 ID | 艺人 ID | 歌单 ID |
| --- | --- | --- | --- | --- |
| 网易云音乐 | 数字 ID，例如 `186664` | 数字 ID | 数字 ID | 数字 ID |
| QQ 音乐 | song MID，例如 `002B2EAA3brD5b`，部分接口也支持数字 ID | album MID | singer MID | 数字 disstid |
| 酷狗音乐 | Hash，例如 `6E44CE19...` | 数字 album ID | 数字 author ID | collection ID，例如 `collection_3_2132040296_8_0` |

## 歌词处理

库会尽量返回统一的歌词结构：

```go
type Lyric struct {
	Raw          string
	Lines        []LyricLine
	Translation  []LyricLine
	Romanization []LyricLine
}
```

平台差异：

| 平台 | 常见格式 | 处理方式 |
| --- | --- | --- |
| 网易云音乐 | LRC / YRC | 解析逐行歌词、翻译和罗马音 |
| QQ 音乐 | QRC / LRC | QRC 解密，失败时 fallback 到 LRC |
| 酷狗音乐 | KRC / LRC | KRC 解密解压，转换为统一时间轴 |

默认歌词清洗会去掉常见制作信息、版权声明、发行信息等元数据行。需要原始内容时：

```go
client, _ := musicapi.Get("netease", musicapi.WithLyricClean(false))
```

## 示例

示例放在 `examples/`：

```bash
go run examples/search/main.go
go run examples/song_lyric/main.go
go run examples/playlist/main.go
go run examples/artist_album/main.go
go run examples/recommend/main.go
go run examples/options/main.go
go run examples/extension/main.go
```

示例说明：

| 示例 | 内容 |
| --- | --- |
| `search` | 跨平台搜索歌曲 |
| `song_lyric` | 获取歌曲详情和歌词，展示清洗前后对比 |
| `playlist` | 获取三平台歌单详情和歌曲列表 |
| `artist_album` | 获取艺人热门歌曲和专辑详情 |
| `recommend` | 获取推荐歌曲，演示 Cookie 参数 |
| `options` | 演示 HTTP Client、歌词清洗、Cookie、代理等选项 |
| `extension` | 演示如何注册自定义平台 |

## 扩展新平台

实现 `MusicProvider` 并注册即可：

```go
package myplatform

import musicapi "github.com/xiaowumin-mark/AMLX-MUSIC-API"

type Client struct{}

func init() {
	musicapi.Register("myplatform", func(opts ...musicapi.Option) (musicapi.MusicProvider, error) {
		return New(opts...)
	})
}
```

完整示例见 `examples/extension/main.go`。

## 测试

```bash
go test ./...

go test ./netease -v
go test ./qqmusic -v
go test ./kugou -v
```

本地验证过的示例：

- `examples/search`
- `examples/song_lyric`
- `examples/playlist`
- `examples/artist_album`
- `examples/recommend`
- `examples/options`
- `examples/extension`

## 项目结构

```text
AMLX-MUSIC-API/
├── models.go              # 统一数据模型
├── provider.go            # MusicProvider 接口和注册机制
├── options.go             # Option 配置
├── lyric.go               # 歌词清洗接口和默认实现
├── netease/               # 网易云音乐实现
├── qqmusic/               # QQ 音乐实现
├── kugou/                 # 酷狗音乐实现
├── internal/httpclient/   # HTTP Client 封装
├── internal/util/         # 加密、解密、压缩等工具
├── examples/              # 可运行示例
└── testdata/              # 歌词解密测试数据
```

## 免责声明

本项目仅用于学习和研究各音乐平台接口调用、数据建模、歌词解密和 Go SDK 设计。请遵守相关平台服务条款、版权要求和当地法律法规，不要将本项目用于侵权、批量抓取或绕过访问限制。

## 许可证

MIT
