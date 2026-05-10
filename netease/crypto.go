// Package netease implements the MusicProvider interface for NetEase Cloud Music.
//
// It supports two API encryption schemes:
//   - weapi: Used for album, artist, playlist, song detail endpoints.
//     Two-layer AES-128-CBC encryption with RSA-secured random key.
//   - eapi: Used for search and lyrics endpoints.
//     MD5 digest + AES-128-ECB encryption.
//
// These algorithms are derived from the NeteaseCloudMusicApi project:
//
//	https://github.com/Binaryify/NeteaseCloudMusicApi
package netease

import (
	"encoding/base64"
	"fmt"

	"github.com/xiaowumin-mark/AMLX-MUSIC-API/internal/util"
)

const (
	// Preset AES key for the first layer of weapi encryption.
	nonceStr = "0CoJUm6Qyw8W8jud"
	// Fixed IV for all AES-CBC operations.
	viStr = "0102030405060708"
	// Fixed AES key for eapi encryption.
	eapiKeyStr = "e82ckenh8dichen8"
	// RSA public exponent for weapi.
	pubKeyStr = "010001"
	// RSA modulus for weapi (hex encoded).
	modulusStr = "00e0b509f6259df8642dbc35662901477df22677ec152b5ff68ace615bb7b725152b3ab17a876aea8a5aa76d2e417629ec4ee341f56135fccf695280104e0312ecbda92557c93870114af6c9d05c4f7f0c3685b7a46bee255932575cce10b424d813cfe4875d3e82047b97ddef52741d546b8e289dc6935b3ece0462db0a22b8e7"
	// Device ID XOR key for anonymous login.
	idXorKey = "3go8&$8*3*3h0k(2)2"
)

// WeapiEncrypt encrypts a JSON body using the weapi scheme.
// Returns (params, encSecKey) form fields.
//
//  1. First-layer AES-CBC with nonceStr key → Base64
//  2. Generate random 16-char secret key
//  3. Second-layer AES-CBC with secret key → Base64 (final params)
//  4. Reverse secret key, RSA encrypt with hardcoded public key → hex (encSecKey)
func WeapiEncrypt(jsonBody string) (params string, encSecKey string, _ error) {
	key := []byte(nonceStr)
	iv := []byte(viStr)

	// First layer
	first, err := util.AesCBCEncrypt([]byte(jsonBody), key, iv)
	if err != nil {
		return "", "", fmt.Errorf("weapi layer1: %w", err)
	}

	// Generate random 16-char key
	secretKey, err := util.RandomString(16)
	if err != nil {
		return "", "", fmt.Errorf("weapi random key: %w", err)
	}

	// Second layer
	params, err = util.AesCBCEncrypt([]byte(first), []byte(secretKey), iv)
	if err != nil {
		return "", "", fmt.Errorf("weapi layer2: %w", err)
	}

	// RSA encrypt the reversed secret key
	encSecKey, err = util.RSAEncryptBI(secretKey, pubKeyStr, modulusStr)
	if err != nil {
		return "", "", fmt.Errorf("weapi rsa: %w", err)
	}

	return params, encSecKey, nil
}

// EapiEncrypt encrypts parameters for the eapi scheme.
// Returns the hex-encoded params string.
//
//  1. Serialize params to JSON
//  2. message = "nobody" + urlPath + "use" + json + "md5forencrypt"
//  3. MD5(message) → digest
//  4. data = urlPath + "-36cd479b6b5-" + json + "-36cd479b6b5-" + digest
//  5. AES-128-ECB encrypt with eapiKeyStr → uppercase hex
func EapiEncrypt(urlPath string, jsonBody string) (string, error) {
	message := fmt.Sprintf("nobody%suse%s%s", urlPath, jsonBody, "md5forencrypt")
	digest := util.MD5Hex(message)

	data := fmt.Sprintf("%s-36cd479b6b5-%s-36cd479b6b5-%s", urlPath, jsonBody, digest)

	return util.AesECBEncrypt([]byte(data), []byte(eapiKeyStr))
}

// CloudMusicDLLEncodeId computes the anonymous login username from a device ID.
// Algorithm: XOR device ID bytes with idXorKey cyclically, MD5, Base64,
// then prefix with the original device ID and Base64 encode again.
func CloudMusicDLLEncodeId(deviceID string) string {
	b := []byte(deviceID)
	key := []byte(idXorKey)
	xored := make([]byte, len(b))
	for i := range b {
		xored[i] = b[i] ^ key[i%len(key)]
	}

	md5Digest := util.MD5Bytes(string(xored))
	b64Hash := base64.StdEncoding.EncodeToString(md5Digest)

	combined := b64Hash + " " + deviceID
	return base64.StdEncoding.EncodeToString([]byte(combined))
}
