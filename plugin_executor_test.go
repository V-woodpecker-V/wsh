package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPluginExecution_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_exec_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple successful script
	script := filepath.Join(tmpDir, "test.sh")
	content := `#!/bin/bash
exit 0
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	ctx := &PluginContext{
		Context:     'T',
		ContextLong: "test",
		Script:      script,
	}

	exitCode := ExecutePlugin(ctx, map[string]string{}, []string{})
	if exitCode != 0 {
		t.Errorf("ExecutePlugin() exit code = %d, want 0", exitCode)
	}
}

func TestPluginExecution_NonZeroExit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_exec_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a script that exits with code 42
	script := filepath.Join(tmpDir, "test.sh")
	content := `#!/bin/bash
exit 42
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	ctx := &PluginContext{
		Context:     'T',
		ContextLong: "test",
		Script:      script,
	}

	exitCode := ExecutePlugin(ctx, map[string]string{}, []string{})
	if exitCode != 42 {
		t.Errorf("ExecutePlugin() exit code = %d, want 42", exitCode)
	}
}

func TestPluginExecution_MissingScript(t *testing.T) {
	ctx := &PluginContext{
		Context:     'T',
		ContextLong: "test",
		Script:      "",
	}

	exitCode := ExecutePlugin(ctx, map[string]string{}, []string{})
	if exitCode == 0 {
		t.Error("ExecutePlugin() should fail with empty script path")
	}
}

func TestPluginExecution_NonExistentScript(t *testing.T) {
	ctx := &PluginContext{
		Context:     'T',
		ContextLong: "test",
		Script:      "/nonexistent/script.sh",
	}

	exitCode := ExecutePlugin(ctx, map[string]string{}, []string{})
	if exitCode == 0 {
		t.Error("ExecutePlugin() should fail with non-existent script")
	}
}

func TestPluginExecution_WithFlags(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_exec_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a script that checks environment variables
	script := filepath.Join(tmpDir, "test.sh")
	content := `#!/bin/bash
if [ "$format" != "json" ]; then
    echo "Expected format=json, got format=$format" >&2
    exit 1
fi

if [ "$verbose" != "true" ]; then
    echo "Expected verbose=true, got verbose=$verbose" >&2
    exit 1
fi

exit 0
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	ctx := &PluginContext{
		Context:     'T',
		ContextLong: "test",
		Script:      script,
	}

	flags := map[string]string{
		"format":  "json",
		"verbose": "true",
	}

	exitCode := ExecutePlugin(ctx, flags, []string{})
	if exitCode != 0 {
		t.Errorf("ExecutePlugin() exit code = %d, want 0", exitCode)
	}
}

func TestPluginExecution_WithArgs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_exec_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a script that checks arguments
	script := filepath.Join(tmpDir, "test.sh")
	content := `#!/bin/bash
if [ "$#" -ne 3 ]; then
    echo "Expected 3 arguments, got $#" >&2
    exit 1
fi

if [ "$1" != "arg1" ] || [ "$2" != "arg2" ] || [ "$3" != "arg3" ]; then
    echo "Arguments mismatch: $*" >&2
    exit 1
fi

exit 0
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	ctx := &PluginContext{
		Context:     'T',
		ContextLong: "test",
		Script:      script,
	}

	args := []string{"arg1", "arg2", "arg3"}

	exitCode := ExecutePlugin(ctx, map[string]string{}, args)
	if exitCode != 0 {
		t.Errorf("ExecutePlugin() exit code = %d, want 0", exitCode)
	}
}

func TestPluginExecution_WithFlagsAndArgs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_exec_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a script that checks both env vars and arguments
	script := filepath.Join(tmpDir, "test.sh")
	content := `#!/bin/bash
if [ "$output" != "result.txt" ]; then
    echo "Expected output=result.txt, got output=$output" >&2
    exit 1
fi

if [ "$#" -ne 2 ]; then
    echo "Expected 2 arguments, got $#" >&2
    exit 1
fi

if [ "$1" != "file1" ] || [ "$2" != "file2" ]; then
    echo "Arguments mismatch: $*" >&2
    exit 1
fi

exit 0
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	ctx := &PluginContext{
		Context:     'T',
		ContextLong: "test",
		Script:      script,
	}

	flags := map[string]string{
		"output": "result.txt",
	}
	args := []string{"file1", "file2"}

	exitCode := ExecutePlugin(ctx, flags, args)
	if exitCode != 0 {
		t.Errorf("ExecutePlugin() exit code = %d, want 0", exitCode)
	}
}

func TestPluginExecution_FlagNamePreservation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_exec_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a script that verifies exact flag names (lowercase preserved)
	script := filepath.Join(tmpDir, "test.sh")
	content := `#!/bin/bash
# Check that lowercase flag names are preserved
if [ -z "$offline" ]; then
    echo "Expected 'offline' env var to be set" >&2
    exit 1
fi

if [ -z "$from" ]; then
    echo "Expected 'from' env var to be set" >&2
    exit 1
fi

exit 0
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	ctx := &PluginContext{
		Context:     'T',
		ContextLong: "test",
		Script:      script,
	}

	flags := map[string]string{
		"offline": "true",
		"from":    "5",
	}

	exitCode := ExecutePlugin(ctx, flags, []string{})
	if exitCode != 0 {
		t.Errorf("ExecutePlugin() exit code = %d, want 0", exitCode)
	}
}
