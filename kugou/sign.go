// Package kugou implements the MusicProvider interface for KuGou Music.
//
// Signature schemes (from Unilyric lyrics_helper_rs kugou signature.rs):
//   - android: MD5(salt + sorted_key_value_pairs + body + salt)
//     where salt = "OIlwieks28dk2k092lksi2UIkp"
//   - register: MD5("1014" + sorted_values + "1014")
//   - sign_params_key: MD5(appid + salt + clientver + clienttime)
//
// KRC lyrics decryption (from Unilyric lyrics_helper_rs kugou decrypter.rs):
//   - Base64 decode
//   - Strip 4-byte "krc1" header
//   - XOR with 16-byte key cyclically
//   - Zlib decompress
//
// API references:
//   - https://github.com/MakcRe/KuGouMusicApi
package kugou

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xiaowumin-mark/AMLX-MUSIC-API/internal/util"
)

const (
	androidSalt      = "OIlwieks28dk2k092lksi2UIkp"
	liteAndroidSalt  = "LnT6xpN3khm36zse0QzvmgTZ3waWdRSA"
	registerSaltPre  = "1014"
	registerSaltPost = "1014"

	appID     = "1005"
	clientVer = "12569"
)

// SignatureAndroidParams computes the MD5 signature for android-type API requests.
// params is sorted by key automatically (via map iteration order is NOT guaranteed,
// but we sort explicitly). body is "" for GET requests, or the JSON body string for POST.
func signatureAndroidParams(params map[string]string, body string, isLite bool) string {
	salt := androidSalt
	if isLite {
		salt = liteAndroidSalt
	}
	// Sort by key
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(params[k])
	}
	return util.MD5Hex(salt + sb.String() + body + salt)
}

// signatureRegisterParams computes the MD5 signature for device registration.
func signatureRegisterParams(params map[string]string) string {
	vals := make([]string, 0, len(params))
	for _, v := range params {
		vals = append(vals, v)
	}
	sort.Strings(vals)
	return util.MD5Hex(registerSaltPre + strings.Join(vals, "") + registerSaltPost)
}

// signParamsKey computes the key parameter for /kmr/ endpoints.
func signParamsKey(appid, clientver, clienttime string) string {
	input := fmt.Sprintf("%s%s%s%s", appid, androidSalt, clientver, clienttime)
	return util.MD5Hex(input)
}

// signKey computes the key for song URL retrieval.
func signKey(hash, mid, userid string) string {
	const str = "57ae12eb6890223e355ccfcb74edf70d"
	return util.MD5Hex(hash + str + appID + mid + userid)
}
