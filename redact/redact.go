package redact

import (
	"regexp"
	"sort"

	"github.com/sonnes/chitragupt/core"
)

// Config controls which rules the Redactor applies.
type Config struct {
	Secrets    bool
	PII        bool
	ExtraRules []Rule
	Allowlist  []string // regex patterns to skip
}

// Redactor applies redaction rules to all string content in a Transcript.
type Redactor struct {
	rules     []Rule
	allowlist []*regexp.Regexp
}

// New creates a Redactor from the given config.
func New(cfg Config) *Redactor {
	var rules []Rule
	if cfg.Secrets {
		rules = append(rules, SecretRules()...)
	}
	if cfg.PII {
		rules = append(rules, PIIRules()...)
	}
	rules = append(rules, cfg.ExtraRules...)

	allowlist := make([]*regexp.Regexp, 0, len(cfg.Allowlist))
	for _, pattern := range cfg.Allowlist {
		if re, err := regexp.Compile(pattern); err == nil {
			allowlist = append(allowlist, re)
		}
	}

	return &Redactor{rules: rules, allowlist: allowlist}
}

func (r *Redactor) Transform(t *core.Transcript) error {
	for i := range t.Messages {
		for j := range t.Messages[i].Content {
			r.redactBlock(&t.Messages[i].Content[j])
		}
	}
	return nil
}

func (r *Redactor) redactBlock(b *core.ContentBlock) {
	switch b.Type {
	case core.BlockText, core.BlockThinking:
		b.Text = r.redactString(b.Text)
	case core.BlockToolUse:
		b.Input = walkAny(b.Input, r.redactString)
	case core.BlockToolResult:
		b.Content = r.redactString(b.Content)
	}
}

// redactString applies all rules to s. Overlapping matches resolve to
// earliest start, then longest. Allowlisted values are skipped.
func (r *Redactor) redactString(s string) string {
	if len(s) == 0 {
		return s
	}

	type replacement struct {
		start int
		end   int
		text  string
	}

	var reps []replacement
	for _, rule := range r.rules {
		for _, m := range rule.Detect(s) {
			if r.isAllowed(m.Value) {
				continue
			}
			reps = append(reps, replacement{
				start: m.Start,
				end:   m.End,
				text:  rule.Replacement(m),
			})
		}
	}

	if len(reps) == 0 {
		return s
	}

	// Sort by start position, then longest match first for ties.
	sort.Slice(reps, func(i, j int) bool {
		if reps[i].start != reps[j].start {
			return reps[i].start < reps[j].start
		}
		return reps[i].end > reps[j].end
	})

	// Apply non-overlapping replacements.
	var result []byte
	pos := 0
	for _, rep := range reps {
		if rep.start < pos {
			continue // overlaps with a previous replacement
		}
		result = append(result, s[pos:rep.start]...)
		result = append(result, rep.text...)
		pos = rep.end
	}
	result = append(result, s[pos:]...)
	return string(result)
}

func (r *Redactor) isAllowed(value string) bool {
	for _, re := range r.allowlist {
		if re.MatchString(value) {
			return true
		}
	}
	return false
}
