// Package textnorm normalises titles so searches ignore case, accents and punctuation.
package textnorm

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Normalize lowercases s, strips accents, removes apostrophes, turns other
// punctuation into spaces and collapses whitespace. Letters of any script are kept.
func Normalize(s string) string {
	decomposed := norm.NFKD.String(s)
	var b strings.Builder
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	composed := norm.NFC.String(b.String())

	var out strings.Builder
	space := false
	for _, r := range composed {
		switch {
		case r == '\'' || r == '’' || r == '‘':
			continue
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			if space && out.Len() > 0 {
				out.WriteByte(' ')
			}
			space = false
			out.WriteRune(unicode.ToLower(r))
		default:
			space = true
		}
	}
	return out.String()
}
