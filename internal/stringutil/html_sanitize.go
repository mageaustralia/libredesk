package stringutil

import (
	"regexp"
	"strings"
)

var (
	// Match multiple consecutive <br> tags (with optional whitespace/attributes)
	multipleBrRegex = regexp.MustCompile(`(<br\s*/?>\s*){2,}`)

	// Match empty divs (with optional attributes but no content)
	emptyDivRegex = regexp.MustCompile(`<div[^>]*>\s*</div>`)

	// Match divs containing only whitespace or <br>
	whitespaceDivRegex = regexp.MustCompile(`<div[^>]*>\s*(<br\s*/?>)?\s*</div>`)

	// Match multiple newlines
	multipleNewlinesRegex = regexp.MustCompile(`\n{3,}`)

	// Match Outlook-specific empty elements
	outlookEmptyRegex = regexp.MustCompile(`<div[^>]*class="elementToProof"[^>]*>\s*(<br\s*/?>)?\s*</div>`)
)

// SanitizeEmailHTML cleans up messy HTML from email clients like Outlook.
// It removes excessive whitespace, empty divs, and multiple consecutive <br> tags.
func SanitizeEmailHTML(html string) string {
	// Remove Outlook's empty "elementToProof" divs
	html = outlookEmptyRegex.ReplaceAllString(html, "")

	// Remove empty divs
	html = emptyDivRegex.ReplaceAllString(html, "")

	// Remove divs with only whitespace or single <br>
	html = whitespaceDivRegex.ReplaceAllString(html, "")

	// Collapse multiple <br> tags to single <br>
	html = multipleBrRegex.ReplaceAllString(html, "<br>")

	// Collapse multiple newlines to double newline
	html = multipleNewlinesRegex.ReplaceAllString(html, "\n\n")

	return strings.TrimSpace(html)
}
