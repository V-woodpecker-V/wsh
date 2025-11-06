package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ScriptExecutor executes a single script and returns its environment
type ScriptExecutor func(zshPath, scriptPath string) (map[string]string, error)

// ExecutionStrategy executes multiple scripts and returns merged environment
type ExecutionStrategy func(zshPath string, scripts []string, executor ScriptExecutor) (map[string]string, error)

// ScriptMiddleware wraps a ScriptExecutor with additional functionality
type ScriptMiddleware func(ScriptExecutor) ScriptExecutor

// WshrcLoader handles loading and processing .wshrc files and directories
type WshrcLoader struct {
	ZshPath        string
	Env            *Environment
	Strategy       ExecutionStrategy
	ScriptExecutor ScriptExecutor
}

// WshrcLoaderOption configures a WshrcLoader instance
type WshrcLoaderOption func(*WshrcLoader) error

// NewWshrcLoader creates a new WshrcLoader with optional configuration
func NewWshrcLoader(zshPath string, opts ...WshrcLoaderOption) *WshrcLoader {
	loader := &WshrcLoader{
		ZshPath:        zshPath,
		Env:            NewEnvironment(),
		Strategy:       ParallelExecutionStrategy,
		ScriptExecutor: defaultScriptExecutor,
	}

	// Apply options
	for _, opt := range opts {
		_ = opt(loader) // Ignoring errors for now, can be improved
	}

	return loader
}

// WshrcLoader options

// WithExecutionStrategy sets the strategy for executing multiple scripts
func WithExecutionStrategy(strategy ExecutionStrategy) WshrcLoaderOption {
	return func(w *WshrcLoader) error {
		if strategy == nil {
			return fmt.Errorf("execution strategy cannot be nil")
		}
		w.Strategy = strategy
		return nil
	}
}

// WithScriptExecutor sets the base script executor
func WithScriptExecutor(executor ScriptExecutor) WshrcLoaderOption {
	return func(w *WshrcLoader) error {
		if executor == nil {
			return fmt.Errorf("script executor cannot be nil")
		}
		w.ScriptExecutor = executor
		return nil
	}
}

// WithMiddleware wraps the script executor with middleware (applied in reverse order)
func WithMiddleware(middleware ...ScriptMiddleware) WshrcLoaderOption {
	return func(w *WshrcLoader) error {
		for i := len(middleware) - 1; i >= 0; i-- {
			w.ScriptExecutor = middleware[i](w.ScriptExecutor)
		}
		return nil
	}
}

// Load processes the .wshrc file or directory and returns the initialization script
func (w *WshrcLoader) Load(wshrcPath string) (string, error) {
	info, err := os.Stat(wshrcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("error reading .wshrc: %w", err)
	}

	if info.IsDir() {
		return w.loadDirectory(wshrcPath)
	}

	return w.loadFile(wshrcPath), nil
}

// loadFile returns a script to source a single .wshrc file
func (w *WshrcLoader) loadFile(path string) string {
	return fmt.Sprintf("source %s 2>/dev/null", path)
}

// loadDirectory processes a .wshrc directory by executing all scripts
// Special files: _pre.sh runs first, _post.sh runs last, others run in parallel
func (w *WshrcLoader) loadDirectory(dirPath string) (string, error) {
	allScripts, err := w.findScripts(dirPath)
	if err != nil {
		return "", fmt.Errorf("error reading directory: %w", err)
	}

	if len(allScripts) == 0 {
		return "", nil
	}

	// Separate special files from regular scripts
	var preScript, postScript string
	var regularScripts []string

	for _, script := range allScripts {
		name := filepath.Base(script)
		switch name {
		case "_pre.sh":
			preScript = script
		case "_post.sh":
			postScript = script
		default:
			regularScripts = append(regularScripts, script)
		}
	}

	currentEnv := w.Env.GetCurrent()
	var mergedEnv map[string]string

	// Execute _pre.sh first if it exists
	if preScript != "" {
		env, err := w.ScriptExecutor(w.ZshPath, preScript)
		if err != nil {
			return "", fmt.Errorf("error executing _pre.sh: %w", err)
		}
		currentEnv = env
	}

	// Execute regular scripts in parallel (or according to strategy)
	if len(regularScripts) > 0 {
		env, err := w.Strategy(w.ZshPath, regularScripts, w.ScriptExecutor)
		if err != nil {
			return "", err
		}
		// Merge with environment from _pre.sh
		for k, v := range env {
			currentEnv[k] = v
		}
	}

	// Execute _post.sh last if it exists
	if postScript != "" {
		env, err := w.ScriptExecutor(w.ZshPath, postScript)
		if err != nil {
			return "", fmt.Errorf("error executing _post.sh: %w", err)
		}
		mergedEnv = env
	} else {
		mergedEnv = currentEnv
	}

	return w.Env.BuildExportScript(w.Env.GetCurrent(), mergedEnv), nil
}

// findScripts returns all regular, non-hidden files in a directory
func (w *WshrcLoader) findScripts(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var scripts []string
	for _, entry := range entries {
		if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			scripts = append(scripts, filepath.Join(dirPath, entry.Name()))
		}
	}

	return scripts, nil
}

// Execution Strategies

// ParallelExecutionStrategy executes scripts concurrently
// TODO: Add proper Ctrl-C (SIGINT) handling to gracefully cancel long-running scripts
func ParallelExecutionStrategy(zshPath string, scripts []string, executor ScriptExecutor) (map[string]string, error) {
	var wg sync.WaitGroup
	envChan := make(chan map[string]string, len(scripts))
	errChan := make(chan error, len(scripts))

	for _, script := range scripts {
		wg.Add(1)
		go func(scriptPath string) {
			defer wg.Done()

			env, err := executor(zshPath, scriptPath)
			if err != nil {
				errChan <- fmt.Errorf("error executing %s: %w", scriptPath, err)
				return
			}
			envChan <- env
		}(script)
	}

	wg.Wait()
	close(envChan)
	close(errChan)

	if len(errChan) > 0 {
		return nil, <-errChan
	}

	return NewEnvironment().Merge(envChan), nil
}

// SequentialExecutionStrategy executes scripts one at a time (useful for debugging)
func SequentialExecutionStrategy(zshPath string, scripts []string, executor ScriptExecutor) (map[string]string, error) {
	merged := make(map[string]string)

	for _, script := range scripts {
		env, err := executor(zshPath, script)
		if err != nil {
			return nil, fmt.Errorf("error executing %s: %w", script, err)
		}

		// Merge this script's environment
		for k, v := range env {
			merged[k] = v
		}
	}

	return merged, nil
}

// Default script executor
func defaultScriptExecutor(zshPath, scriptPath string) (map[string]string, error) {
	return NewEnvironment().ExecuteAndCapture(zshPath, scriptPath)
}
