package parser_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ma-tf/ogle/internal/services/parser"
)

const (
	shortPort8080_80   = "8080:80"
	expectedFwd80TCP   = "8080→80/tcp"
	expectedFwdEmpty80 = "→80/tcp"
	publishedKey       = "published"
	targetKey          = "target"
	protocolKey        = "protocol"
)

func TestSplitByColon(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		input    string
		expected []string
	}

	tt := []testCase{
		{name: "empty string", input: "", expected: []string{""}},
		{name: "no colon", input: "abc", expected: []string{"abc"}},
		{name: "two parts", input: "a:b", expected: []string{"a", "b"}},
		{name: "three parts", input: "a:b:c", expected: []string{"a", "b", "c"}},
		{name: "trailing colon", input: "a:", expected: []string{"a", ""}},
		{name: "leading colon", input: ":a", expected: []string{"", "a"}},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := parser.SplitByColon(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFindSlash(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		input    string
		expected int
	}

	tt := []testCase{
		{name: "empty string", input: "", expected: -1},
		{name: "no slash", input: "tcp", expected: -1},
		{name: "slash in middle", input: "80/tcp", expected: 2},
		{name: "leading slash", input: "/tcp", expected: 0},
		{name: "trailing slash", input: "tcp/", expected: 3},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := parser.FindSlash(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNormaliseShortPort(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		input    string
		expected string
	}

	tt := []testCase{
		{name: "host:container", input: shortPort8080_80, expected: expectedFwd80TCP},
		{name: "host:container with tcp", input: "8080:80/tcp", expected: expectedFwd80TCP},
		{name: "host:container with udp", input: "8080:80/udp", expected: "8080→80/udp"},
		{
			name:     "bind address host:container",
			input:    "127.0.0.1:8080:80",
			expected: expectedFwd80TCP,
		},
		{name: "container port only", input: "80", expected: expectedFwdEmpty80},
		{name: "container port with protocol", input: "80/tcp", expected: expectedFwdEmpty80},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := parser.NormaliseShortPort(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNormaliseLongPort(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		input    map[string]any
		expected string
	}

	tt := []testCase{
		{
			name: "target and published",
			input: map[string]any{
				targetKey:    float64(80),
				publishedKey: float64(8080),
			},
			expected: expectedFwd80TCP,
		},
		{
			name: "target, published, udp protocol",
			input: map[string]any{
				targetKey:    float64(80),
				publishedKey: float64(8080),
				protocolKey:  "udp",
			},
			expected: "8080→80/udp",
		},
		{
			name: "target only",
			input: map[string]any{
				targetKey: float64(80),
			},
			expected: expectedFwdEmpty80,
		},
		{
			name:     "empty map",
			input:    map[string]any{},
			expected: "→/tcp",
		},
		{
			name: "string target and published",
			input: map[string]any{
				targetKey:    "80",
				publishedKey: "8080",
			},
			expected: expectedFwd80TCP,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := parser.NormaliseLongPort(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNormalisePorts(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		input    []any
		expected []string
	}

	tt := []testCase{
		{name: "nil input", input: nil, expected: nil},
		{name: "empty slice", input: []any{}, expected: nil},
		{
			name:  "short form ports",
			input: []any{shortPort8080_80, "3000:3000"},
			expected: []string{
				expectedFwd80TCP,
				"3000→3000/tcp",
			},
		},
		{
			name: "long form ports",
			input: []any{
				map[string]any{
					targetKey:    float64(80),
					publishedKey: float64(8080),
				},
			},
			expected: []string{expectedFwd80TCP},
		},
		{
			name: "mixed short and long",
			input: []any{
				shortPort8080_80,
				map[string]any{
					targetKey:    float64(443),
					publishedKey: float64(8443),
				},
			},
			expected: []string{expectedFwd80TCP, "8443→443/tcp"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := parser.NormalisePorts(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
