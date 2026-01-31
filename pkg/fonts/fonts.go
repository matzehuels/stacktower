// Package fonts provides embedded font files for SVG rendering.
//
// The fonts are embedded directly into the binary using go:embed,
// making them available without external dependencies.
package fonts

import (
	_ "embed"
	"encoding/base64"
	"sync"
)

// XKCDScript is the xkcd-script font from https://github.com/ipython/xkcd-font
// This handwriting-style font is used for the hand-drawn visual style.

//go:embed xkcd-script.woff
var xkcdScriptWOFF []byte

//go:embed xkcd-script.ttf
var xkcdScriptTTF []byte

// XKCDScriptWOFF returns the WOFF font data.
func XKCDScriptWOFF() []byte {
	return xkcdScriptWOFF
}

// XKCDScriptTTF returns the TTF font data.
func XKCDScriptTTF() []byte {
	return xkcdScriptTTF
}

// Cache for base64-encoded fonts (computed once on first access).
var (
	woffBase64     string
	woffBase64Once sync.Once
)

// XKCDScriptWOFFBase64 returns the WOFF font data as a base64 string.
// The result is cached after first computation.
func XKCDScriptWOFFBase64() string {
	woffBase64Once.Do(func() {
		woffBase64 = base64.StdEncoding.EncodeToString(xkcdScriptWOFF)
	})
	return woffBase64
}

// FontFamily is the CSS font-family name for the xkcd-script font.
const FontFamily = "xkcd Script"

// FallbackFontFamily provides fallback fonts for systems without the embedded font.
const FallbackFontFamily = `'xkcd Script', 'Comic Sans MS', 'Bradley Hand', 'Segoe Script', sans-serif`
