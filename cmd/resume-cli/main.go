package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"resume-cli/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	cmd := cli.New(os.Stdout, os.Stderr, os.Getenv)
	if e := cmd.ExecuteContext(ctx); e != nil {
		fmt.Fprintln(os.Stderr, "resume-cli:", e)
		os.Exit(1)
	}
}
