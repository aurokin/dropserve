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
	"dropserve/internal/control"
)

const version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
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
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func runServe() error {
	addr := controlAddrFromEnv()
	store := control.NewStore()
	logger := log.New(os.Stdout, "control ", log.LstdFlags)

	server := &http.Server{
		Addr:              addr,
		Handler:           control.NewServer(store, logger).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	logger.Printf("control api listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("control api failed: %w", err)
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

func usage() {
	fmt.Fprintln(os.Stderr, "DropServe CLI")
	fmt.Fprintln(os.Stderr, "\nUsage:")
	fmt.Fprintln(os.Stderr, "  dropserve open [--minutes N] [--reusable] [--policy overwrite|autorename] [--host HOST]")
	fmt.Fprintln(os.Stderr, "  dropserve serve")
	fmt.Fprintln(os.Stderr, "  dropserve version")
}
