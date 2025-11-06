package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// GetPluginDir returns the plugin directory path
// Checks WSH_PLUGIN_DIR env var, falls back to ./plugins
// TODO: Change default to ~/.config/wsh/plugins
func GetPluginDir() string {
	if dir := os.Getenv("WSH_PLUGIN_DIR"); dir != "" {
		return dir
	}
	// TODO: Change to ~/.config/wsh/plugins
	return "./plugins"
}

// FindPluginScripts discovers all executable scripts in the plugin directory
func FindPluginScripts(dir string) ([]string, error) {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Plugin directory doesn't exist, that's okay
			return []string{}, nil
		}
		return nil, fmt.Errorf("error accessing plugin directory: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	// Read directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error reading plugin directory: %w", err)
	}

	var scripts []string
	for _, entry := range entries {
		// Skip directories and hidden files
		if entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}

		// Get full path
		fullPath := filepath.Join(dir, entry.Name())

		// Check if executable
		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}

		// On Unix, check executable bit
		if fileInfo.Mode()&0111 != 0 {
			scripts = append(scripts, fullPath)
		}
	}

	return scripts, nil
}

// LoadPlugins loads all plugins by executing them in parallel
// Each plugin script should call: wsh args --register ...
func LoadPlugins(registry *PluginRegistry, wshBinary string, timeout time.Duration) error {
	pluginDir := GetPluginDir()

	scripts, err := FindPluginScripts(pluginDir)
	if err != nil {
		return fmt.Errorf("error finding plugins: %w", err)
	}

	if len(scripts) == 0 {
		// No plugins found, that's okay
		return nil
	}

	// Execute all plugins in parallel
	var wg sync.WaitGroup
	ctxChan := make(chan *PluginContext, len(scripts))
	errChan := make(chan error, len(scripts))

	for _, script := range scripts {
		wg.Add(1)
		go func(scriptPath string) {
			defer wg.Done()

			ctx, err := executePlugin(scriptPath, wshBinary, timeout)
			if err != nil {
				errChan <- fmt.Errorf("plugin %s: %w", filepath.Base(scriptPath), err)
				return
			}

			if ctx != nil {
				ctxChan <- ctx
			}
		}(script)
	}

	// Wait for all plugins to complete
	wg.Wait()
	close(errChan)
	close(ctxChan)

	// Register all collected contexts
	for ctx := range ctxChan {
		if err := registry.Register(ctx); err != nil {
			// Ignore registration errors (already handled by plugin)
			fmt.Fprintf(os.Stderr, "wsh: warning: failed to register plugin %c: %v\n", ctx.Context, err)
		}
	}

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		// Return first error (could collect all if needed)
		return errors[0]
	}

	return nil
}

// executePlugin executes a single plugin script and returns the registered context
// The script should call: wsh args --register ...
func executePlugin(scriptPath, wshBinary string, timeout time.Duration) (*PluginContext, error) {
	// Create command to execute the plugin script
	cmd := exec.Command(scriptPath)

	// Set environment variable so plugin knows its own path
	cmd.Env = append(os.Environ(), fmt.Sprintf("WSH_PLUGIN_SCRIPT=%s", scriptPath))
	cmd.Env = append(cmd.Env, fmt.Sprintf("WSH_BINARY=%s", wshBinary))

	// Capture stdout to parse JSON output
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	// Create timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-time.After(timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return nil, fmt.Errorf("plugin execution timed out after %v", timeout)
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("plugin execution failed: %w", err)
		}
	}

	// Parse JSON output
	var ctx PluginContext
	if err := json.Unmarshal(stdout.Bytes(), &ctx); err != nil {
		return nil, fmt.Errorf("failed to parse plugin output: %w", err)
	}

	return &ctx, nil
}
