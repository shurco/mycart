package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/internal/testutil"
	"github.com/shurco/mycart/pkg/logging"
)

// startupBanner renders the startup banner for the given arguments and returns
// what it printed.
//
// The banner goes to an explicit writer, so the test reads it back without
// swapping os.Stdout: that is process-wide state the server goroutines print
// through too, and redirecting it makes this test race with any of them.
func startupBanner(t *testing.T, schema, mainAddr string, noSite bool, dbCfg database.Config) string {
	t.Helper()

	var buf bytes.Buffer
	printStartupInfo(&buf, schema, mainAddr, noSite, dbCfg)
	return buf.String()
}

// captureStdout runs fn with stdout redirected to a pipe and returns what it
// printed.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	previous := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	// Restoring in Cleanup keeps a panicking fn from leaving stdout swapped for
	// the rest of the test binary; the assignment below brings stdout back before
	// the caller's own output.
	t.Cleanup(func() { os.Stdout = previous })

	fn()

	os.Stdout = previous
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return string(out)
}

func TestMigrate(t *testing.T) {
	// Not parallel: the test asserts on a file it creates, and Migrate opens
	// the database through the process-wide helpers.
	dsn := filepath.Join(t.TempDir(), "migrated.db")
	cfg := database.Config{Driver: database.DriverSQLite, DSN: dsn}

	if err := Migrate(cfg); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if _, err := os.Stat(dsn); err != nil {
		t.Fatalf("the database file was not created: %v", err)
	}

	// The schema is what makes the file usable: the migration seeds the
	// settings row the app reads on the next start.
	conn, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect to the migrated database: %v", err)
	}
	defer func() { _ = conn.Close() }()

	var settings int
	if err := conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM setting`).Scan(&settings); err != nil {
		t.Fatalf("query the migrated schema: %v", err)
	}
	if settings == 0 {
		t.Error("the migrated database has no settings; the schema did not run")
	}

	// Running it again must be a no-op rather than an error: every process
	// start calls it.
	if err := Migrate(cfg); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}

func TestPrintStartupInfo(t *testing.T) {
	cfg := database.Config{
		Driver: database.DriverPostgres,
		DSN:    "postgres://cart:secret@db.example.com:5432/cart?sslmode=disable",
	}

	t.Run("production", func(t *testing.T) {
		setDevMode(false)
		out := startupBanner(t, "http", "site.com:80", false, cfg)

		for _, want := range []string{"Cart UI: http://site.com:80/", "Admin UI: http://site.com:80/_/", "Database: postgres"} {
			if !strings.Contains(out, want) {
				t.Errorf("startup output does not mention %q:\n%s", want, out)
			}
		}
		if !strings.Contains(out, "Swagger UI: disabled") {
			t.Errorf("production output should say swagger is disabled:\n%s", out)
		}
		if strings.Contains(out, "secret") {
			t.Errorf("the connection string password leaked into the startup output:\n%s", out)
		}
	})

	t.Run("development", func(t *testing.T) {
		setDevMode(true)
		defer func() { setDevMode(false) }()

		out := startupBanner(t, "https", "site.com:443", false, cfg)
		if !strings.Contains(out, "API Docs: https://site.com:443/swagger/index.html") {
			t.Errorf("dev output should point at the swagger UI:\n%s", out)
		}
	})

	t.Run("no site", func(t *testing.T) {
		setDevMode(false)
		out := startupBanner(t, "http", "site.com:80", true, cfg)
		if strings.Contains(out, "Cart UI") {
			t.Errorf("--no-site must not advertise a cart UI:\n%s", out)
		}
		if !strings.Contains(out, "Admin UI") {
			t.Errorf("--no-site still serves the admin UI:\n%s", out)
		}
	})
}

func TestSetupRoutes(t *testing.T) {
	// The fixtures leave the cart installed, which is what makes the API
	// routes answerable instead of redirecting to the wizard. The cleanup has
	// to be registered: a pool left open keeps pgtestdb from dropping the
	// database, which fails the test on PostgreSQL for a reason that has
	// nothing to do with routes.
	t.Cleanup(testutil.SetupTestDB(t))

	setLogger(logging.New())
	t.Cleanup(func() { setLogger(logging.New()) })

	build := func(t *testing.T, noSite bool) *fiber.App {
		t.Helper()

		app, err := setupFiberApp(noSite)
		if err != nil {
			t.Fatalf("setupFiberApp: %v", err)
		}
		t.Cleanup(func() { _ = app.Shutdown() })

		setupRoutes(app, noSite)
		return app
	}

	t.Run("the full app serves the api and the storefront", func(t *testing.T) {
		app := build(t, false)

		for _, path := range []string{"/ping", "/api/products"} {
			resp := testutil.DoRequest(t, app, http.MethodGet, path, "", "")
			testutil.AssertStatus(t, resp, http.StatusOK)
		}
	})

	t.Run("--no-site leaves the storefront out", func(t *testing.T) {
		app := build(t, true)

		// /ping and the public API belong to the storefront, which this mode
		// does not serve at all.
		for _, path := range []string{"/ping", "/api/products"} {
			resp := testutil.DoRequest(t, app, http.MethodGet, path, "", "")
			testutil.AssertStatus(t, resp, http.StatusNotFound)
		}

		// The admin API is still registered — it answers 401 without a
		// session, which is what distinguishes it from an unrouted 404.
		resp := testutil.DoRequest(t, app, http.MethodGet, "/api/_/products", "", "")
		testutil.AssertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("the install guard runs before the spa handler", func(t *testing.T) {
		// A path the guard redirects must not be answered by the SPA, which
		// serves index.html without calling Next. The fixtures are installed,
		// so /_/install is the path that gets redirected away.
		app := build(t, false)

		resp := testutil.DoRequest(t, app, http.MethodGet, "/_/install", "", "")
		testutil.AssertStatus(t, resp, http.StatusFound, http.StatusSeeOther)
		if loc := resp.Header.Get("Location"); loc != "/_" {
			t.Errorf("Location = %q, want the admin UI", loc)
		}
	})
}

// subscribeToInterrupt makes the test process survive SIGINT whatever else
// happens, and returns the channel that receives it.
//
// signal.Notify disables the default action for the signal process-wide and
// delivers it to every registered channel, so the interrupt is delivered to
// this channel as well as to the code under test even if the code has not
// registered yet. Without this, a test that sends SIGINT would kill the test
// binary.
func subscribeToInterrupt(t *testing.T) chan os.Signal {
	t.Helper()

	ch := make(chan os.Signal, 4)
	signal.Notify(ch, os.Interrupt)
	t.Cleanup(func() { signal.Stop(ch) })

	return ch
}

func TestHandleShutdown(t *testing.T) {
	setLogger(logging.New())
	subscribeToInterrupt(t)

	app := fiber.New()
	app.Get("/ping", func(c fiber.Ctx) error { return c.SendString("pong") })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	served := make(chan error, 1)
	go func() { served <- app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()

	// Wait for the server to answer before interrupting it: a shutdown of a
	// server that never started would prove nothing.
	url := "http://" + ln.Addr().String() + "/ping"
	waitForServer(t, url)

	connsClosed := make(chan struct{})
	go handleShutdown(context.Background(), app, connsClosed)

	// handleShutdown registers its signal channel from inside that goroutine, so
	// an interrupt that arrives before it gets there is delivered to this test's
	// own subscriber instead and never reaches the code under test. Sending it
	// again until the shutdown is reported closes that window: the delay is a
	// scheduling one, so one send is normally enough, and the test no longer
	// depends on how long it is.
	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
			t.Fatalf("send SIGINT: %v", err)
		}
		select {
		case <-connsClosed:
		case <-time.After(50 * time.Millisecond):
			if time.Now().Before(deadline) {
				continue
			}
			t.Fatal("handleShutdown did not report the shutdown within 5s")
		}
		break
	}

	select {
	case <-served:
	case <-time.After(5 * time.Second):
		t.Fatal("the server did not stop after the interrupt")
	}

	if _, err := http.Get(url); err == nil {
		t.Error("the server still answers after shutdown")
	}
}

func TestStartHTTP(t *testing.T) {
	setLogger(logging.New())
	t.Cleanup(func() { setDevMode(false) })

	t.Run("dev mode reports a failed listen", func(t *testing.T) {
		setDevMode(true)

		// Holding the port makes Listen fail immediately, which is the only
		// way this branch returns without a running server.
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		defer func() { _ = ln.Close() }()

		if err := startHTTP(ln.Addr().String(), fiber.New()); err == nil {
			t.Fatal("expected the address collision to be reported")
		}
	})

	t.Run("serves until interrupted", func(t *testing.T) {
		setDevMode(false)
		subscribeToInterrupt(t)

		// startHTTP picks its own listener, so the address has to be free
		// before it is handed over.
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		addr := ln.Addr().String()
		_ = ln.Close()

		// Small delay to let the kernel release the port. Without this,
		// the Listen call inside startHTTP can race with the Close above
		// and fail with "address already in use" if another process or
		// test claims the port first.
		time.Sleep(50 * time.Millisecond)

		app := fiber.New()
		app.Get("/ping", func(c fiber.Ctx) error { return c.SendString("pong") })

		done := make(chan error, 1)
		go func() { done <- startHTTP(addr, app) }()

		waitForServer(t, "http://"+addr+"/ping")

		// startHTTP's handleShutdown registers its signal channel from inside its
		// own goroutine, so an interrupt that arrives before it gets there is
		// delivered to this test's own subscriber instead and never reaches the
		// code under test. Sending it again until startHTTP returns closes that
		// window: the delay is a scheduling one, so one send is normally enough,
		// and the test no longer depends on how long it is.
		deadline := time.Now().Add(5 * time.Second)
		for {
			if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
				t.Fatalf("send SIGINT: %v", err)
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("startHTTP returned %v, want a clean shutdown", err)
				}
			case <-time.After(50 * time.Millisecond):
				if time.Now().Before(deadline) {
					continue
				}
				t.Fatal("startHTTP did not return after the interrupt")
			}
			break
		}
	})
}

// waitForServer polls url until it answers, failing the test if it never does.
func waitForServer(t *testing.T, url string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no server answered %s within 5s", url)
}

func TestNewApp(t *testing.T) {
	// NewApp creates the directory layout in the working directory, so the
	// test needs one of its own.
	t.Cleanup(testutil.WithCmdTestDir(t))
	setDevMode(false)
	t.Cleanup(func() { setDevMode(false) })

	// Hold a port so the listen fails: this way NewApp runs the whole
	// production path — init, app setup, routes, the banner — and stops at the
	// listener instead of serving until the test times out.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	cfg := database.Config{Driver: database.DriverSQLite, DSN: "./lc_base/data.db"}

	var err2 error
	out := captureStdout(t, func() {
		err2 = NewApp(cfg, ln.Addr().String(), "", true, true)
	})

	if err2 == nil {
		t.Fatal("expected NewApp to report that it could not listen")
	}
	if !strings.Contains(out, "myCart") || !strings.Contains(out, "Admin UI") {
		t.Errorf("NewApp did not print the startup banner:\n%s", out)
	}

	// The first run has to leave a usable installation behind.
	for _, path := range []string{"lc_uploads", "lc_digitals", "lc_base/data.db"} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("NewApp did not create %s: %v", path, err)
		}
	}
}

// startHTTPS exits the process when it cannot listen, so that path is only
// observable in a child process. This test is that child when the environment
// says so, and a no-op otherwise.
func TestStartHTTPSHelperProcess(t *testing.T) {
	if os.Getenv("MYCART_TEST_START_HTTPS_CHILD") == "" {
		t.Skip("only runs as the child process of TestStartHTTPS")
	}

	setLogger(logging.New())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "child could not take a port: %v\n", err)
		os.Exit(3)
	}

	// The port is already held, so tls.Listen must fail — and startHTTPS must
	// exit the process rather than return.
	_ = startHTTPS(fiber.New(), ln.Addr().String(), ln.Addr().String())

	fmt.Fprintln(os.Stderr, "startHTTPS returned instead of exiting")
	os.Exit(4)
}

func TestStartHTTPS(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestStartHTTPSHelperProcess")
	cmd.Env = append(os.Environ(), "MYCART_TEST_START_HTTPS_CHILD=1")

	out, err := cmd.CombinedOutput()

	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("startHTTPS did not exit when it could not listen: err=%v, output: %s", err, out)
	}
	if exit.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1; output: %s", exit.ExitCode(), out)
	}
	if strings.Contains(string(out), "returned instead of exiting") {
		t.Error("startHTTPS returned after a failed listen; it must exit")
	}
}
