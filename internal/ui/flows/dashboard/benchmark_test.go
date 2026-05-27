package dashboard_test

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/stretchr/testify/mock"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	dockermocks "github.com/ma-tf/ogle/internal/services/docker/mocks"
	parsermocks "github.com/ma-tf/ogle/internal/services/parser/mocks"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// benchmarkDone is a sentinel message type used by the concurrent benchmark
// to signal that all mouse events have been processed.
type benchmarkDone struct{}

func benchmarkModel(b *testing.B) dashboard.Model {
	b.Helper()

	mockD := dockermocks.NewMockDocker(b)
	mockP := parsermocks.NewMockParser(b)

	mockD.EXPECT().Ps(mock.Anything, mock.Anything, mock.Anything).
		Return(func() tea.Msg {
			return msgs.ServicesPolled{Runtimes: map[string]*domain.ServiceRuntimeData{}}
		}).Maybe()

	mockD.EXPECT().Connect(mock.Anything).Maybe().
		Return(func() tea.Msg { return msgs.DaemonUnavailable{Err: nil} })

	m := dashboard.New(
		b.Context(),
		&domain.Project{
			Name: "bench",
			File: "/dev/null/compose.yaml",
			Services: []domain.ServiceDef{
				{Name: svcWeb, Image: "nginx"},
				{Name: svcAPI, Image: "api"},
				{Name: "db", Image: "postgres"},
			},
		},
		theme.Default(),
		config.Defaults(),
		zone.New(),
		b.TempDir(),
		120,
		80,
		mockD,
		mockP,
	)

	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 80})

	return m
}

// BenchmarkMouseMotionPerFrame measures the synchronous cost of a single
// Update(MouseMotionMsg) + View() cycle at the dashboard level.
func BenchmarkMouseMotionPerFrame(b *testing.B) {
	m := benchmarkModel(b)

	m.Update(tea.MouseMotionMsg{X: 0, Y: 0})
	m.View()

	b.ResetTimer()

	for i := range b.N {
		x := (i * 7) % 100
		y := (i * 13) % 50

		m, _ = m.Update(tea.MouseMotionMsg{X: x, Y: y})
		_ = m.View()
	}
}

// BenchmarkMouseMotionBgContention measures throughput when background messages
// are sent concurrently from separate goroutines, competing with mouse events
// on an unbuffered channel (simulating Bubble Tea p.msgs contention).
func BenchmarkMouseMotionBgContention(b *testing.B) {
	m := benchmarkModel(b)

	msgCh := make(chan tea.Msg) // unbuffered, like p.msgs

	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()

	for range 3 {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case msgCh <- msgs.LogStreamContainerNotFound{ServiceName: svcWeb}:
				}
			}
		}()
	}

	done := make(chan struct{})

	go func() {
		for i := range b.N {
			x := (i * 7) % 100
			y := (i * 13) % 50

			msgCh <- tea.MouseMotionMsg{X: x, Y: y}
		}

		msgCh <- benchmarkDone{}

		close(done)
	}()

	b.ResetTimer()

	for {
		msg := <-msgCh
		if _, ok := msg.(benchmarkDone); ok {
			break
		}

		m, _ = m.Update(msg)
		_ = m.View()
	}

	<-done
}
