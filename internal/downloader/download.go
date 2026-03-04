package downloader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// DownloadFolder downloads a GitHub folder to outputDirName atomically.
func (d *Downloader) DownloadFolder(ctx context.Context, info *GitHubURLInfo, outputDirName string) error {
	if fi, err := os.Stat(outputDirName); err == nil && !fi.IsDir() {
		return fmt.Errorf("output path '%s' exists and is a file; choose a different -o", outputDirName)
	}

	d.logger.Info("collecting file list from GitHub API")
	files, err := d.CollectFiles(ctx, info)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}
	d.logger.Info("file list collected", "count", len(files))

	if len(files) == 0 {
		return fmt.Errorf("no files found at the specified path")
	}

	tempDir := outputDirName + ".tmp"
	if err := os.RemoveAll(tempDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean temp directory: %w", err)
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("could not create temp directory: %w", err)
	}

	basePrefix := info.Path
	if basePrefix != "" && !strings.HasSuffix(basePrefix, "/") {
		basePrefix += "/"
	}

	if err := d.downloadFiles(ctx, files, tempDir, basePrefix); err != nil {
		os.RemoveAll(tempDir)
		return err
	}

	if err := os.RemoveAll(outputDirName); err != nil && !os.IsNotExist(err) {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to remove existing output directory: %w", err)
	}
	if err := os.Rename(tempDir, outputDirName); err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to rename temp directory: %w", err)
	}

	return nil
}

func (d *Downloader) downloadFiles(ctx context.Context, files []GitHubContent, outputDir, basePrefix string) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(d.concurrency)

	total := len(files)
	var completed atomic.Int64

	for _, file := range files {
		f := file
		g.Go(func() error {
			rel := f.Path
			if basePrefix != "" {
				rel = strings.TrimPrefix(rel, basePrefix)
			}
			rel = strings.TrimLeft(rel, "/")

			destPath := filepath.Join(outputDir, rel)
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", rel, err)
			}

			d.logger.Debug("downloading file", "path", rel, "url", f.DownloadURL)

			if err := d.streamFile(ctx, f.DownloadURL, destPath); err != nil {
				return fmt.Errorf("failed to download %s: %w", rel, err)
			}

			n := int(completed.Add(1))
			if d.onProgress != nil {
				d.onProgress(Progress{
					TotalFiles:     total,
					CompletedFiles: n,
					CurrentFile:    rel,
				})
			}
			return nil
		})
	}

	return g.Wait()
}

// streamFile downloads a URL and streams it directly to destPath via a temp file.
func (d *Downloader) streamFile(ctx context.Context, downloadURL, destPath string) error {
	resp, err := d.doWithRetry(ctx, downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tmpPath := destPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()

	if copyErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write file: %w", copyErr)
	}
	if closeErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close file: %w", closeErr)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize file: %w", err)
	}
	return nil
}
