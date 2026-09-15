package observability

import (
	"regexp"
	"strings"
)

// maxMessageRunes caps one log line so a stack trace cannot flood the agent context.
const maxMessageRunes = 2000

// redactions run in order over every message and context before a log line leaves
// this process. Logs are the one place where store PII and credentials leak by
// accident, so this repeats the store-side strip instead of trusting it.
var redactions = []struct {
	pattern *regexp.Regexp
	replace string
}{
	{regexp.MustCompile(`(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`), "[redacted:email]"},
	{regexp.MustCompile(`(?i)\b(authorization|cookie|set-cookie|x-wp-nonce)\b\s*[:=]\s*(?:bearer\s+|basic\s+)?\S+`), "$1: [redacted]"},
	{regexp.MustCompile(`(?i)\b(bearer|basic)\s+[a-z0-9._\-/+=]{8,}`), "$1 [redacted]"},
	{regexp.MustCompile(`(?i)\b(api[_\-]?key|consumer_key|consumer_secret|client_secret|secret|token|password|passwd|pwd|pass)\b["']?\s*[:=]\s*["']?[^\s"',;&)]+`), "$1=[redacted]"},
	{regexp.MustCompile(`(?i)\b(ck|cs)_[a-f0-9]{20,}`), "[redacted:wc-key]"},
	{regexp.MustCompile(`\b\d(?:[ \-]?\d){12,18}\b`), "[redacted:pan]"},
	{regexp.MustCompile(`(?i)\+\d[\d \-().]{7,}\d`), "[redacted:phone]"},
	{regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`), "[redacted:ip]"},
}

// SanitizeLine strips credentials and PII from one log line and caps its length.
func SanitizeLine(s string) string {
	if s == "" {
		return s
	}
	for _, r := range redactions {
		s = r.pattern.ReplaceAllString(s, r.replace)
	}
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) > maxMessageRunes {
		s = string(runes[:maxMessageRunes]) + " […truncated]"
	}
	return s
}

func sanitizeEntry(e LogEntry) LogEntry {
	e.Message = SanitizeLine(e.Message)
	e.Context = SanitizeLine(e.Context)
	e.Channel = SanitizeLine(e.Channel)
	return e
}
