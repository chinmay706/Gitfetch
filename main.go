package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/chinmay706/gitf/cmd"
	"github.com/joho/godotenv"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	_ = godotenv.Load()

	cmd.SetVersionInfo(version, commit, date)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cmd.Execute(ctx)
}
