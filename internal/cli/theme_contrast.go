package cli

import (
	"image/color"
	"math"
)

// minTextContrast is the WCAG 2.x AA ratio for normal text. Every built-in
// role that colors text, including secondary (Muted) text, must reach it on
// the theme background.
const minTextContrast = 4.5

// minRoleDistance is how far apart (RGB distance, see colorDistance) two
// colors must be to read as different roles, such as two file formats.
const minRoleDistance = 60

// readableTheme raises every text role of t to minTextContrast against its
// background. Syntax palettes keep comment colors dim on purpose, which left
// zlib's secondary text below 3:1 in most themes.
func readableTheme(t Theme) Theme {
	if t.Background == nil {
		return t
	}
	for _, role := range []*color.Color{
		&t.Accent, &t.Link, &t.Title, &t.Success, &t.Error, &t.Warning,
		&t.Info, &t.Muted, &t.ExtEPUB, &t.ExtPDF, &t.ExtMOBI, &t.ExtFB2,
	} {
		*role = readableOn(*role, t.Background, minTextContrast)
	}
	// Selected rows draw Title on Surface, which sits a step off Background.
	if t.Surface != nil {
		t.Title = readableOn(t.Title, t.Surface, minTextContrast)
	}
	return t
}

// readableOn returns fg, moved toward black or white (whichever the
// background is not) in small steps until it reaches ratio against bg.
func readableOn(fg, bg color.Color, ratio float64) color.Color {
	if fg == nil || contrastRatio(fg, bg) >= ratio {
		return fg
	}
	target := color.Color(color.White)
	if relativeLuminance(bg) > 0.5 {
		target = color.Black
	}
	for step := 1; step <= 20; step++ {
		mixed := mixColor(fg, target, float64(step)/20)
		if contrastRatio(mixed, bg) >= ratio {
			return mixed
		}
	}
	return target
}

// distinctColors picks n colors from candidates, in order, skipping any that
// sit within minRoleDistance of one already picked. When too few qualify, the
// remaining slots take the farthest leftover candidates.
func distinctColors(n int, candidates ...color.Color) []color.Color {
	picked := make([]color.Color, 0, n)
	var rest []color.Color
	for _, c := range candidates {
		if len(picked) < n && nearestDistance(c, picked) >= minRoleDistance {
			picked = append(picked, c)
		} else {
			rest = append(rest, c)
		}
	}
	for len(picked) < n && len(rest) > 0 {
		best := 0
		for i, c := range rest {
			if nearestDistance(c, picked) > nearestDistance(rest[best], picked) {
				best = i
			}
		}
		picked = append(picked, rest[best])
		rest = append(rest[:best], rest[best+1:]...)
	}
	return picked
}

// distinctFrom returns the first candidate at least minRoleDistance from
// every color in others, or the farthest candidate when none is.
func distinctFrom(candidates []color.Color, others ...color.Color) color.Color {
	best := candidates[0]
	for _, c := range candidates {
		if nearestDistance(c, others) >= minRoleDistance {
			return c
		}
		if nearestDistance(c, others) > nearestDistance(best, others) {
			best = c
		}
	}
	return best
}

// nearestDistance is the distance from c to the closest color in others.
func nearestDistance(c color.Color, others []color.Color) float64 {
	nearest := math.Inf(1)
	for _, o := range others {
		nearest = min(nearest, colorDistance(c, o))
	}
	return nearest
}

func mixColor(a, b color.Color, t float64) color.Color {
	ar, ag, ab := rgb8(a)
	br, bg, bb := rgb8(b)
	mix := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.RGBA{R: mix(ar, br), G: mix(ag, bg), B: mix(ab, bb), A: 0xff}
}

func rgb8(c color.Color) (uint8, uint8, uint8) {
	r, g, b, _ := c.RGBA()
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}

// relativeLuminance is the WCAG 2.x relative luminance of c.
func relativeLuminance(c color.Color) float64 {
	r, g, b := rgb8(c)
	linear := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
}

// contrastRatio is the WCAG 2.x contrast ratio between a and b (1–21).
func contrastRatio(a, b color.Color) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}
