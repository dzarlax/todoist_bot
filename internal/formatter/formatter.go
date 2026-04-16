package formatter

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// FindProjectNameForUser returns the project a given user is mapped to,
// or "" if no project matches. Matching is exact on the user identifier
// (e.g. "@username" or "First Last").
func FindProjectNameForUser(user string, mapping map[string][]string) string {
	for project, users := range mapping {
		for _, u := range users {
			if u == user {
				return project
			}
		}
	}
	return ""
}

var (
	// priorityRe matches `!p1`..`!p4` as a standalone token. (^|\s) captures the prefix
	// so replacement can preserve the whitespace boundary (RE2 has no lookaround).
	priorityRe = regexp.MustCompile(`(?i)(^|\s)!p([1-4])\b`)

	// labelRe matches `#tag` tokens. \w in RE2 is [0-9A-Za-z_].
	labelRe = regexp.MustCompile(`(^|\s)#(\w+)`)

	// Either an already-formed markdown link, or a bare URL.
	// Groups: 1 = full markdown-link match (if any), 2 = bare URL (if any).
	urlRe = regexp.MustCompile(`(?i)(\[[^\]]+\]\([^)]+\))|((?:https?://|www\.)[^\s]+(?:\.[^\s]+)+)`)

	whitespaceRe = regexp.MustCompile(`\s+`)
)

// ParseTaskMeta extracts priority markers (`!p1`..`!p4`) and labels (`#tag`)
// from the input text and returns the cleaned text with those markers removed.
// Priority defaults to 1 if no marker is present.
func ParseTaskMeta(text string) (priority int, labels []string, cleanText string) {
	priority = 1
	if text == "" {
		return
	}

	if m := priorityRe.FindStringSubmatch(text); m != nil {
		if n, err := strconv.Atoi(m[2]); err == nil && n >= 1 && n <= 4 {
			priority = n
		}
	}

	for _, m := range labelRe.FindAllStringSubmatch(text, -1) {
		labels = append(labels, m[2])
	}

	// Strip markers, preserving the boundary whitespace that was consumed by (^|\s).
	cleaned := priorityRe.ReplaceAllString(text, "$1")
	cleaned = labelRe.ReplaceAllString(cleaned, "$1")
	cleaned = whitespaceRe.ReplaceAllString(cleaned, " ")
	cleanText = strings.TrimSpace(cleaned)
	return
}

// FormatTextWithLinks inlines `text_link` Telegram entities as Markdown links
// (`[text](url)`) and converts bare URLs in the remaining text to Markdown too.
// Existing `[text](url)` fragments are left untouched.
func FormatTextWithLinks(text string, entities []tgbotapi.MessageEntity) string {
	if text == "" {
		return text
	}

	// 1. Apply text_link entities, last-first so earlier offsets remain valid.
	runes := []rune(text)
	relevant := make([]tgbotapi.MessageEntity, 0, len(entities))
	for _, e := range entities {
		if e.Type == "text_link" && e.URL != "" {
			relevant = append(relevant, e)
		}
	}
	sort.Slice(relevant, func(i, j int) bool { return relevant[i].Offset > relevant[j].Offset })
	for _, e := range relevant {
		start := e.Offset
		end := e.Offset + e.Length
		if start < 0 || end > len(runes) || start > end {
			continue
		}
		linkText := string(runes[start:end])
		replacement := []rune(fmt.Sprintf("[%s](%s)", linkText, e.URL))
		runes = append(runes[:start], append(replacement, runes[end:]...)...)
	}
	processed := string(runes)

	// 2. Convert bare URLs outside of existing markdown links.
	return urlRe.ReplaceAllStringFunc(processed, func(match string) string {
		subs := urlRe.FindStringSubmatch(match)
		if subs[1] != "" {
			// already a markdown link
			return subs[1]
		}
		bare := subs[2]
		full := bare
		if strings.HasPrefix(strings.ToLower(bare), "www.") {
			full = "https://" + bare
		}
		return fmt.Sprintf("[%s](%s)", bare, full)
	})
}
