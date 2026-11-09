package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/grimange/almadial/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// Closing stdin wakes a blocked terminal read; only wizard consumes stdin.
	wake := context.AfterFunc(ctx, func() { _ = os.Stdin.Close() })
	code := cli.RunContext(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	wake()
	stop()
	os.Exit(code)
}
