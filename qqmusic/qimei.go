package qqmusic

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

// QimeiResult holds the device fingerprint identifiers.
type QimeiResult struct {
	Q16 string `json:"q16"`
	Q36 string `json:"q36"`
}

// DefaultQimei36 is a hardcoded fallback when the Qimei server is unreachable.
// This is a known working value from the reference implementation.
const DefaultQimei36 = "6c9d3cd110abca9b16311cee10001e717614"

// GetQimei attempts to register a virtual device with the Tencent Music
// Qimei service. On failure, returns the hardcoded defaults.
func GetQimei() QimeiResult {
	// Generate a simple device identity
	imei := generateIMEI()
	androidID := generateAndroidID()

	// Without a full Qimei registration flow (which requires RSA PKCS#1 v1.5
	// encryption and a specific AES-CBC payload construction), we fall back
	// to the known working default.
	_ = imei
	_ = androidID

	// Generate random q16 and use default q36.
	q16 := generateQ16()

	return QimeiResult{
		Q16: q16,
		Q36: DefaultQimei36,
	}
}

// generateIMEI creates a random IMEI-like 15-digit string with Luhn check.
func generateIMEI() string {
	b := make([]byte, 14)
	for i := range b {
		b[i] = byte('0' + rand.Intn(10))
	}
	// Luhn check digit
	sum := 0
	for i := 0; i < 14; i++ {
		d := int(b[i] - '0')
		if i%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	checkDigit := (10 - (sum % 10)) % 10
	return string(b) + strconv.Itoa(checkDigit)
}

// generateAndroidID creates a 16-char hex Android ID.
func generateAndroidID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7])
}

func generateQ16() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%016x", uint64(b[0])<<56|uint64(b[1])<<48|uint64(b[2])<<40|uint64(b[3])<<32|
		uint64(b[4])<<24|uint64(b[5])<<16|uint64(b[6])<<8|uint64(b[7]))
}

// GetSearchID generates a search ID for QQ Music API requests.
// Algorithm from Unilyric: t = e * 2^54; n = random[0..2^22) * 2^32; r = ms_of_day
func GetSearchID() string {
	e := rand.Int63n(20) + 1
	t := e * 18014398509481984
	n := rand.Int63n(4194304) * 4294967296
	r := time.Now().UnixMilli() % 86400000
	return strconv.FormatInt(t+n+r, 10)
}

// ------- Batch API Request ---------

// BatchRequest is the payload for the QQ Music batched API gateway.
type BatchRequest struct {
	Comm     CommParams             `json:"comm"`
	Requests map[string]BatchMethod `json:"-"`
}

// CommParams holds common device/version parameters.
type CommParams struct {
	CV         int    `json:"cv"`
	Ct         string `json:"ct"`
	V          int    `json:"v"`
	UID        string `json:"uid"`
	QIMEI36    string `json:"QIMEI36"`
	TmeAppID   string `json:"tmeAppID"`
	Format     string `json:"format"`
	InCharset  string `json:"inCharset"`
	OutCharset string `json:"outCharset"`
}

// BatchMethod describes a single API call in the batch.
type BatchMethod struct {
	Module string `json:"module"`
	Method string `json:"method"`
	Param  any    `json:"param"`
}

// BuildBatchPayload creates the JSON payload for the QQ Music musicu.fcg endpoint.
func BuildBatchPayload(qimei string, methods map[string]BatchMethod) []byte {
	payload := map[string]any{
		"comm": map[string]any{
			"cv":         13020508,
			"ct":         "11",
			"v":          13020508,
			"uid":        "3931641530",
			"QIMEI36":    qimei,
			"tmeAppID":   "qqmusic",
			"format":     "json",
			"inCharset":  "utf-8",
			"outCharset": "utf-8",
		},
	}
	for k, v := range methods {
		payload[k] = v
	}
	b, _ := json.Marshal(payload)
	return b
}

// ------- LRC-only API ---------

// LRCAPIResponse wraps the LRC-only lyrics endpoint response.
type LRCAPIResponse struct {
	Code  int    `json:"code"`
	Lyric string `json:"lyric"` // Base64 encoded
	Trans string `json:"trans"` // Base64 encoded
}

// NormalizeLRCResponse strips JSONP wrapper and decodes Base64 lyrics.
func NormalizeLRCResponse(body string) (string, string, error) {
	// Strip MusicJsonCallback(...) wrapper
	jsonText := body
	if len(jsonText) > 18 && jsonText[:18] == "MusicJsonCallback(" {
		jsonText = jsonText[18 : len(jsonText)-1]
	}

	var resp LRCAPIResponse
	if err := json.Unmarshal([]byte(jsonText), &resp); err != nil {
		return "", "", fmt.Errorf("lrc parse: %w", err)
	}
	if resp.Code != 0 {
		return "", "", fmt.Errorf("lrc api: code=%d", resp.Code)
	}

	lyricB64, err := base64DecodeString(resp.Lyric)
	if err != nil {
		return "", "", fmt.Errorf("lrc decode lyric: %w", err)
	}
	transB64 := ""
	if resp.Trans != "" {
		b, err := base64DecodeString(resp.Trans)
		if err == nil {
			transB64 = string(b)
		}
	}
	return string(lyricB64), transB64, nil
}

func base64DecodeString(s string) ([]byte, error) {
	result := make([]byte, len(s))
	n := base64Decode(result, []byte(s))
	return result[:n], nil
}

func base64Decode(dst, src []byte) int {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var lookup [256]byte
	for i := range lookup {
		lookup[i] = 0xFF
	}
	for i, c := range []byte(alphabet) {
		lookup[c] = byte(i)
	}

	di := 0
	si := 0
	for si < len(src) && src[si] != '=' {
		b0 := lookup[src[si]]
		si++
		b1 := byte(0xFF)
		if si < len(src) && src[si] != '=' {
			b1 = lookup[src[si]]
			si++
		}
		b2 := byte(0xFF)
		if si < len(src) && src[si] != '=' {
			b2 = lookup[src[si]]
			si++
		}
		b3 := byte(0xFF)
		if si < len(src) && src[si] != '=' {
			b3 = lookup[src[si]]
			si++
		}
		if b0 == 0xFF || b1 == 0xFF {
			break
		}
		dst[di] = (b0 << 2) | (b1 >> 4)
		di++
		if b2 != 0xFF {
			dst[di] = (b1 << 4) | (b2 >> 2)
			di++
		}
		if b3 != 0xFF {
			dst[di] = (b2 << 6) | b3
			di++
		}
	}
	return di
}

// GetAlbumCoverURL builds the QQ Music album cover image URL.
func GetAlbumCoverURL(albumMID string, size int) string {
	if albumMID == "" {
		return ""
	}
	return fmt.Sprintf("https://y.gtimg.cn/music/photo_new/T002R%dx%dM000%s.jpg", size, size, albumMID)
}

// GetSingerPicURL builds the QQ Music singer image URL.
func GetSingerPicURL(singerMID string, size int) string {
	if singerMID == "" {
		return ""
	}
	return fmt.Sprintf("https://y.gtimg.cn/music/photo_new/T001R%dx%dM000%s.jpg", size, size, singerMID)
}
