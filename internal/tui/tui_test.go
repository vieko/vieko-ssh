package tui

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "hel\u2026"},
		{"hello", 1, "h"},
		{"hello", 0, ""},
	}
	for _, c := range cases {
		got := truncate(c.in, c.max)
		if got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
		if runewidth.StringWidth(got) > c.max && c.max > 0 {
			t.Errorf("truncate(%q, %d) width %d exceeds max", c.in, c.max, runewidth.StringWidth(got))
		}
	}
}

func TestSplitRowFits(t *testing.T) {
	left, gap, right := splitRow("A Post Title", "June 23, 2026", 40)
	total := runewidth.StringWidth(left) + gap + runewidth.StringWidth(right)
	if total != 40 {
		t.Errorf("splitRow total width = %d, want 40", total)
	}
	if gap < 1 {
		t.Errorf("gap = %d, want >= 1", gap)
	}
}

func TestSplitRowTruncatesLongTitle(t *testing.T) {
	long := strings.Repeat("x", 100)
	left, gap, right := splitRow(long, "Jan 1, 2026", 30)
	total := runewidth.StringWidth(left) + gap + runewidth.StringWidth(right)
	if total > 30 {
		t.Errorf("splitRow total width = %d, want <= 30", total)
	}
	if !strings.HasSuffix(left, "\u2026") {
		t.Errorf("expected truncated title to end with ellipsis, got %q", left)
	}
}

func TestHyperlinkWrapsText(t *testing.T) {
	got := hyperlink("https://vieko.dev/x", "click")
	if !strings.Contains(got, "click") {
		t.Errorf("hyperlink missing visible text: %q", got)
	}
	if !strings.HasPrefix(got, "\x1b]8;;https://vieko.dev/x\x1b\\") {
		t.Errorf("hyperlink missing OSC-8 opener: %q", got)
	}
	if !strings.HasSuffix(got, "\x1b]8;;\x1b\\") {
		t.Errorf("hyperlink missing OSC-8 closer: %q", got)
	}
}
