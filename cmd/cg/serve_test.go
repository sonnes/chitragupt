package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/sonnes/chitragupt/core"
	readerpkg "github.com/sonnes/chitragupt/reader"
	"github.com/sonnes/chitragupt/redact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServeSourceDefaultsToAllAgentsAndMapsProject(t *testing.T) {
	a := &app{
		readers: map[string]func() readerpkg.Reader{
			"codex": func() readerpkg.Reader {
				return &fakeServeReader{}
			},
			"claude": func() readerpkg.Reader {
				return &fakeServeReader{}
			},
		},
	}

	source, err := newServeSource(a, serveOptions{
		projectPath: "/work/chitragupt",
	})
	require.NoError(t, err)

	require.Len(t, source.readers, 2)
	assert.Equal(t, []string{"claude", "codex"}, source.agentNames())
	assert.Equal(t, "-work-chitragupt", source.readers[0].project)
	assert.Equal(t, "/work/chitragupt", source.readers[1].project)
}

func TestResolveServeScopeDefaultsToCurrentProject(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	projectPath, all, err := resolveServeScope("", false)
	require.NoError(t, err)

	assert.False(t, all)
	assert.Equal(t, cwd, projectPath)
}

func TestResolveServeScopeRejectsProjectWithAll(t *testing.T) {
	_, _, err := resolveServeScope(".", true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--project and --all are mutually exclusive")
}

func TestRedactorTransformerKeepsNilInterfaceNil(t *testing.T) {
	var redactor *redact.Redactor

	assert.Nil(t, redactorTransformer(redactor))
}

func TestServeSourceReadsOnEachSnapshot(t *testing.T) {
	createdAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	calls := 0

	reader := &fakeServeReader{
		readProject: func(project string) ([]*core.Transcript, error) {
			calls++
			return []*core.Transcript{
				testServeTranscript("claude", fmt.Sprintf("claude-%d", calls), createdAt),
			}, nil
		},
	}

	source := &serveSource{
		readers: []serveReader{
			{
				agent:   "claude",
				reader:  reader,
				project: "-work-chitragupt",
			},
		},
	}

	first, err := source.snapshot()
	require.NoError(t, err)

	second, err := source.snapshot()
	require.NoError(t, err)

	require.Equal(t, 2, calls)
	require.Len(t, first.entries, 1)
	require.Len(t, second.entries, 1)
	assert.Equal(t, "claude-1", first.entries[0].SessionID)
	assert.Equal(t, "claude-2", second.entries[0].SessionID)
}

func TestServeMuxRereadsIndexOnEachRequest(t *testing.T) {
	createdAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	calls := 0

	source := &serveSource{
		readers: []serveReader{
			{
				agent: "claude",
				reader: &fakeServeReader{
					readProject: func(project string) ([]*core.Transcript, error) {
						calls++
						return []*core.Transcript{
							testServeTranscript("claude", fmt.Sprintf("claude-%d", calls), createdAt),
						}, nil
					},
				},
				project: "-work-chitragupt",
			},
		},
	}

	handler := newServeMux(source)

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, http.StatusOK, second.Code)
	require.Equal(t, 2, calls)
	assert.Equal(t, "no-store", first.Header().Get("Cache-Control"))
	assert.Contains(t, first.Body.String(), "claude claude-1")
	assert.Contains(t, second.Body.String(), "claude claude-2")
}

func TestServeSourceCombinesAgentsWithDistinctHrefs(t *testing.T) {
	createdAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)

	source := &serveSource{
		readers: []serveReader{
			{
				agent: "claude",
				reader: &fakeServeReader{
					readAll: func() ([]*core.Transcript, error) {
						return []*core.Transcript{
							testServeTranscript("claude", "shared", createdAt),
						}, nil
					},
				},
				all: true,
			},
			{
				agent: "codex",
				reader: &fakeServeReader{
					readAll: func() ([]*core.Transcript, error) {
						return []*core.Transcript{
							testServeTranscript("codex", "shared", createdAt.Add(time.Minute)),
						}, nil
					},
				},
				all: true,
			},
		},
	}

	snapshot, err := source.snapshot()
	require.NoError(t, err)

	require.Len(t, snapshot.entries, 2)
	require.Len(t, snapshot.byKey, 2)

	assert.Equal(t, "/session/codex/shared", snapshot.entries[0].Href)
	assert.Equal(t, "/session/claude/shared", snapshot.entries[1].Href)
	assert.NotNil(t, snapshot.byKey[serveSessionKey("claude", "shared")])
	assert.NotNil(t, snapshot.byKey[serveSessionKey("codex", "shared")])
}

func TestServeSourceIgnoresMissingAgentStores(t *testing.T) {
	createdAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)

	source := &serveSource{
		readers: []serveReader{
			{
				agent: "claude",
				reader: &fakeServeReader{
					readProject: func(project string) ([]*core.Transcript, error) {
						return nil, fmt.Errorf("read project directory: %w", os.ErrNotExist)
					},
				},
				project: "-work-chitragupt",
			},
			{
				agent: "codex",
				reader: &fakeServeReader{
					readProject: func(project string) ([]*core.Transcript, error) {
						return []*core.Transcript{
							testServeTranscript("codex", "codex-1", createdAt),
						}, nil
					},
				},
				project: "/work/chitragupt",
			},
		},
	}

	snapshot, err := source.snapshot()
	require.NoError(t, err)

	require.Len(t, snapshot.entries, 1)
	assert.Equal(t, "codex", snapshot.entries[0].Agent)
	assert.Equal(t, "codex-1", snapshot.entries[0].SessionID)
}

type fakeServeReader struct {
	readFile    func(path string) (*core.Transcript, error)
	readSession func(sessionID string) (*core.Transcript, error)
	readProject func(project string) ([]*core.Transcript, error)
	readAll     func() ([]*core.Transcript, error)
}

func (r *fakeServeReader) ReadFile(path string) (*core.Transcript, error) {
	if r.readFile == nil {
		return nil, fmt.Errorf("unexpected ReadFile")
	}
	return r.readFile(path)
}

func (r *fakeServeReader) ReadSession(sessionID string) (*core.Transcript, error) {
	if r.readSession == nil {
		return nil, fmt.Errorf("unexpected ReadSession")
	}
	return r.readSession(sessionID)
}

func (r *fakeServeReader) ReadProject(project string) ([]*core.Transcript, error) {
	if r.readProject == nil {
		return nil, fmt.Errorf("unexpected ReadProject")
	}
	return r.readProject(project)
}

func (r *fakeServeReader) ReadAll() ([]*core.Transcript, error) {
	if r.readAll == nil {
		return nil, fmt.Errorf("unexpected ReadAll")
	}
	return r.readAll()
}

func testServeTranscript(agent, sessionID string, createdAt time.Time) *core.Transcript {
	return &core.Transcript{
		SessionID: sessionID,
		Agent:     agent,
		Title:     agent + " " + sessionID,
		CreatedAt: createdAt,
		Messages: []core.Message{
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{
						Type: core.BlockText,
						Text: "hello",
					},
				},
			},
		},
	}
}
