package zlib

import (
	"reflect"
	"strings"
	"unicode"
)

// cleanText removes control characters, other than newline and tab, from every
// exported string reachable from v. EAPI text such as titles, authors and
// error messages ends up on the user's terminal, where ESC, CSI and OSC
// sequences from a hostile or compromised mirror could rewrite the screen or
// set the clipboard (OSC 52). Without the ESC byte what remains is plain text.
func cleanText(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			cleanText(v.Elem())
		}
	case reflect.Interface:
		// The dynamic value of an interface cannot be set in place; clean a
		// copy and store it back, keeping its type (string or json.Number).
		if v.IsNil() || !v.CanSet() {
			return
		}
		value := reflect.New(v.Elem().Type()).Elem()
		value.Set(v.Elem())
		cleanText(value)
		v.Set(value)
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Field(i).CanSet() {
				cleanText(v.Field(i))
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			cleanText(v.Index(i))
		}
	case reflect.Map:
		// Keys are server text too. Rebuild the map so a cleaned key
		// replaces the old one. Interface values arrive here as settable
		// copies, so v can always be replaced.
		if v.IsNil() || !v.CanSet() {
			return
		}
		cleaned := reflect.MakeMapWithSize(v.Type(), v.Len())
		for _, key := range v.MapKeys() {
			k := reflect.New(key.Type()).Elem()
			k.Set(key)
			cleanText(k)
			value := reflect.New(v.Type().Elem()).Elem()
			value.Set(v.MapIndex(key))
			cleanText(value)
			cleaned.SetMapIndex(k, value)
		}
		v.Set(cleaned)
	case reflect.String:
		if v.CanSet() {
			v.SetString(stripControl(v.String(), "\n\t"))
		}
	}
}

// stripControl drops every Unicode control character from s except those in
// keep. strings.Map also rewrites invalid UTF-8 as U+FFFD: a lone 0x9b byte is
// not a control rune to the decoder, but an 8-bit terminal reads it as CSI.
// Do not add a fast path that returns s unchanged; it would keep those bytes.
func stripControl(s, keep string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && !strings.ContainsRune(keep, r) {
			return -1
		}
		return r
	}, s)
}

// isBidiControl reports the Unicode bidirectional formatting characters,
// which can make a file name such as "book\u202Ebupe.exe" display as
// "bookexe.epub".
func isBidiControl(r rune) bool {
	return r == '\u200E' || r == '\u200F' || r == '\u061C' ||
		(r >= '\u202A' && r <= '\u202E') || (r >= '\u2066' && r <= '\u2069')
}
