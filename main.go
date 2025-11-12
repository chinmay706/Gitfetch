package main

import (
	"github.com/DibyashaktiMoharana/gitf/cmd"
	"github.com/joho/godotenv"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Load .env file if present (silently ignore if missing)
	_ = godotenv.Load()
	
	cmd.Execute()
}
