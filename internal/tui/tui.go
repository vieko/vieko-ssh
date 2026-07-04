// Package tui renders the vieko.dev one-pager as a Bubble Tea model served
// over SSH. It is a faithful port of vieko-cli's src/app.js render loop:
// same layout, same keys, same chrome. The one semantic difference is that we
// cannot open a browser on the server, so post/link activation emits an OSC-8
// hyperlink (clickable in supporting terminals) plus a copyable URL status
// line instead of shelling out to `open`/`xdg-open`.
package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/ssh"
	"github.com/mattn/go-runewidth"

	"github.com/vieko/vieko-ssh/internal/content"
)

// ANSI SGR codes (mirrors app.js).
const (
	reset   = "\x1b[0m"
	bold    = "\x1b[1m"
	dim     = "\x1b[2m"
	reverse = "\x1b[7m"
	white   = "\x1b[97m"
	gray    = "\x1b[90m"

	navSymbols = "\u2191\u2193" // ↑↓
)

// Layout constants (mirrors app.js).
const (
	minWidth      = 60
	maxWidth      = 78
	minVisibleRow = 3
)

// fixedChromeLines: blank + logo + blank + tagline + blank + header + blank +
// footer + blank + status line.
var fixedChromeLines = 9 + len(content.LogoLines)

// Model is the Bubble Tea model for a single SSH session.
type Model struct {
	width, height int
	selected      int
	scrollOffset  int
	status        string // URL surfaced after an action (Enter/g/x/e); "" hides it
}

// New builds a Model seeded with the session's initial PTY dimensions.
func New(s ssh.Session) Model {
	pty, _, _ := s.Pty()
	return Model{width: pty.Window.Width, height: pty.Window.Height}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(content.Posts)-1 {
				m.selected++
			}
		case "enter":
			m.status = content.Posts[m.selected].URL()
		case "g":
			m.status = content.GitHubURL
		case "x":
			m.status = content.XURL
		case "e":
			m.status = content.EmailURL
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	v := tea.NewView(m.frame())
	v.AltScreen = true
	return v
}

// --- rendering ---------------------------------------------------------------

func (m Model) contentWidth() int {
	cols := m.width
	if cols <= 0 {
		cols = maxWidth
	}
	w := cols - 2
	if w > maxWidth {
		w = maxWidth
	}
	if w < minWidth {
		w = minWidth
	}
	return w
}

func (m Model) visibleRows() int {
	rows := m.height
	if rows <= 0 {
		rows = 24
	}
	max := rows - fixedChromeLines
	if max < minVisibleRow {
		max = minVisibleRow
	}
	if max > len(content.Posts) {
		max = len(content.Posts)
	}
	return max
}

func (m *Model) clampScroll() {
	visible := m.visibleRows()
	if m.selected < m.scrollOffset {
		m.scrollOffset = m.selected
	} else if m.selected >= m.scrollOffset+visible {
		m.scrollOffset = m.selected - visible + 1
	}
	maxOffset := len(content.Posts) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
}

func (m Model) frame() string {
	m.clampScroll()
	w := m.contentWidth()
	visible := m.visibleRows()
	var lines []string

	lines = append(lines, "")

	for _, logo := range content.LogoLines {
		lines = append(lines, "  "+bold+white+logo+reset)
	}

	lines = append(lines, "")
	lines = append(lines, "  "+dim+gray+truncate(content.Tagline, w)+reset)
	lines = append(lines, "")

	// RECENT WRITING ───────────
	const headerLabel = "RECENT WRITING"
	ruleLen := w - runewidth.StringWidth(headerLabel) - 1
	if ruleLen < 0 {
		ruleLen = 0
	}
	lines = append(lines, "  "+dim+gray+headerLabel+" "+strings.Repeat("\u2500", ruleLen)+reset)

	// Post rows (windowed).
	start := m.scrollOffset
	end := start + visible
	if end > len(content.Posts) {
		end = len(content.Posts)
	}
	for i := start; i < end; i++ {
		post := content.Posts[i]
		left, gap, right := splitRow(post.Title, post.Date, w)
		linkedLeft := hyperlink(post.URL(), left)
		if i == m.selected {
			// Reverse-video the whole row, padded to width.
			row := linkedLeft + strings.Repeat(" ", gap) + right
			pad := w - (runewidth.StringWidth(left) + gap + runewidth.StringWidth(right))
			if pad > 0 {
				row += strings.Repeat(" ", pad)
			}
			lines = append(lines, "  "+reverse+row+reset)
		} else {
			lines = append(lines, "  "+white+linkedLeft+strings.Repeat(" ", gap)+right+reset)
		}
	}
	for i := end - start; i < visible; i++ {
		lines = append(lines, "")
	}

	// Blank separator, then footer hints.
	lines = append(lines, "")
	lines = append(lines, "  "+m.footer(w))

	// Blank spacer, then the action/status line below the footer. The row is
	// always reserved (empty when no action is active) so the layout (and
	// thus visibleRows math) stays stable.
	lines = append(lines, "")
	lines = append(lines, m.statusLine(w))

	return strings.Join(lines, "\n")
}

func (m Model) statusLine(w int) string {
	if m.status == "" {
		return ""
	}
	prefix := "\u2197 " // ↗
	suffix := "  \u00b7  cmd/ctrl-click or copy"
	// Budget the URL so the whole line fits.
	budget := w - runewidth.StringWidth(prefix) - runewidth.StringWidth(suffix)
	if budget < 8 {
		budget = 8
	}
	shown := truncate(m.status, budget)
	return "  " + dim + gray + prefix + reset + hyperlink(m.status, white+shown+reset+dim+gray) + suffix + reset
}

func (m Model) footer(w int) string {
	label := func(t string) string { return dim + gray + t + reset }
	mnemonic := func(text string, i int) string {
		before, char, after := text[:i], string(text[i]), text[i+1:]
		out := ""
		if before != "" {
			out += label(before)
		}
		out += bold + white + char + reset
		if after != "" {
			out += label(after)
		}
		return out
	}

	type seg struct{ styled, plain string }
	segs := []seg{
		{bold + white + navSymbols + reset + " " + bold + white + "Enter" + reset + " " + label("open"), navSymbols + " Enter open"},
		{hyperlink(content.GitHubURL, mnemonic(content.GitHubDisplay, 0)), content.GitHubDisplay},
		{hyperlink(content.XURL, mnemonic(content.XDisplay, 0)), content.XDisplay},
		{hyperlink(content.EmailURL, mnemonic(content.Email, 1)), content.Email},
		{mnemonic("quit", 0), "quit"},
	}

	var styled, plain strings.Builder
	for i, s := range segs {
		styled.WriteString(s.styled)
		plain.WriteString(s.plain)
		if i < len(segs)-1 {
			styled.WriteString("   ")
			plain.WriteString("   ")
		}
	}
	gap := w - runewidth.StringWidth(plain.String())
	if gap < 0 {
		gap = 0
	}
	return styled.String() + strings.Repeat(" ", gap)
}

// --- helpers -----------------------------------------------------------------

// hyperlink wraps text in an OSC-8 terminal hyperlink. Terminals that don't
// support OSC-8 ignore the wrapper and render text plainly.
func hyperlink(url, text string) string {
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// truncate shortens s to display width max, appending an ellipsis.
func truncate(s string, max int) string {
	if runewidth.StringWidth(s) <= max {
		return s
	}
	if max <= 1 {
		return runewidth.Truncate(s, max, "")
	}
	return runewidth.Truncate(s, max, "\u2026")
}

// splitRow lays out left (title) and right (date) within total display width,
// truncating the left as needed. It returns the (possibly truncated) left, the
// gap width between them, and the right, so callers can inject styling/links
// without disturbing the width math.
func splitRow(left, right string, total int) (string, int, string) {
	rightLen := runewidth.StringWidth(right)
	leftMax := total - rightLen - 1
	if leftMax < 0 {
		leftMax = 0
	}
	leftTrunc := truncate(left, leftMax)
	gap := total - runewidth.StringWidth(leftTrunc) - rightLen
	if gap < 1 {
		gap = 1
	}
	return leftTrunc, gap, right
}
