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

Config via env: `VIEKO_SSH_HOST` (default `0.0.0.0`), `VIEKO_SSH_PORT`
(default `2222`), `VIEKO_SSH_HOST_KEY` (default `.ssh/id_ed25519`, generated on
first run).

## Layout

```
main.go                     wish server + hardened middleware stack
internal/content/           the data — hand-mirrored from vieko.dev
internal/tui/               bubbletea model (port of vieko-cli's render loop)
deploy/                     systemd unit, nftables ruleset, deploy script
```

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

## Deploy to Hetzner

One-time setup on a fresh box (Debian/Ubuntu):

```sh
# 0. Move your admin SSH off :22 BEFORE touching the firewall, so you don't
#    lock yourself out. Edit /etc/ssh/sshd_config: `Port 2200`, `PasswordAuthentication no`.
sudo systemctl reload ssh
# reconnect on :2200 and confirm it works before continuing.

# 1. Service user + state dir
sudo useradd --system --home /var/lib/vieko-ssh --create-home \
  --shell /usr/sbin/nologin vieko-ssh

# 2. Firewall (redirects :22 -> :2222, rate limits, keeps :2200 for admin)
sudo cp deploy/nftables.conf /etc/nftables.conf
sudo systemctl enable --now nftables

# 3. First deploy (builds, uploads, installs the unit, starts it)
VIEKO_SSH_DEPLOY_HOST=root@vieko.sh VIEKO_SSH_ADMIN_PORT=2200 ./deploy/deploy.sh
```

Then point DNS `A`/`AAAA` for the `vieko.sh` apex at the box so `ssh vieko.sh`
reaches the front door.

Finally, add the **Hetzner Cloud Firewall** (network-edge layer, in front of
nftables) so admin SSH is invisible to internet scanners:

```sh
hcloud context create vieko-ssh --token-from-env   # HCLOUD_TOKEN=<read-write token>
ADMIN_SRC_V4=<your-ip>/32 ADMIN_SRC_V6=<your-ip6>/64 ./deploy/hcloud-firewall.sh
```

This opens `:22`/`:80`/`:443` to the world and restricts `:2200` to your
source only. `:2222` is deliberately not exposed (the front door is `:22`; the
`:22`->`:2222` DNAT happens on the box, after the edge firewall). If your admin
source IP changes you can always recover via the Hetzner Console (edit the
rule) or the server's VNC Console (bypasses the network firewall).

Subsequent deploys are just:

```sh
VIEKO_SSH_DEPLOY_HOST=root@vieko.sh ./deploy/deploy.sh
```

## Updating content

`internal/content/content.go` is hand-mirrored from
[`vieko.dev`'s `src/lib/posts.ts`](https://github.com/vieko/vieko.dev/blob/main/src/lib/posts.ts)
and kept in sync with `vieko-cli`'s `src/data.js`. No fetch/build coupling —
update the `Posts` slice by hand when new writing ships.

## License

MIT
