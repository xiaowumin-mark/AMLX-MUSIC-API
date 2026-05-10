# AMLX-MUSIC-API 完整使用文档

本文档面向库使用者和后续维护者，覆盖安装、Provider 注册、统一数据模型、各平台能力差异、歌词处理、认证、错误处理、示例运行和扩展新平台。

## 1. 项目定位

`AMLX-MUSIC-API` 是一个 Go 多平台音乐 API 聚合库。它把网易云音乐、QQ 音乐、酷狗音乐的接口封装成统一的 `MusicProvider`，调用方可以用同一套方法完成搜索、获取歌曲详情、歌词、歌单、艺人、专辑和推荐。

当前内置平台：

| 平台 | 包路径 | Provider 名 |
| --- | --- | --- |
| 网易云音乐 | `github.com/xiaowumin-mark/AMLX-MUSIC-API/netease` | `netease` |
| QQ 音乐 | `github.com/xiaowumin-mark/AMLX-MUSIC-API/qqmusic` | `qq` |
| 酷狗音乐 | `github.com/xiaowumin-mark/AMLX-MUSIC-API/kugou` | `kugou` |

## 2. 安装和导入

```bash
go get github.com/xiaowumin-mark/AMLX-MUSIC-API
```

基础导入：

```go
import musicapi "github.com/xiaowumin-mark/AMLX-MUSIC-API"
```

如果要使用内置平台，需要匿名导入对应 provider 包：

```go
import (
	musicapi "github.com/xiaowumin-mark/AMLX-MUSIC-API"

	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/kugou"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/netease"
	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/qqmusic"
)
```

匿名导入会触发 provider 包里的 `init()`，将平台注册到全局 registry。没有导入 provider 包时，`musicapi.Get("netease")` 会返回 unknown provider 错误。

## 3. 快速示例

```go
package main

import (
	"context"
	"fmt"

	musicapi "github.com/xiaowumin-mark/AMLX-MUSIC-API"

	_ "github.com/xiaowumin-mark/AMLX-MUSIC-API/netease"
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
}
```

## 4. Provider 获取和注册机制

获取平台客户端：

```go
client, err := musicapi.Get("qq")
```

查看已注册平台：

```go
providers := musicapi.ListProviders()
```

注册新平台：

```go
func init() {
	musicapi.Register("myplatform", func(opts ...musicapi.Option) (musicapi.MusicProvider, error) {
		return New(opts...)
	})
}
```

`Register` 使用 provider 名作为唯一 key，重复注册同名 provider 会 panic。

## 5. 统一接口

所有平台都实现：

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

### 5.1 Search

```go
result, err := client.Search(ctx, "周杰伦", musicapi.SearchTypeSong, 1, 20)
```

参数：

| 参数 | 说明 |
| --- | --- |
| `keyword` | 搜索关键词 |
| `searchType` | 搜索类型 |
| `page` | 页码，从 1 开始 |
| `limit` | 每页数量，不同平台会有上限 |

搜索类型：

| 常量 | 说明 |
| --- | --- |
| `SearchTypeSong` | 歌曲 |
| `SearchTypeAlbum` | 专辑 |
| `SearchTypeArtist` | 艺人 |
| `SearchTypePlaylist` | 歌单 |

注意：当前 QQ 音乐歌曲搜索接口可能不返回总数。此时 `SearchResult.Total` 会是 `-1`，调用方应使用 `len(result.Songs)` 展示本页数量。

### 5.2 GetSong

```go
song, err := client.GetSong(ctx, "002B2EAA3brD5b")
```

返回统一歌曲模型，包含歌曲名、艺人、专辑、时长、封面、付费状态和平台扩展字段。

### 5.3 GetLyric

```go
lyric, err := client.GetLyric(ctx, song.ID)
```

返回：

- `Raw`：原始歌词文本，通常是解密后的原文。
- `Lines`：主歌词时间轴。
- `Translation`：翻译歌词时间轴，平台有返回时才有。
- `Romanization`：罗马音时间轴，平台有返回时才有。

### 5.4 GetPlaylist

```go
playlist, err := client.GetPlaylist(ctx, "3778678")
```

返回歌单基础信息和曲目列表。部分平台的创建者昵称可能为空，这是接口返回限制。

### 5.5 GetArtist

```go
artist, err := client.GetArtist(ctx, "6452")
```

返回艺人基础信息和热门歌曲。网易云通常能返回较完整的艺人介绍；QQ 和酷狗当前主要返回热门歌曲，并会从歌曲艺人信息中补齐艺人名。

### 5.6 GetAlbum

```go
album, err := client.GetAlbum(ctx, "001m1S8d09nqfp")
```

返回专辑基础信息、艺人和曲目列表。

### 5.7 GetDailyRecommend

```go
songs, err := client.GetDailyRecommend(ctx)
```

推荐接口最依赖平台策略：

| 平台 | 当前行为 |
| --- | --- |
| 网易云音乐 | 匿名可尝试，登录 Cookie 更稳定 |
| QQ 音乐 | 当前接口返回推荐歌单，不直接转成歌曲，所以可能返回空 |
| 酷狗音乐 | 支持匿名推荐 |

生产代码应把空推荐视为正常结果，而不是错误。

## 6. 统一数据模型

### 6.1 Song

```go
type Song struct {
	ID            string
	Name          string
	Artists       []ArtistBrief
	Album         *AlbumBrief
	Duration      int
	CoverURL      string
	PayStatus     string
	PlatformExtra map[string]any
}
```

说明：

- `Duration` 单位为秒。
- `CoverURL` 是歌曲所属专辑封面，`Album.CoverURL` 会尽量同步填充。
- `PayStatus` 常见值包括 `free`、`vip`、`only`，不是所有平台都会返回。
- `PlatformExtra` 存放平台原生 ID、付费字段等额外信息。

### 6.2 Album

```go
type Album struct {
	ID          string
	Name        string
	Artists     []ArtistBrief
	Songs       []*Song
	Description string
	ReleaseDate string
	CoverURL    string
}
```

### 6.3 Artist

```go
type Artist struct {
	ID          string
	Name        string
	PicURL      string
	Description string
	HotSongs    []*Song
	Albums      []*Album
}
```

### 6.4 Playlist

```go
type Playlist struct {
	ID          string
	Name        string
	CoverURL    string
	CreatorName string
	Description string
	TrackCount  int
	Songs       []*Song
}
```

### 6.5 Lyric

```go
type Lyric struct {
	Raw           string
	Lines         []LyricLine
	Translation   []LyricLine
	Romanization  []LyricLine
	PlatformExtra map[string]any
}
```

`LyricLine.Time` 单位是毫秒。

逐字/音节时间轴放在 `LyricLine.Syllables`：

```go
type LyricLine struct {
	Time      int64
	Duration  int64
	Text      string
	Syllables []LyricSyllable
}

type LyricSyllable struct {
	Time     int64
	Duration int64
	Text     string
}
```

规则：

- 平台返回 YRC、QRC、KRC 等逐字格式时，优先填充 `Syllables`。
- 平台只返回 LRC 时，保留逐行歌词，`Syllables` 为空。
- 调用方可以先检查 `len(line.Syllables)`，没有逐字片段时直接使用 `line.Time` 和 `line.Text`。

## 7. 配置选项

创建 provider 时可以传入 `Option`：

```go
client, err := musicapi.Get("netease",
	musicapi.WithLyricClean(true),
	musicapi.WithCookie("MUSIC_U=xxxx;"),
)
```

| Option | 说明 |
| --- | --- |
| `WithHTTPClient(*http.Client)` | 使用自定义 HTTP client |
| `WithLyricClean(bool)` | 开启或关闭歌词清洗，默认开启 |
| `WithCustomLyricCleaner(LyricCleaner)` | 注入自定义歌词清洗器 |
| `WithCookie(string)` | 设置 Cookie |
| `WithProxy(string)` | 使用 HTTP 代理 |

### 7.1 自定义 HTTP Client

```go
httpClient := &http.Client{
	Timeout: 10 * time.Second,
}

client, err := musicapi.Get("qq", musicapi.WithHTTPClient(httpClient))
```

### 7.2 代理

```go
client, err := musicapi.Get("netease",
	musicapi.WithProxy("http://127.0.0.1:7890"),
)
```

如果你需要同时设置代理和更复杂的超时、Transport 参数，建议自己构造 `*http.Client` 后使用 `WithHTTPClient`。

### 7.3 Cookie

```go
client, err := musicapi.Get("netease",
	musicapi.WithCookie("MUSIC_U=xxxx;"),
)
```

Cookie 主要用于登录态接口，例如每日推荐。不同平台对 Cookie 的要求不同：

| 平台 | 常见 Cookie |
| --- | --- |
| 网易云音乐 | `MUSIC_U=...` |
| QQ 音乐 | 完整 QQ 音乐 Cookie 串 |
| 酷狗音乐 | token 或平台登录相关 Cookie |

## 8. 歌词处理

不同平台的歌词格式差异较大：

| 平台 | 格式 | 当前处理 |
| --- | --- | --- |
| 网易云音乐 | LRC / YRC | 解析主歌词、翻译、罗马音 |
| QQ 音乐 | QRC / LRC | QRC 解密，失败时 fallback 到 LRC |
| 酷狗音乐 | KRC / LRC | KRC 解密解压，转换为统一时间轴 |

逐字能力：

| 平台 | 逐字来源 | fallback |
| --- | --- | --- |
| 网易云音乐 | YRC | LRC |
| QQ 音乐 | QRC | LRC |
| 酷狗音乐 | KRC | LRC/空结果 |

默认启用歌词清洗，会移除常见元数据行，例如：

- 作词、作曲、编曲、制作人、录音、混音、母带等制作信息。
- 出品、发行、版权、宣推等说明。
- 常见版权警告语。

关闭清洗：

```go
client, _ := musicapi.Get("netease", musicapi.WithLyricClean(false))
```

自定义清洗：

```go
type MyCleaner struct{}

func (c *MyCleaner) Clean(raw string) (string, error) {
	return raw, nil
}

client, _ := musicapi.Get("netease",
	musicapi.WithCustomLyricCleaner(&MyCleaner{}),
)
```

## 9. 平台 ID 格式

| 平台 | 歌曲 ID | 专辑 ID | 艺人 ID | 歌单 ID |
| --- | --- | --- | --- | --- |
| 网易云音乐 | 数字 ID，如 `186664` | 数字 ID | 数字 ID | 数字 ID |
| QQ 音乐 | song MID，如 `002B2EAA3brD5b` | album MID | singer MID | 数字 disstid |
| 酷狗音乐 | Hash，如 `6E44CE19...` | 数字 album ID | 数字 author ID | collection ID，如 `collection_3_2132040296_8_0` |

从搜索结果拿到的 `Song.ID`、`Album.ID`、`Artist.ID`、`Playlist.ID` 可以直接传给对应详情方法。

## 10. 错误处理建议

平台接口可能因为网络、风控、版权、地区、会员权限、Cookie 过期等原因返回错误或空数据。建议调用方按以下方式处理：

```go
result, err := client.Search(ctx, "keyword", musicapi.SearchTypeSong, 1, 10)
if err != nil {
	// 网络、解析、平台业务错误
	return err
}

if result.Total == -1 {
	// 平台没有返回总数，展示本页数量
	fmt.Println(len(result.Songs))
}

if len(result.Songs) == 0 {
	// 空结果不是错误
	return nil
}
```

不要假设：

- `Artists` 一定非空。
- `Album` 一定非 nil。
- `CoverURL` 一定存在。
- `SearchResult.Total` 一定大于等于 0。
- 推荐接口一定返回歌曲。

## 11. 示例程序

所有示例都在 `examples/`。

```bash
go run examples/search/main.go
go run examples/song_lyric/main.go
go run examples/playlist/main.go
go run examples/artist_album/main.go
go run examples/recommend/main.go
go run examples/options/main.go
go run examples/extension/main.go
```

| 示例 | 说明 |
| --- | --- |
| `search` | 三平台搜索歌曲，并展示搜索结果 |
| `song_lyric` | 展示歌曲详情、歌词、翻译、罗马音和清洗前后差异 |
| `playlist` | 获取三平台歌单详情和曲目 |
| `artist_album` | 获取艺人热门歌曲和专辑曲目 |
| `recommend` | 获取推荐歌曲并演示 Cookie 参数 |
| `options` | 演示配置项 |
| `extension` | 演示自定义平台注册 |

推荐接口示例支持参数：

```bash
go run examples/recommend/main.go --netease-cookie="MUSIC_U=xxxx"
go run examples/recommend/main.go --qq-cookie="..."
go run examples/recommend/main.go --kugou-cookie="..."
```

## 12. 平台实现说明

### 12.1 网易云音乐

网易云实现包含：

- weapi/eapi 请求加密。
- 匿名设备注册。
- 搜索、歌曲详情、歌词、歌单、艺人、专辑、推荐。
- LRC/YRC 解析。

常见注意点：

- 推荐接口登录后更稳定。
- 部分歌词会返回 YRC，其中包含逐字时间轴，本库会转成统一行级歌词。

### 12.2 QQ 音乐

QQ 实现包含：

- `musicu.fcg` batch 请求。
- Qimei 设备信息。
- QRC 解密，失败时 fallback 到 LRC。
- 搜索、歌曲详情、歌词、歌单、艺人、专辑。

常见注意点：

- 搜索接口可能不返回总数，`Total` 会是 `-1`。
- 歌单接口对 Referer 有要求，当前实现使用 QQ 播放页 Referer。
- 推荐接口当前不直接返回歌曲列表。

### 12.3 酷狗音乐

酷狗实现包含：

- Android 风格请求参数。
- 设备注册和 dfid/mid/uuid。
- MD5/signature 签名。
- KRC 歌词解密解压。
- 搜索、歌曲详情、歌词、歌单、艺人、专辑、推荐。

常见注意点：

- 酷狗多个接口字段大小写、数字/字符串类型不稳定，模型中做了兼容。
- 歌单 ID 通常是 `collection_...` 格式。

## 13. 扩展新平台

新平台需要完成：

1. 新建 provider 包。
2. 定义 `Client`。
3. 实现 `MusicProvider` 的全部方法。
4. 在 `init()` 中调用 `musicapi.Register`。
5. 把平台原始字段转换成统一模型。

最小结构：

```text
myplatform/
├── client.go
├── models.go
└── doc.go
```

示例：

```go
package myplatform

import (
	"context"

	musicapi "github.com/xiaowumin-mark/AMLX-MUSIC-API"
)

type Client struct {
	cfg *musicapi.ClientConfig
}

func init() {
	musicapi.Register("myplatform", func(opts ...musicapi.Option) (musicapi.MusicProvider, error) {
		return New(opts...)
	})
}

func New(opts ...musicapi.Option) (*Client, error) {
	cfg := musicapi.DefaultClientConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return &Client{cfg: cfg}, nil
}

func (c *Client) Name() string { return "myplatform" }

func (c *Client) Search(ctx context.Context, keyword string, searchType musicapi.SearchType, page, limit int) (*musicapi.SearchResult, error) {
	return &musicapi.SearchResult{Page: page, Limit: limit}, nil
}

func (c *Client) GetSong(ctx context.Context, songID string) (*musicapi.Song, error) {
	return nil, nil
}

func (c *Client) GetLyric(ctx context.Context, songID string) (*musicapi.Lyric, error) {
	return nil, nil
}

func (c *Client) GetPlaylist(ctx context.Context, playlistID string) (*musicapi.Playlist, error) {
	return nil, nil
}

func (c *Client) GetArtist(ctx context.Context, artistID string) (*musicapi.Artist, error) {
	return nil, nil
}

func (c *Client) GetAlbum(ctx context.Context, albumID string) (*musicapi.Album, error) {
	return nil, nil
}

func (c *Client) GetDailyRecommend(ctx context.Context) ([]*musicapi.Song, error) {
	return nil, nil
}
```

完整可运行示例见 `examples/extension/main.go`。

## 14. 测试和验证

运行全部测试：

```bash
go test ./...
```

运行单个平台测试：

```bash
go test ./netease -v
go test ./qqmusic -v
go test ./kugou -v
```

运行示例：

```bash
go run examples/search/main.go
go run examples/song_lyric/main.go
go run examples/playlist/main.go
go run examples/artist_album/main.go
go run examples/recommend/main.go
go run examples/options/main.go
go run examples/extension/main.go
```

如果在 Windows 上连续运行多个 `go run` 时遇到 Go build cache 文件占用，可以临时设置项目内缓存：

```powershell
$env:GOCACHE=(Resolve-Path .).Path + '\.gocache'
```

## 15. 目录结构

```text
AMLX-MUSIC-API/
├── README.md
├── docs/
│   └── USER_GUIDE.md
├── models.go
├── provider.go
├── options.go
├── lyric.go
├── netease/
├── qqmusic/
├── kugou/
├── internal/
│   ├── httpclient/
│   └── util/
├── examples/
└── testdata/
```

## 16. 维护建议

平台接口会变化，维护时建议优先检查：

- 请求参数是否仍然有效。
- Header、Referer、User-Agent 是否被平台风控。
- 签名算法是否变化。
- 响应字段是否从字符串变成数字，或从对象变成数组。
- 歌词格式是否新增字段。
- 示例 ID 是否失效。

对外行为上，尽量保持统一模型不变，把平台变化收敛在 provider 内部。

## 17. 免责声明

本项目仅用于学习和研究接口调用、数据建模、歌词解密和 Go SDK 设计。请遵守相关平台服务条款、版权要求和当地法律法规，不要将本项目用于侵权、批量抓取或绕过访问限制。
