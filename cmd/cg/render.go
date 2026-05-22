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
		Name:      "render",
		Usage:     "Generate transcript output",
		UsageText: "cg render --agent claude (--file PATH | --session ID | --project PATH | --all) [--format FORMAT] [--out DIR]",
		Description: `Generate transcripts from Claude Code session logs.

Pick exactly one input:
  --file PATH       Render one raw session file.
  --session ID      Find and render one saved session.
  --project PATH    Render every session for a project directory.
  --all             Render every discoverable session.

Pick output:
  no --format       Terminal output to stdout.
  --format FORMAT   terminal, html, or markdown.
  --out DIR         Write index.{ext} and agent-{id}.{ext} files.
  --compact         Summarize verbose tool content.

Examples:
  cg render -a claude -f session.jsonl
  cg render -a claude -f session.jsonl --format markdown
  cg render -a claude -f session.jsonl --format html --out transcript
  cg render -a claude --project . --format html --out transcripts
  cg render -a claude --all --format html --format markdown --out transcripts`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "agent",
				Aliases:  []string{"a"},
				Usage:    "Reader to use. Valid value: claude",
				Required: true,
				Category: "Required",
			},
			&cli.StringFlag{
				Name:      "file",
				Aliases:   []string{"f"},
				Usage:     "Render one raw session file at `PATH`",
				Category:  "Input: choose one",
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:     "session",
				Aliases:  []string{"s"},
				Usage:    "Render one saved session by `ID`",
				Category: "Input: choose one",
			},
			&cli.StringFlag{
				Name:     "project",
				Aliases:  []string{"p"},
				Usage:    "Render all sessions for project directory `PATH`",
				Category: "Input: choose one",
			},
			&cli.BoolFlag{
				Name:     "all",
				Usage:    "Render every discoverable session",
				Category: "Input: choose one",
			},
			&cli.StringSliceFlag{
				Name:     "format",
				Aliases:  []string{"fmt"},
				Usage:    "Output `FORMAT`: terminal, html, markdown. Repeat for multiple formats",
				Category: "Output",
			},
			&cli.BoolFlag{
				Name:     "no-redact",
				Usage:    "Disable default redaction of secrets and PII",
				Category: "Privacy",
			},
			&cli.StringSliceFlag{
				Name:     "redact",
				Aliases:  []string{"r"},
				Usage:    "Redact only these `RULES`: secrets, pii. Repeat or comma-separate",
				Category: "Privacy",
			},
			&cli.BoolFlag{
				Name:     "compact",
				Aliases:  []string{"c"},
				Usage:    "Replace verbose tool inputs and results with short summaries",
				Category: "Output",
			},
			&cli.BoolFlag{
				Name:     "strip-thinking",
				Usage:    "Also remove thinking blocks; implies --compact",
				Category: "Output",
			},
			&cli.StringFlag{
				Name:      "out",
				Aliases:   []string{"o"},
				Usage:     "Write files into output directory `DIR` instead of stdout",
				Category:  "Output",
				TakesFile: true,
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

			if cmd.Bool("compact") || cmd.Bool("strip-thinking") {
				cfg := compact.Config{
					StripThinking: cmd.Bool("strip-thinking"),
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

// computeDiffStatsTree computes derived stats for a transcript and all its sub-agents.
func computeDiffStatsTree(t *core.Transcript) {
	t.DiffStats = core.ComputeDiffStats(t)
	t.Stats = core.ComputeSessionStats(t)
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
