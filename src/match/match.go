package match

import (
	"regexp"
	"strings"

	"github.com/anyascii/go"
)

func LooksAlike(a, b string) bool {
	if strings.EqualFold(a, b) {
		return true
	}

	a = washText(a)
	b = washText(b)

	if strings.EqualFold(a, b) {
		return true
	}

	return false
}

func ContainsAlike(content, key string) bool {
	const boundary = `(^|$|[^\pL\pN])`

	re := regexp.MustCompile(`(?i)` + boundary + regexp.QuoteMeta(key) + boundary)
	if re.MatchString(content) {
		return true
	}

	content = washText(content)
	key = washText(key)

	re = regexp.MustCompile(`(?i)` + boundary + regexp.QuoteMeta(key) + boundary)
	if re.MatchString(content) {
		return true
	}

	return false
}

var (
	punctRe = regexp.MustCompile(`[^\s\pL\pN]+`)
	spaceRe = regexp.MustCompile(`\s+`)
)

func washText(s string) string {
	// remove everything except alnums and spaces
	s = punctRe.ReplaceAllString(s, "")
	// convert unicode to ascii
	s = anyascii.Transliterate(s)
	// squash subsequent spaces into one
	s = spaceRe.ReplaceAllString(s, " ")
	// trim surrounding spaces
	s = strings.TrimSpace(s)

	return s
}
