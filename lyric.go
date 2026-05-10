package musicapi

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

// defaultLyricCleaner is the built-in MetadataStripper that removes
// lines containing production credits, composer/lyricist info, copyright
// notices, and other non-lyric metadata.
//
// The stripping logic is derived from the metadata_stripper module in
// the Unilyric project (lyrics_helper_rs/src/converter/processors/).
type defaultLyricCleaner struct{}

// Clean implements LyricCleaner by removing header and footer metadata
// lines while preserving timed lyric content.
func (d *defaultLyricCleaner) Clean(raw string) (string, error) {
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return raw, nil
	}

	cleaned := stripMetadataLines(lines)
	return strings.Join(cleaned, "\n"), nil
}

// stripMetadataLines removes header metadata lines (before the first real
// lyric line) and footer metadata lines (after the last real lyric line).
// Lines that match keyword rules or generic copyright patterns are considered
// metadata. Lines that look like timed lyrics (containing [...] timestamps
// or no colons/dashes in a metadata-like context) are preserved.
func stripMetadataLines(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}

	n := len(lines)

	// Build a set of indices that are strict metadata matches.
	isMeta := make([]bool, n)
	hasTimedContent := make([]bool, n)
	for i, line := range lines {
		isMeta[i] = matchesMetadataRule(line)
		hasTimedContent[i] = isTimedLyricLine(line)
	}

	// Find first lyric line: scan from top, stop at first line that is
	// NOT metadata and has timed content or looks like a real lyric.
	firstLyric := 0
	for i := 0; i < n; i++ {
		if !isMeta[i] && (hasTimedContent[i] || !looksLikeMetadata(lineCleanForCheck(lines[i]))) {
			firstLyric = i
			break
		}
		if isMeta[i] {
			// Keep going, but remember we may have passed valid meta lines.
		} else if looksLikeMetadata(lineCleanForCheck(lines[i])) {
			// Weak match — could be a role marker like "男：" which we keep.
			continue
		} else {
			firstLyric = i
			break
		}
	}

	// Find last lyric line: scan from bottom.
	lastLyric := n
	for i := n - 1; i >= firstLyric; i-- {
		if !isMeta[i] && (hasTimedContent[i] || !looksLikeMetadata(lineCleanForCheck(lines[i]))) {
			lastLyric = i + 1
			break
		}
		if isMeta[i] {
			continue
		} else if looksLikeMetadata(lineCleanForCheck(lines[i])) {
			continue
		} else {
			lastLyric = i + 1
			break
		}
	}

	if firstLyric < lastLyric {
		return lines[firstLyric:lastLyric]
	}
	if firstLyric > 0 || lastLyric < n {
		// All lines are metadata — return empty.
		return nil
	}
	return lines
}

// matchesMetadataRule checks whether a line matches a known metadata keyword
// pattern (e.g. "作词：xxx", "Composed by: xxx") or a copyright notice regex.
func matchesMetadataRule(line string) bool {
	text := lineCleanForCheck(line)
	textLower := strings.ToLower(text)

	// Check keywords from metadataStripperKeywords.
	for _, kw := range metadataStripperKeywords {
		if after, ok := strings.CutPrefix(textLower, kw); ok {
			after = strings.TrimLeft(after, " \t")
			if after == "" || strings.HasPrefix(after, ":") || strings.HasPrefix(after, "\uFF1A") {
				return true
			}
		}
	}

	// Check copyright regex patterns.
	for _, re := range copyrightPatterns {
		if re.MatchString(text) {
			return true
		}
	}

	return false
}

// lineCleanForCheck strips surrounding brackets and leading dash/separator
// characters so that the keyword matching is robust against common formatting.
func lineCleanForCheck(line string) string {
	text := strings.TrimSpace(line)
	if text == "" {
		return text
	}

	pairs := [][2]string{
		{"(", ")"},
		{"（", "）"},
		{"【", "】"},
	}
	for _, p := range pairs {
		if strings.HasPrefix(text, p[0]) {
			if strings.HasSuffix(text, p[1]) {
				text = strings.TrimSpace(text[len(p[0]) : len(text)-len(p[1])])
				break
			}
			if idx := strings.Index(text, p[1]); idx != -1 {
				text = strings.TrimLeft(text[idx+len(p[1]):], " \t")
				break
			}
		}
	}

	// Also strip leading "·" or "-" commonly used as separators.
	text = strings.TrimLeft(text, "·-\u2010\u2011\u2012\u2013\u2014\u2015")

	// Strip leading script markers like Roman numerals with dots.
	if len(text) > 0 && text[0] >= 'A' && text[0] <= 'Z' {
		idx := 0
		for idx < len(text) && ((text[idx] >= 'A' && text[idx] <= 'Z') || text[idx] == '.') {
			idx++
		}
		if idx > 0 && idx < len(text) && text[idx] == ' ' {
			text = strings.TrimLeft(text[idx:], " ")
		}
	}

	return strings.TrimSpace(text)
}

// isTimedLyricLine returns true if the line contains a lyric timestamp
// in [mm:ss.xx] or [mm:ss] format.
func isTimedLyricLine(line string) bool {
	return timedLyricPattern.MatchString(line)
}

// looksLikeMetadata returns true if a line looks like it could be
// metadata (contains colon, dash as separator pattern).
func looksLikeMetadata(text string) bool {
	if text == "" {
		return false
	}
	return strings.Contains(text, ":") || strings.Contains(text, "：") || strings.Contains(text, "-")
}

// timedLyricPattern matches lines that start with a lyric timestamp.
var timedLyricPattern = regexp.MustCompile(`^\s*\[\d{2}:\d{2}(?:[\.:]\d+)?\]`)

// metadataStripperKeywords defines the lowercase keyword prefixes that
// indicate a metadata line. Derived from default_stripper_config.toml in
// the Unilyric project.
var metadataStripperKeywords = []string{
	"作词", "作曲", "编曲", "演唱", "歌手", "歌名", "专辑",
	"发行", "出品", "监制", "录音", "混音", "母带", "吉他",
	"贝斯", "鼓", "键盘", "弦乐", "和声", "版权", "制作人",
	"原唱", "翻唱", "词", "曲", "发行人", "发行公司", "宣推",
	"录音制作", "制作发行", "制作团队", "音乐制作", "录音师",
	"混音工程师", "混音师", "母带工程师", "母带处理工程师",
	"制作统筹", "艺术指导", "出品团队", "发行方", "和声编写",
	"封面设计", "策划", "营销推广", "总策划", "特别鸣谢",
	"出品人", "出品公司", "联合出品", "词曲提供", "制作公司",
	"推广策划", "乐器演奏", "第一小提琴", "第二小提琴",
	"中提琴", "大提琴", "和音", "配唱制作人", "文案",
	"设计", "策划统筹", "企划宣传", "企划营销", "录音室",
	"混音室", "鸣谢", "工作室", "特别企划", "音频编辑",
	"词曲协力", "企划", "宣传", "统筹", "推广", "封面",
	"总企划", "混缩", "联合策划", "联合推广", "题记",
	"项目统筹", "合声", "合声编写", "op", "sp",
	"artist", "songs title",
	"lyrics by", "composed by", "produced by", "published by",
	"vocals by", "background vocals by", "additional vocal by",
	"mixing engineer", "mastered by", "executive producer",
	"vocal engineer", "vocals produced by", "recorded at",
	"repertoire owner", "co-producer", "mastering engineer",
	"written by", "lyrics", "composer", "arranged by",
	"record producer", "guitar", "music production",
	"recording engineer", "backing vocal", "art director",
	"chief producer", "production team", "publisher",
	"lyricist", "arranger", "producer", "backing vocals",
	"backing vocals design", "cover design", "planner",
	"marketing promotion", "chref planner", "acknowledgement",
	"production company", "jointly produced by", "co-production",
	"presenter", "presented by", "co-produced by",
	"lyrics and composition provided by",
	"music and lyrics provided by",
	"words and music by",
	"distribution", "release", "distributed by", "released by",
	"produce company", "promotion planning",
	"strings", "first violin", "second violin", "viola", "cello",
	"vocal producer", "supervised production",
	"copywriting", "propaganda", "arrangement",
	"guitars", "bass", "drums",
	"backing vocal arrangement", "strings arrangement",
	"recording studio", "mixing studio",
}

// copyrightPatterns matches copyright disclaimers and usage restriction
// notices commonly embedded in lyrics.
var copyrightPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i).*未经.*许可.*不得.*使用.*`),
	regexp.MustCompile(`(?i).*未经著作权人.*`),
}

// NewDefaultLyricCleaner returns a new instance of the built-in metadata
// stripper lyric cleaner.
func NewDefaultLyricCleaner() LyricCleaner {
	return &defaultLyricCleaner{}
}

// It extracts [mm:ss.xx] timestamps and maps each to a LyricLine.
func LRCParse(raw string) []LyricLine {
	var lines []LyricLine
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		matches := timedLyricPattern.FindAllStringSubmatch(line, -1)
		if len(matches) == 0 {
			continue
		}
		text := timedLyricPattern.ReplaceAllString(line, "")
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		for _, m := range matches {
			t := parseLRCTimestamp(m[0])
			lines = append(lines, LyricLine{Time: t, Text: text})
		}
	}
	return lines
}

// parseLRCTimestamp converts a [mm:ss.xx] or [mm:ss] string to milliseconds.
func parseLRCTimestamp(s string) int64 {
	s = strings.Trim(s, "[]")
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0
	}
	min, _ := strconv.Atoi(parts[0])
	secPart := parts[1]
	if idx := strings.IndexByte(secPart, '.'); idx != -1 {
		sec, _ := strconv.Atoi(secPart[:idx])
		frac, _ := strconv.Atoi(secPart[idx+1:])
		return int64(min*60*1000 + sec*1000 + frac*10)
	}
	sec, _ := strconv.Atoi(secPart)
	return int64(min*60*1000 + sec*1000)
}
