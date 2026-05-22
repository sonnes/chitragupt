package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sonnes/chitragupt/core"
	htmlrender "github.com/sonnes/chitragupt/render/html"
	"github.com/urfave/cli/v3"
)

func serveCmd() *cli.Command {
	return &cli.Command{
		Name:      "serve",
		Usage:     "Browse sessions in a local web UI",
		UsageText: "cg serve --agent AGENT [--project PATH | --all] [--port PORT]",
		Description: `Read saved sessions and render them on demand in a local browser.

Input:
  no input flag     Serve sessions for the current working directory.
  --project PATH    Serve sessions for a project directory.
  --all             Serve every discoverable session.

Examples:
  cg serve --agent claude
  cg serve --agent codex
  cg serve --agent claude --project .
  cg serve --agent claude --all --port 3000`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "agent",
				Aliases:  []string{"a"},
				Usage:    "Reader to use. Valid values: claude, codex",
				Required: true,
				Category: "Required",
			},
			&cli.StringFlag{
				Name:     "project",
				Aliases:  []string{"p"},
				Usage:    "Serve sessions for project directory `PATH`",
				Category: "Input: choose at most one",
			},
			&cli.BoolFlag{
				Name:     "all",
				Usage:    "Serve every discoverable session",
				Category: "Input: choose at most one",
			},
			&cli.IntFlag{
				Name:        "port",
				Usage:       "HTTP port to listen on",
				Value:       8080,
				DefaultText: "8080",
				Category:    "Server",
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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			agentName := cmd.String("agent")
			project := cmd.String("project")
			all := cmd.Bool("all")

			if project != "" && all {
				return fmt.Errorf("--project and --all are mutually exclusive")
			}

			if project == "" && !all {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("get working directory: %w", err)
				}
				project = projectForAgent(agentName, cwd)
			} else if project != "" {
				projectPath, err := filepath.Abs(project)
				if err != nil {
					return fmt.Errorf("resolve project path: %w", err)
				}
				project = projectForAgent(agentName, projectPath)
			}

			a := newApp()

			r, err := a.reader(agentName)
			if err != nil {
				return err
			}

			var transcripts []*core.Transcript
			if all {
				transcripts, err = r.ReadAll()
			} else {
				transcripts, err = r.ReadProject(project)
			}
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

			for _, t := range transcripts {
				computeDiffStatsTree(t)
			}

			sort.Slice(transcripts, func(i, j int) bool {
				return transcripts[i].CreatedAt.After(transcripts[j].CreatedAt)
			})

			byID := make(map[string]*core.Transcript)
			var indexAll func(t *core.Transcript)
			indexAll = func(t *core.Transcript) {
				byID[t.SessionID] = t
				for _, sub := range t.SubAgents {
					indexAll(sub)
				}
			}
			for _, t := range transcripts {
				indexAll(t)
			}

			renderer := htmlrender.New()
			renderer.SubAgentHref = func(agentID string) string {
				return "/session/" + agentID
			}

			entries := make([]core.SessionEntry, len(transcripts))
			for i, t := range transcripts {
				entries[i] = core.NewSessionEntry(t, "/session/"+t.SessionID)
			}

			mux := http.NewServeMux()

			mux.HandleFunc("GET /", func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				if err := renderer.RenderIndex(w, entries); err != nil {
					slog.Error("render index", "error", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			})

			mux.HandleFunc("GET /session/{id}", func(w http.ResponseWriter, req *http.Request) {
				id := req.PathValue("id")
				t, ok := byID[id]
				if !ok {
					http.NotFound(w, req)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				if err := renderer.Render(w, t); err != nil {
					slog.Error("render session", "session_id", id, "error", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			})

			addr := fmt.Sprintf(":%d", cmd.Int("port"))
			slog.Info("serving", "addr", "http://localhost"+addr, "sessions", len(transcripts))
			return http.ListenAndServe(addr, mux)
		},
	}
}

// cwdToProject converts an absolute path to Claude's project directory name.
func cwdToProject(cwd string) string {
	return strings.ReplaceAll(cwd, "/", "-")
}

func projectForAgent(agent, cwd string) string {
	if agent == "claude" {
		return cwdToProject(cwd)
	}
	return cwd
}
