package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chinmay706/gitf/internal/cache"
	"github.com/chinmay706/gitf/internal/downloader"
	"github.com/chinmay706/gitf/internal/server"
	"github.com/spf13/cobra"
)

var (
	servePort       int
	serveCORSOrigin string
	serveNoCache    bool
	serveCacheDir   string
	serveVerbose    bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the gitf HTTP API server",
	Long: `Starts an HTTP server that exposes the gitf downloader as a REST API.

Endpoints:
  GET  /api/v1/health           Health check
  GET  /api/v1/preview?url=...  List files in a GitHub folder
  POST /api/v1/download         Download a GitHub folder as a ZIP

Example:
  gitf serve
  gitf serve --port 9090 --cors-origin http://localhost:8505`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		logger := buildLogger(serveVerbose)

		dlOpts := []downloader.Option{
			downloader.WithVerifySHA(true),
			downloader.WithLogger(logger),
		}

		if !serveNoCache {
			dir := serveCacheDir
			if dir == "" {
				home, _ := os.UserHomeDir()
				dir = filepath.Join(home, ".gitf", "cache")
			}
			c, err := cache.New(dir)
			if err != nil {
				logger.Warn("could not create cache, proceeding without", "error", err)
			} else {
				dlOpts = append(dlOpts, downloader.WithCache(c))
			}
		}

		dl := downloader.New(dlOpts...)

		srv := server.New(dl,
			server.WithPort(servePort),
			server.WithCORSOrigin(serveCORSOrigin),
			server.WithLogger(logger),
		)

		fmt.Printf("gitf server listening on :%d\n", servePort)
		return srv.ListenAndServe(ctx)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVar(&servePort, "port", 8080, "Port to listen on")
	serveCmd.Flags().StringVar(&serveCORSOrigin, "cors-origin", "*", "Allowed CORS origin")
	serveCmd.Flags().BoolVar(&serveNoCache, "no-cache", false, "Disable ETag-based response caching")
	serveCmd.Flags().StringVar(&serveCacheDir, "cache-dir", "", "Cache directory (default ~/.gitf/cache)")
	serveCmd.Flags().BoolVarP(&serveVerbose, "verbose", "v", false, "Enable debug-level logging")
}
