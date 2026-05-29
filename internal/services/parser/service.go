// Package parser provides parsing and validation of Docker Compose files.
package parser

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"go.yaml.in/yaml/v3"

	"github.com/ma-tf/ogle/internal/domain"
)

var (
	ErrReadComposeFile  = errors.New("failed to read compose file")
	ErrParseComposeFile = errors.New("failed to parse compose file")
)

// Port parsing constants.
const (
	portPartsBindAddrMin   = 3 // minimum parts to have a bind address (addr:host:container)
	portPartsHostContainer = 2 // parts count for host:container format
)

var _ Parser = Service{} //nolint:exhaustruct // readFileFn set by New

// Option configures a Service.
type Option func(*Service)

// WithReadFile sets the readFileFn on a Service. Useful in tests.
func WithReadFile(fn func(string) ([]byte, error)) Option {
	return func(s *Service) {
		s.readFileFn = fn
	}
}

// Parser validates and parses Compose Files into Projects.
//
//mockery:generate: true
type Parser interface {
	Parse(path string) (*domain.Project, error)
}

// composeFile is the minimal YAML structure required for parsing.
type composeFile struct {
	Name     string `yaml:"name"`
	Services map[string]struct {
		Image         string            `yaml:"image"`
		ContainerName string            `yaml:"container_name"` //nolint:tagliatelle // Docker Compose uses snake_case
		Build         any               `yaml:"build"`
		Labels        map[string]string `yaml:"labels"`
		Ports         []any             `yaml:"ports"`
	} `yaml:"services"`
}

// Service exposes compose file validation and parsing.
type Service struct {
	readFileFn func(string) ([]byte, error)
}

// New constructs a Service with the given options.
func New(opts ...Option) Service {
	s := Service{
		readFileFn: os.ReadFile,
	}
	for _, opt := range opts {
		opt(&s)
	}

	return s
}

// Parse reads and parses the compose file at path into a Project. path must
// be an absolute path to an existing, valid compose file; callers should use
// ScanAll and Validate before calling Parse.
func (s Service) Parse(path string) (*domain.Project, error) {
	cf, err := s.readAndUnmarshal(path)
	if err != nil {
		return nil, err
	}

	name := cf.Name
	if name == "" {
		absPath, errPath := filepath.Abs(path)
		if errPath != nil {
			return nil, fmt.Errorf("resolve absolute path: %w", errPath)
		}

		name = filepath.Base(filepath.Dir(absPath))
	}

	services := make([]domain.ServiceDef, 0, len(cf.Services))
	for serviceName, svc := range cf.Services {
		services = append(services, domain.ServiceDef{
			Name:          serviceName,
			Image:         svc.Image,
			ContainerName: svc.ContainerName,
			Labels:        svc.Labels,
			Ports:         NormalisePorts(svc.Ports),
		})
	}

	slices.SortFunc(services, func(a, b domain.ServiceDef) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return &domain.Project{
		Name:     name,
		File:     path,
		Services: services,
	}, nil
}

func (s Service) readAndUnmarshal(path string) (composeFile, error) {
	data, err := s.readFileFn(path)
	if err != nil {
		return composeFile{}, fmt.Errorf("%w: %w", ErrReadComposeFile, err)
	}

	var cf composeFile
	if unmarshalErr := yaml.Unmarshal(data, &cf); unmarshalErr != nil {
		return composeFile{}, fmt.Errorf("%w: %w", ErrParseComposeFile, unmarshalErr)
	}

	return cf, nil
}

// NormalisePorts converts Docker Compose port declarations into normalised
// display format "host→container/protocol". Returns an empty slice if ports
// is nil or empty.
func NormalisePorts(ports []any) []string {
	if len(ports) == 0 {
		return nil
	}

	result := make([]string, 0, len(ports))
	for _, p := range ports {
		switch v := p.(type) {
		case string:
			// Short form: "8080:80", "8080:80/tcp", "127.0.0.1:8080:80/tcp", "80", etc.
			result = append(result, NormaliseShortPort(v))
		case map[string]any:
			// Long form: {target: 80, published: 8080, protocol: tcp}
			result = append(result, NormaliseLongPort(v))
		}
	}

	return result
}

// NormaliseShortPort converts a short-form port string to "host→container/proto" format.
// Handles: "8080:80", "8080:80/tcp", "127.0.0.1:8080:80/tcp", "80", etc.
func NormaliseShortPort(s string) string {
	// Strip bind address (e.g., "127.0.0.1:8080:80/tcp" → "8080:80/tcp")
	parts := SplitByColon(s)
	protocol := "tcp" // default

	// Check if the last part contains a protocol
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if idx := FindSlash(lastPart); idx >= 0 {
			protocol = lastPart[idx+1:]
			parts[len(parts)-1] = lastPart[:idx]
		}
	}

	// Now parts should be like ["8080", "80"] or ["80"] or ["127.0.0.1", "8080", "80"]
	// If we have 3+ parts, the first is the bind address, skip it
	if len(parts) >= portPartsBindAddrMin {
		parts = parts[1:]
	}

	// Normalize based on what we have
	if len(parts) == portPartsHostContainer {
		// "8080:80" → "8080→80/tcp"
		return parts[0] + "→" + parts[1] + "/" + protocol
	} else if len(parts) == 1 {
		// "80" → "→80/tcp" (no host port)
		return "→" + parts[0] + "/" + protocol
	}

	// Fallback (shouldn't happen with valid input)
	return s
}

// NormaliseLongPort converts a long-form port object to "host→container/proto" format.
func NormaliseLongPort(m map[string]any) string {
	protocol := "tcp"
	if p, ok := m["protocol"].(string); ok && p != "" {
		protocol = p
	}

	container := ""

	switch t := m["target"].(type) {
	case float64:
		container = strconv.Itoa(int(t))
	case string:
		container = t
	}

	host := ""

	switch pub := m["published"].(type) {
	case float64:
		host = strconv.Itoa(int(pub))
	case string:
		host = pub
	}

	if host != "" {
		return host + "→" + container + "/" + protocol
	}

	return "→" + container + "/" + protocol
}

// SplitByColon splits a string by colons. Simple helper to avoid importing strings unnecessarily.
func SplitByColon(s string) []string {
	var (
		result  []string
		current string
	)

	for _, ch := range s {
		if ch == ':' {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}

	result = append(result, current)

	return result
}

// FindSlash returns the index of the first '/' in s, or -1 if not found.
func FindSlash(s string) int {
	for i, ch := range s {
		if ch == '/' {
			return i
		}
	}

	return -1
}
