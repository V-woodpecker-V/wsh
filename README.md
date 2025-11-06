# wsh - Woodpecker Shell

A zsh wrapper with support for modular configuration through `.wshrc` files or directories, and a powerful plugin system for extending functionality.

## Project Structure

```
wsh/
├── main.go                 # Entry point and CLI routing
├── shell.go                # Shell struct and execution logic
├── wshrc.go                # .wshrc loading and processing
├── env.go                  # Environment variable manipulation
├── middleware.go           # Middleware for script execution
├── plugin.go               # Plugin registry and parsing
├── args.go                 # Args subcommand for plugin registration
├── plugin_loader.go        # External plugin discovery and loading
├── plugin_executor.go      # Plugin script execution
├── help.go                 # Context-aware help generation
├── *_test.go               # Comprehensive unit and integration tests
├── README.md               # This file
├── EXAMPLES.md             # Usage examples
└── plugins/                # External plugin directory
```

## Architecture

### Core Components

#### 1. Shell (`shell.go`)
The main shell orchestrator that:
- Manages zsh path and wshrc configuration
- Coordinates initialization and execution
- Handles command vs interactive mode
- Manages exit codes

**Key struct:**
```go
type Shell struct {
    ZshPath     string        // Path to zsh executable
    WshrcPath   string        // Path to .wshrc (file or directory)
    Env         *Environment  // Environment handler
    WshrcLoader *WshrcLoader  // Configuration loader
}
```

#### 2. WshrcLoader (`wshrc.go`)
Handles loading and processing of `.wshrc` configurations:
- Detects if `.wshrc` is a file or directory
- For files: generates a simple source command
- For directories: executes all scripts in parallel and merges environments
- Filters hidden files and subdirectories

**Key methods:**
- `Load(path)` - Main entry point for loading configuration
- `loadFile(path)` - Handles single file configuration
- `loadDirectory(path)` - Handles directory-based configuration
- `executeScriptsInParallel(scripts)` - Runs multiple scripts concurrently

#### 3. Environment (`env.go`)
Manages environment variable operations:
- Captures current environment
- Executes scripts and captures their environment
- Parses null-delimited environment output
- Builds export scripts for environment changes
- Merges multiple environment maps

**Key methods:**
- `GetCurrent()` - Returns current environment as map
- `ExecuteAndCapture(zshPath, scriptPath)` - Runs script and captures env
- `BuildExportScript(current, new)` - Generates export statements
- `Merge(envChan)` - Combines multiple environment maps

#### 4. Plugin System (`plugin.go`, `args.go`, `plugin_loader.go`)
Provides a powerful plugin architecture with:
- **Context-based routing**: Capital letters (A-Z) define plugin contexts
- **Hierarchical contexts**: Nest contexts infinitely (e.g., `-TO` for Time->Overtime)
- **Dynamic registration**: Plugins register themselves via `wsh args --register`
- **Parallel loading**: All plugins load concurrently for fast startup
- **Automatic discovery**: Plugins auto-discovered from `./plugins/` directory

**Key structs:**
```go
type PluginContext struct {
    Context     rune                        // Single capital letter
    ContextLong string                      // Long name (e.g., "time")
    Description string                      // Plugin description
    Script      string                      // Path to plugin script
    Flags       []Flag                      // Plugin-specific flags
    SubContexts map[rune]*PluginContext    // Nested contexts
}

type PluginRegistry struct {
    contexts map[rune]*PluginContext        // Thread-safe registry
    mu       sync.RWMutex                   // Concurrent access
}
```

**Plugin flow:**
1. wsh starts and creates PluginRegistry
2. Internal plugins (-S/--shell, -A/--args) register
3. External plugins discovered in `./plugins/`
4. Each plugin script executes and calls `wsh args --register`
5. Registration outputs JSON captured by parent process
6. Registry populated with all plugin contexts

## Usage

### Command Line

```bash
# Interactive mode
./wsh

# Execute command with -c flag
./wsh -c "echo hello"
./wsh -S -c "echo hello"    # Equivalent (explicit shell context)

# Execute with arguments
./wsh -c "echo $1" arg1

# Get help
./wsh -h                    # Top-level help
./wsh -Th                   # Context-specific help (e.g., Time plugin)

# Use plugins (examples assume Time plugin exists)
./wsh -T --format json      # Use Time plugin with format flag
./wsh -TO --hours 5         # Use nested Overtime context
```

### Configuration

#### Single File Mode
Create `~/.wshrc` as a file:
```bash
export MY_VAR="value"
alias ll="ls -la"
```

#### Directory Mode
Create `~/.wshrc/` as a directory with multiple scripts:
```bash
~/.wshrc/
├── 01-env.sh      # Environment variables
├── 02-paths.sh    # PATH configuration
└── 03-aliases.sh  # Aliases and functions
```

**Key features:**
- All scripts execute in parallel for faster startup
- Environment changes are merged automatically
- Hidden files (starting with `.`) are ignored
- Subdirectories are ignored
- Only regular files are processed

### Creating Plugins

Create executable scripts in `./plugins/` directory (or `$WSH_PLUGIN_DIR`):

```bash
#!/bin/bash
# ./plugins/time.sh

# Register plugin with wsh
$WSH_BINARY args --register \
    -T --time "Time tracking and management" \
    -f --format "fmt" "Output format (json, text)" \
    -o --offline "Work offline"

# Plugin logic goes here
# Environment variables are set for each flag:
# - $format (if --format was provided)
# - $offline (if --offline was provided)

if [ "$format" = "json" ]; then
    echo '{"time": "12:34:56"}'
else
    echo "Current time: 12:34:56"
fi
```

**Plugin registration syntax:**
```bash
$WSH_BINARY args --register \
    -X --context-name "Description" \
    -f --flag-name "arg" "Flag description" \
    -b --bool-flag "Boolean flag description"
```

**Nested contexts:**
```bash
# In parent plugin
$WSH_BINARY args --register \
    -T --time "Time management" \
    -f --format "fmt" "Output format"

# In nested plugin (registers under -T)
$WSH_BINARY args --register \
    -TO --overtime "Overtime tracking" \
    -h --hours "num" "Hours worked"
```

**Environment variables available to plugins:**
- `$WSH_BINARY` - Path to wsh executable
- `$WSH_PLUGIN_SCRIPT` - Path to current plugin script
- `$<flagname>` - Value for each flag provided by user

See `PLUGINS.md` for detailed plugin development guide.

## Testing

Run all tests:
```bash
go test -v ./...
```

Check coverage:
```bash
go test -cover ./...
```

Run specific test:
```bash
go test -v -run TestEnvironment_ParseEnvLine
```

### Test Coverage

**113 tests** covering all major functionality:

Test files:
- `env_test.go` - Environment variable operations
- `wshrc_test.go` - Configuration loading
- `shell_test.go` - Shell execution
- `middleware_test.go` - Middleware composition
- `plugin_test.go` - Plugin registry and parsing
- `args_test.go` - Plugin registration and parsing
- `plugin_loader_test.go` - Plugin discovery and loading (includes integration tests)
- `plugin_executor_test.go` - Plugin execution
- `help_test.go` - Help generation

## Building

```bash
go build -o wsh
```

## Functional Programming Patterns

wsh leverages several functional programming patterns for flexibility and composability:

### 1. Functional Options Pattern

Constructors accept variadic option functions for flexible configuration:

```go
shell, err := NewShell(
    WithZshPath("/custom/zsh"),
    WithWshrcPath("~/.customrc"),
)
```

**Benefits:**
- Backward compatible (default usage stays simple)
- Easy to test with custom configurations
- Self-documenting API
- Optional parameters without bloat

### 2. Strategy Pattern

Different execution strategies can be swapped at runtime:

```go
loader := NewWshrcLoader(zshPath,
    WithExecutionStrategy(SequentialExecutionStrategy), // or ParallelExecutionStrategy
)
```

**Use cases:**
- Parallel execution (default) for speed
- Sequential execution for debugging
- Custom strategies for dependency management

### 3. Middleware Pattern

Wrap script executors with composable middleware:

```go
loader := NewWshrcLoader(zshPath,
    WithMiddleware(
        WithTimeout(10 * time.Second),
        WithLogging(log.Printf),
        WithErrorRecovery(),
    ),
)
```

**Available middleware:**
- `WithLogging` - Log execution with timing
- `WithTimeout` - Prevent hanging scripts
- `WithErrorRecovery` - Recover from panics
- `WithRetry` - Retry failed executions
- `WithEnvFilter` - Filter environment variables
- `WithCaching` - Cache script results

**Benefits:**
- Composable - mix and match as needed
- Reusable - write once, use everywhere
- Testable - easy to verify behavior
- Non-invasive - doesn't change core logic

See [EXAMPLES.md](EXAMPLES.md) for detailed usage examples.

## Design Decisions

### Why Parallel Execution?
When `.wshrc` is a directory, scripts are executed in parallel by default to minimize startup time. Each script runs in isolation and their environments are merged at the end. This can be changed to sequential execution via the strategy pattern.

### Why Null-Delimited Environment Parsing?
Using `env -0` ensures environment variables with newlines or special characters are parsed correctly.

### Why Separate Structs?
The code is organized into three main components (Shell, WshrcLoader, Environment) for:
- **Separation of concerns**: Each struct has a single responsibility
- **Testability**: Individual components can be tested in isolation
- **Maintainability**: Changes to one component don't affect others
- **Extensibility**: New features can be added without major refactoring

### Why Functional Patterns?
Functional programming patterns provide:
- **Flexibility**: Easy to customize behavior without modifying code
- **Composability**: Combine small pieces to create complex behavior
- **Testability**: Easy to mock and test individual components
- **Maintainability**: Clear separation of concerns and responsibilities

## Future Enhancements

Areas for potential expansion:
- Plugin marketplace/repository
- Plugin dependency management
- Configuration validation
- Performance profiling via middleware
- Shell function support
- Plugin versioning
- Auto-update mechanism for plugins
