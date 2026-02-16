package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sonnes/chitragupt/core"
	"github.com/sonnes/chitragupt/manifest"
	"github.com/sonnes/chitragupt/reader"
	"github.com/urfave/cli/v3"
)

func manifestCmd() *cli.Command {
	return &cli.Command{
		Name:  "manifest",
		Usage: "Manage the session manifest",
		Commands: []*cli.Command{
			manifestUpsertCmd(),
			manifestRepairCmd(),
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

func manifestRepairCmd() *cli.Command {
	return &cli.Command{
		Name:  "repair",
		Usage: "Rebuild manifest.json by scanning the transcripts directory",
		Description: `Scans the transcripts directory for session directories, re-parses their
raw source files, and rebuilds the manifest from scratch.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "dir",
				Aliases:  []string{"d"},
				Usage:    "Path to the .transcripts/ directory",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "agent",
				Aliases: []string{"a"},
				Usage:   "Agent name, selects the reader for raw sources",
				Value:   "claude",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			a := newApp()

			r, err := a.reader(cmd.String("agent"))
			if err != nil {
				return err
			}

			dir := cmd.String("dir")
			m, skipped, err := repairManifest(dir, r)
			if err != nil {
				return err
			}

			manifestPath := filepath.Join(dir, "manifest.json")
			if err := m.WriteFile(manifestPath); err != nil {
				return fmt.Errorf("write manifest: %w", err)
			}

			fmt.Printf("Repaired manifest: %d entries (%d skipped)\n", len(m.Entries), skipped)
			return nil
		},
	}
}

// repairManifest scans dir for session directories, re-parses raw sources via
// the reader, and builds a new manifest. Returns the manifest, count of skipped
// sessions, and any fatal error.
func repairManifest(dir string, r reader.Reader) (*manifest.Manifest, int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, fmt.Errorf("read transcripts directory: %w", err)
	}

	m := &manifest.Manifest{}
	skipped := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == ".git" {
			continue
		}

		sessionID := name
		sessionDir := filepath.Join(dir, name)

		href := detectSessionHref(sessionID, sessionDir)
		if href == "" {
			skipped++
			continue
		}

		t, err := r.ReadSession(sessionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skip %s: %v\n", sessionID, err)
			skipped++
			continue
		}

		computeDiffStatsTree(t)
		me := core.NewManifestEntry(t, href)
		m.Upsert(me)
	}

	return m, skipped, nil
}

// detectSessionHref checks for index files in priority order and returns
// the relative href (e.g. "{sessionID}/index.html"), or empty string if none found.
func detectSessionHref(sessionID, sessionDir string) string {
	for _, ext := range []string{".html", ".jsonl", ".json", ".md"} {
		path := filepath.Join(sessionDir, "index"+ext)
		if _, err := os.Stat(path); err == nil {
			return sessionID + "/index" + ext
		}
	}
	return ""
}
