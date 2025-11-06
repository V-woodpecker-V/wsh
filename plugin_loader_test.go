package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetPluginDir_Default(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("WSH_PLUGIN_DIR")

	dir := GetPluginDir()
	if dir != "./plugins" {
		t.Errorf("GetPluginDir() = %s, want ./plugins", dir)
	}
}

func TestGetPluginDir_EnvVar(t *testing.T) {
	customDir := "/custom/plugins"
	os.Setenv("WSH_PLUGIN_DIR", customDir)
	defer os.Unsetenv("WSH_PLUGIN_DIR")

	dir := GetPluginDir()
	if dir != customDir {
		t.Errorf("GetPluginDir() = %s, want %s", dir, customDir)
	}
}

func TestFindPluginScripts_NonExistent(t *testing.T) {
	scripts, err := FindPluginScripts("/nonexistent/directory")
	if err != nil {
		t.Errorf("FindPluginScripts() should not error on non-existent directory, got: %v", err)
	}

	if len(scripts) != 0 {
		t.Errorf("Expected 0 scripts for non-existent directory, got %d", len(scripts))
	}
}

func TestFindPluginScripts_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	scripts, err := FindPluginScripts(tmpDir)
	if err != nil {
		t.Fatalf("FindPluginScripts() error = %v", err)
	}

	if len(scripts) != 0 {
		t.Errorf("Expected 0 scripts in empty directory, got %d", len(scripts))
	}
}

func TestFindPluginScripts_WithScripts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create executable scripts
	script1 := filepath.Join(tmpDir, "plugin1.sh")
	script2 := filepath.Join(tmpDir, "plugin2.sh")
	nonExec := filepath.Join(tmpDir, "readme.txt")
	hidden := filepath.Join(tmpDir, ".hidden.sh")

	// Create files
	if err := os.WriteFile(script1, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script2, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nonExec, []byte("readme"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hidden, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create subdirectory
	if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	scripts, err := FindPluginScripts(tmpDir)
	if err != nil {
		t.Fatalf("FindPluginScripts() error = %v", err)
	}

	// Should find only the 2 executable scripts (not hidden, not subdir, not non-exec)
	if len(scripts) != 2 {
		t.Errorf("Expected 2 scripts, got %d", len(scripts))
	}

	// Verify paths are absolute
	for _, script := range scripts {
		if !filepath.IsAbs(script) {
			t.Errorf("Expected absolute path, got: %s", script)
		}
	}
}

func TestExecutePlugin_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple script that outputs valid JSON
	script := filepath.Join(tmpDir, "test.sh")
	content := `#!/bin/bash
echo '{"Context":84,"ContextLong":"test","Description":"Test plugin","Script":"` + script + `","Flags":null,"SubContexts":null}'
exit 0
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	ctx, err := executePlugin(script, "/bin/wsh", 5*time.Second)
	if err != nil {
		t.Errorf("executePlugin() error = %v", err)
	}

	if ctx == nil {
		t.Error("Expected context to be returned")
	}

	if ctx != nil && ctx.Context != 'T' {
		t.Errorf("Context = %c, want T", ctx.Context)
	}
}

func TestExecutePlugin_Timeout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a script that sleeps
	script := filepath.Join(tmpDir, "slow.sh")
	content := `#!/bin/bash
sleep 10
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	ctx, err := executePlugin(script, "/bin/wsh", 100*time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error")
	}

	if ctx != nil {
		t.Error("Expected nil context on timeout")
	}
}

func TestLoadPlugins_NoDirectory(t *testing.T) {
	registry := NewPluginRegistry()

	// Use non-existent directory
	os.Setenv("WSH_PLUGIN_DIR", "/nonexistent/plugins")
	defer os.Unsetenv("WSH_PLUGIN_DIR")

	err := LoadPlugins(registry, "/bin/wsh", 5*time.Second)
	if err != nil {
		t.Errorf("LoadPlugins() should not error on non-existent directory, got: %v", err)
	}
}

func TestLoadPlugins_EmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	registry := NewPluginRegistry()

	os.Setenv("WSH_PLUGIN_DIR", tmpDir)
	defer os.Unsetenv("WSH_PLUGIN_DIR")

	err = LoadPlugins(registry, "/bin/wsh", 5*time.Second)
	if err != nil {
		t.Errorf("LoadPlugins() error = %v", err)
	}
}

func TestLoadPlugins_Integration(t *testing.T) {
	// Skip if wsh binary doesn't exist yet
	wshBinary, err := filepath.Abs("./wsh")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(wshBinary); os.IsNotExist(err) {
		t.Skip("wsh binary not built yet")
	}

	tmpDir, err := os.MkdirTemp("", "plugin_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test plugin that registers itself
	plugin := filepath.Join(tmpDir, "test_plugin.sh")
	content := `#!/bin/bash
# Test plugin
$WSH_BINARY args --register \
    -T --test "Test plugin" \
    -x --example "Example flag"
`
	if err := os.WriteFile(plugin, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}

	registry := NewPluginRegistry()

	os.Setenv("WSH_PLUGIN_DIR", tmpDir)
	defer os.Unsetenv("WSH_PLUGIN_DIR")

	err = LoadPlugins(registry, wshBinary, 5*time.Second)
	if err != nil {
		t.Fatalf("LoadPlugins() error = %v", err)
	}

	// Verify plugin was registered
	ctx := registry.Lookup([]rune{'T'})
	if ctx == nil {
		t.Fatal("Expected test plugin to be registered")
	}

	if ctx.ContextLong != "test" {
		t.Errorf("ContextLong = %s, want test", ctx.ContextLong)
	}

	if len(ctx.Flags) != 1 {
		t.Errorf("Expected 1 flag, got %d", len(ctx.Flags))
	}
}

func TestLoadPlugins_Parallel(t *testing.T) {
	// Skip if wsh binary doesn't exist yet
	wshBinary, err := filepath.Abs("./wsh")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(wshBinary); os.IsNotExist(err) {
		t.Skip("wsh binary not built yet")
	}

	tmpDir, err := os.MkdirTemp("", "plugin_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple plugins
	for i := 1; i <= 3; i++ {
		plugin := filepath.Join(tmpDir, fmt.Sprintf("plugin%d.sh", i))
		content := fmt.Sprintf(`#!/bin/bash
$WSH_BINARY args --register \
    -%c --plugin%d "Plugin %d"
`, rune('P'+i-1), i, i)
		if err := os.WriteFile(plugin, []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	}

	registry := NewPluginRegistry()

	os.Setenv("WSH_PLUGIN_DIR", tmpDir)
	defer os.Unsetenv("WSH_PLUGIN_DIR")

	err = LoadPlugins(registry, wshBinary, 5*time.Second)
	if err != nil {
		t.Fatalf("LoadPlugins() error = %v", err)
	}

	// Verify all plugins were registered
	contexts := registry.GetAllContexts()
	if len(contexts) != 3 {
		t.Errorf("Expected 3 contexts, got %d", len(contexts))
	}
}
