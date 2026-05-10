package musicapi

import (
	"strings"
	"testing"
)

func TestLRCTimestampParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"[00:00.00]", 0},
		{"[01:30.50]", 90500},
		{"[03:45.00]", 225000},
		{"[00:05.12]", 5120},
		{"[10:00.00]", 600000},
	}

	for _, tt := range tests {
		got := parseLRCTimestamp(tt.input)
		if got != tt.expected {
			t.Errorf("parseLRCTimestamp(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestLRCParse(t *testing.T) {
	input := `[00:00.00]First line
[00:10.50]Second line
[00:20.00]第三行`

	lines := LRCParse(input)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].Text != "First line" {
		t.Errorf("line 0 text = %q, want %q", lines[0].Text, "First line")
	}
	if lines[0].Time != 0 {
		t.Errorf("line 0 time = %d, want 0", lines[0].Time)
	}
	if lines[1].Time != 10500 {
		t.Errorf("line 1 time = %d, want 10500", lines[1].Time)
	}
}

func TestLRCParseEmptyLine(t *testing.T) {
	input := `[00:00.00]
[00:05.00]real text
[00:10.00]  `
	lines := LRCParse(input)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
}

func TestDefaultLyricCleaner(t *testing.T) {
	cleaner := NewDefaultLyricCleaner()

	tests := []struct {
		name   string
		input  string
		expect []string // expected lines after cleaning
	}{
		{
			name: "basic metadata strip",
			input: `作词：张三
作曲：李四
编曲：王五
真正的歌词第一行
真正的歌词第二行`,
			expect: []string{"真正的歌词第一行", "真正的歌词第二行"},
		},
		{
			name: "english metadata",
			input: `Composed by: John
Lyrics by: Jane
Produced by: Mike
Hello world
This is a song`,
			expect: []string{"Hello world", "This is a song"},
		},
		{
			name: "copyright notice",
			input: `未经著作权人书面许可，不得以任何方式使用
真正的歌词`,
			expect: []string{"真正的歌词"},
		},
		{
			name: "footer metadata",
			input: `Lyric line 1
Lyric line 2
制作人：X
发行：Y公司`,
			expect: []string{"Lyric line 1", "Lyric line 2"},
		},
		{
			name:   "no metadata",
			input:  `Just a plain lyric without any metadata at all`,
			expect: []string{"Just a plain lyric without any metadata at all"},
		},
		{
			name: "LRC format preserved",
			input: `[00:00.00]First line
[00:10.00]Second line`,
			expect: []string{"[00:00.00]First line", "[00:10.00]Second line"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cleaner.Clean(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			lines := strings.Split(strings.TrimSpace(result), "\n")
			if result == "" {
				lines = nil
			}
			if len(lines) != len(tt.expect) {
				t.Fatalf("expected %d lines, got %d: %v", len(tt.expect), len(lines), lines)
			}
			for i := range lines {
				if lines[i] != tt.expect[i] {
					t.Errorf("line %d: got %q, want %q", i, lines[i], tt.expect[i])
				}
			}
		})
	}
}

func TestLyricCleanerDisabled(t *testing.T) {
	// When cleaning is disabled, raw lyrics should be preserved.
	cfg := DefaultClientConfig()
	cfg.EnableLyricClean = false

	if cfg.EnableLyricClean {
		t.Error("Expected EnableLyricClean to be false")
	}
}

func TestWithCustomLyricCleaner(t *testing.T) {
	customCleaner := &customTestCleaner{}

	cfg := DefaultClientConfig()
	WithCustomLyricCleaner(customCleaner)(cfg)

	if cfg.CustomLyricCleaner != customCleaner {
		t.Error("Custom lyric cleaner not set")
	}
	if !cfg.EnableLyricClean {
		t.Error("EnableLyricClean should be true when custom cleaner is set")
	}
}

type customTestCleaner struct{}

func (c *customTestCleaner) Clean(raw string) (string, error) {
	return "cleaned by custom", nil
}
