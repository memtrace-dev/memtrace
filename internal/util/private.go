package util

import (
	"regexp"
	"strings"
)

var privateTagRe = regexp.MustCompile(`(?i)<private>[\s\S]*?</private>`)

// StripPrivate removes all <private>...</private> blocks from s and
// normalises the resulting whitespace. It is case-insensitive.
//
// This is applied to memory content before persistence so that agents can
// include sensitive details in a save request without those details ever
// reaching the database.
func StripPrivate(s string) string {
	stripped := privateTagRe.ReplaceAllString(s, "")

	// Collapse runs of 3+ newlines down to two (one blank line).
	stripped = regexp.MustCompile(`\n{3,}`).ReplaceAllString(stripped, "\n\n")

	return strings.TrimSpace(stripped)
}
