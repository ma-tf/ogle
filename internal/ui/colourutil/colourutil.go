// Package colourutil provides colour manipulation helpers for the UI layer.
package colourutil

import (
	"fmt"
	"image/color"
	"math"

	"charm.land/lipgloss/v2"
)

const (
	bitsPerChannel = 8
	maxRGBA        = 0xFFFF // 65535, the maximum value returned by color.Color.RGBA()
)

// Brighten multiplies each RGB channel of c by factor and returns a new
// lipgloss colour. Values are clamped to the valid range. Alpha is preserved.
func Brighten(c color.Color, factor float64) color.Color {
	r, g, b, _ := c.RGBA()

	br := clampChannel(float64(r) * factor)
	bg := clampChannel(float64(g) * factor)
	bb := clampChannel(float64(b) * factor)

	return lipgloss.Color(fmt.Sprintf(
		"#%02x%02x%02x",
		br>>bitsPerChannel,
		bg>>bitsPerChannel,
		bb>>bitsPerChannel,
	))
}

func clampChannel(v float64) uint32 {
	return uint32(math.Max(0, math.Min(maxRGBA, v)))
}
