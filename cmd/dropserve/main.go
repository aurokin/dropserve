package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dropserve/internal/cli"
	"dropserve/internal/config"
	"dropserve/internal/control"
	"dropserve/internal/publicapi"
	"dropserve/internal/sweeper"
)

const version = "dev"

func main() {
	if len(os.Args) < 2 {
		if err := cli.RunOpen(nil, os.Stdout, os.Stderr); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	switch os.Args[1] {
	case "open":
		if err := cli.RunOpen(os.Args[2:], os.Stdout, os.Stderr); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "serve":
		if err := runServe(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "version":
		fmt.Fprintln(os.Stdout, version)
	case "help", "-h", "--help", "-help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var port int
	fs.IntVar(&port, "port", 0, "Override server port")
	if err := fs.Parse(args); err != nil {
		return err
	}

	addr := addrFromEnv(port)
	store := control.NewStore()
	publicLogger := log.New(os.Stdout, "public ", log.LstdFlags)

	controlLogger := log.New(os.Stdout, "control ", log.LstdFlags)
	publicHandler := publicapi.NewServer(store, publicLogger).Handler()
	controlHandler := control.NewServer(store, controlLogger).Handler()
	mux := http.NewServeMux()
	mux.Handle("/api/control/", controlHandler)
	mux.Handle("/", publicHandler)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if host, _, err := net.SplitHostPort(addr); err == nil {
		if host == "" || host == "0.0.0.0" || host == "::" {
			publicLogger.Printf("warning: binding to %s; ensure /api/control/* is blocked at the proxy", addr)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sweepLogger := log.New(os.Stdout, "sweep ", log.LstdFlags)
	sweeper := sweeper.New(sweeper.Config{
		TempDirName:      config.TempDirName(),
		SweepInterval:    config.SweepInterval(),
		PartMaxAge:       config.PartMaxAge(),
		PortalIdleMaxAge: config.PortalIdleMaxAge(),
		Roots:            config.SweepRoots(),
	}, store, sweepLogger)
	if err := sweeper.RunOnce(ctx); err != nil {
		sweepLogger.Printf("startup sweep failed: %v", err)
	}
	go sweeper.Run(ctx)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	errCh := make(chan error, 1)
	go func() {
		publicLogger.Printf("server listening on %s", addr)
		errCh <- server.ListenAndServe()
	}()

	err := <-errCh
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

func addrFromEnv(portOverride int) string {
	if portOverride > 0 {
		return fmt.Sprintf("0.0.0.0:%d", portOverride)
	}
	addr := os.Getenv("DROPSERVE_ADDR")
	if addr == "" {
		addr = os.Getenv("DROPSERVE_PUBLIC_ADDR")
	}
	if addr == "" {
		addr = os.Getenv("DROPSERVE_PORT")
		if addr != "" {
			return "0.0.0.0:" + addr
		}
	}
	if addr == "" {
		return "0.0.0.0:8080"
	}
	return addr
}

func usage() {
	fmt.Fprintln(os.Stderr, "DropServe CLI")
	fmt.Fprintln(os.Stderr, "\nUsage:")
	fmt.Fprintln(os.Stderr, "  dropserve (defaults to: open)")
	fmt.Fprintln(os.Stderr, "  dropserve open [--minutes N] [--reusable] [--policy overwrite|autorename] [--host HOST] [--port N]")
	fmt.Fprintln(os.Stderr, "  dropserve serve [--port N]")
	fmt.Fprintln(os.Stderr, "  dropserve version")
}
