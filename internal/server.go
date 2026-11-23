package app

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"os"
	"os/signal"
	"strings"

	"golang.org/x/crypto/acme/autocert"

	"github.com/gofiber/fiber/v3"
)

// startHTTPS starts the server with HTTPS support and automatic TLS.
func startHTTPS(app *fiber.App, mainAddr, httpsAddr string) error {
	hostOnly := extractHostOnly(mainAddr)
	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(hostOnly),
		Cache:      autocert.DirCache("./lc_certs"),
	}

	cfgTLS := &tls.Config{
		GetCertificate: manager.GetCertificate,
		NextProtos:     []string{"http/1.1", "acme-tls/1"},
	}

	listenAddr := DefaultHTTPSPort
	if httpsAddr != "" {
		listenAddr = httpsAddr
	}

	ln, err := tls.Listen("tcp", listenAddr, cfgTLS)
	if err != nil {
		logger().Err(err).Send()
		os.Exit(1)
	}

	if err := app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}); err != nil {
		logger().Err(err).Send()
		os.Exit(1)
	}

	return nil
}

// extractHostOnly extracts only the host from the address, removing the port.
func extractHostOnly(addr string) string {
	if !strings.Contains(addr, ":") {
		return addr
	}

	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}

	return addr
}

// startHTTP starts the HTTP server with graceful shutdown support.
func startHTTP(mainAddr string, app *fiber.App) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if DevMode() {
		return StartServer(ctx, mainAddr, app)
	}

	idleConnsClosed := make(chan struct{})

	go handleShutdown(ctx, app, idleConnsClosed)
	go func() {
		if err := StartServer(ctx, mainAddr, app); err != nil {
			logger().Err(err).Send()
		}
	}()

	<-idleConnsClosed
	return nil
}

// handleShutdown handles application shutdown signals.
func handleShutdown(ctx context.Context, app *fiber.App, idleConnsClosed chan struct{}) {
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	<-sigint

	if err := app.Shutdown(); err != nil {
		logger().Err(err).Send()
	}

	close(idleConnsClosed)
}

// StartServer starts the server and handles graceful shutdown.
func StartServer(ctx context.Context, addr string, a *fiber.App) error {
	// Buffered, so the listen goroutine can always report its error and exit.
	// Unbuffered it blocks on the send forever when the shutdown path below has
	// already returned — a goroutine leaked for the life of the process.
	errCh := make(chan error, 1)

	go func() {
		if err := a.Listen(addr, fiber.ListenConfig{DisableStartupMessage: true}); err != nil {
			logger().Err(err).Send()
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		err := errors.New("shutdown signal received, closing server")
		logger().Err(err).Send()
		return a.Shutdown()
	case err := <-errCh:
		return err
	}
}
