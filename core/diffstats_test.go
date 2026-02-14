package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeDiffStats(t *testing.T) {
	tests := []struct {
		name string
		msgs []Message
		want *DiffStats
	}{
		{
			name: "write adds lines",
			msgs: []Message{{
				Role: RoleAssistant,
				Content: []ContentBlock{{
					Type: BlockToolUse, Name: "Write",
					Input: map[string]any{
						"file_path": "/tmp/foo.go",
						"content":   "line1\nline2\nline3\n",
					},
				}},
			}},
			want: &DiffStats{Added: 3, Removed: 0, Changed: 1},
		},
		{
			name: "edit adds and removes",
			msgs: []Message{{
				Role: RoleAssistant,
				Content: []ContentBlock{{
					Type: BlockToolUse, Name: "Edit",
					Input: map[string]any{
						"file_path":  "/tmp/bar.go",
						"old_string": "old1\nold2\n",
						"new_string": "new1\nnew2\nnew3\n",
					},
				}},
			}},
			want: &DiffStats{Added: 3, Removed: 2, Changed: 1},
		},
		{
			name: "multiple files counted",
			msgs: []Message{{
				Role: RoleAssistant,
				Content: []ContentBlock{
					{Type: BlockToolUse, Name: "Write", Input: map[string]any{"file_path": "/a.go", "content": "x\n"}},
					{Type: BlockToolUse, Name: "Edit", Input: map[string]any{"file_path": "/b.go", "old_string": "y\n", "new_string": "z\n"}},
				},
			}},
			want: &DiffStats{Added: 2, Removed: 1, Changed: 2},
		},
		{
			name: "same file counted once",
			msgs: []Message{{
				Role: RoleAssistant,
				Content: []ContentBlock{
					{Type: BlockToolUse, Name: "Write", Input: map[string]any{"file_path": "/a.go", "content": "x\n"}},
					{Type: BlockToolUse, Name: "Edit", Input: map[string]any{"file_path": "/a.go", "old_string": "y\n", "new_string": "z\n"}},
				},
			}},
			want: &DiffStats{Added: 2, Removed: 1, Changed: 1},
		},
		{
			name: "no edits returns nil",
			msgs: []Message{{
				Role: RoleAssistant,
				Content: []ContentBlock{
					{Type: BlockToolUse, Name: "Bash", Input: map[string]any{"command": "ls"}},
				},
			}},
			want: nil,
		},
		{
			name: "nil input handled",
			msgs: []Message{{
				Role: RoleAssistant,
				Content: []ContentBlock{
					{Type: BlockToolUse, Name: "Write", Input: nil},
				},
			}},
			want: nil,
		},
		{
			name: "case insensitive tool names",
			msgs: []Message{{
				Role: RoleAssistant,
				Content: []ContentBlock{{
					Type: BlockToolUse, Name: "write",
					Input: map[string]any{"file_path": "/x.go", "content": "a\nb\n"},
				}},
			}},
			want: &DiffStats{Added: 2, Removed: 0, Changed: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Transcript{
				SessionID: "test",
				Agent:     "claude",
				CreatedAt: time.Now(),
				Messages:  tt.msgs,
			}
			got := ComputeDiffStats(tr)
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.want.Added, got.Added)
				assert.Equal(t, tt.want.Removed, got.Removed)
				assert.Equal(t, tt.want.Changed, got.Changed)
			}
		})
	}
}

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{"just now", 30 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "5m ago"},
		{"hours", 3 * time.Hour, "3h ago"},
		{"days", 2 * 24 * time.Hour, "2d ago"},
		{"weeks", 14 * 24 * time.Hour, "2w ago"},
		{"months", 60 * 24 * time.Hour, "2mo ago"},
		{"years", 400 * 24 * time.Hour, "1y ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := time.Now().Add(-tt.ago)
			assert.Equal(t, tt.want, RelativeTime(ts))
		})
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hello", 1},
		{"hello\n", 1},
		{"a\nb\n", 2},
		{"a\nb\nc", 3},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, countLines(tt.input), "countLines(%q)", tt.input)
	}
}
