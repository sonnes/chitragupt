package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanUserText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "slash command with args",
			in:   "<command-message>git:commit</command-message>\n<command-name>/git:commit</command-name>\n<command-args>everything</command-args>",
			want: "/git:commit everything",
		},
		{
			name: "slash command without args",
			in:   "<command-message>commit</command-message>\n<command-name>/commit</command-name>\n<command-args></command-args>",
			want: "/commit",
		},
		{
			name: "ide_selection stripped",
			in:   `<ide_selection>The user selected the lines 41 to 45 from /path/to/file.go:
some code here
</ide_selection>`,
			want: "",
		},
		{
			name: "ide_opened_file stripped",
			in:   `<ide_opened_file>The user opened the file /path/to/file.go in the IDE. This may or may not be related to the current task.</ide_opened_file>`,
			want: "",
		},
		{
			name: "system-reminder stripped",
			in:   "<system-reminder>\nSome system reminder text\n</system-reminder>",
			want: "",
		},
		{
			name: "plain text unchanged",
			in:   "Fix the bug in the login handler",
			want: "Fix the bug in the login handler",
		},
		{
			name: "mixed tag and text",
			in:   "<ide_opened_file>opened file</ide_opened_file>\nActual user prompt here",
			want: "Actual user prompt here",
		},
		{
			name: "multiple tags stripped",
			in:   "<ide_selection>selected code</ide_selection>\n<system-reminder>reminder</system-reminder>\nDo the thing",
			want: "Do the thing",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
		{
			name: "whitespace only after strip",
			in:   "<ide_opened_file>some content</ide_opened_file>\n  \n",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanUserText(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}
