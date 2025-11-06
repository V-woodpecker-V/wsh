package main

import (
	"fmt"
	"os"
	"os/exec"
)

// ExecutePlugin executes a plugin with the given context, flags, and arguments
func ExecutePlugin(ctx *PluginContext, flags map[string]string, args []string) int {
	// Verify script path exists
	if ctx.Script == "" {
		fmt.Fprintf(os.Stderr, "wsh: internal error: no script for context %c\n", ctx.Context)
		return 1
	}

	// Check if script is executable
	if _, err := os.Stat(ctx.Script); err != nil {
		fmt.Fprintf(os.Stderr, "wsh: plugin script not found: %s\n", ctx.Script)
		return 1
	}

	// Create command
	cmd := exec.Command(ctx.Script, args...)

	// Set environment variables for flags
	cmd.Env = os.Environ()
	for flagName, flagValue := range flags {
		envVar := fmt.Sprintf("%s=%s", flagName, flagValue)
		cmd.Env = append(cmd.Env, envVar)
	}

	// Connect stdio
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Execute
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "wsh: failed to execute plugin: %v\n", err)
		return 1
	}

	return 0
}
