package util

import (
	"bytes"
	gozlib "compress/zlib"
	"encoding/base64"
	"os"
	"testing"
)

func TestMD5Hex(t *testing.T) {
	result := MD5Hex("hello")
	if len(result) != 32 {
		t.Errorf("MD5 hex length = %d, want 32", len(result))
	}
	// Known MD5 of "hello"
	expected := "5d41402abc4b2a76b9719d911017c592"
	if result != expected {
		t.Errorf("MD5('hello') = %s, want %s", result, expected)
	}
}

func TestAesCBCEncryptDecrypt(t *testing.T) {
	plaintext := []byte("test data for aes cbc")
	key := []byte("0123456789abcdef") // 16 bytes
	iv := []byte("0102030405060708")  // 16 bytes

	encrypted, err := AesCBCEncrypt(plaintext, key, iv)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	decrypted, err := AesCBCDecrypt(encrypted, key, iv)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if decrypted != string(plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, string(plaintext))
	}
}

func TestAesECBEncryptDecrypt(t *testing.T) {
	plaintext := []byte("hello world ecb test!!")
	key := []byte("e82ckenh8dichen8")

	encrypted, err := AesECBEncrypt(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	decrypted, err := AesECBDecrypt(encrypted, key)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if decrypted != string(plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, string(plaintext))
	}
}

func TestZlibDecompress(t *testing.T) {
	// Generate zlib compressed data using the standard library
	original := []byte("hello world!")
	compressed := zlibCompress(original)
	result, err := ZlibDecompress(compressed)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}
	if string(result) != string(original) {
		t.Errorf("decompressed = %q, want %q", string(result), string(original))
	}
}

func zlibCompress(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gozlib.NewWriterLevel(&buf, 6)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

func TestKRCDecryptFromFile(t *testing.T) {
	data, err := os.ReadFile("../../testdata/kugou_lyrics.krc")
	if err != nil {
		t.Skipf("test data not found: %v", err)
	}

	payload := data[4:]
	XorBytes(payload, KRCDecryptKey)
	decompressed, err := ZlibDecompress(payload)
	if err != nil {
		t.Fatalf("krc decompress failed: %v", err)
	}
	resultStr := string(decompressed)
	if resultStr == "" {
		t.Error("decrypted content should not be empty")
	}
	// KRC may start with BOM - strip if present
	if len(resultStr) >= 3 && resultStr[0] == 0xEF && resultStr[1] == 0xBB && resultStr[2] == 0xBF {
		resultStr = resultStr[3:]
	}
	if resultStr != "" && resultStr[0] != '[' {
		t.Errorf("decrypted content should start with '[' (KRC format), got: %c (0x%02x)", resultStr[0], resultStr[0])
	}
	t.Logf("Decrypted KRC (first 200 chars): %s", truncate(resultStr, 200))
}

func TestKRCDecryptBase64(t *testing.T) {
	b64Data, err := os.ReadFile("../../testdata/kugou_lyrics.b64")
	if err != nil {
		t.Skipf("test data not found: %v", err)
	}

	result, err := KRCDecrypt(string(b64Data))
	if err != nil {
		t.Fatalf("krc decrypt failed: %v", err)
	}
	if result == "" {
		t.Error("decrypted content should not be empty")
	}
	t.Logf("Decrypted KRC (first 200 chars): %s", truncate(result, 200))
}

func TestKRCDecryptBinary(t *testing.T) {
	data, err := os.ReadFile("../../testdata/kugou_lyrics.krc")
	if err != nil {
		t.Skipf("test data not found: %v", err)
	}
	if len(data) < 4 {
		t.Fatal("test data too short")
	}

	payload := data[4:]
	XorBytes(payload, KRCDecryptKey)
	result, err := ZlibDecompress(payload)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}
	if len(result) == 0 {
		t.Error("result should not be empty")
	}
}

func TestXorBytes(t *testing.T) {
	data := []byte{0x41, 0x42, 0x43, 0x44}
	key := []byte{0x01, 0x02}

	expectedFirst := data[0] ^ key[0]
	expectedSecond := data[1] ^ key[1]
	expectedThird := data[2] ^ key[0]

	XorBytes(data, key)

	if data[0] != expectedFirst {
		t.Errorf("XOR [0] = %02x, want %02x", data[0], expectedFirst)
	}
	if data[1] != expectedSecond {
		t.Errorf("XOR [1] = %02x, want %02x", data[1], expectedSecond)
	}
	if data[2] != expectedThird {
		t.Errorf("XOR [2] = %02x, want %02x", data[2], expectedThird)
	}
}

func TestRSAEncryptBI(t *testing.T) {
	// Test with known NetEase parameters
	pubKey := "010001"
	modulus := "00e0b509f6259df8642dbc35662901477df22677ec152b5ff68ace615bb7b725152b3ab17a876aea8a5aa76d2e417629ec4ee341f56135fccf695280104e0312ecbda92557c93870114af6c9d05c4f7f0c3685b7a46bee255932575cce10b424d813cfe4875d3e82047b97ddef52741d546b8e289dc6935b3ece0462db0a22b8e7"

	result, err := RSAEncryptBI("testkey123456789", pubKey, modulus)
	if err != nil {
		t.Fatalf("RSA encrypt failed: %v", err)
	}
	if len(result) != 256 {
		t.Errorf("RSA result length = %d, want 256", len(result))
	}
	t.Logf("RSA result: %s", result[:32]+"...")
}

func TestRandomString(t *testing.T) {
	s, err := RandomString(16)
	if err != nil {
		t.Fatalf("random string failed: %v", err)
	}
	if len(s) != 16 {
		t.Errorf("random string length = %d, want 16", len(s))
	}

	// Test uniqueness
	s2, _ := RandomString(16)
	if s == s2 {
		t.Log("warning: random strings collided (very unlikely)")
	}
}

func TestBase64Encoding(t *testing.T) {
	input := []byte("test")
	encoded := base64.StdEncoding.EncodeToString(input)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if string(decoded) != string(input) {
		t.Errorf("roundtrip failed: %q != %q", decoded, input)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
