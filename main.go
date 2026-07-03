// Command vieko-ssh serves the vieko.dev one-pager over SSH.
//
// Connect with: ssh <host> -p 2222
//
// Security posture (see README for the full rationale):
//   - Anonymous by design: no auth handler is set, so clients connect with the
//     SSH "none" method. This is an intentionally public, read-only TUI. It
//     exposes no shell, no filesystem, no exec.
//   - activeterm requires a PTY, so non-interactive `exec`/subsystem attempts
//     are rejected before reaching the app.
//   - Port forwarding (direct-tcpip / tcpip-forward) is disabled: no forwarding
//     callbacks are registered, which is deny-by-default in charmbracelet/ssh.
//   - recover wraps the app so a panic in one session cannot crash the process
//     (which would drop every other connected user).
//   - A per-IP rate limiter caps connection churn from any single source.
//   - Idle and absolute timeouts bound session lifetime.
//   - Crypto algorithms are the modern hardened defaults of the pinned
//     golang.org/x/crypto (>= 0.50): curve25519 KEX, AES-GCM / ChaCha20-Poly1305
//     ciphers, ed25519 host key. Terrapin (CVE-2023-48795) is mitigated upstream.
package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"charm.land/wish/v2/ratelimiter"
	"charm.land/wish/v2/recover"
	"github.com/charmbracelet/ssh"
	"golang.org/x/time/rate"

	"github.com/vieko/vieko-ssh/internal/tui"
)

func main() {
	host := env("VIEKO_SSH_HOST", "0.0.0.0")
	port := env("VIEKO_SSH_PORT", "2222")
	hostKey := env("VIEKO_SSH_HOST_KEY", ".ssh/id_ed25519")

	// Per-IP connection rate limiter: 1/s sustained, burst 3, tracking up to
	// 4096 distinct IPs (LRU). Blunt but effective against connection floods
	// from a single source; the network firewall is the primary DoS defense.
	limiter := ratelimiter.NewRateLimiter(rate.Limit(1), 3, 4096)

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(hostKey),
		wish.WithIdleTimeout(5*time.Minute),
		wish.WithMaxTimeout(30*time.Minute),
		wish.WithMiddleware(
			// Applied bottom-up: logging is outermost (logs every connection,
			// including rejected ones), then rate limiting, then the recover-
			// wrapped app (PTY gate + Bubble Tea program).
			recover.Middleware(
				bubbletea.Middleware(teaHandler),
				activeterm.Middleware(),
			),
			ratelimiter.Middleware(limiter),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Fatal("could not create server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("starting vieko-ssh", "addr", net.JoinHostPort(host, port))
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("server error", "error", err)
			done <- syscall.SIGTERM
		}
	}()

	<-done
	log.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("shutdown error", "error", err)
	}
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	return tui.New(s), nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
