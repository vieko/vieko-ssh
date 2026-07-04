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

Config via env: `VIEKO_SSH_HOST` (default `::`, dual-stack), `VIEKO_SSH_PORT`
(default `2222`), `VIEKO_SSH_HOST_KEY` (default `.ssh/id_ed25519`, generated on
first run).

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

- **Visitor** (`ssh vieko.sh`): the edge allows `:22`, the box's nftables DNATs
  `:22` to `:2222`, and the `vieko-ssh` service renders the TUI. The app binds a
  high port, so it never needs root or `CAP_NET_BIND_SERVICE`; `:22` is just a
  redirect, not a listener.
- **Web** (`https://vieko.sh`): the edge allows `:80`/`:443`, Caddy
  302-redirects to `vieko.dev` with the path preserved and an auto-renewing
  Let's Encrypt cert.
- **Admin** (`ssh -p 2200`): the edge allows `:2200` from one source only,
  key-only OpenSSH, moved off `:22` so the front door is free for the redirect.

Everything is dual-stack (IPv4 and IPv6). Nothing in this repo is a secret:
what protects the box is the SSH keys and the admin allowlist, both of which
live on the box, not here. The `deploy/` files (systemd unit, nftables ruleset,
Hetzner Cloud Firewall script, deploy script) carry their own setup steps in
header comments.

## Updating content

`internal/content/content.go` is hand-mirrored from
[`vieko.dev`'s `src/lib/posts.ts`](https://github.com/vieko/vieko.dev/blob/main/src/lib/posts.ts)
and kept in sync with `vieko-cli`'s `src/data.js`. No fetch/build coupling, so
update the `Posts` slice by hand when new writing ships.

## License

MIT
