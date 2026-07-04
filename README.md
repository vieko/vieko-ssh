# vieko-ssh

The [vieko.dev](https://vieko.dev) one-pager, served over SSH.

```
ssh vieko.sh
```

Browse recent writing without leaving the terminal. No install, just `ssh`.
It's the sibling of [`@vieko/cli`](https://github.com/vieko/vieko-cli) (the
`npx` version): same layout, same keys, delivered as a public SSH TUI instead
of a package. Built with [Charm's `wish`](https://github.com/charmbracelet/wish)
and [`bubbletea`](https://github.com/charmbracelet/bubbletea).

## Keys

| Key         | Action                    |
| ----------- | ------------------------- |
| `↑`/`k`     | Move up                   |
| `↓`/`j`     | Move down                 |
| `Enter`     | Surface selected post URL |
| `g`         | Surface GitHub URL        |
| `x`         | Surface X URL             |
| `e`         | Surface email             |
| `q` / `Esc` | Quit                      |

Post titles and footer links are emitted as OSC-8 hyperlinks, so in a
supporting terminal you can ⌘/ctrl-click them directly. Activating an item also
prints its URL in a status line for easy copying. There's no browser to launch
on the server side, so we hand the URL back to *your* terminal instead.

## Run locally

```sh
go run .
# then, in another shell:
ssh localhost -p 2222
```

## Architecture

Three things share one small box behind one domain. The SSH TUI is the point;
the web redirect and admin SSH are along for the ride.

```
              ssh vieko.sh        https://vieko.sh
                    │                     │
                    ▼                     ▼
        ┌──────────────────────────────────────────────┐
        │        Hetzner Cloud Firewall (edge)         │
        │  :22, :80, :443 → anyone                     │
        │  :2200 (admin)  → one allowlisted source only│
        └──────────────────────────────────────────────┘
                    │            │            │
                  :22        :80/:443       :2200
                    ▼            ▼            ▼
        ┌──────────────────────────────────────────────┐
        │  the box: nftables (default-drop) + services │
        │                                              │
        │  :22      → :2222  vieko-ssh (wish/tea)      │
        │  :80/:443 → Caddy → vieko.dev                │
        │  :2200    → OpenSSH (key-only)               │
        └──────────────────────────────────────────────┘
```

The app binds an unprivileged high port (`:2222`); nftables DNATs `:22` to it,
so the front door needs no root. Caddy owns `:80`/`:443` and redirects the web
hit to `vieko.dev` (path preserved, auto-renewing TLS). Admin SSH sits on
`:2200`, key-only and allowlisted to one source, moved off `:22` to leave the
front door for the TUI.

Everything is dual-stack (IPv4 and IPv6). Nothing in this repo is a secret:
what protects the box is the SSH keys and the admin allowlist, both of which
live on the box, not here. The `deploy/` files carry their own setup steps in
header comments.

## Updating content

`internal/content/content.go` is hand-mirrored from
[`vieko.dev`'s `src/lib/posts.ts`](https://github.com/vieko/vieko.dev/blob/main/src/lib/posts.ts)
and kept in sync with `vieko-cli`'s `src/data.js`. No fetch/build coupling, so
update the `Posts` slice by hand when new writing ships.

## License

MIT
