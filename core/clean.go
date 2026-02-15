package core

import (
	"regexp"
	"strings"
)

// commandNameRE extracts the slash command name from <command-name>/foo</command-name>.
var commandNameRE = regexp.MustCompile(`<command-name>(/[^<]+)</command-name>`)

// commandArgsRE extracts arguments from <command-args>...</command-args>.
var commandArgsRE = regexp.MustCompile(`<command-args>([^<]*)</command-args>`)

// openTagRE matches an XML opening tag like <tag-name> or <tag_name attr="val">.
var openTagRE = regexp.MustCompile(`<([a-zA-Z_][a-zA-Z0-9_-]*)[^>]*>`)

// CleanUserText strips system-injected XML from user text for rendering.
//
// Slash commands (containing <command-name>) are shortened to "/name args".
// All other XML block elements are removed entirely (tag + content).
func CleanUserText(s string) string {
	// Slash commands: extract /name and optional args.
	if m := commandNameRE.FindStringSubmatch(s); m != nil {
		name := m[1]
		if a := commandArgsRE.FindStringSubmatch(s); a != nil && strings.TrimSpace(a[1]) != "" {
			return name + " " + strings.TrimSpace(a[1])
		}
		return name
	}

	// Strip all <tag>…</tag> blocks by finding opening tags and their
	// matching closing tags. Go regexp doesn't support backreferences,
	// so we walk matches manually.
	for {
		loc := openTagRE.FindStringSubmatchIndex(s)
		if loc == nil {
			break
		}
		tagName := s[loc[2]:loc[3]]
		closeTag := "</" + tagName + ">"
		closeIdx := strings.Index(s[loc[1]:], closeTag)
		if closeIdx < 0 {
			// No matching close tag — strip just the open tag.
			s = s[:loc[0]] + s[loc[1]:]
			continue
		}
		// Remove from open tag start through end of close tag.
		end := loc[1] + closeIdx + len(closeTag)
		s = s[:loc[0]] + s[end:]
	}

	return strings.TrimSpace(s)
}
