package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sonnes/chitragupt/core"
	"github.com/sonnes/chitragupt/reader"
	"github.com/sonnes/chitragupt/redact"
	htmlrender "github.com/sonnes/chitragupt/render/html"
	"github.com/urfave/cli/v3"
)

func serveCmd() *cli.Command {
	return &cli.Command{
		Name:      "serve",
		Usage:     "Browse sessions in a local web UI",
		UsageText: "cg serve [--agent AGENT] [--project PATH | --all] [--port PORT]",
		Description: `Read saved sessions from supported agents and render them in a local browser.

Input:
  no input flag     Serve sessions for the current working directory.
  --project PATH    Serve sessions for a project directory.
  --all             Serve every discoverable session.

Examples:
  cg serve
  cg serve --project .
  cg serve --all
  cg serve --agent claude --port 3000`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "agent",
				Aliases:     []string{"a"},
				Usage:       "Agent filter. Valid values: all, claude, codex",
				Value:       "all",
				DefaultText: "all",
				Category:    "Filter",
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
			projectPath := cmd.String("project")
			all := cmd.Bool("all")

			projectPath, all, err := resolveServeScope(projectPath, all)
			if err != nil {
				return err
			}

			a := newApp()

			redactor, err := newRedactor(cmd)
			if err != nil {
				return err
			}

			source, err := newServeSource(a, serveOptions{
				agent:       agentName,
				projectPath: projectPath,
				all:         all,
				transformer: redactorTransformer(redactor),
			})
			if err != nil {
				return err
			}

			mux := newServeMux(source)

			addr := fmt.Sprintf(":%d", cmd.Int("port"))
			slog.Info(
				"serving live sessions",
				"addr",
				"http://localhost"+addr,
				"agents",
				strings.Join(source.agentNames(), ","),
			)
			return http.ListenAndServe(addr, mux)
		},
	}
}

func resolveServeScope(projectPath string, all bool) (string, bool, error) {
	if projectPath != "" && all {
		return "", false, fmt.Errorf("--project and --all are mutually exclusive")
	}

	if projectPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", false, fmt.Errorf("get working directory: %w", err)
		}
		return cwd, false, nil
	}

	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", false, fmt.Errorf("resolve project path: %w", err)
	}
	return absProjectPath, false, nil
}

func redactorTransformer(redactor *redact.Redactor) core.Transformer {
	if redactor == nil {
		return nil
	}
	return redactor
}

type serveOptions struct {
	agent       string
	projectPath string
	all         bool
	transformer core.Transformer
}

type serveReader struct {
	agent   string
	reader  reader.Reader
	project string
	all     bool
}

type serveSource struct {
	readers     []serveReader
	transformer core.Transformer
}

type serveSnapshot struct {
	entries []core.SessionEntry
	byKey   map[string]*core.Transcript
}

func newServeSource(a *app, opts serveOptions) (*serveSource, error) {
	agentName := opts.agent
	if agentName == "" {
		agentName = "all"
	}

	var names []string
	if agentName == "all" {
		names = allReaderNames(a)
	} else {
		names = []string{agentName}
	}

	var readers []serveReader
	for _, name := range names {
		r, err := a.reader(name)
		if err != nil {
			return nil, err
		}

		source := serveReader{
			agent:  name,
			reader: r,
			all:    opts.all,
		}
		if !opts.all {
			source.project = projectForAgent(name, opts.projectPath)
		}
		readers = append(readers, source)
	}

	return &serveSource{
		readers:     readers,
		transformer: opts.transformer,
	}, nil
}

func allReaderNames(a *app) []string {
	names := make([]string, 0, len(a.readers))
	for name := range a.readers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *serveSource) agentNames() []string {
	names := make([]string, 0, len(s.readers))
	for _, source := range s.readers {
		names = append(names, source.agent)
	}
	return names
}

func (s *serveSource) snapshot() (*serveSnapshot, error) {
	var transcripts []*core.Transcript

	for _, source := range s.readers {
		next, err := source.read()
		if err != nil {
			if shouldIgnoreServeReadError(err) {
				continue
			}
			return nil, err
		}

		for _, t := range next {
			ensureTranscriptAgent(t, source.agent)
		}
		transcripts = append(transcripts, next...)
	}

	if s.transformer != nil {
		for _, t := range transcripts {
			if err := core.Chain(t, s.transformer); err != nil {
				return nil, fmt.Errorf("redact: %w", err)
			}
		}
	}

	for _, t := range transcripts {
		computeDiffStatsTree(t)
	}

	sort.SliceStable(transcripts, func(i, j int) bool {
		left := transcripts[i]
		right := transcripts[j]

		if !left.CreatedAt.Equal(right.CreatedAt) {
			return left.CreatedAt.After(right.CreatedAt)
		}
		if left.Agent != right.Agent {
			return left.Agent < right.Agent
		}
		return left.SessionID < right.SessionID
	})

	byKey := make(map[string]*core.Transcript)
	for _, t := range transcripts {
		indexServeTranscript(byKey, t)
	}

	entries := make([]core.SessionEntry, len(transcripts))
	for i, t := range transcripts {
		entries[i] = core.NewSessionEntry(t, serveSessionHref(t))
	}

	return &serveSnapshot{
		entries: entries,
		byKey:   byKey,
	}, nil
}

func (s serveReader) read() ([]*core.Transcript, error) {
	if s.all {
		return s.reader.ReadAll()
	}
	return s.reader.ReadProject(s.project)
}

func shouldIgnoreServeReadError(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

func ensureTranscriptAgent(t *core.Transcript, agent string) {
	if t.Agent == "" {
		t.Agent = agent
	}

	for _, sub := range t.SubAgents {
		ensureTranscriptAgent(sub, t.Agent)
	}
}

func indexServeTranscript(byKey map[string]*core.Transcript, t *core.Transcript) {
	byKey[serveSessionKey(t.Agent, t.SessionID)] = t
	for _, sub := range t.SubAgents {
		indexServeTranscript(byKey, sub)
	}
}

func newServeMux(source *serveSource) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, req *http.Request) {
		snapshot, err := source.snapshot()
		if err != nil {
			slog.Error("load sessions", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		renderer := htmlrender.New()
		setServeHTMLHeaders(w)
		if err := renderer.RenderIndex(w, snapshot.entries); err != nil {
			slog.Error("render index", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("GET /session/{agent}/{id}", func(w http.ResponseWriter, req *http.Request) {
		agent := req.PathValue("agent")
		id := req.PathValue("id")
		renderServeSession(w, req, source, agent, id)
	})

	mux.HandleFunc("GET /session/{id}", func(w http.ResponseWriter, req *http.Request) {
		id := req.PathValue("id")

		snapshot, err := source.snapshot()
		if err != nil {
			slog.Error("load sessions", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		t, ok := snapshot.findUniqueSession(id)
		if !ok {
			http.NotFound(w, req)
			return
		}

		renderLoadedServeSession(w, t)
	})

	return mux
}

func renderServeSession(
	w http.ResponseWriter,
	req *http.Request,
	source *serveSource,
	agent string,
	id string,
) {
	snapshot, err := source.snapshot()
	if err != nil {
		slog.Error("load sessions", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	t, ok := snapshot.byKey[serveSessionKey(agent, id)]
	if !ok {
		http.NotFound(w, req)
		return
	}

	renderLoadedServeSession(w, t)
}

func renderLoadedServeSession(w http.ResponseWriter, t *core.Transcript) {
	renderer := htmlrender.New()
	renderer.SubAgentHref = func(agentID string) string {
		for _, sub := range t.SubAgents {
			if sub.SessionID == agentID {
				return serveSessionHref(sub)
			}
		}
		return "/session/" + url.PathEscape(t.Agent) + "/" + url.PathEscape(agentID)
	}

	setServeHTMLHeaders(w)
	if err := renderer.Render(w, t); err != nil {
		slog.Error("render session", "session_id", t.SessionID, "agent", t.Agent, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func setServeHTMLHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
}

func (s *serveSnapshot) findUniqueSession(sessionID string) (*core.Transcript, bool) {
	var found *core.Transcript
	for _, t := range s.byKey {
		if t.SessionID != sessionID {
			continue
		}
		if found != nil {
			return nil, false
		}
		found = t
	}
	return found, found != nil
}

func serveSessionKey(agent, sessionID string) string {
	return agent + "\x00" + sessionID
}

func serveSessionHref(t *core.Transcript) string {
	return "/session/" + url.PathEscape(t.Agent) + "/" + url.PathEscape(t.SessionID)
}

// cwdToProject converts an absolute path to Claude's project directory name.
func cwdToProject(cwd string) string {
	return strings.ReplaceAll(cwd, "/", "-")
}

func projectForAgent(agent, projectPath string) string {
	if agent == "claude" {
		return cwdToProject(projectPath)
	}
	return projectPath
}
