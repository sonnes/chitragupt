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
		Usage: "Set up git infrastructure for storing agent session transcripts",
		Description: `Creates an orphan branch and git worktree to store agent session
transcripts alongside your repository. Installs hooks to automatically
capture sessions and commit them when you commit code.

After install, session transcripts are saved to .transcripts/<agent>/
on a separate branch that does not pollute your main history.`,
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
				Name:  "branch",
				Usage: "Branch name for transcripts",
				Value: "transcripts",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg := install.Config{
				Agent:   cmd.String("agent"),
				Formats: cmd.StringSlice("format"),
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
			fmt.Printf("  Branch:    %s (orphan)\n", cfg.Branch)
			fmt.Printf("  Worktree:  .transcripts/\n")
			fmt.Printf("  Agent:     %s\n", cfg.Agent)
			fmt.Println()
			fmt.Println("Sessions will be saved to .transcripts/claude/ when a session ends.")
			fmt.Println("Transcripts are auto-committed when you run git commit.")
			return nil
		},
	}
}
