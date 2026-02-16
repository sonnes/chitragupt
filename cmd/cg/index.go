package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sonnes/chitragupt/manifest"
	htmlrender "github.com/sonnes/chitragupt/render/html"
	"github.com/urfave/cli/v3"
)

func indexCmd() *cli.Command {
	return &cli.Command{
		Name:  "index",
		Usage: "Generate an index page from the manifest",
		Description: `Reads manifest.json from the given directory and writes index.html
alongside it. Typically called from the post-commit hook to regenerate
the session listing after new transcripts are committed.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "dir",
				Aliases:  []string{"d"},
				Usage:    "Directory containing manifest.json (writes index.html there)",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dir := cmd.String("dir")

			m, err := manifest.ReadFile(filepath.Join(dir, "manifest.json"))
			if err != nil {
				return fmt.Errorf("read manifest: %w", err)
			}

			if len(m.Entries) == 0 {
				return nil
			}

			outPath := filepath.Join(dir, "index.html")
			f, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("create %s: %w", outPath, err)
			}
			defer f.Close()

			renderer := htmlrender.New()
			return renderer.RenderIndex(f, m.Entries)
		},
	}
}
