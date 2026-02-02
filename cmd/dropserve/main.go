package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
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
		if err := runServe(); err != nil {
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

func runServe() error {
	controlAddr := controlAddrFromEnv()
	publicAddr := publicAddrFromEnv()
	store := control.NewStore()
	controlLogger := log.New(os.Stdout, "control ", log.LstdFlags)
	publicLogger := log.New(os.Stdout, "public ", log.LstdFlags)

	controlServer := &http.Server{
		Addr:              controlAddr,
		Handler:           control.NewServer(store, controlLogger).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	publicServer := &http.Server{
		Addr:              publicAddr,
		Handler:           publicapi.NewServer(store, publicLogger).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
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
		_ = controlServer.Shutdown(shutdownCtx)
		_ = publicServer.Shutdown(shutdownCtx)
	}()

	errCh := make(chan error, 2)
	go func() {
		controlLogger.Printf("control api listening on %s", controlAddr)
		errCh <- controlServer.ListenAndServe()
	}()
	go func() {
		publicLogger.Printf("public api listening on %s", publicAddr)
		errCh <- publicServer.ListenAndServe()
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server failed: %w", err)
		}
	}

	return nil
}

func controlAddrFromEnv() string {
	addr := os.Getenv("DROPSERVE_CONTROL_ADDR")
	if addr == "" {
		return "127.0.0.1:9090"
	}
	return addr
}

func publicAddrFromEnv() string {
	addr := os.Getenv("DROPSERVE_PUBLIC_ADDR")
	if addr == "" {
		return "127.0.0.1:8080"
	}
	return addr
}

func usage() {
	fmt.Fprintln(os.Stderr, "DropServe CLI")
	fmt.Fprintln(os.Stderr, "\nUsage:")
	fmt.Fprintln(os.Stderr, "  dropserve (defaults to: open)")
	fmt.Fprintln(os.Stderr, "  dropserve open [--minutes N] [--reusable] [--policy overwrite|autorename] [--host HOST]")
	fmt.Fprintln(os.Stderr, "  dropserve serve")
	fmt.Fprintln(os.Stderr, "  dropserve version")
}
