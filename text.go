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
		for _, key := range v.MapKeys() {
			value := reflect.New(v.Type().Elem()).Elem()
			value.Set(v.MapIndex(key))
			cleanText(value)
			v.SetMapIndex(key, value)
		}
	case reflect.String:
		if v.CanSet() {
			v.SetString(stripControl(v.String(), "\n\t"))
		}
	}
}

// stripControl drops every Unicode control character from s except those in
// keep.
func stripControl(s, keep string) string {
	unsafe := func(r rune) bool { return unicode.IsControl(r) && !strings.ContainsRune(keep, r) }
	if !strings.ContainsFunc(s, unsafe) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if unsafe(r) {
			return -1
		}
		return r
	}, s)
}
