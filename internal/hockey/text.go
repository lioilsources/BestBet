package hockey

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var foldReplacer = strings.NewReplacer("ø", "o", "Ø", "O", "æ", "ae", "Æ", "AE", "ß", "ss", "ł", "l", "Ł", "L")

// Fold strips diacritics: "Östersunds" -> "Ostersunds", "Baráž" -> "Baraz".
func Fold(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, foldReplacer.Replace(s))
	if err != nil {
		return s
	}
	return out
}
