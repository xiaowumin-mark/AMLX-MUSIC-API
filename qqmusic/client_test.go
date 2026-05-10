package qqmusic

import (
	"encoding/json"
	"testing"

	"github.com/xiaowumin-mark/AMLX-MUSIC-API"
)

func TestQQSongConversion(t *testing.T) {
	songJSON := `{
		"mid": "001ABC", "id": 123456, "name": "测试歌曲", "title": "测试歌曲",
		"interval": 240, "pay": {"payplay": 0},
		"singer": [{"mid": "002DEF", "name": "测试歌手"}],
		"album": {"mid": "003GHI", "name": "测试专辑"}
	}`
	var song QQSearchSong
	if err := json.Unmarshal([]byte(songJSON), &song); err != nil {
		t.Fatal(err)
	}

	result := convertQQSearchSong(song)

	if result.ID != "001ABC" {
		t.Errorf("ID = %s, want 001ABC", result.ID)
	}
	if result.Name != "测试歌曲" {
		t.Errorf("Name = %s", result.Name)
	}
	if result.Duration != 240 {
		t.Errorf("Duration = %d", result.Duration)
	}
	if result.PayStatus != "free" {
		t.Errorf("PayStatus = %s, want free", result.PayStatus)
	}
}

func TestQQTrackConversion(t *testing.T) {
	trackJSON := `{
		"id": 123456, "mid": "001ABC", "name": "测试歌曲", "interval": 300,
		"pay": {"payplay": 5},
		"singer": [{"mid": "002DEF", "name": "歌手1"}, {"mid": "003GHI", "name": "歌手2"}],
		"album": {"id": 100, "mid": "004JKL", "name": "专辑名"}
	}`
	var track QQTrackInfo
	if err := json.Unmarshal([]byte(trackJSON), &track); err != nil {
		t.Fatal(err)
	}

	result := convertQQTrack(track)

	if result.ID != "001ABC" {
		t.Errorf("ID mismatch")
	}
	if len(result.Artists) != 2 {
		t.Errorf("Expected 2 artists, got %d", len(result.Artists))
	}
	if result.PayStatus != "only" {
		t.Errorf("PayStatus for payplay=5 should be 'only', got %s", result.PayStatus)
	}
}

func TestPayStatusMapping(t *testing.T) {
	tests := []struct {
		payPlay  int
		expected string
	}{
		{0, "free"},
		{1, "vip"},
		{2, "vip"},
		{3, "vip"},
		{4, "only"},
		{5, "only"},
		{6, "only"},
		{8, "only"},
	}

	for _, tt := range tests {
		result := payStatus(tt.payPlay)
		if result != tt.expected {
			t.Errorf("payStatus(%d) = %s, want %s", tt.payPlay, result, tt.expected)
		}
	}
}

func TestClientRegistration(t *testing.T) {
	client, err := musicapi.Get("qq")
	if err != nil {
		t.Fatalf("Get qq: %v", err)
	}
	if client.Name() != "qq" {
		t.Errorf("Name = %q, want qq", client.Name())
	}
}

func TestBuildBatchPayload(t *testing.T) {
	payload := BuildBatchPayload("testqimei36", map[string]BatchMethod{
		"test": {Module: "test.module", Method: "TestMethod", Param: map[string]any{"key": "value"}},
	})

	if len(payload) == 0 {
		t.Error("Payload should not be empty")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
