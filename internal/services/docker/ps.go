package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
)

// psLine maps a single JSON line from "docker compose ps --format json".
type psLine struct {
	ID        string `json:"id"`
	Service   string `json:"service"`
	State     string `json:"state"`
	CreatedAt string `json:"createdat"`
	Status    string `json:"status"`
}

// Ps returns a Cmd that runs docker compose ps and returns a ServicesPolled
// message with the parsed runtime data for every service.
func (s *Service) Ps(ctx context.Context, composeFile, projectName string) tea.Cmd {
	return func() tea.Msg {
		cmd := s.commander.CommandContext(
			ctx,
			"docker", "compose",
			"-f", composeFile,
			"-p", projectName,
			"ps", "--format", "json",
		)
		cmd.Dir = filepath.Dir(composeFile)

		out, err := cmd.Output()
		if err != nil {
			return msgs.ServicesPolled{
				Runtimes: nil,
				Err:      fmt.Errorf("docker compose ps: %w", err),
			}
		}

		runtimes, err := ParsePsOutput(out)
		if err != nil {
			return msgs.ServicesPolled{
				Runtimes: nil,
				Err:      fmt.Errorf("parse compose ps output: %w", err),
			}
		}

		return msgs.ServicesPolled{Runtimes: runtimes, Err: nil}
	}
}

// ParsePsOutput parses the JSON-lines output of "docker compose ps --format json"
// into a map keyed by service name.
func ParsePsOutput(data []byte) (map[string]*domain.ServiceRuntimeData, error) {
	runtimes := make(map[string]*domain.ServiceRuntimeData)

	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) == 1 && len(lines[0]) == 0 {
		return runtimes, nil
	}

	for _, line := range lines {
		var entry psLine
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("decode json line: %w", err)
		}

		name := strings.TrimSpace(entry.Service)
		if name == "" {
			continue
		}

		createdAt, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", entry.CreatedAt)

		runtimes[name] = &domain.ServiceRuntimeData{
			ContainerID: entry.ID,
			State:       ParseState(entry.State),
			Status:      entry.Status,
			CreatedAt:   createdAt,
		}
	}

	return runtimes, nil
}

// ParseState maps a Docker container state string to a domain.ServiceState.
func ParseState(s string) domain.ServiceState {
	switch s {
	case "running":
		return domain.ServiceStateRunning
	case "exited":
		return domain.ServiceStateExited
	case "paused":
		return domain.ServiceStatePaused
	case "restarting":
		return domain.ServiceStateRestarting
	case "dead":
		return domain.ServiceStateDead
	default:
		return domain.ServiceStateUnknown
	}
}
