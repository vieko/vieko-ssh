package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

var ansiRE = regexp.MustCompile("\x1b\\][0-9];;[^\x1b]*\x1b\\\\|\x1b\\[[0-9;?]*[A-Za-z]")

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

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

func TestStatusLineBelowFooter(t *testing.T) {
	m := Model{width: 100, height: 30, status: "https://vieko.dev/say-no"}
	lines := strings.Split(m.frame(), "\n")

	footerIdx, statusIdx := -1, -1
	for i, ln := range lines {
		plain := stripANSI(ln)
		if strings.Contains(plain, "Enter open") && strings.Contains(plain, "quit") {
			footerIdx = i
		}
		if strings.Contains(plain, "cmd/ctrl-click or copy") {
			statusIdx = i
		}
	}

	if footerIdx == -1 {
		t.Fatal("footer line not found")
	}
	if statusIdx == -1 {
		t.Fatal("status line not found")
	}
	if statusIdx <= footerIdx {
		t.Fatalf("status (%d) should be below footer (%d)", statusIdx, footerIdx)
	}
	if statusIdx-footerIdx != 2 {
		t.Fatalf("expected exactly one blank line between footer (%d) and status (%d)", footerIdx, statusIdx)
	}
	if blank := stripANSI(lines[footerIdx+1]); strings.TrimSpace(blank) != "" {
		t.Fatalf("line between footer and status should be blank, got %q", blank)
	}
	if statusIdx != len(lines)-1 {
		t.Fatalf("status line should be last (idx %d of %d)", statusIdx, len(lines))
	}
}

func TestStatusLineEmptyWhenNoAction(t *testing.T) {
	m := Model{width: 100, height: 30}
	lines := strings.Split(m.frame(), "\n")
	if got := stripANSI(lines[len(lines)-1]); strings.TrimSpace(got) != "" {
		t.Fatalf("last line should be empty with no action, got %q", got)
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
