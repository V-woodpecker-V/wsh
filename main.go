package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Special case: handle "args" subcommand before any other processing
	if len(os.Args) > 1 && os.Args[1] == "args" {
		registry := NewPluginRegistry()
		exitCode := HandleArgs(registry, os.Args[2:])
		os.Exit(exitCode)
	}

	// Create shell instance
	shell, err := NewShell()
	if err != nil {
		fmt.Fprintf(os.Stderr, "wsh: %v\n", err)
		os.Exit(1)
	}

	// Register internal plugins
	if err := RegisterShellPlugin(shell.PluginRegistry); err != nil {
		fmt.Fprintf(os.Stderr, "wsh: failed to register shell plugin: %v\n", err)
		os.Exit(1)
	}

	if err := RegisterArgsPlugin(shell.PluginRegistry); err != nil {
		fmt.Fprintf(os.Stderr, "wsh: failed to register args plugin: %v\n", err)
		os.Exit(1)
	}

	// Parse command line arguments
	if len(os.Args) == 1 {
		// No arguments - run interactive shell
		// Load plugins before running shell
		loadExternalPlugins(shell.PluginRegistry)
		exitCode := shell.Run("", []string{})
		os.Exit(exitCode)
	}

	// Parse flags using plugin registry
	result, err := shell.PluginRegistry.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "wsh: %v\n", err)
		os.Exit(1)
	}

	// Handle help
	if result.ShowHelp {
		ShowHelp(shell.PluginRegistry, result.ContextPath)
		os.Exit(0)
	}

	// Handle shell plugin explicitly (-S or no context)
	// Plugins are a shell feature - only load them when running shell mode
	if result.Context == nil || result.Context.Context == 'S' {
		// Load plugins when using shell
		loadExternalPlugins(shell.PluginRegistry)

		// Check for -c/--command flag
		if cmdStr, hasCmd := result.Flags["command"]; hasCmd {
			exitCode := shell.Run(cmdStr, result.Args)
			os.Exit(exitCode)
		}

		// Check for -r/--reload flag
		if _, hasReload := result.Flags["reload"]; hasReload {
			// Reload: re-load .wshrc and plugins
			// For now, just print a message
			// TODO: Implement actual reload logic
			fmt.Println("Reload functionality coming soon...")
			os.Exit(0)
		}

		// No specific flags - run interactive shell
		exitCode := shell.Run("", result.Args)
		os.Exit(exitCode)
	}

	// Non-shell contexts are not supported outside of shell mode
	// Plugins are only available when using wsh as a shell (interactive or -c mode)
	if result.Context != nil {
		fmt.Fprintf(os.Stderr, "wsh: plugins are only available in shell mode\n")
		fmt.Fprintf(os.Stderr, "wsh: use 'wsh -c \"<your command>\"' to access plugins\n")
		os.Exit(1)
	}

	// Should never reach here
	fmt.Fprintf(os.Stderr, "wsh: internal error: no handler for parsed result\n")
	os.Exit(1)
}

// loadExternalPlugins loads external plugins from the plugin directory
func loadExternalPlugins(registry *PluginRegistry) {
	wshBinary, err := os.Executable()
	if err != nil {
		// Fallback to argv[0]
		wshBinary, _ = filepath.Abs(os.Args[0])
	}

	if err := LoadPlugins(registry, wshBinary, 10*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "wsh: warning: plugin loading failed: %v\n", err)
		// Continue anyway - plugins are optional
	}
}
