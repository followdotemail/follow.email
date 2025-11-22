package email

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
)

// EmailRenderOptions controls how raw Gmail HTML should be normalized for the frontend.
type EmailRenderOptions struct {
	ShouldLoadImages bool
	Theme            string
}

// ProcessedEmailContent captures sanitized HTML output and metadata.
type ProcessedEmailContent struct {
	HTML             string
	HasBlockedImages bool
}

var (
	unsafeCSSPattern   = regexp.MustCompile(`(?is)@import|expression|url\(`)
	invisiblePixelRule = regexp.MustCompile(`(?i)(display\s*:\s*none|height\s*:\s*0|width\s*:\s*0)`)
)

// ProcessEmailHTML sanitizes Gmail HTML, removes trackers, optionally blocks images, and adds theming.
func ProcessEmailHTML(rawHTML string, opts EmailRenderOptions) (ProcessedEmailContent, error) {
	if strings.TrimSpace(rawHTML) == "" {
		return ProcessedEmailContent{}, nil
	}

	safeHTML := sanitizeHTML(rawHTML)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(wrapWithHTML(safeHTML)))
	if err != nil {
		return ProcessedEmailContent{}, fmt.Errorf("failed to parse sanitized HTML: %w", err)
	}

	removeDisallowedElements(doc)
	cleanStyleBlocks(doc)
	wrapQuotedReplies(doc)

	hasBlockedImages := applyImagePreferences(doc, opts.ShouldLoadImages)
	stripTrackingPixels(doc)

	injectTheme(doc, opts.Theme)

	finalHTML, err := extractBodyHTML(doc)
	if err != nil {
		return ProcessedEmailContent{}, fmt.Errorf("failed to serialize sanitized HTML: %w", err)
	}

	return ProcessedEmailContent{
		HTML:             finalHTML,
		HasBlockedImages: hasBlockedImages,
	}, nil
}

func sanitizeHTML(input string) string {
	policy := bluemonday.UGCPolicy()
	policy.AllowElements("style", "details", "summary", "table", "thead", "tbody", "tfoot", "tr", "th", "td", "colgroup", "col")
	policy.AllowAttrs("class", "style", "id", "data-*", "dir").Globally()
	policy.AllowAttrs("href", "title", "target", "rel").OnElements("a")
	policy.AllowURLSchemes("http", "https", "mailto", "tel", "data", "cid")
	policy.AllowAttrs("src", "srcset", "alt", "title", "width", "height").OnElements("img")
	policy.AllowAttrs("type", "media").OnElements("style")
	policy.AllowRelativeURLs(true)
	return policy.Sanitize(input)
}

func wrapWithHTML(body string) string {
	return "<html><body>" + body + "</body></html>"
}

func removeDisallowedElements(doc *goquery.Document) {
	doc.Find("title, meta, link").Each(func(_ int, s *goquery.Selection) {
		s.Remove()
	})
}

func cleanStyleBlocks(doc *goquery.Document) {
	doc.Find("style").Each(func(_ int, s *goquery.Selection) {
		cleaned := unsafeCSSPattern.ReplaceAllString(s.Text(), "")
		s.SetText(cleaned)
	})
}

func wrapQuotedReplies(doc *goquery.Document) {
	doc.Find("blockquote, .gmail_quote").Each(func(_ int, block *goquery.Selection) {
		if parent := block.Parent(); parent.HasClass("quoted-toggle") || parent.Is("details.quoted-toggle") {
			return
		}

		html, err := goquery.OuterHtml(block)
		if err != nil {
			return
		}

		wrapper := fmt.Sprintf(`<details class="quoted-toggle"><summary>Show previous message</summary>%s</details>`, html)
		block.ReplaceWithHtml(wrapper)
	})
}

func applyImagePreferences(doc *goquery.Document, shouldLoadImages bool) bool {
	if shouldLoadImages {
		return false
	}

	hasBlocked := false
	doc.Find("img").Each(func(_ int, img *goquery.Selection) {
		src := strings.TrimSpace(img.AttrOr("src", ""))
		if strings.HasPrefix(strings.ToLower(src), "cid:") {
			return
		}
		hasBlocked = true
		img.ReplaceWithHtml(`<span class="fe-image-placeholder">Remote image blocked</span>`)
	})
	return hasBlocked
}

func stripTrackingPixels(doc *goquery.Document) {
	doc.Find("img").Each(func(_ int, img *goquery.Selection) {
		if isTrackingPixel(img) {
			img.Remove()
		}
	})
	// Remove hidden spans frequently used as invisible preheaders
	doc.Find("span, div").Each(func(_ int, node *goquery.Selection) {
		style := node.AttrOr("style", "")
		if invisiblePixelRule.MatchString(style) {
			node.Remove()
		}
	})
}

func isTrackingPixel(img *goquery.Selection) bool {
	widthAttr := strings.TrimSpace(img.AttrOr("width", ""))
	heightAttr := strings.TrimSpace(img.AttrOr("height", ""))
	styleAttr := img.AttrOr("style", "")

	width := parseDimension(widthAttr)
	height := parseDimension(heightAttr)

	if (width <= 1 && width > 0) && (height <= 1 && height > 0) {
		return true
	}

	return invisiblePixelRule.MatchString(styleAttr)
}

func parseDimension(value string) int {
	if value == "" {
		return 0
	}
	value = strings.TrimSpace(strings.TrimSuffix(value, "px"))
	var parsed int
	fmt.Sscanf(value, "%d", &parsed)
	return parsed
}

func injectTheme(doc *goquery.Document, theme string) {
	theme = strings.ToLower(theme)
	if theme == "" {
		theme = "dark"
	}

	common := `
:host {
	display: block;
	font-family: Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}
body {
	margin: 0;
	padding: 0;
	line-height: 1.5;
}
table {
	border-collapse: collapse;
	width: 100%;
}
img {
	max-width: 100%;
	height: auto;
}
.quoted-toggle {
	margin: 1rem 0;
	padding: 0.5rem 0.75rem;
	border-left: 3px solid rgba(148, 163, 184, 0.6);
	border-radius: 0.5rem;
	background: rgba(148, 163, 184, 0.1);
}
.quoted-toggle summary {
	cursor: pointer;
	font-weight: 600;
	outline: none;
}
.fe-image-placeholder {
	display: inline-flex;
	padding: 0.25rem 0.5rem;
	margin: 0.25rem 0;
	border-radius: 0.375rem;
	background: rgba(244, 114, 182, 0.15);
	color: rgba(244, 114, 182, 1);
	font-size: 0.85rem;
}
`

	var palette string
	if theme == "light" {
		palette = `
body {
	background: #ffffff;
	color: #0f172a;
}
a {
	color: #2563eb;
}
`
	} else {
		palette = `
body {
	background: #0b0f19;
	color: #e2e8f0;
}
a {
	color: #93c5fd;
}
`
	}

	styleBlock := fmt.Sprintf("<style>%s%s</style>", common, palette)
	doc.Find("body").PrependHtml(styleBlock)
}

func extractBodyHTML(doc *goquery.Document) (string, error) {
	body := doc.Find("body")
	if body.Length() == 0 {
		return doc.Html()
	}

	html, err := body.Html()
	if err != nil {
		return "", err
	}
	return html, nil
}
