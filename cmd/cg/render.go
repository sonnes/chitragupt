package main

import (
	"context"
	"fmt"

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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("render: not implemented")
			return nil
		},
	}
}
