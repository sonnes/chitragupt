// Package redact provides a reusable redaction layer for sanitizing secrets
// and PII from core.Transcript values.
package redact

import (
	"fmt"
	"regexp"
)

// Rule detects sensitive data in a string and provides a replacement.
type Rule interface {
	Name() string
	Kind() string
	Detect(s string) []Match
	Replacement(m Match) string
}

// Match represents a detected occurrence within a string.
type Match struct {
	Start int
	End   int
	Value string
}

type regexRule struct {
	name    string
	kind    string
	pattern *regexp.Regexp
}

func (r *regexRule) Name() string { return r.name }
func (r *regexRule) Kind() string { return r.kind }

func (r *regexRule) Detect(s string) []Match {
	locs := r.pattern.FindAllStringIndex(s, -1)
	matches := make([]Match, len(locs))
	for i, loc := range locs {
		matches[i] = Match{Start: loc[0], End: loc[1], Value: s[loc[0]:loc[1]]}
	}
	return matches
}

func (r *regexRule) Replacement(_ Match) string {
	return fmt.Sprintf("[REDACTED:%s]", r.name)
}

// SecretRules returns the built-in secret detection rules.
func SecretRules() []Rule {
	return []Rule{
		&regexRule{
			name:    "aws_key",
			kind:    "secret",
			pattern: regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		},
		&regexRule{
			name:    "api_key",
			kind:    "secret",
			pattern: regexp.MustCompile(`(?:sk-[a-zA-Z0-9]{32,}|ghp_[a-zA-Z0-9]{36,}|gho_[a-zA-Z0-9]{36,}|glpat-[a-zA-Z0-9\-]{20,})`),
		},
		&regexRule{
			name:    "private_key",
			kind:    "secret",
			pattern: regexp.MustCompile(`-----BEGIN [A-Z ]+PRIVATE KEY-----`),
		},
		&regexRule{
			name:    "connection_string",
			kind:    "secret",
			pattern: regexp.MustCompile(`(?:postgres|mongodb|mysql|redis)://[^\s"'` + "`" + `]+`),
		},
		&regexRule{
			name:    "jwt",
			kind:    "secret",
			pattern: regexp.MustCompile(`eyJ[A-Za-z0-9\-_]+\.eyJ[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_.+/=]+`),
		},
	}
}

// PIIRules returns the built-in PII detection rules.
func PIIRules() []Rule {
	return []Rule{
		&regexRule{
			name:    "email",
			kind:    "pii",
			pattern: regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
		},
		&regexRule{
			name:    "ipv4",
			kind:    "pii",
			pattern: regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
		},
		&regexRule{
			name:    "phone",
			kind:    "pii",
			pattern: regexp.MustCompile(`(?:\+\d{1,3}[\s\-]?)?\(?\d{3}\)?[\s\-]?\d{3}[\s\-]?\d{4}`),
		},
	}
}
