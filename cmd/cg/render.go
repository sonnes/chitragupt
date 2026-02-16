package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sonnes/chitragupt/compact"
	"github.com/sonnes/chitragupt/core"
	"github.com/sonnes/chitragupt/render"
	"github.com/urfave/cli/v3"
)

func renderCmd() *cli.Command {
	return &cli.Command{
		Name:  "render",
		Usage: "Convert a session file to a transcript",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "agent",
				Aliases:  []string{"a"},
				Usage:    "Agent name (claude, codex, opencode, cursor)",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "Path to a session file",
			},
			&cli.StringFlag{
				Name:    "session",
				Aliases: []string{"s"},
				Usage:   "Session ID to convert",
			},
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Project name (converts all sessions in the project)",
			},
			&cli.BoolFlag{
				Name:  "all",
				Usage: "Convert all sessions",
			},
			&cli.StringSliceFlag{
				Name:    "format",
				Aliases: []string{"fmt"},
				Usage:   "Output format(s): html, markdown, json, terminal (repeatable)",
			},
			&cli.BoolFlag{
				Name:  "no-redact",
				Usage: "Disable redaction of secrets and PII",
			},
			&cli.StringSliceFlag{
				Name:    "redact",
				Aliases: []string{"r"},
				Usage:   "Allowlist of rules to redact. Example: --redact=secrets,pii",
			},
			&cli.StringFlag{
				Name:    "compact",
				Aliases: []string{"c"},
				Usage:   "Enable compact mode. Use --compact=no-thinking to also strip thinking blocks",
			},
			&cli.StringFlag{
				Name:    "out",
				Aliases: []string{"o"},
				Usage:   "Output directory (writes index.{ext} + agent-{id}.{ext} for each format)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			a := newApp()

			r, err := a.reader(cmd.String("agent"))
			if err != nil {
				return err
			}

			transcripts, err := readTranscripts(r, cmd)
			if err != nil {
				return err
			}

			redactor, err := newRedactor(cmd)
			if err != nil {
				return err
			}
			if redactor != nil {
				for _, t := range transcripts {
					if err := core.Chain(t, redactor); err != nil {
						return fmt.Errorf("redact: %w", err)
					}
				}
			}

			// Compute diff stats BEFORE compact, which mutates tool input strings.
			// Apply to sub-agents too.
			for _, t := range transcripts {
				computeDiffStatsTree(t)
			}

			if v := cmd.String("compact"); v != "" {
				cfg := compact.Config{}
				if v == "no-thinking" {
					cfg.StripThinking = true
				}
				compactor := compact.New(cfg)
				for _, t := range transcripts {
					if err := core.Chain(t, compactor); err != nil {
						return fmt.Errorf("compact: %w", err)
					}
				}
			}

			formats := cmd.StringSlice("format")
			if len(formats) == 0 {
				formats = []string{"terminal"}
			}

			outDir := cmd.String("out")

			if len(formats) > 1 && outDir == "" {
				return fmt.Errorf("--out is required when specifying multiple formats")
			}

			if outDir == "" {
				rnd, err := a.renderer(formats[0])
				if err != nil {
					return err
				}
				for _, t := range transcripts {
					if err := rnd.Render(os.Stdout, t); err != nil {
						return fmt.Errorf("render: %w", err)
					}
				}
				return nil
			}

			for _, format := range formats {
				rnd, err := a.renderer(format)
				if err != nil {
					return err
				}
				for _, t := range transcripts {
					// Each session gets its own subdirectory,
					// matching the install hook layout: $SESSION_ID/index.{ext}
					dir := outDir
					if len(transcripts) > 1 && t.SessionID != "" {
						dir = filepath.Join(outDir, t.SessionID)
					}
					if err := renderToDir(rnd, t, dir, format); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

// computeDiffStatsTree computes DiffStats for a transcript and all its sub-agents.
func computeDiffStatsTree(t *core.Transcript) {
	t.DiffStats = core.ComputeDiffStats(t)
	for _, sub := range t.SubAgents {
		computeDiffStatsTree(sub)
	}
}

// renderToDir writes the main transcript as index.html and each sub-agent as
// agent-{SessionID}.html in the output directory.
func renderToDir(rnd render.Renderer, t *core.Transcript, outDir, format string) error {
	ext := formatExtension(format)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Write main transcript.
	mainPath := filepath.Join(outDir, "index"+ext)
	f, err := os.Create(mainPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", mainPath, err)
	}
	defer f.Close()

	if err := rnd.Render(f, t); err != nil {
		return fmt.Errorf("render main transcript: %w", err)
	}

	// Write each sub-agent transcript.
	for _, sub := range t.SubAgents {
		safeName := filepath.Base(sub.SessionID)
		subPath := filepath.Join(outDir, "agent-"+safeName+ext)
		if err := renderFile(rnd, sub, subPath); err != nil {
			return err
		}
	}

	return nil
}

// formatExtension maps a format name to its file extension (with leading dot).
func formatExtension(format string) string {
	switch format {
	case "html":
		return ".html"
	case "terminal":
		return ".txt"
	case "markdown":
		return ".md"
	case "json":
		return ".json"
	default:
		return "." + format
	}
}

func renderFile(rnd render.Renderer, t *core.Transcript, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	if err := rnd.Render(f, t); err != nil {
		return fmt.Errorf("render %s: %w", path, err)
	}
	return nil
}
