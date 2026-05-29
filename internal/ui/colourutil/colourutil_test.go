package colourutil_test

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ma-tf/ogle/internal/ui/colourutil"
)

func TestBrighten(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		c        color.Color
		factor   float64
		expected color.Color
	}

	cases := []testCase{
		{
			name:     "factor 1.0 returns same color",
			c:        color.RGBA{255, 255, 255, 255},
			factor:   1.0,
			expected: color.RGBA{255, 255, 255, 255},
		},
		{
			name:     "factor 0.0 returns black",
			c:        color.RGBA{255, 255, 255, 255},
			factor:   0.0,
			expected: color.RGBA{0, 0, 0, 255},
		},
		{
			name:     "black unchanged by any factor",
			c:        color.RGBA{0, 0, 0, 255},
			factor:   2.0,
			expected: color.RGBA{0, 0, 0, 255},
		},
		{
			name:     "brightening past max clamps at white",
			c:        color.RGBA{128, 128, 128, 255},
			factor:   3.0,
			expected: color.RGBA{255, 255, 255, 255},
		},
		{
			name:     "factor 0.5 halves to mid-grey",
			c:        color.RGBA{255, 255, 255, 255},
			factor:   0.5,
			expected: color.RGBA{127, 127, 127, 255},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := colourutil.Brighten(tc.c, tc.factor)

			assert.Equal(t, tc.expected, got)
		})
	}
}
