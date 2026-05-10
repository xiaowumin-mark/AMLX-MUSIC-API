// Package util provides cryptographic and encoding utilities shared across
// music platform providers.
package util

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"strings"
)

// ---------- MD5 ----------

// MD5Hex returns the lowercase hex-encoded MD5 hash of the input string.
func MD5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// MD5Bytes returns the raw MD5 hash bytes.
func MD5Bytes(s string) []byte {
	h := md5.Sum([]byte(s))
	return h[:]
}

// ---------- SHA1 ----------

// SHA1Hex returns the lowercase hex-encoded SHA1 hash of the input string.
func SHA1Hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// ---------- AES-128-CBC ----------

// AesCBCEncrypt encrypts plaintext using AES-128-CBC with PKCS7 padding,
// returning the Base64-encoded ciphertext.
//
// key and iv must be exactly 16 bytes.
func AesCBCEncrypt(plaintext, key, iv []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cbc encrypt: %w", err)
	}
	plaintext = pkcs7Pad(plaintext, aes.BlockSize)

	ciphertext := make([]byte, len(plaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AesCBCDecrypt decrypts Base64-encoded ciphertext using AES-128-CBC
// with PKCS7 padding.
func AesCBCDecrypt(b64Ciphertext string, key, iv []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64Ciphertext)
	if err != nil {
		return "", fmt.Errorf("aes cbc decrypt base64: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cbc decrypt: %w", err)
	}
	if len(data)%aes.BlockSize != 0 {
		return "", fmt.Errorf("aes cbc decrypt: ciphertext not aligned")
	}
	plaintext := make([]byte, len(data))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, data)
	plaintext, err = pkcs7Unpad(plaintext)
	if err != nil {
		return "", fmt.Errorf("aes cbc decrypt unpad: %w", err)
	}
	return string(plaintext), nil
}

// AesCBCEncryptRaw encrypts and returns raw bytes (not Base64).
func AesCBCEncryptRaw(plaintext, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cbc encrypt: %w", err)
	}
	plaintext = pkcs7Pad(plaintext, aes.BlockSize)
	ciphertext := make([]byte, len(plaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)
	return ciphertext, nil
}

// ---------- AES-128-ECB ----------

// AesECBEncrypt encrypts plaintext using AES-128-ECB with PKCS7 padding,
// returning the uppercase hex-encoded ciphertext.
func AesECBEncrypt(plaintext []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes ecb encrypt: %w", err)
	}
	plaintext = pkcs7Pad(plaintext, aes.BlockSize)
	ciphertext := make([]byte, len(plaintext))

	for i := 0; i < len(plaintext); i += aes.BlockSize {
		block.Encrypt(ciphertext[i:i+aes.BlockSize], plaintext[i:i+aes.BlockSize])
	}
	return strings.ToUpper(hex.EncodeToString(ciphertext)), nil
}

// AesECBDecrypt decrypts hex-encoded ciphertext using AES-128-ECB
// with PKCS7 padding.
func AesECBDecrypt(hexCiphertext string, key []byte) (string, error) {
	data, err := hex.DecodeString(hexCiphertext)
	if err != nil {
		return "", fmt.Errorf("aes ecb decrypt hex: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes ecb decrypt: %w", err)
	}
	if len(data)%aes.BlockSize != 0 {
		return "", fmt.Errorf("aes ecb decrypt: ciphertext not aligned")
	}
	plaintext := make([]byte, len(data))
	for i := 0; i < len(data); i += aes.BlockSize {
		block.Decrypt(plaintext[i:i+aes.BlockSize], data[i:i+aes.BlockSize])
	}
	plaintext, err = pkcs7Unpad(plaintext)
	if err != nil {
		return "", fmt.Errorf("aes ecb decrypt unpad: %w", err)
	}
	return string(plaintext), nil
}

// ---------- PKCS7 Padding ----------

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("pkcs7 unpad: empty data")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > len(data) || padding > aes.BlockSize {
		return nil, fmt.Errorf("pkcs7 unpad: invalid padding %d", padding)
	}
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, fmt.Errorf("pkcs7 unpad: invalid padding at %d", i)
		}
	}
	return data[:len(data)-padding], nil
}

// ---------- RSA ----------

// RSAEncryptBI encrypts using RSA with raw big.Int exponent and modulus.
// This is used for the NetEase weapi encryption scheme.
//
// text is the plaintext, pubKeyHex is the hex-encoded exponent (e.g. "010001"),
// modulusHex is the hex-encoded modulus.
func RSAEncryptBI(text, pubKeyHex, modulusHex string) (string, error) {
	// Reverse the plaintext (NetEase specific)
	reversed := reverseString(text)
	textHex := hex.EncodeToString([]byte(reversed))

	a := new(big.Int)
	a.SetString(textHex, 16)

	b := new(big.Int) // exponent
	b.SetString(pubKeyHex, 16)

	c := new(big.Int) // modulus
	c.SetString(modulusHex, 16)

	result := new(big.Int).Exp(a, b, c)
	keyHex := fmt.Sprintf("%x", result)

	// Pad or truncate to 256 hex chars
	if len(keyHex) < 256 {
		keyHex = strings.Repeat("0", 256-len(keyHex)) + keyHex
	} else if len(keyHex) > 256 {
		keyHex = keyHex[len(keyHex)-256:]
	}
	return keyHex, nil
}

// RSAEncryptPKCS encrypts data using RSA PKCS#1 v1.5 with the given
// PEM-encoded public key. Returns Base64-encoded ciphertext.
func RSAEncryptPKCS(plaintext []byte, pubKeyPEM string) (string, error) {
	block, _ := pem.Decode([]byte(pubKeyPEM))
	if block == nil {
		return "", fmt.Errorf("rsa encrypt: failed to decode PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// Try PKCS1 format
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("rsa encrypt: parse key: %w", err)
		}
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("rsa encrypt: not an RSA public key")
	}
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPub, plaintext)
	if err != nil {
		return "", fmt.Errorf("rsa encrypt: %w", err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ---------- Zlib ----------

// ZlibDecompress decompresses zlib-compressed data.
func ZlibDecompress(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("zlib decompress: %w", err)
	}
	defer r.Close()
	return io.ReadAll(r)
}

// ---------- XOR ----------

// XorBytes XORs each byte in data with the cyclic key.
func XorBytes(data []byte, key []byte) {
	for i := range data {
		data[i] ^= key[i%len(key)]
	}
}

// ---------- KRC Decrypt ----------

// KRCDecryptKey is the fixed 16-byte XOR key used to decrypt KuGou KRC
// lyrics. Source: Unilyric lyrics_helper_rs kugou decrypter.
var KRCDecryptKey = []byte{0x40, 0x47, 0x61, 0x77, 0x5E, 0x32, 0x74, 0x47, 0x51, 0x36, 0x31, 0x2D, 0xCE, 0xD2, 0x6E, 0x69}

// KRCDecrypt decrypts a Base64-encoded KuGou KRC lyric string.
// Steps: Base64 decode → strip 4-byte "krc1" header → XOR with key → Zlib decompress.
func KRCDecrypt(b64Input string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64Input)
	if err != nil {
		return "", fmt.Errorf("krc decrypt base64: %w", err)
	}
	if len(data) < 4 {
		return "", fmt.Errorf("krc decrypt: data too short (%d bytes)", len(data))
	}
	payload := data[4:]
	XorBytes(payload, KRCDecryptKey)
	decompressed, err := ZlibDecompress(payload)
	if err != nil {
		return "", fmt.Errorf("krc decrypt zlib: %w", err)
	}
	return string(decompressed), nil
}

// ---------- Hex ----------

// HexDecode is a convenience wrapper around hex.DecodeString.
func HexDecode(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// ---------- Random String ----------

// RandomString generates a random alphanumeric string of the given length.
func RandomString(n int) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b), nil
}

// RandomHex generates a random hex string of the given byte length (2x hex chars).
func RandomHex(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
