package kugou

import (
	"testing"
)

func TestSignatureAndroidParams(t *testing.T) {
	params := map[string]string{
		"appid":      "1005",
		"clientver":  "11083",
		"clienttime": "1678886400",
	}
	body := `{"data":[{"album_id":"12345"}],"is_buy":0}`

	result := signatureAndroidParams(params, body, false)
	// Expected value from Rust reference implementation
	expected := "f02fe39da9cc0f24a97aa5063da7de2f"
	if result != expected {
		t.Errorf("signature = %s, want %s", result, expected)
	}
}

func TestSignatureAndroidParamsEmptyBody(t *testing.T) {
	params := map[string]string{
		"appid":      "1005",
		"clientver":  "12569",
		"clienttime": "1678886400",
	}
	result := signatureAndroidParams(params, "", false)
	if len(result) != 32 {
		t.Errorf("signature length = %d, want 32", len(result))
	}
}

func TestSignatureRegisterParams(t *testing.T) {
	params := map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
	}
	result := signatureRegisterParams(params)
	if len(result) != 32 {
		t.Errorf("register signature length = %d, want 32", len(result))
	}
}

func TestSignParamsKey(t *testing.T) {
	// Test from Rust reference
	key := signParamsKey("1005", "12569", "1678886400")
	expected := "d750b0eda64e5a973df8ecca4b0d2e80"
	if key != expected {
		t.Errorf("signParamsKey = %s, want %s", key, expected)
	}
}

func TestSignKey(t *testing.T) {
	result := signKey("abc123", "mid456", "0")
	if len(result) != 32 {
		t.Errorf("signKey length = %d, want 32", len(result))
	}
}

func TestSignatureKeyOrdering(t *testing.T) {
	// Parameters should be sorted alphabetically for consistent results
	params1 := map[string]string{
		"zzz": "1",
		"aaa": "2",
	}
	params2 := map[string]string{
		"aaa": "2",
		"zzz": "1",
	}
	r1 := signatureAndroidParams(params1, "", false)
	r2 := signatureAndroidParams(params2, "", false)
	if r1 != r2 {
		t.Error("signatures should be identical regardless of map insertion order")
	}
}

func TestBuildQuery(t *testing.T) {
	params := map[string]string{"a": "1", "b": "2"}
	result := buildQuery(params)
	// Should contain key=value pairs
	if result == "" {
		t.Error("buildQuery result should not be empty")
	}
}
