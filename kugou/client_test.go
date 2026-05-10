package kugou

import (
	"testing"

	"github.com/xiaowumin-mark/AMLX-MUSIC-API"
)

func TestSongConversion(t *testing.T) {
	song := KuGouSong{
		Hash:       "abc123",
		SongName:   "测试歌曲",
		AlbumName:  "测试专辑",
		AlbumID:    "456",
		AudioID:    789,
		SingerName: "测试歌手",
		SingerID:   "012",
		Duration:   240,
		FeeType:    0,
	}

	result := convertSongFromSearch(song)

	if result.ID != "abc123" {
		t.Errorf("ID = %s, want abc123", result.ID)
	}
	if result.Name != "测试歌曲" {
		t.Errorf("Name = %s", result.Name)
	}
	if result.Duration != 240 {
		t.Errorf("Duration = %d", result.Duration)
	}
	if len(result.Artists) != 1 || result.Artists[0].Name != "测试歌手" {
		t.Error("Artist conversion failed")
	}
}

func TestPlaylistConversion(t *testing.T) {
	pl := PlaylistInfo{
		GlobalCollectionID: "9999",
		Name:               "推荐歌单",
		ImgURL:             "https://example.com/img.jpg",
		Username:           "创建者",
		Intro:              "好听的歌",
		Count:              50,
		MusicList: []KuGouSongItem{
			{Hash: "abc", SongName: "歌1", Duration: 200, SingerName: "歌手1", SingerID: "1", AlbumName: "专辑1", AlbumID: "1"},
		},
	}

	result := convertPlaylist(pl)

	if result.ID != "9999" {
		t.Errorf("ID mismatch")
	}
	if result.TrackCount != 50 {
		t.Errorf("TrackCount = %d, want 50", result.TrackCount)
	}
	if len(result.Songs) != 1 {
		t.Errorf("Expected 1 song, got %d", len(result.Songs))
	}
}

func TestKRCParse(t *testing.T) {
	content := "[1000,5000]第一行歌词<1000,100,0>气...\n[6000,4000]第二行歌词"

	lines := parseKRCLines(content)

	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}
	if lines[0].Time != 1000 {
		t.Errorf("Line 0 time = %d, want 1000", lines[0].Time)
	}
	if lines[0].Text != "第一行歌词气..." {
		t.Errorf("Line 0 text = %q, want '第一行歌词气...'", lines[0].Text)
	}
	if len(lines[0].Syllables) != 1 || lines[0].Syllables[0].Text != "气..." {
		t.Errorf("Line 0 syllables = %+v, want one '气...' syllable", lines[0].Syllables)
	}
	if lines[1].Time != 6000 {
		t.Errorf("Line 1 time = %d, want 6000", lines[1].Time)
	}
}

func TestKRCParseSyllableStrip(t *testing.T) {
	content := "[0,3000]我<0,500,0>想<500,500,0>你<1000,500,0>了"

	lines := parseKRCLines(content)

	if len(lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(lines))
	}
	if lines[0].Text != "我想你了" {
		t.Errorf("Text = %q, want '我想你了'", lines[0].Text)
	}
	if len(lines[0].Syllables) != 3 {
		t.Fatalf("Expected 3 syllables, got %d", len(lines[0].Syllables))
	}
	if lines[0].Syllables[1].Time != 500 || lines[0].Syllables[1].Text != "你" {
		t.Errorf("Unexpected syllable: %+v", lines[0].Syllables[1])
	}
}

func TestClientRegistration(t *testing.T) {
	client, err := musicapi.Get("kugou")
	if err != nil {
		t.Fatalf("Get kugou: %v", err)
	}
	if client.Name() != "kugou" {
		t.Errorf("Name = %q, want kugou", client.Name())
	}
}
