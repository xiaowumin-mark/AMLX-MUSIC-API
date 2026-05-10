package netease

import (
	"encoding/base64"
	"testing"
)

func TestWeapiEncrypt(t *testing.T) {
	jsonBody := `{"s":"test","type":1,"limit":10,"offset":0}`

	params, encSecKey, err := WeapiEncrypt(jsonBody)
	if err != nil {
		t.Fatalf("weapi encrypt failed: %v", err)
	}

	if params == "" {
		t.Error("params should not be empty")
	}
	if encSecKey == "" {
		t.Error("encSecKey should not be empty")
	}
	if len(encSecKey) != 256 {
		t.Errorf("encSecKey length = %d, want 256", len(encSecKey))
	}

	t.Logf("params length: %d", len(params))
	t.Logf("encSecKey length: %d", len(encSecKey))
}

func TestWeapiEncryptConsistent(t *testing.T) {
	jsonBody := `{"test":true}`

	p1, k1, err := WeapiEncrypt(jsonBody)
	if err != nil {
		t.Fatal(err)
	}
	p2, k2, err := WeapiEncrypt(jsonBody)
	if err != nil {
		t.Fatal(err)
	}

	if p1 == p2 {
		t.Log("params unexpectedly identical (unlikely but possible)")
	}
	if k1 == k2 {
		t.Log("encSecKey unexpectedly identical (unlikely but possible)")
	}
}

func TestEapiEncrypt(t *testing.T) {
	urlPath := "/api/song/lyric/v1"
	jsonBody := `{"id":12345,"cp":false,"lv":0}`

	result, err := EapiEncrypt(urlPath, jsonBody)
	if err != nil {
		t.Fatalf("eapi encrypt failed: %v", err)
	}
	if result == "" {
		t.Error("result should not be empty")
	}
	for _, c := range result {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')) {
			t.Errorf("unexpected char in hex result: %c", c)
		}
	}
}

func TestEapiEncryptRoundtrip(t *testing.T) {
	urlPath := "/api/search"
	jsonBody := `{"keyword":"hello"}`

	r1, err := EapiEncrypt(urlPath, jsonBody)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := EapiEncrypt(urlPath, jsonBody)
	if err != nil {
		t.Fatal(err)
	}

	if r1 != r2 {
		t.Error("eapi encrypt should be deterministic for same input")
	}
}

func TestCloudMusicDLLEncodeId(t *testing.T) {
	deviceID := "abcdef1234567890abcdef1234567890"
	result := CloudMusicDLLEncodeId(deviceID)

	if result == "" {
		t.Error("encoded ID should not be empty")
	}
	if result == deviceID {
		t.Error("encoded ID should differ from input")
	}

	result2 := CloudMusicDLLEncodeId(deviceID)
	if result != result2 {
		t.Error("encoding should be deterministic")
	}

	t.Logf("encoded: %s", result)
}

func TestBase64Std(t *testing.T) {
	input := []byte("hello")
	result := base64.StdEncoding.EncodeToString(input)
	if result != "aGVsbG8=" {
		t.Errorf("base64('hello') = %s, want 'aGVsbG8='", result)
	}
}
