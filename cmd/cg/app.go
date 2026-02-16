package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sonnes/chitragupt/core"
	"github.com/sonnes/chitragupt/reader"
	"github.com/sonnes/chitragupt/reader/claude"
	"github.com/sonnes/chitragupt/redact"
	"github.com/sonnes/chitragupt/render"
	htmlrender "github.com/sonnes/chitragupt/render/html"
	"github.com/sonnes/chitragupt/render/terminal"
	"github.com/urfave/cli/v3"
)

// app holds reader and renderer registries used by CLI commands.
type app struct {
	readers   map[string]func() reader.Reader
	renderers map[string]func() render.Renderer
}

func newApp() *app {
	return &app{
		readers: map[string]func() reader.Reader{
			"claude": func() reader.Reader { return &claude.Reader{} },
		},
		renderers: map[string]func() render.Renderer{
			"terminal": func() render.Renderer { return terminal.New() },
			"html":     func() render.Renderer { return htmlrender.New() },
		},
	}
}

func (a *app) reader(name string) (reader.Reader, error) {
	fn, ok := a.readers[name]
	if !ok {
		return nil, fmt.Errorf("unknown agent %q", name)
	}
	return fn(), nil
}

func (a *app) renderer(name string) (render.Renderer, error) {
	fn, ok := a.renderers[name]
	if !ok {
		return nil, fmt.Errorf("unknown output format %q", name)
	}
	return fn(), nil
}

// readTranscripts dispatches to the appropriate Reader method based on CLI flags.
// Exactly one of --file, --session, --project, or --all must be set.
func readTranscripts(r reader.Reader, cmd *cli.Command) ([]*core.Transcript, error) {
	file := cmd.String("file")
	session := cmd.String("session")
	project := cmd.String("project")
	all := cmd.Bool("all")

	n := 0
	if file != "" {
		n++
	}
	if session != "" {
		n++
	}
	if project != "" {
		n++
	}
	if all {
		n++
	}

	if n == 0 {
		return nil, fmt.Errorf("one of --file, --session, --project, or --all is required")
	}
	if n > 1 {
		return nil, fmt.Errorf("only one of --file, --session, --project, or --all may be specified")
	}

	switch {
	case file != "":
		t, err := r.ReadFile(file)
		if err != nil {
			return nil, err
		}
		return []*core.Transcript{t}, nil
	case session != "":
		t, err := r.ReadSession(session)
		if err != nil {
			return nil, err
		}
		return []*core.Transcript{t}, nil
	case project != "":
		project, err := filepath.Abs(project)
		if err != nil {
			return nil, err
		}

		project = strings.ReplaceAll(project, "/", "-")
		return r.ReadProject(project)
	default:
		return r.ReadAll()
	}
}

// newRedactor builds a Redactor from CLI flags. Returns nil when --no-redact is set.
func newRedactor(cmd *cli.Command) (*redact.Redactor, error) {
	if cmd.Bool("no-redact") {
		return nil, nil
	}

	cfg := redact.Config{}
	rules := cmd.StringSlice("redact")

	if len(rules) == 0 {
		cfg.Secrets = true
		cfg.PII = true
	} else {
		for _, r := range rules {
			switch r {
			case "secrets":
				cfg.Secrets = true
			case "pii":
				cfg.PII = true
			default:
				return nil, fmt.Errorf("unknown redaction rule %q", r)
			}
		}
	}

	return redact.New(cfg), nil
}
