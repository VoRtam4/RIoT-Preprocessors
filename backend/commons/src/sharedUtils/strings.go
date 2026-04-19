package sharedUtils

import (
	"strings"
	"unicode/utf8"

	"github.com/dchest/uniuri"
)

func GenerateRandomAlphanumericString(length int) string {
	return uniuri.NewLen(length)
}

func SafeLabel(s string, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	if utf8.RuneCountInString(s) == 1 {
		return strings.ToUpper(s)
	}
	firstRune, size := utf8.DecodeRuneInString(s)
	if firstRune == utf8.RuneError && size == 1 {
		return s
	}
	return strings.ToUpper(string(firstRune)) + s[size:]
}
