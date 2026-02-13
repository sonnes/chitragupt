package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sonnes/chitragupt/core"
	"github.com/urfave/cli/v3"
)

func renderCmd() *cli.Command {
	return &cli.Command{
		Name:  "render",
		Usage: "Convert a session file to a transcript",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "agent",
				Usage:    "Agent name (claude, codex, opencode, cursor)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "file",
				Usage: "Path to a session file",
			},
			&cli.StringFlag{
				Name:  "session",
				Usage: "Session ID to convert",
			},
			&cli.StringFlag{
				Name:  "project",
				Usage: "Project name (converts all sessions in the project)",
			},
			&cli.BoolFlag{
				Name:  "all",
				Usage: "Convert all sessions",
			},
			&cli.StringFlag{
				Name:  "o",
				Usage: "Output format: html, markdown, json, terminal",
				Value: "terminal",
			},
			&cli.BoolFlag{
				Name:  "no-redact",
				Usage: "Disable redaction of secrets and PII",
			},
			&cli.StringSliceFlag{
				Name:  "redact",
				Usage: "Allowlist of rules to redact. Example: --redact=secrets,pii",
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

			rnd, err := a.renderer(cmd.String("o"))
			if err != nil {
				return err
			}

			for _, t := range transcripts {
				if err := rnd.Render(os.Stdout, t); err != nil {
					return fmt.Errorf("render: %w", err)
				}
			}

			return nil
		},
	}
}
