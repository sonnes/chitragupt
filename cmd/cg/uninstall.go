package main

import (
	"context"
	"fmt"

	"github.com/sonnes/chitragupt/install"
	"github.com/urfave/cli/v3"
)

func uninstallCmd() *cli.Command {
	return &cli.Command{
		Name:  "uninstall",
		Usage: "Remove hooks and configuration installed by cg install",
		Description: `Removes the Claude Code session hook, post-commit hook, and .gitignore entry.
Transcript data is preserved unless --purge is set.

With --purge, also deletes the output directory and orphan branch.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "out",
				Usage: "Output directory name (must match what was passed to install)",
				Value: ".transcripts",
			},
			&cli.BoolFlag{
				Name:  "purge",
				Usage: "Also delete transcript data and orphan branch",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg := install.UninstallConfig{
				OutDir: cmd.String("out"),
				Purge:  cmd.Bool("purge"),
			}

			if err := install.Uninstall(cfg); err != nil {
				return err
			}

			fmt.Println("Uninstalled successfully.")
			if cfg.Purge {
				fmt.Println("Transcript data and orphan branch have been removed.")
			} else {
				fmt.Printf("Transcript data in %s/ has been preserved.\n", cfg.OutDir)
				fmt.Println("Use --purge to also delete transcript data.")
			}
			return nil
		},
	}
}
