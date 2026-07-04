// Package content holds the site data rendered by the TUI.
//
// This is hand-mirrored from vieko.dev's src/lib/posts.ts (and kept in sync
// with the sibling vieko-cli package). There is intentionally no build step or
// fetch dependency on the site. Update POSTS by hand when new writing ships.
package content

const (
	SiteURL       = "https://vieko.dev"
	Tagline       = "Developer and game maker summoning demons at \u25b2 Vercel"
	GitHubURL     = "https://github.com/vieko"
	GitHubDisplay = "github.com/vieko"
	XURL          = "https://x.com/vieko"
	XDisplay      = "x.com/vieko"
	Email         = "hello@vieko.dev"
	EmailURL      = "mailto:hello@vieko.dev"
)

// Post is a single writing entry.
type Post struct {
	Title string
	Slug  string
	Date  string
}

// URL returns the absolute URL for a post.
func (p Post) URL() string { return SiteURL + p.Slug }

// Posts is the recent-writing list, newest first.
var Posts = []Post{
	{Title: "Say No", Slug: "/say-no", Date: "June 23, 2026"},
	{Title: "Ask Again", Slug: "/ask-again", Date: "May 27, 2026"},
	{Title: "Prompt Bench", Slug: "/prompt-bench", Date: "March 6, 2026"},
	{Title: "Making Games with Agents", Slug: "/reach", Date: "February 23, 2026"},
	{Title: "The Orchestrator I Didn't Build", Slug: "/outcomes", Date: "February 13, 2026"},
	{Title: "Save Your Progress", Slug: "/bonfire", Date: "January 2, 2026"},
	{Title: "There Are Only Artists", Slug: "/artists", Date: "December 31, 2025"},
	{Title: "Changing Everything at Once", Slug: "/mothership", Date: "November 23, 2025"},
	{Title: "Pairing with a Partner Who Forgets Everything", Slug: "/sessions-directory-pattern", Date: "November 16, 2025"},
	{Title: "Thirteen Years at Devolver Digital", Slug: "/devolver", Date: "October 31, 2025"},
}

// LogoLines is the active "vieko" wordmark (LOGO_CLASSY from vieko-cli).
var LogoLines = []string{
	"       \u2580\u2580       \u2584\u2584          ",
	"\u2580\u2588\u2584 \u2588\u2588\u2580\u2588\u2588 \u2584\u2588\u2580\u2588\u2584 \u2588\u2588 \u2584\u2588\u2580 \u2584\u2588\u2588\u2588\u2584",
	" \u2588\u2588\u2584\u2588\u2588 \u2588\u2588 \u2588\u2588\u2584\u2588\u2580 \u2588\u2588\u2588\u2588   \u2588\u2588 \u2588\u2588",
	"  \u2580\u2588\u2580 \u2584\u2588\u2588\u2584\u2580\u2588\u2584\u2584\u2584\u2584\u2588\u2588 \u2580\u2588\u2584\u2584\u2580\u2588\u2588\u2588\u2580",
}
