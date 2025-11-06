package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewShell(t *testing.T) {
	shell, err := NewShell()
	if err != nil {
		t.Fatalf("NewShell() error = %v", err)
	}

	if shell.ZshPath == "" {
		t.Error("Expected ZshPath to be set")
	}

	if shell.WshrcPath == "" {
		t.Error("Expected WshrcPath to be set")
	}

	if shell.Env == nil {
		t.Error("Expected Env to be initialized")
	}

	if shell.WshrcLoader == nil {
		t.Error("Expected WshrcLoader to be initialized")
	}

	// Check that wshrc path is in home directory
	homeDir, _ := os.UserHomeDir()
	expectedPath := filepath.Join(homeDir, ".wshrc")
	if shell.WshrcPath != expectedPath {
		t.Errorf("WshrcPath = %q, want %q", shell.WshrcPath, expectedPath)
	}
}

func TestNewShell_WithOptions(t *testing.T) {
	customWshrcPath := "/tmp/custom_wshrc"

	shell, err := NewShell(
		WithWshrcPath(customWshrcPath),
	)
	if err != nil {
		t.Fatalf("NewShell() error = %v", err)
	}

	if shell.WshrcPath != customWshrcPath {
		t.Errorf("WshrcPath = %q, want %q", shell.WshrcPath, customWshrcPath)
	}
}

func TestNewShell_WithInvalidZshPath(t *testing.T) {
	_, err := NewShell(
		WithZshPath("/nonexistent/zsh"),
	)
	if err == nil {
		t.Error("Expected error for non-existent zsh path")
	}
}

func TestWithZshPath(t *testing.T) {
	zshPath, err := findZsh()
	if err != nil {
		t.Skip("zsh not available")
	}

	shell, err := NewShell(
		WithZshPath(zshPath),
	)
	if err != nil {
		t.Fatalf("NewShell() error = %v", err)
	}

	if shell.ZshPath != zshPath {
		t.Errorf("ZshPath = %q, want %q", shell.ZshPath, zshPath)
	}

	// Verify loader also has the correct zsh path
	if shell.WshrcLoader.ZshPath != zshPath {
		t.Errorf("WshrcLoader.ZshPath = %q, want %q", shell.WshrcLoader.ZshPath, zshPath)
	}
}

func TestWithEnvironment(t *testing.T) {
	customEnv := NewEnvironment()

	shell, err := NewShell(
		WithEnvironment(customEnv),
	)
	if err != nil {
		t.Fatalf("NewShell() error = %v", err)
	}

	if shell.Env != customEnv {
		t.Error("Expected custom environment to be set")
	}
}

func TestWithEnvironment_Nil(t *testing.T) {
	_, err := NewShell(
		WithEnvironment(nil),
	)
	if err == nil {
		t.Error("Expected error for nil environment")
	}
}

func TestShell_BuildScript(t *testing.T) {
	shell, err := NewShell()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		initScript     string
		cmdToExecute   string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:           "command mode",
			initScript:     "export FOO=bar",
			cmdToExecute:   "echo hello",
			wantContains:   []string{"export FOO=bar", "echo hello"},
			wantNotContain: []string{"exec zsh"},
		},
		{
			name:           "interactive mode",
			initScript:     "export FOO=bar",
			cmdToExecute:   "",
			wantContains:   []string{"export FOO=bar", "exec zsh"},
			wantNotContain: []string{},
		},
		{
			name:           "empty init script",
			initScript:     "",
			cmdToExecute:   "pwd",
			wantContains:   []string{"pwd"},
			wantNotContain: []string{"exec zsh"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := shell.buildScript(tt.initScript, tt.cmdToExecute)

			for _, want := range tt.wantContains {
				if !strings.Contains(script, want) {
					t.Errorf("buildScript() should contain %q, got: %q", want, script)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(script, notWant) {
					t.Errorf("buildScript() should not contain %q, got: %q", notWant, script)
				}
			}
		})
	}
}

func TestFindZsh(t *testing.T) {
	zshPath, err := findZsh()
	if err != nil {
		t.Skip("zsh not available in test environment")
	}

	if zshPath == "" {
		t.Error("findZsh() returned empty path")
	}

	// Check that the path exists
	if _, err := os.Stat(zshPath); err != nil {
		t.Errorf("zsh path does not exist: %s", zshPath)
	}
}

func TestGetDefaultWshrcPath(t *testing.T) {
	path, err := getDefaultWshrcPath()
	if err != nil {
		t.Fatalf("getDefaultWshrcPath() error = %v", err)
	}

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".wshrc")

	if path != expected {
		t.Errorf("getDefaultWshrcPath() = %q, want %q", path, expected)
	}
}

// Integration test for Run method
func TestShell_Run_WithCommand(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	shell, err := NewShell()
	if err != nil {
		t.Fatal(err)
	}

	// Run a simple command
	exitCode := shell.Run("echo test", []string{})
	if exitCode != 0 {
		t.Errorf("Run() exit code = %d, want 0", exitCode)
	}
}

func TestShell_Run_WithFailingCommand(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	shell, err := NewShell()
	if err != nil {
		t.Fatal(err)
	}

	// Run a command that fails
	exitCode := shell.Run("exit 42", []string{})
	if exitCode != 42 {
		t.Errorf("Run() exit code = %d, want 42", exitCode)
	}
}

func TestRegisterShellPlugin(t *testing.T) {
	registry := NewPluginRegistry()

	err := RegisterShellPlugin(registry)
	if err != nil {
		t.Fatalf("RegisterShellPlugin() error = %v", err)
	}

	// Verify registration
	ctx := registry.Lookup([]rune{'S'})
	if ctx == nil {
		t.Fatal("Expected shell plugin to be registered")
	}

	if ctx.Context != 'S' {
		t.Errorf("Context = %c, want S", ctx.Context)
	}

	if ctx.ContextLong != "shell" {
		t.Errorf("ContextLong = %s, want shell", ctx.ContextLong)
	}

	// Check flags
	if len(ctx.Flags) != 2 {
		t.Fatalf("Expected 2 flags, got %d", len(ctx.Flags))
	}

	// Verify command flag exists
	hasCommand := false
	hasReload := false
	for _, flag := range ctx.Flags {
		if flag.Short == 'c' && flag.Long == "command" {
			hasCommand = true
		}
		if flag.Short == 'r' && flag.Long == "reload" {
			hasReload = true
		}
	}

	if !hasCommand {
		t.Error("Expected command flag (-c/--command) to exist")
	}

	if !hasReload {
		t.Error("Expected reload flag (-r/--reload) to exist")
	}
}
