package qqmusic

import (
	"os"
	"testing"
)

func TestQrcDecryptFromHex(t *testing.T) {
	hexData, err := os.ReadFile("../testdata/encrypted_lyrics.hex")
	if err != nil {
		t.Skipf("test data not found: %v", err)
	}

	result, err := QrcDecrypt(string(hexData))
	if err != nil {
		t.Fatalf("QRC decrypt failed: %v", err)
	}
	if result == "" {
		t.Error("decrypted result should not be empty")
	}

	// Should start with XML wrapper
	if len(result) < 5 || result[:5] != "<?xml" {
		t.Errorf("decrypted content should start with '<?xml', got: %s", trunc(result, 50))
	}

	t.Logf("Decrypted QRC (first 300 chars):\n%s", trunc(result, 300))
}

func TestQrcDecryptFromBinary(t *testing.T) {
	binData, err := os.ReadFile("../testdata/encrypted_lyrics.bin")
	if err != nil {
		t.Skipf("test data not found: %v", err)
	}

	result, err := QrcDecryptLocal(binData)
	if err != nil {
		t.Fatalf("QRC local decrypt failed: %v", err)
	}
	if result == "" {
		t.Error("decrypted result should not be empty")
	}

	t.Logf("Decrypted local QRC (first 300 chars):\n%s", trunc(result, 300))
}

func TestQrcDecryptWithFallback(t *testing.T) {
	hexData, err := os.ReadFile("../testdata/encrypted_lyrics.hex")
	if err != nil {
		t.Skipf("test data not found: %v", err)
	}

	result, err := QrcDecryptWithFallback(string(hexData))
	if err != nil {
		t.Fatalf("QRC decrypt with fallback failed: %v", err)
	}
	if result == "" {
		t.Error("decrypted result should not be empty")
	}
}

func TestQrcParseSyllables(t *testing.T) {
	lines := parseQrcLines("[1000,1200](1000,300,0)海(1300,300,0)阔(1600,300,0)天(1900,300,0)空")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}
	if lines[0].Text != "海阔天空" {
		t.Fatalf("Text = %q, want 海阔天空", lines[0].Text)
	}
	if len(lines[0].Syllables) != 4 {
		t.Fatalf("Expected 4 syllables, got %d", len(lines[0].Syllables))
	}
	if lines[0].Syllables[1].Time != 1300 || lines[0].Syllables[1].Text != "阔" {
		t.Errorf("Unexpected syllable: %+v", lines[0].Syllables[1])
	}
}

func TestExtractFromQRcWrapper(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="utf-8"?>
<QrcLyric LyricContent="[0,1000]test lyric content" QrcType="1"/>`

	content := ExtractFromQRcWrapper(xmlContent)
	expected := "[0,1000]test lyric content"
	if content != expected {
		t.Errorf("extracted = %q, want %q", content, expected)
	}
}

func TestExtractFromQRcWrapperNoXML(t *testing.T) {
	// If content doesn't start with <?xml, it should be returned as-is
	content := ExtractFromQRcWrapper("plain text content")
	if content != "plain text content" {
		t.Errorf("extracted = %q, want %q", content, "plain text content")
	}
}

func TestExtractFromQRcWrapperEmpty(t *testing.T) {
	if result := ExtractFromQRcWrapper(""); result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestCustomDESVerifyKeySchedule(t *testing.T) {
	// Verify key_1 encryption schedule matches Rust reference
	schedule := desKeySchedule(qrcKey1, true)

	// Known round keys from Rust test
	expectedR1 := [6]byte{0x40, 0x0C, 0x26, 0x10, 0x28, 0x08}
	if schedule[0] != expectedR1 {
		t.Errorf("Round 1: got %02X, want %02X", schedule[0], expectedR1)
	}

	expectedR16 := [6]byte{0xD0, 0x2C, 0x04, 0x00, 0xCA, 0x82}
	if schedule[15] != expectedR16 {
		t.Errorf("Round 16: got %02X, want %02X", schedule[15], expectedR16)
	}

	t.Log("DES key schedule verified")
}

func TestCustomDESBlockEncryptDecrypt(t *testing.T) {
	plaintext := [8]byte{'t', 'e', 's', 't', 'm', 's', 'g', '!'}

	// Encrypt with KEY_1 encrypt schedule
	encryptSchedule := desKeySchedule(qrcKey1, true)
	var encrypted [8]byte
	desBlock(plaintext[:], encrypted[:], encryptSchedule)

	// Decrypt with KEY_1 decrypt schedule
	decryptSchedule := desKeySchedule(qrcKey1, false)
	var decrypted [8]byte
	desBlock(encrypted[:], decrypted[:], decryptSchedule)

	if decrypted != plaintext {
		t.Errorf("round-trip failed: decrypted=%v, want=%v", decrypted, plaintext)
	}
}

func TestQMC1Decrypt(t *testing.T) {
	// Test with small data (less than threshold)
	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	original := make([]byte, len(data))
	copy(original, data)

	qmc1Decrypt(data)

	// Each byte should be XORed with privKey
	for i := range data {
		if data[i] != original[i]^privKey[i&0x7F] {
			t.Errorf("qmc1 byte %d: got %02x, want %02x", i, data[i], original[i]^privKey[i&0x7F])
		}
	}

	// Double XOR should restore original
	qmc1Decrypt(data)
	for i := range data {
		if data[i] != original[i] {
			t.Errorf("double qmc1 byte %d: got %02x, want %02x", i, data[i], original[i])
		}
	}
}

func TestQMC1DecryptLargeFile(t *testing.T) {
	// Test with data larger than threshold (0x8000 = 32768)
	size := 0x8000 + 100
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i & 0xFF)
	}
	original := make([]byte, size)
	copy(original, data)

	qmc1Decrypt(data)

	// First 0x8000 bytes use index & 0x7F
	for i := 0; i < 0x8000; i++ {
		if data[i] != original[i]^privKey[i&0x7F] {
			t.Errorf("large qmc1 byte %d (first segment): wrong value", i)
			break
		}
	}
	// Remaining use (index % 0x7FFF) & 0x7F
	for i := 0x8000; i < size; i++ {
		expected := original[i] ^ privKey[(i%0x7FFF)&0x7F]
		if data[i] != expected {
			t.Errorf("large qmc1 byte %d (second segment): got %02x, want %02x", i, data[i], expected)
			break
		}
	}
}

func TestGetSearchID(t *testing.T) {
	id1 := GetSearchID()
	id2 := GetSearchID()

	if id1 == "" || id2 == "" {
		t.Error("search ID should not be empty")
	}
	if id1 == id2 {
		t.Log("warning: search IDs identical (very unlikely)")
	}
}

func TestGetQimei(t *testing.T) {
	result := GetQimei()
	if result.Q16 == "" {
		t.Error("q16 should not be empty")
	}
	if result.Q36 == "" {
		t.Error("q36 should not be empty")
	}
	if result.Q36 == DefaultQimei36 {
		t.Log("using default qimei36 fallback")
	}
}

func TestGetAlbumCoverURL(t *testing.T) {
	url := GetAlbumCoverURL("001ABC", 300)
	expected := "https://y.gtimg.cn/music/photo_new/T002R300x300M000001ABC.jpg"
	if url != expected {
		t.Errorf("cover URL = %s, want %s", url, expected)
	}
}

func TestGenerateIMEI(t *testing.T) {
	imei := generateIMEI()
	if len(imei) != 15 {
		t.Errorf("IMEI length = %d, want 15", len(imei))
	}
	// Basic Luhn check
	sum := 0
	for i := 0; i < 14; i++ {
		d := int(imei[i] - '0')
		if i%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	checkDigit := (10 - (sum % 10)) % 10
	if int(imei[14]-'0') != checkDigit {
		t.Errorf("IMEI Luhn check failed: check digit = %c, expected %d", imei[14], checkDigit)
	}
}

func TestBase64Decode(t *testing.T) {
	input := "aGVsbG8="
	result, err := base64DecodeString(input)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if string(result) != "hello" {
		t.Errorf("decoded = %q, want 'hello'", string(result))
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
