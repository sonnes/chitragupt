package main

import (
	"context"
	"fmt"

	"github.com/sonnes/chitragupt/core"
	"github.com/sonnes/chitragupt/manifest"
	"github.com/urfave/cli/v3"
)

func manifestCmd() *cli.Command {
	return &cli.Command{
		Name:  "manifest",
		Usage: "Manage the session manifest",
		Commands: []*cli.Command{
			manifestUpsertCmd(),
		},
	}
}

func manifestUpsertCmd() *cli.Command {
	return &cli.Command{
		Name:  "upsert",
		Usage: "Add or update a session entry in the manifest",
		Description: `Parses a raw session file, extracts metadata, and upserts the entry
into the manifest file. Called by the SessionEnd hook after rendering.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "agent",
				Aliases:  []string{"a"},
				Usage:    "Agent name (claude, codex, opencode, cursor)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "file",
				Aliases:  []string{"f"},
				Usage:    "Path to the raw session file",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "manifest",
				Aliases:  []string{"m"},
				Usage:    "Path to manifest.json",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "href",
				Usage:    "Relative link to the rendered transcript page",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			a := newApp()

			r, err := a.reader(cmd.String("agent"))
			if err != nil {
				return err
			}

			t, err := r.ReadFile(cmd.String("file"))
			if err != nil {
				return fmt.Errorf("read session: %w", err)
			}

			computeDiffStatsTree(t)

			entry := core.NewManifestEntry(t, cmd.String("href"))

			m, err := manifest.ReadFile(cmd.String("manifest"))
			if err != nil {
				return fmt.Errorf("read manifest: %w", err)
			}

			m.Upsert(entry)

			if err := m.WriteFile(cmd.String("manifest")); err != nil {
				return fmt.Errorf("write manifest: %w", err)
			}

			return nil
		},
	}
}
