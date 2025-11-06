package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Shell represents a wsh shell instance
type Shell struct {
	ZshPath       string
	WshrcPath     string
	Env           *Environment
	WshrcLoader   *WshrcLoader
	PluginRegistry *PluginRegistry
}

// ShellOption configures a Shell instance
type ShellOption func(*Shell) error

// NewShell creates a new Shell instance with optional configuration
func NewShell(opts ...ShellOption) (*Shell, error) {
	zshPath, err := findZsh()
	if err != nil {
		return nil, fmt.Errorf("zsh not found: %w", err)
	}

	wshrcPath, err := getDefaultWshrcPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get wshrc path: %w", err)
	}

	shell := &Shell{
		ZshPath:        zshPath,
		WshrcPath:      wshrcPath,
		Env:            NewEnvironment(),
		WshrcLoader:    NewWshrcLoader(zshPath),
		PluginRegistry: NewPluginRegistry(),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(shell); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	return shell, nil
}

// RegisterShellPlugin registers the shell plugin (-S/--shell) as an internal plugin
func RegisterShellPlugin(registry *PluginRegistry) error {
	shellCtx := &PluginContext{
		Context:     'S',
		ContextLong: "shell",
		Description: "Shell operations (alias for bare wsh)",
		Script:      "", // Internal plugin, no script
		Flags: []Flag{
			{Short: 'c', Long: "command", ArgName: "cmd", Description: "Execute command and exit"},
			{Short: 'r', Long: "reload", Description: "Reload .wshrc and plugins"},
		},
	}

	return registry.Register(shellCtx)
}

// Shell options

// WithZshPath sets a custom zsh executable path
func WithZshPath(path string) ShellOption {
	return func(s *Shell) error {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("zsh path does not exist: %s", path)
		}
		s.ZshPath = path
		// Update the loader with the new zsh path
		s.WshrcLoader.ZshPath = path
		return nil
	}
}

// WithWshrcPath sets a custom .wshrc location
func WithWshrcPath(path string) ShellOption {
	return func(s *Shell) error {
		s.WshrcPath = path
		return nil
	}
}

// WithEnvironment sets a custom environment handler
func WithEnvironment(env *Environment) ShellOption {
	return func(s *Shell) error {
		if env == nil {
			return fmt.Errorf("environment cannot be nil")
		}
		s.Env = env
		return nil
	}
}

// WithWshrcLoader sets a custom wshrc loader
func WithWshrcLoader(loader *WshrcLoader) ShellOption {
	return func(s *Shell) error {
		if loader == nil {
			return fmt.Errorf("wshrc loader cannot be nil")
		}
		s.WshrcLoader = loader
		return nil
	}
}

// Run executes the shell with the given command and arguments
func (s *Shell) Run(cmdToExecute string, args []string) int {
	initScript, err := s.WshrcLoader.Load(s.WshrcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wsh: error loading .wshrc: %v\n", err)
		return 1
	}

	script := s.buildScript(initScript, cmdToExecute)
	return s.executeZsh(script, args)
}

// buildScript constructs the final script to execute
func (s *Shell) buildScript(initScript, cmdToExecute string) string {
	if cmdToExecute != "" {
		return fmt.Sprintf("%s\n%s", initScript, cmdToExecute)
	}
	return fmt.Sprintf("%s\nexec zsh", initScript)
}

// executeZsh runs zsh with the given script
func (s *Shell) executeZsh(script string, args []string) int {
	cmd := exec.Command(s.ZshPath, "-c", script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if len(args) > 0 {
		cmd.Args = append(cmd.Args, args...)
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}

	return 0
}

// Helper functions

// findZsh locates the zsh executable
func findZsh() (string, error) {
	zshPath, err := exec.LookPath("zsh")
	if err != nil {
		// Try fallback path
		fallback := "/bin/zsh"
		if _, err := os.Stat(fallback); err == nil {
			return fallback, nil
		}
		return "", err
	}
	return zshPath, nil
}

// getDefaultWshrcPath returns the default .wshrc path
func getDefaultWshrcPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".wshrc"), nil
}
