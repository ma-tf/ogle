package layout_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ma-tf/ogle/internal/ui/layout"
)

func TestSidebarWidth(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 40, layout.SidebarWidth)
}

func TestSidebarMinTermWidth(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 80, layout.SidebarMinTermWidth)
}
