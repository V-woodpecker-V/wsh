package main

import (
	"os"
	"strings"
	"testing"
)

func TestRegisterArgsPlugin(t *testing.T) {
	registry := NewPluginRegistry()

	err := RegisterArgsPlugin(registry)
	if err != nil {
		t.Fatalf("RegisterArgsPlugin() error = %v", err)
	}

	// Verify registration
	ctx := registry.Lookup([]rune{'A'})
	if ctx == nil {
		t.Fatal("Expected args plugin to be registered")
	}

	if ctx.Context != 'A' {
		t.Errorf("Context = %c, want A", ctx.Context)
	}

	if ctx.ContextLong != "args" {
		t.Errorf("ContextLong = %s, want args", ctx.ContextLong)
	}
}

func TestParsePluginDefinition_Simple(t *testing.T) {
	args := []string{
		"-T", "--time", "Time tracking",
		"-o", "--offline", "Offline mode",
		"-f", "--from", "days", "Days ago",
	}

	ctx, err := parsePluginDefinition(args)
	if err != nil {
		t.Fatalf("parsePluginDefinition() error = %v", err)
	}

	if ctx.Context != 'T' {
		t.Errorf("Context = %c, want T", ctx.Context)
	}

	if ctx.ContextLong != "time" {
		t.Errorf("ContextLong = %s, want time", ctx.ContextLong)
	}

	if ctx.Description != "Time tracking" {
		t.Errorf("Description = %s, want 'Time tracking'", ctx.Description)
	}

	if len(ctx.Flags) != 2 {
		t.Fatalf("Expected 2 flags, got %d", len(ctx.Flags))
	}

	// Check first flag
	if ctx.Flags[0].Short != 'o' || ctx.Flags[0].Long != "offline" {
		t.Errorf("First flag = -%c/--%s, want -o/--offline", ctx.Flags[0].Short, ctx.Flags[0].Long)
	}

	// Check second flag
	if ctx.Flags[1].Short != 'f' || ctx.Flags[1].Long != "from" || ctx.Flags[1].ArgName != "days" {
		t.Errorf("Second flag = -%c/--%s <%s>, want -f/--from <days>",
			ctx.Flags[1].Short, ctx.Flags[1].Long, ctx.Flags[1].ArgName)
	}
}

func TestParsePluginDefinition_WithSubContext(t *testing.T) {
	args := []string{
		"-T", "--time", "Time tracking",
		"-o", "--offline", "Offline mode",
		"-O", "--overtime", "Overtime calculations",
		"-s", "--start", "time", "Start time",
		"-e", "--end", "time", "End time",
	}

	ctx, err := parsePluginDefinition(args)
	if err != nil {
		t.Fatalf("parsePluginDefinition() error = %v", err)
	}

	if len(ctx.Flags) != 1 {
		t.Errorf("Expected 1 top-level flag, got %d", len(ctx.Flags))
	}

	if len(ctx.SubContexts) != 1 {
		t.Fatalf("Expected 1 sub-context, got %d", len(ctx.SubContexts))
	}

	subCtx, exists := ctx.SubContexts['O']
	if !exists {
		t.Fatal("Expected sub-context 'O' to exist")
	}

	if subCtx.ContextLong != "overtime" {
		t.Errorf("SubContext long = %s, want overtime", subCtx.ContextLong)
	}

	if len(subCtx.Flags) != 2 {
		t.Errorf("Expected 2 flags in sub-context, got %d", len(subCtx.Flags))
	}
}

func TestParseFlag_ShortAndLong(t *testing.T) {
	args := []string{"-o", "--offline", "Offline mode"}

	flag, consumed, err := parseFlag(args)
	if err != nil {
		t.Fatalf("parseFlag() error = %v", err)
	}

	if consumed != 3 {
		t.Errorf("consumed = %d, want 3", consumed)
	}

	if flag.Short != 'o' {
		t.Errorf("Short = %c, want o", flag.Short)
	}

	if flag.Long != "offline" {
		t.Errorf("Long = %s, want offline", flag.Long)
	}

	if flag.Description != "Offline mode" {
		t.Errorf("Description = %s, want 'Offline mode'", flag.Description)
	}
}

func TestParseFlag_WithArg(t *testing.T) {
	args := []string{"-f", "--from", "days", "Days ago"}

	flag, consumed, err := parseFlag(args)
	if err != nil {
		t.Fatalf("parseFlag() error = %v", err)
	}

	if consumed != 4 {
		t.Errorf("consumed = %d, want 4", consumed)
	}

	if flag.ArgName != "days" {
		t.Errorf("ArgName = %s, want days", flag.ArgName)
	}
}

func TestParseFlag_LongOnly(t *testing.T) {
	args := []string{"--offline", "Offline mode"}

	flag, consumed, err := parseFlag(args)
	if err != nil {
		t.Fatalf("parseFlag() error = %v", err)
	}

	if consumed != 2 {
		t.Errorf("consumed = %d, want 2", consumed)
	}

	if flag.Short != 0 {
		t.Errorf("Short = %c, want 0", flag.Short)
	}

	if flag.Long != "offline" {
		t.Errorf("Long = %s, want offline", flag.Long)
	}
}

func TestParseFlag_ShortOnly(t *testing.T) {
	args := []string{"-o", "Offline mode"}

	flag, consumed, err := parseFlag(args)
	if err != nil {
		t.Fatalf("parseFlag() error = %v", err)
	}

	if consumed != 2 {
		t.Errorf("consumed = %d, want 2", consumed)
	}

	if flag.Short != 'o' {
		t.Errorf("Short = %c, want o", flag.Short)
	}

	if flag.Long != "" {
		t.Errorf("Long = %s, want empty", flag.Long)
	}
}

func TestHandleRegister_Integration(t *testing.T) {
	registry := NewPluginRegistry()

	// Set env var that plugins would set
	os.Setenv("WSH_PLUGIN_SCRIPT", "./plugins/time.sh")
	defer os.Unsetenv("WSH_PLUGIN_SCRIPT")

	args := []string{
		"--register",
		"-T", "--time", "Time tracking",
		"-o", "--offline", "Offline mode",
	}

	exitCode := HandleArgs(registry, args)
	if exitCode != 0 {
		t.Errorf("HandleArgs() exit code = %d, want 0", exitCode)
	}

	// Verify registration
	ctx := registry.Lookup([]rune{'T'})
	if ctx == nil {
		t.Fatal("Expected plugin to be registered")
	}

	if ctx.Context != 'T' {
		t.Errorf("Context = %c, want T", ctx.Context)
	}

	// Test idempotent registration
	exitCode = HandleArgs(registry, args)
	if exitCode != 0 {
		t.Errorf("Second HandleArgs() exit code = %d, want 0 (idempotent)", exitCode)
	}
}

func TestHandleRegister_Idempotent(t *testing.T) {
	registry := NewPluginRegistry()

	os.Setenv("WSH_PLUGIN_SCRIPT", "./plugins/test.sh")
	defer os.Unsetenv("WSH_PLUGIN_SCRIPT")

	args := []string{
		"--register",
		"-T", "--time", "Time tracking",
	}

	// Register twice
	exitCode1 := HandleArgs(registry, args)
	exitCode2 := HandleArgs(registry, args)

	if exitCode1 != 0 {
		t.Errorf("First HandleArgs() exit code = %d, want 0", exitCode1)
	}

	if exitCode2 != 0 {
		t.Errorf("Second HandleArgs() exit code = %d, want 0", exitCode2)
	}

	// Should only have one registration
	contexts := registry.GetAllContexts()
	if len(contexts) != 1 {
		t.Errorf("Expected 1 context after duplicate registration, got %d", len(contexts))
	}
}

func TestHandleParse_OutputsEnvVars(t *testing.T) {
	registry := NewPluginRegistry()

	// Register a test plugin
	ctx := &PluginContext{
		Context:     'T',
		ContextLong: "time",
		Flags: []Flag{
			{Short: 'o', Long: "offline", Description: "Offline"},
			{Short: 'f', Long: "from", ArgName: "days", Description: "Days"},
		},
	}
	registry.Register(ctx)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	args := []string{"-Tof", "5"}
	HandleArgs(registry, args)

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check output contains env vars
	if !strings.Contains(output, "offline=true") {
		t.Errorf("Expected output to contain 'offline=true', got: %s", output)
	}

	if !strings.Contains(output, "from=5") {
		t.Errorf("Expected output to contain 'from=5', got: %s", output)
	}
}

func TestParsePluginDefinition_InvalidContext(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "lowercase context",
			args: []string{"-t", "--time", "Time"},
		},
		{
			name: "no description",
			args: []string{"-T", "--time"},
		},
		{
			name: "no context flag",
			args: []string{"Time"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePluginDefinition(tt.args)
			if err == nil {
				t.Error("Expected error for invalid definition")
			}
		})
	}
}
