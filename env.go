package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Environment handles environment variable operations
type Environment struct{}

// NewEnvironment creates a new Environment instance
func NewEnvironment() *Environment {
	return &Environment{}
}

// GetCurrent returns the current process environment as a map
func (e *Environment) GetCurrent() map[string]string {
	env := make(map[string]string)
	for _, envVar := range os.Environ() {
		if key, value, ok := e.parseEnvLine(envVar); ok {
			env[key] = value
		}
	}
	return env
}

// ExecuteAndCapture sources a script and captures its resulting environment
func (e *Environment) ExecuteAndCapture(zshPath, scriptPath string) (map[string]string, error) {
	// Redirect script output to stderr so it's visible to user
	// while env -0 output goes to stdout for capture
	script := fmt.Sprintf("source %s >&2 && env -0", scriptPath)

	cmd := exec.Command(zshPath, "-c", script)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return e.parseNullDelimited(&out)
}

// BuildExportScript creates export statements for new or changed environment variables
func (e *Environment) BuildExportScript(current, new map[string]string) string {
	var builder strings.Builder

	for key, value := range new {
		if currentVal, exists := current[key]; !exists || currentVal != value {
			builder.WriteString(e.buildExportLine(key, value))
		}
	}

	return builder.String()
}

// Merge combines multiple environment maps into one
func (e *Environment) Merge(envChan chan map[string]string) map[string]string {
	merged := make(map[string]string)
	for env := range envChan {
		for key, value := range env {
			merged[key] = value
		}
	}
	return merged
}

// parseEnvLine splits an environment line into key and value
func (e *Environment) parseEnvLine(line string) (string, string, bool) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return "", "", false
}

// buildExportLine creates a single export statement with proper escaping
func (e *Environment) buildExportLine(key, value string) string {
	escapedValue := strings.ReplaceAll(value, "'", "'\"'\"'")
	return fmt.Sprintf("export %s='%s'\n", key, escapedValue)
}

// parseNullDelimited parses null-delimited environment output from env -0
func (e *Environment) parseNullDelimited(reader *bytes.Buffer) (map[string]string, error) {
	env := make(map[string]string)
	scanner := bufio.NewScanner(reader)
	scanner.Split(scanNullTerminated)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if key, value, ok := e.parseEnvLine(line); ok {
			env[key] = value
		}
	}

	return env, scanner.Err()
}

// scanNullTerminated is a split function for bufio.Scanner that splits on null bytes
func scanNullTerminated(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.IndexByte(data, 0); i >= 0 {
		return i + 1, data[0:i], nil
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}
