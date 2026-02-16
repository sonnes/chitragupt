package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/sonnes/chitragupt/core"
	htmlrender "github.com/sonnes/chitragupt/render/html"
	"github.com/urfave/cli/v3"
)

func serveCmd() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Serve sessions for browsing in a local web UI",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "agent",
				Aliases:  []string{"a"},
				Usage:    "Agent name (claude, codex, opencode, cursor)",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "Project name (serve all sessions in the project)",
			},
			&cli.BoolFlag{
				Name:  "all",
				Usage: "Serve all sessions",
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "Port to listen on",
				Value: 8080,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			project := cmd.String("project")
			all := cmd.Bool("all")

			if project != "" && all {
				return fmt.Errorf("--project and --all are mutually exclusive")
			}

			// Default to cwd-based project when neither flag is set.
			if project == "" && !all {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("get working directory: %w", err)
				}
				project = cwdToProject(cwd)
			}

			a := newApp()

			r, err := a.reader(cmd.String("agent"))
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

			// Build lookup map for all transcripts (including sub-agents).
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

			mux := http.NewServeMux()

			entries := make([]core.ManifestEntry, len(transcripts))
			for i, t := range transcripts {
				entries[i] = core.NewManifestEntry(t, "/session/"+t.SessionID)
			}

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
// Claude uses the path with "/" replaced by "-", e.g. "/Users/foo/bar" → "-Users-foo-bar".
func cwdToProject(cwd string) string {
	return strings.ReplaceAll(cwd, "/", "-")
}
