package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestShowHelp_TopLevel(t *testing.T) {
	registry := NewPluginRegistry()

	// Register some test contexts
	registry.Register(&PluginContext{
		Context:     'S',
		ContextLong: "shell",
		Description: "Shell operations",
		Flags: []Flag{
			{Short: 'c', Long: "command", ArgName: "cmd", Description: "Execute command"},
		},
	})

	registry.Register(&PluginContext{
		Context:     'T',
		ContextLong: "time",
		Description: "Time management",
		Flags: []Flag{
			{Short: 'f', Long: "format", ArgName: "fmt", Description: "Time format"},
		},
	})

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ShowHelp(registry, []rune{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for expected content
	expectedStrings := []string{
		"Usage: wsh [OPTIONS] [COMMAND]",
		"A zsh wrapper with plugin support",
		"Contexts:",
		"-S",
		"--shell",
		"Shell operations",
		"-T",
		"--time",
		"Time management",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestShowHelp_ContextSpecific(t *testing.T) {
	registry := NewPluginRegistry()

	// Register test context
	registry.Register(&PluginContext{
		Context:     'T',
		ContextLong: "time",
		Description: "Time management operations",
		Flags: []Flag{
			{Short: 'f', Long: "format", ArgName: "fmt", Description: "Time format string"},
			{Short: 'z', Long: "timezone", ArgName: "tz", Description: "Timezone to use"},
		},
	})

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ShowHelp(registry, []rune{'T'})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for expected content
	expectedStrings := []string{
		"Usage: wsh -T [OPTIONS] [ARGS]",
		"Time management operations",
		"Options:",
		"-f, --format <fmt>",
		"Time format string",
		"-z, --timezone <tz>",
		"Timezone to use",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestShowHelp_WithSubContexts(t *testing.T) {
	registry := NewPluginRegistry()

	// Create sub-context
	overtimeCtx := &PluginContext{
		Context:     'O',
		ContextLong: "overtime",
		Description: "Overtime tracking",
		Flags: []Flag{
			{Short: 'h', Long: "hours", ArgName: "num", Description: "Hours worked"},
		},
	}

	// Register parent context with sub-context
	registry.Register(&PluginContext{
		Context:     'T',
		ContextLong: "time",
		Description: "Time management",
		Flags: []Flag{
			{Short: 'f', Long: "format", ArgName: "fmt", Description: "Time format"},
		},
		SubContexts: map[rune]*PluginContext{
			'O': overtimeCtx,
		},
	})

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ShowHelp(registry, []rune{'T'})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for expected content
	expectedStrings := []string{
		"Usage: wsh -T [OPTIONS] [ARGS]",
		"Time management",
		"Sub-contexts:",
		"-O",
		"--overtime",
		"Overtime tracking",
		"Use 'wsh -T<subcontext>h' for sub-context help",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
		}
	}

	// Should NOT contain sub-context flags
	if strings.Contains(output, "--hours") {
		t.Errorf("Output should not contain sub-context flags, got:\n%s", output)
	}
}

func TestShowHelp_NestedContext(t *testing.T) {
	registry := NewPluginRegistry()

	// Create sub-context
	overtimeCtx := &PluginContext{
		Context:     'O',
		ContextLong: "overtime",
		Description: "Overtime tracking",
		Flags: []Flag{
			{Short: 'h', Long: "hours", ArgName: "num", Description: "Hours worked"},
			{Short: 'r', Long: "rate", ArgName: "rate", Description: "Hourly rate"},
		},
	}

	// Register parent context with sub-context
	registry.Register(&PluginContext{
		Context:     'T',
		ContextLong: "time",
		Description: "Time management",
		SubContexts: map[rune]*PluginContext{
			'O': overtimeCtx,
		},
	})

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ShowHelp(registry, []rune{'T', 'O'})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for expected content
	expectedStrings := []string{
		"Usage: wsh -TO [OPTIONS] [ARGS]",
		"Overtime tracking",
		"-h, --hours <num>",
		"Hours worked",
		"-r, --rate <rate>",
		"Hourly rate",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
		}
	}
}

func TestShowHelp_UnknownContext(t *testing.T) {
	registry := NewPluginRegistry()

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ShowHelp(registry, []rune{'X'})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "unknown context") {
		t.Errorf("Expected error message about unknown context, got: %s", output)
	}
}

func TestShowHelp_ExternalPlugin(t *testing.T) {
	registry := NewPluginRegistry()

	// Register external plugin
	registry.Register(&PluginContext{
		Context:     'T',
		ContextLong: "time",
		Description: "Time management",
		Script:      "/usr/local/bin/time_plugin.sh",
		Flags: []Flag{
			{Short: 'f', Long: "format", ArgName: "fmt", Description: "Time format"},
		},
	})

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ShowHelp(registry, []rune{'T'})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should show script path
	if !strings.Contains(output, "Plugin script: /usr/local/bin/time_plugin.sh") {
		t.Errorf("Expected plugin script path in output, got:\n%s", output)
	}
}

func TestFormatContextPath(t *testing.T) {
	tests := []struct {
		name string
		path []rune
		want string
	}{
		{
			name: "empty path",
			path: []rune{},
			want: "",
		},
		{
			name: "single context",
			path: []rune{'T'},
			want: "T",
		},
		{
			name: "nested context",
			path: []rune{'T', 'O'},
			want: "TO",
		},
		{
			name: "deeply nested",
			path: []rune{'A', 'B', 'C'},
			want: "ABC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatContextPath(tt.path)
			if got != tt.want {
				t.Errorf("formatContextPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSortContexts(t *testing.T) {
	contexts := []*PluginContext{
		{Context: 'Z', ContextLong: "zulu"},
		{Context: 'A', ContextLong: "alpha"},
		{Context: 'M', ContextLong: "mike"},
	}

	sorted := sortContexts(contexts)

	if sorted[0].Context != 'A' {
		t.Errorf("First context should be 'A', got %c", sorted[0].Context)
	}

	if sorted[1].Context != 'M' {
		t.Errorf("Second context should be 'M', got %c", sorted[1].Context)
	}

	if sorted[2].Context != 'Z' {
		t.Errorf("Third context should be 'Z', got %c", sorted[2].Context)
	}

	// Verify original not modified
	if contexts[0].Context != 'Z' {
		t.Error("Original slice should not be modified")
	}
}

func TestSortSubContexts(t *testing.T) {
	subContexts := map[rune]*PluginContext{
		'Z': {Context: 'Z', ContextLong: "zulu"},
		'A': {Context: 'A', ContextLong: "alpha"},
		'M': {Context: 'M', ContextLong: "mike"},
	}

	sorted := sortSubContexts(subContexts)

	if len(sorted) != 3 {
		t.Fatalf("Expected 3 contexts, got %d", len(sorted))
	}

	if sorted[0].Context != 'A' {
		t.Errorf("First context should be 'A', got %c", sorted[0].Context)
	}

	if sorted[1].Context != 'M' {
		t.Errorf("Second context should be 'M', got %c", sorted[1].Context)
	}

	if sorted[2].Context != 'Z' {
		t.Errorf("Third context should be 'Z', got %c", sorted[2].Context)
	}
}

func TestShowFlagHelp_Variations(t *testing.T) {
	tests := []struct {
		name     string
		flag     Flag
		contains []string
	}{
		{
			name: "short and long with arg",
			flag: Flag{
				Short:       'f',
				Long:        "format",
				ArgName:     "fmt",
				Description: "Format string",
			},
			contains: []string{"-f", "--format <fmt>", "Format string"},
		},
		{
			name: "short and long without arg",
			flag: Flag{
				Short:       'v',
				Long:        "verbose",
				Description: "Enable verbose output",
			},
			contains: []string{"-v", "--verbose", "Enable verbose output"},
		},
		{
			name: "long only with arg",
			flag: Flag{
				Long:        "config",
				ArgName:     "file",
				Description: "Config file path",
			},
			contains: []string{"--config <file>", "Config file path"},
		},
		{
			name: "short only with arg",
			flag: Flag{
				Short:       'o',
				ArgName:     "output",
				Description: "Output file",
			},
			contains: []string{"-o <output>", "Output file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			showFlagHelp(tt.flag)

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got: %q", expected, output)
				}
			}
		})
	}
}
