package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

const (
	inspectPath = "http://localhost/containers/%s/json"
)

// inspectResponse maps the relevant part of the Docker Engine API
// GET /containers/{id}/json response.
type inspectResponse struct {
	Config struct {
		Labels map[string]string `json:"labels"`
	} `json:"config"`
}

// Inspect returns a Cmd that fetches container metadata via the Docker Engine
// API and returns container labels via ContainerLabelsPolled.
func (s *Service) Inspect(ctx context.Context, containerID string) tea.Cmd {
	return func() tea.Msg {
		path := fmt.Sprintf(inspectPath, containerID)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
		if err != nil {
			return msgs.ContainerLabelsPolled{
				Labels: nil,
				Err:    fmt.Errorf("build inspect request: %w", err),
			}
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return msgs.ContainerLabelsPolled{
				Labels: nil,
				Err:    fmt.Errorf("inspect container: %w", err),
			}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return msgs.ContainerLabelsPolled{
				Labels: nil,
				Err:    fmt.Errorf("%w: %d", ErrUnexpectedInspectStatus, resp.StatusCode),
			}
		}

		var data inspectResponse
		if decErr := json.NewDecoder(resp.Body).Decode(&data); decErr != nil {
			return msgs.ContainerLabelsPolled{
				Labels: nil,
				Err:    fmt.Errorf("decode inspect response: %w", decErr),
			}
		}

		labels := data.Config.Labels

		return msgs.ContainerLabelsPolled{Labels: labels, Err: nil}
	}
}
