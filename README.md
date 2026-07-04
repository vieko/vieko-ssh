# vieko-ssh

The [vieko.dev](https://vieko.dev) one-pager, served over SSH.

```
ssh vieko.sh
```

Browse recent writing without leaving the terminal — no install, just `ssh`.
It's the sibling of [`@vieko/cli`](https://github.com/vieko/vieko-cli) (the
`npx` version): same layout, same keys, delivered as a public SSH TUI instead
of a package. Built with [Charm's `wish`](https://github.com/charmbracelet/wish)
+ [`bubbletea`](https://github.com/charmbracelet/bubbletea).

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
prints its URL in a status line for easy copying — there's no browser to launch
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

## Layout

```
main.go                     wish server + hardened middleware stack
internal/content/           the data — hand-mirrored from vieko.dev
internal/tui/               bubbletea model (port of vieko-cli's render loop)
deploy/                     systemd unit, nftables ruleset, deploy script
```

## Architecture

Three things share one small box behind one domain. The SSH TUI is the point;
the web redirect and admin SSH are along for the ride.

```
                        ssh vieko.sh                 https://vieko.sh
                            │                              │
                            ▼                              ▼
                  ┌───────────────────────────────────────────────┐
                  │        Hetzner Cloud Firewall (edge)           │
                  │  :22, :80, :443  → anyone                      │
                  │  :2200 (admin)   → one allowlisted source only │
                  └───────────────────────────────────────────────┘
                            │            │              │
                    :22     │      :80/:443             │ :2200
                            ▼            ▼              ▼
                  ┌───────────────────────────────────────────────┐
                  │   the box: nftables (default-drop) + services  │
                  │                                                │
                  │   :22 ──DNAT──▶ :2222   vieko-ssh (wish/tea)   │
                  │   :80/:443 ───────────▶ Caddy → vieko.dev      │
                  │   :2200 ──────────────▶ OpenSSH (key-only)     │
                  └───────────────────────────────────────────────┘
```

**Request paths**
- **Visitor** (`ssh vieko.sh`): edge allows `:22` → box's nftables DNATs
  `:22`→`:2222` → the `vieko-ssh` service renders the TUI. The app binds a high
  port so it never needs root or `CAP_NET_BIND_SERVICE`; the privileged port
  `:22` is just a redirect, not a listener.
- **Web** (`https://vieko.sh`): edge allows `:80`/`:443` → Caddy 302-redirects
  to `vieko.dev` (path preserved) with an auto-renewing Let's Encrypt cert.
- **Admin** (`ssh -p 2200`): edge allows `:2200` only from one source → OpenSSH,
  key-only. Moved off `:22` so the front door is free for the TUI redirect.

All three are dual-stack (IPv4 + IPv6). Nothing here is a secret — see the
threat model below for why that's fine.

## Security posture

This is an intentionally public, **anonymous**, read-only TUI pointed at the
open internet. That's close to a worst-case exposure profile, so hardening is
layered — no single control is load-bearing:

**Application (`main.go`)**
- **No auth by design.** Clients connect with the SSH `none` method. There is
  no shell, no filesystem access, no `exec` — the only thing a session can do is
  drive the TUI.
- **PTY gate** (`activeterm`): non-interactive `exec`/subsystem attempts are
  rejected before reaching the app.
- **Port forwarding disabled**: no `direct-tcpip`/`tcpip-forward` callbacks are
  registered, which is deny-by-default in `charmbracelet/ssh`. No tunneling.
- **Panic isolation** (`recover`): one bad session can't crash the process and
  drop everyone else.
- **Per-IP rate limiting** (`ratelimiter`): caps connection churn from a single
  source (1/s, burst 3).
- **Timeouts**: 5-minute idle, 30-minute absolute.
- **Modern crypto**: the pinned `golang.org/x/crypto` (≥ 0.50) defaults to
  curve25519 KEX, AES-GCM / ChaCha20-Poly1305, ed25519 host key. Terrapin
  (CVE-2023-48795) is mitigated upstream. Keeping this dependency current is the
  single most important ongoing task — Go's SSH transport has a mature
  disclose-and-patch pipeline, which only helps if you pull the patches.

**OS (`deploy/vieko-ssh.service`)**
- Dedicated unprivileged `vieko-ssh` user, `nologin` shell.
- Binds a high port (2222), so **zero capabilities** are needed
  (`CapabilityBoundingSet=`).
- Full systemd sandbox: `ProtectSystem=strict`, `NoNewPrivileges`,
  `PrivateTmp/Devices`, `MemoryDenyWriteExecute`, `RestrictAddressFamilies`,
  `SystemCallFilter=@system-service`, memory/task/fd caps.
- `Restart=on-failure` with a start-limit — supervised recovery without a hot
  crash loop.

**Network (`deploy/nftables.conf`)**
- Per-source new-connection rate limiting at the firewall (primary DoS defense).
- Public `:22` is redirected to `:2222`; your **admin OpenSSH is moved to
  `:2200`, key-only** — fail2ban belongs on *that* port (failed auth is a
  concept there; it isn't on the anonymous TUI port).
- Pair with the free Hetzner Cloud Firewall in front of the box.

### Why Go + wish (not Node + ssh2)

Both can do this. Go won narrowly on two axes: `x/crypto/ssh` (what `wish` sits
on) has a mature, well-scrutinized security process with a real disclosure
pipeline, and `wish` gives the hardening as composable middleware
(`recover`, `ratelimiter`, `activeterm`) rather than hand-rolled guards — which
matters when the input is anonymous and adversarial. `ssh2` is a fine,
single-maintainer pure-JS library, but its risk surface (a history of
crash-on-malformed-input issues, one process serving everyone) is a poorer fit
for this specific threat model.

### Threat model — why this repo is safe to open-source

The design is fully public on purpose. Security here follows Kerckhoffs's
principle: **knowing exactly how it's built shouldn't help you break in.** What
actually protects it lives *outside* this repo:

- **The SSH host + user keys** (gitignored; the host key is generated on the box
  at first run and never leaves it).
- **The admin allowlist** — `:2200` is reachable only from one source IP at the
  cloud edge, and even then it's key-only (passwords off). The *port number*
  isn't a secret and doesn't need to be; the allowlist and the key are the lock.
- **The Hetzner API token** — never committed; lives in local `hcloud` config /
  a password manager, and is revocable.

So what an attacker learns from this repo is the *shape* of the system, which is
intentionally boring: an anonymous, read-only TUI with no shell, no `exec`, no
filesystem, no forwarding, one hardened process behind default-drop firewalls.
There's no credential, IP allowlist entry, or private key to find here, and no
control relies on you *not* knowing the architecture.

The deploy files (`deploy/`) are the author's actual setup *shape*, not a
turn-key drop-in. If you fork this, bring your own domain, keys, admin source
IP, and — since there's no reason to reuse mine — your own admin port. None of
the security depends on those specific values.

## Deploying

Everything runs on one small VPS. The files in [`deploy/`](deploy/) are
self-documenting — the systemd unit, nftables ruleset, Hetzner Cloud Firewall
script, and a build/upload/restart `deploy.sh` each carry header comments with
their own steps. Day-to-day it's one command:

```sh
VIEKO_SSH_DEPLOY_HOST=root@your-box ./deploy/deploy.sh
```

The one gotcha worth stating up front: **move admin SSH off `:22` and confirm
you can reconnect on the new port _before_ applying the firewall** — otherwise
the `:22`→`:2222` front-door redirect locks you out of admin.

## Updating content

`internal/content/content.go` is hand-mirrored from
[`vieko.dev`'s `src/lib/posts.ts`](https://github.com/vieko/vieko.dev/blob/main/src/lib/posts.ts)
and kept in sync with `vieko-cli`'s `src/data.js`. No fetch/build coupling —
update the `Posts` slice by hand when new writing ships.

## License

MIT
