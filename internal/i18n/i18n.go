// Package i18n picks between the German and English variant of user-facing texts. Texts are
// written inline as pairs, T("Ablage", "File"), so both languages stay side by side in the code.
package i18n

import (
	"os"
	"strings"
	"sync/atomic"
)

// Lang is a supported user interface language.
type Lang int32

const (
	English Lang = iota
	German
)

var current atomic.Int32

func init() { current.Store(int32(FromLocale(envLocale()))) }

// envLocale is the POSIX locale; native front ends override it with the system preference.
func envLocale() string {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// FromLocale maps a locale or language tag ("de_DE.UTF-8", "de-AT", "en") to a language; anything
// that is not German is English.
func FromLocale(tag string) Lang {
	if strings.HasPrefix(strings.ToLower(tag), "de") {
		return German
	}
	return English
}

// Set selects the language.
func Set(l Lang) { current.Store(int32(l)) }

// Current is the selected language.
func Current() Lang { return Lang(current.Load()) }

// T returns the variant for the selected language.
func T(de, en string) string {
	if Current() == German {
		return de
	}
	return en
}
