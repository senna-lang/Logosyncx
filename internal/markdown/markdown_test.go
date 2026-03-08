// Tests for the shared markdown helpers.
package markdown

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Hello World", "hello-world"},
		{"  trim me  ", "trim-me"},
		{"foo--bar", "foo-bar"},
		{"kebab-case", "kebab-case"},
		{"with_underscore", "with_underscore"},
		{"UPPER CASE", "upper-case"},
		{"123 numbers", "123-numbers"},
		{"special!@#chars", "specialchars"},
		{"", ""},
	}
	for _, c := range cases {
		got := Slugify(c.in)
		if got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSplitFrontmatter(t *testing.T) {
	t.Run("valid frontmatter", func(t *testing.T) {
		input := "---\nkey: value\n---\nbody text\n"
		fm, body, err := SplitFrontmatter([]byte(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(fm) != "key: value" {
			t.Errorf("frontmatter = %q, want %q", fm, "key: value")
		}
		if string(body) != "body text\n" {
			t.Errorf("body = %q, want %q", body, "body text\n")
		}
	})

	t.Run("missing opening ---", func(t *testing.T) {
		_, _, err := SplitFrontmatter([]byte("no frontmatter"))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing closing ---", func(t *testing.T) {
		_, _, err := SplitFrontmatter([]byte("---\nkey: value\n"))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestExtractExcerpt(t *testing.T) {
	body := []byte("## Background\n\nSome background text.\n\n## Other\n\nOther text.\n")

	t.Run("named section found", func(t *testing.T) {
		got := ExtractExcerpt(body, "Background")
		if !strings.Contains(got, "Some background text") {
			t.Errorf("excerpt = %q, want to contain 'Some background text'", got)
		}
		if strings.Contains(got, "Other text") {
			t.Errorf("excerpt should not contain 'Other text', got %q", got)
		}
	})

	t.Run("section not found falls back to body", func(t *testing.T) {
		got := ExtractExcerpt(body, "NonExistent")
		if got == "" {
			t.Error("expected non-empty fallback")
		}
	})

	t.Run("empty section falls back to body", func(t *testing.T) {
		got := ExtractExcerpt(body, "")
		if got == "" {
			t.Error("expected non-empty fallback")
		}
	})

	t.Run("truncation", func(t *testing.T) {
		long := []byte("## Sec\n\n" + strings.Repeat("a", 400) + "\n")
		got := ExtractExcerpt(long, "Sec")
		if len([]rune(got)) > ExcerptMaxRunes+1 { // +1 for ellipsis rune
			t.Errorf("excerpt too long: %d runes", len([]rune(got)))
		}
	})
}

func TestParseHeading(t *testing.T) {
	cases := []struct {
		line      string
		wantText  string
		wantLevel int
		wantOk    bool
	}{
		{"# H1", "H1", 1, true},
		{"## H2 text", "H2 text", 2, true},
		{"###### H6", "H6", 6, true},
		{"####### too deep", "", 0, false},
		{"#nospace", "", 0, false},
		{"plain text", "", 0, false},
		{"", "", 0, false},
	}
	for _, c := range cases {
		text, level, ok := ParseHeading(c.line)
		if ok != c.wantOk || level != c.wantLevel || text != c.wantText {
			t.Errorf("ParseHeading(%q) = (%q, %d, %v), want (%q, %d, %v)",
				c.line, text, level, ok, c.wantText, c.wantLevel, c.wantOk)
		}
	}
}

func TestTruncateRunes(t *testing.T) {
	t.Run("no truncation needed", func(t *testing.T) {
		got := TruncateRunes("hello", 10)
		if got != "hello" {
			t.Errorf("got %q, want %q", got, "hello")
		}
	})

	t.Run("truncates with ellipsis", func(t *testing.T) {
		got := TruncateRunes("hello world", 5)
		if got != "hello…" {
			t.Errorf("got %q, want %q", got, "hello…")
		}
	})

	t.Run("exact length not truncated", func(t *testing.T) {
		got := TruncateRunes("hello", 5)
		if got != "hello" {
			t.Errorf("got %q, want %q", got, "hello")
		}
	})
}
