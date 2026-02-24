package main

import (
	"context"
	"fmt"

	"github.com/sonnes/chitragupt/install"
	"github.com/urfave/cli/v3"
)

func installCmd() *cli.Command {
	return &cli.Command{
		Name:  "install",
		Usage: "Set up hooks for storing agent session transcripts",
		Description: `Installs hooks to automatically capture agent session transcripts.

Without --branch, transcripts are saved to a plain directory (default .transcripts/).
With --branch, an orphan branch and git worktree are created, and a post-commit
hook auto-commits transcripts when you commit code.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "agent",
				Usage:   "Agent name (claude)",
				Value:   "claude",
				Aliases: []string{"a"},
			},
			&cli.StringSliceFlag{
				Name:    "format",
				Aliases: []string{"fmt"},
				Usage:   "Output format(s): html, json, markdown (repeatable)",
				Value:   []string{"html"},
			},
			&cli.StringFlag{
				Name:  "out",
				Usage: "Output directory name (relative to repo root)",
				Value: ".transcripts",
			},
			&cli.StringFlag{
				Name:  "branch",
				Usage: "Git branch for transcripts (enables orphan branch + worktree mode)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg := install.Config{
				Agent:   cmd.String("agent"),
				Formats: cmd.StringSlice("format"),
				OutDir:  cmd.String("out"),
				Branch:  cmd.String("branch"),
			}

			if cfg.Agent != "claude" {
				return fmt.Errorf("unsupported agent %q; currently only 'claude' is supported", cfg.Agent)
			}

			if err := install.Run(cfg); err != nil {
				return err
			}

			fmt.Println("Installed successfully.")
			fmt.Println()
			if cfg.Branch != "" {
				fmt.Printf("  Branch:    %s (orphan)\n", cfg.Branch)
				fmt.Printf("  Worktree:  %s/\n", cfg.OutDir)
			} else {
				fmt.Printf("  Output:    %s/\n", cfg.OutDir)
			}
			fmt.Printf("  Agent:     %s\n", cfg.Agent)
			fmt.Println()
			fmt.Printf("Sessions will be saved to %s/ when a session ends.\n", cfg.OutDir)
			if cfg.Branch != "" {
				fmt.Println("Transcripts are auto-committed when you run git commit.")
			}
			return nil
		},
	}
}
