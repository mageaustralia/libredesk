package stringutil

import (
	"regexp"
	"strings"
)

var (
	// Match 4+ consecutive <br> tags — preserve double line breaks (paragraph spacing)
	excessiveBrRegex = regexp.MustCompile(`(<br\s*/?>\s*){4,}`)

	// Match multiple newlines (4+)
	multipleNewlinesRegex = regexp.MustCompile(`\n{4,}`)

	// Match Outlook-specific empty "elementToProof" divs
	outlookEmptyRegex = regexp.MustCompile(`<div[^>]*class="elementToProof"[^>]*>\s*(<br\s*/?>)?\s*</div>`)
)

// SanitizeEmailHTML cleans up messy HTML from email clients like Outlook.
// Preserves intentional formatting (paragraph spacing, empty paragraphs for vertical space).
func SanitizeEmailHTML(html string) string {
	// Remove Outlook's empty "elementToProof" divs (these are rendering artefacts)
	html = outlookEmptyRegex.ReplaceAllString(html, "")

	// Only collapse excessive <br> runs (4+) to double <br> — preserve paragraph breaks
	html = excessiveBrRegex.ReplaceAllString(html, "<br><br>")

	// Collapse excessive newlines (4+) to double newline
	html = multipleNewlinesRegex.ReplaceAllString(html, "\n\n")

	return strings.TrimSpace(html)
}
