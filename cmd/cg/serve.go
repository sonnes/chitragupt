package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func serveCmd() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Serve sessions for browsing in a local web UI",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "agent",
				Usage:    "Agent name (claude, codex, opencode, cursor)",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "all",
				Usage: "Serve all sessions",
			},
			&cli.StringFlag{
				Name:  "dir",
				Usage: "Directory of sessions to serve",
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "Port to listen on",
				Value: 8080,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("serve: not implemented")
			return nil
		},
	}
}
