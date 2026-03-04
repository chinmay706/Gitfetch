package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/chinmay706/gitf/internal/downloader"
	"github.com/spf13/cobra"
)

var (
	outputFileName string
	concurrency    int
	timeout        time.Duration
	verbose        bool
	dryRun         bool
)

var downloadCmd = &cobra.Command{
	Use:   "download [github-folder-url]",
	Short: "Downloads the folder from the provided GitHub URL.",
	Long: `Takes a full GitHub URL to a folder, downloads its contents recursively,
and extracts them into a clean output directory (without the repo root folder).

Example:
  gitf download https://github.com/spf13/cobra/tree/main/docs
  gitf download https://github.com/spf13/cobra/tree/main/docs -o cobra-docs
  gitf download https://github.com/spf13/cobra/tree/main/docs --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		githubURL := args[0]

		fmt.Println("gitf - GitHub Folder Downloader")
		fmt.Println("=================================")
		fmt.Println()

		fmt.Print("Parsing GitHub URL... ")
		urlInfo, err := downloader.ParseGithubURL(githubURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError: Invalid URL. %v\n", err)
			return err
		}
		fmt.Println("OK")

		fmt.Println("Repository Information:")
		fmt.Printf("  Owner:      %s\n", urlInfo.Owner)
		fmt.Printf("  Repository: %s\n", urlInfo.Repo)
		fmt.Printf("  Branch:     %s\n", urlInfo.Branch)
		fmt.Printf("  Path:       %s\n", urlInfo.Path)
		fmt.Println()

		logger := buildLogger(verbose)

		dl := downloader.New(
			downloader.WithConcurrency(concurrency),
			downloader.WithLogger(logger),
			downloader.WithProgress(func(p downloader.Progress) {
				fmt.Printf("\r  [%d/%d] %s", p.CompletedFiles, p.TotalFiles, p.CurrentFile)
			}),
		)

		if dryRun {
			return runDryRun(ctx, dl, urlInfo)
		}

		fmt.Printf("Downloading files to '%s'...\n", outputFileName)

		start := time.Now()
		if err := dl.DownloadFolder(ctx, urlInfo, outputFileName); err != nil {
			fmt.Fprintf(os.Stderr, "\n  Download failed: %v\n", err)
			return err
		}
		elapsed := time.Since(start)

		fmt.Printf("\r  Download completed successfully!          \n")
		fmt.Println()
		fmt.Printf("Output directory: %s\n", outputFileName)
		fmt.Printf("Completed in %s\n", elapsed.Round(time.Millisecond))
		fmt.Println("\nAll done! Your GitHub folder has been downloaded and extracted.")
		return nil
	},
}

func runDryRun(ctx context.Context, dl *downloader.Downloader, info *downloader.GitHubURLInfo) error {
	fmt.Println("[dry-run] Listing files (no download)...")
	fmt.Println()

	files, err := dl.CollectFiles(ctx, info)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
		fmt.Printf("  %s  (%s)\n", f.Path, humanSize(f.Size))
	}
	fmt.Println()
	fmt.Printf("Total: %d files, %s\n", len(files), humanSize(totalSize))
	return nil
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func buildLogger(verbose bool) *slog.Logger {
	if verbose {
		return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func init() {
	rootCmd.AddCommand(downloadCmd)

	downloadCmd.Flags().StringVarP(&outputFileName, "output", "o", "download", "Name of the output directory")
	downloadCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "Max parallel downloads")
	downloadCmd.Flags().DurationVarP(&timeout, "timeout", "t", 5*time.Minute, "Overall operation timeout")
	downloadCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug-level logging")
	downloadCmd.Flags().BoolVar(&dryRun, "dry-run", false, "List files without downloading")
}
