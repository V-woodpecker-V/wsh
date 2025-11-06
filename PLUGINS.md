# wsh Plugin Development Guide

This guide explains how to create plugins for wsh, from basic concepts to advanced techniques.

## Table of Contents

- [Quick Start](#quick-start)
- [Plugin Concepts](#plugin-concepts)
- [Registration Syntax](#registration-syntax)
- [Flag Types](#flag-types)
- [Nested Contexts](#nested-contexts)
- [Environment Variables](#environment-variables)
- [Best Practices](#best-practices)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

## Quick Start

Create a simple plugin in 3 steps:

1. Create an executable script in `./plugins/`:

```bash
#!/bin/bash
# ./plugins/hello.sh

$WSH_BINARY args --register \
    -H --hello "Say hello to someone" \
    -n --name "person" "Name to greet"

# Plugin logic
name="${name:-World}"
echo "Hello, $name!"
```

2. Make it executable:
```bash
chmod +x ./plugins/hello.sh
```

3. Use it:
```bash
./wsh -H --name Alice
# Output: Hello, Alice!
```

## Plugin Concepts

### Contexts

Each plugin defines a **context** - a single capital letter that activates the plugin:

- `-A` through `-Z` are available for plugins
- Context letters can be combined for nested functionality
- `-S` (shell) and `-A` (args) are reserved for internal use

### Flags

Within a context, plugins define **flags** - options that modify behavior:

- Short flags: single lowercase letter (`-f`)
- Long flags: descriptive name (`--format`)
- Flags can accept arguments or be boolean

### Plugin Script

The executable that:
1. Registers the plugin context and flags
2. Implements the plugin functionality
3. Receives flag values via environment variables

## Registration Syntax

### Basic Registration

```bash
$WSH_BINARY args --register \
    -X --context-name "Description"
```

Where:
- `-X` - Single capital letter (A-Z) for the context
- `--context-name` - Long name for the context
- `"Description"` - Brief description of plugin functionality

### Adding Flags

#### Boolean Flags (no argument)

```bash
$WSH_BINARY args --register \
    -X --context-name "Description" \
    -v --verbose "Enable verbose output"
```

#### Flags with Arguments

```bash
$WSH_BINARY args --register \
    -X --context-name "Description" \
    -f --format "fmt" "Output format (json, text, xml)"
```

Where:
- `-f` - Short flag (optional, can be omitted)
- `--format` - Long flag (required)
- `"fmt"` - Argument name (shown in help)
- `"Description"` - Flag description

#### Multiple Flags

```bash
$WSH_BINARY args --register \
    -T --time "Time tracking" \
    -f --format "fmt" "Output format" \
    -o --offline "Work offline" \
    -v --verbose "Verbose output" \
    -d --database "path" "Database file path"
```

## Flag Types

### 1. Boolean Flags

Flags that don't take arguments:

```bash
-v --verbose "Enable verbose output"
```

Usage: `wsh -T --verbose`

Environment variable: `verbose=true`

### 2. Value Flags

Flags that require an argument:

```bash
-f --format "fmt" "Output format"
```

Usage: `wsh -T --format json`

Environment variable: `format=json`

### 3. Optional Short Form

Long-only flags:

```bash
--config "path" "Configuration file"
```

Usage: `wsh -T --config /path/to/config`

### 4. Combined Short Flags

Multiple short flags can be combined:

```bash
wsh -Tov  # Same as: wsh -T -o -v
```

## Nested Contexts

Create hierarchical plugin structures by nesting contexts.

### Parent Context

```bash
#!/bin/bash
# ./plugins/time.sh

$WSH_BINARY args --register \
    -T --time "Time management" \
    -f --format "fmt" "Output format"

# Parent plugin logic
echo "Time plugin activated"
```

### Nested Context

```bash
#!/bin/bash
# ./plugins/time_overtime.sh

$WSH_BINARY args --register \
    -TO --overtime "Overtime tracking" \
    -h --hours "num" "Hours worked" \
    -r --rate "amount" "Hourly rate"

# Nested plugin logic
hours="${hours:-0}"
rate="${rate:-15}"
total=$((hours * rate))
echo "Overtime pay: \$$total"
```

### Using Nested Contexts

```bash
# Use parent context
wsh -T --format json

# Use nested context
wsh -TO --hours 5 --rate 20
# Output: Overtime pay: $100
```

### Deep Nesting

Contexts can nest arbitrarily deep:

```bash
wsh -TOW    # Time -> Overtime -> Weekly
```

## Environment Variables

### Automatic Variables

wsh automatically sets environment variables for your plugin:

#### `$WSH_BINARY`
Path to the wsh executable. Use this for recursive calls:
```bash
$WSH_BINARY args --register ...
```

#### `$WSH_PLUGIN_SCRIPT`
Path to your plugin script. Useful for locating resources:
```bash
PLUGIN_DIR=$(dirname "$WSH_PLUGIN_SCRIPT")
source "$PLUGIN_DIR/lib/common.sh"
```

### Flag Variables

Each flag becomes an environment variable with the **exact flag name** (preserving case):

```bash
# Registration
-f --format "fmt" "Output format"
-o --offline "Work offline"

# In plugin script
if [ "$format" = "json" ]; then
    # format flag was provided
fi

if [ -n "$offline" ]; then
    # offline flag was provided
fi
```

**Important**: Variable names use the **long flag name** exactly as specified.

## Best Practices

### 1. Use Meaningful Names

```bash
# Good
-T --time "Time management"
-f --format "fmt" "Output format"

# Avoid
-X --x "Does stuff"
-a --a "arg" "An argument"
```

### 2. Provide Clear Descriptions

```bash
# Good
-f --format "fmt" "Output format: json, text, or xml"

# Avoid
-f --format "fmt" "Format"
```

### 3. Validate Input

```bash
#!/bin/bash
# ./plugins/validator.sh

$WSH_BINARY args --register \
    -V --validator "Input validation" \
    -f --format "fmt" "Output format (json, text, xml)"

# Validate format
case "$format" in
    json|text|xml)
        # Valid format
        ;;
    *)
        echo "Error: Invalid format '$format'" >&2
        echo "Valid formats: json, text, xml" >&2
        exit 1
        ;;
esac

# Continue with plugin logic
```

### 4. Handle Missing Values

```bash
# Provide defaults
name="${name:-World}"
format="${format:-text}"
count="${count:-10}"

# Or require them
if [ -z "$database" ]; then
    echo "Error: --database is required" >&2
    exit 1
fi
```

### 5. Use Consistent Exit Codes

```bash
# Success
exit 0

# User error (invalid input)
exit 1

# System error (file not found, etc.)
exit 2
```

### 6. Support Help Flag

Your plugin automatically gets `-h/--help` support:

```bash
wsh -Th  # Shows Time plugin help
```

No additional code needed!

## Examples

### Example 1: Simple Greeter

```bash
#!/bin/bash
# ./plugins/greet.sh

$WSH_BINARY args --register \
    -G --greet "Greet someone" \
    -n --name "person" "Name to greet" \
    -f --formal "Use formal greeting"

# Implementation
name="${name:-friend}"

if [ -n "$formal" ]; then
    echo "Good day, ${name}."
else
    echo "Hey ${name}!"
fi
```

Usage:
```bash
./wsh -G --name Alice
# Output: Hey Alice!

./wsh -G --name Bob --formal
# Output: Good day, Bob.
```

### Example 2: File Processor

```bash
#!/bin/bash
# ./plugins/process.sh

$WSH_BINARY args --register \
    -P --process "Process files" \
    -i --input "path" "Input file" \
    -o --output "path" "Output file" \
    -f --format "fmt" "Output format (json, csv)" \
    -v --verbose "Verbose output"

# Validate required args
if [ -z "$input" ]; then
    echo "Error: --input is required" >&2
    exit 1
fi

# Set defaults
output="${output:-output.txt}"
format="${format:-json}"

# Verbose logging
if [ -n "$verbose" ]; then
    echo "Processing $input -> $output (format: $format)" >&2
fi

# Process the file
case "$format" in
    json)
        jq '.' "$input" > "$output"
        ;;
    csv)
        # Convert to CSV
        # ... conversion logic ...
        ;;
    *)
        echo "Error: Unknown format $format" >&2
        exit 1
        ;;
esac

echo "Processed successfully: $output"
```

### Example 3: Nested Configuration

```bash
#!/bin/bash
# ./plugins/config.sh

$WSH_BINARY args --register \
    -C --config "Configuration management" \
    -f --file "path" "Config file path" \
    -l --list "List all settings"

# Parent plugin handles basic config
if [ -n "$list" ]; then
    cat "${file:-~/.config/app.conf}"
    exit 0
fi
```

```bash
#!/bin/bash
# ./plugins/config_set.sh

$WSH_BINARY args --register \
    -CS --set "Set configuration value" \
    -k --key "name" "Configuration key" \
    -v --value "val" "Configuration value"

# Nested plugin handles setting values
if [ -z "$key" ] || [ -z "$value" ]; then
    echo "Error: --key and --value required" >&2
    exit 1
fi

echo "$key=$value" >> ~/.config/app.conf
echo "Set $key = $value"
```

Usage:
```bash
wsh -C --list
wsh -CS --key theme --value dark
```

## Troubleshooting

### Plugin Not Found

**Problem**: Plugin doesn't appear when running `wsh -h`

**Solutions**:
1. Check plugin is executable: `chmod +x ./plugins/myplugin.sh`
2. Check plugin is in the correct directory: `ls -la ./plugins/`
3. Set custom directory: `export WSH_PLUGIN_DIR=/path/to/plugins`
4. Check for errors: Look at stderr during wsh startup

### Registration Failed

**Problem**: Error during plugin registration

**Solutions**:
1. Check registration syntax:
   - Context must be `-X` (single capital letter)
   - Long name must be `--name` (lowercase)
   - Description is required
2. Ensure `$WSH_BINARY` is set correctly
3. Test registration manually:
   ```bash
   WSH_PLUGIN_SCRIPT=./plugins/test.sh ./wsh args --register -T --test "Test"
   ```

### Environment Variables Not Set

**Problem**: Flag values not available in plugin

**Solutions**:
1. Use exact flag name (long form): `$format` not `$fmt`
2. Check if flag was provided: `[ -n "$flagname" ]`
3. Remember boolean flags are set to "true", not the flag name
4. Debug: Add `env | grep -v ^_` to see all variables

### Context Conflicts

**Problem**: Two plugins trying to use the same context letter

**Solution**: Choose a different letter. Only 26 are available (A-Z).

### Nested Context Not Working

**Problem**: `wsh -TO` not recognized

**Solutions**:
1. Ensure parent context (`-T`) is registered
2. Check nested registration uses combined letters: `-TO` not `-T -O`
3. Verify both plugins are executable and in plugins directory

## Advanced Topics

### Sharing Code Between Plugins

Create a library file:

```bash
# ./plugins/lib/common.sh
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" >&2
}

validate_file() {
    if [ ! -f "$1" ]; then
        echo "Error: File not found: $1" >&2
        exit 2
    fi
}
```

Use in plugins:

```bash
#!/bin/bash
# ./plugins/myplugin.sh

PLUGIN_DIR=$(dirname "$WSH_PLUGIN_SCRIPT")
source "$PLUGIN_DIR/lib/common.sh"

# Now use shared functions
log "Starting plugin"
validate_file "$input"
```

### Plugin Configuration Files

```bash
#!/bin/bash
# ./plugins/configurable.sh

# Load plugin-specific config
CONFIG_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/wsh/myplugin.conf"
if [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
fi

# Use config with flag defaults
format="${format:-${DEFAULT_FORMAT:-text}}"
```

### Testing Plugins

Test registration:
```bash
WSH_PLUGIN_SCRIPT=./plugins/test.sh ./wsh args --register -T --test "Test" 2>&1
```

Test execution:
```bash
format=json ./plugins/test.sh
```

### Plugin Templates

Start new plugins from this template:

```bash
#!/bin/bash
# ./plugins/template.sh

set -euo pipefail  # Exit on error, undefined vars, pipe failures

# Register plugin
$WSH_BINARY args --register \
    -X --name "Description" \
    -f --flag "arg" "Flag description"

# Validate inputs
if [ -z "${required_flag:-}" ]; then
    echo "Error: --required-flag is required" >&2
    exit 1
fi

# Set defaults
optional_flag="${optional_flag:-default_value}"

# Main logic
echo "Plugin logic here"

# Success
exit 0
```

## Plugin Distribution

### Sharing Plugins

Package your plugin for distribution:

```bash
# Create plugin package
tar czf myplugin.tar.gz plugins/myplugin.sh plugins/lib/

# Install instructions:
# tar xzf myplugin.tar.gz -C ~/.local/share/wsh/
# export WSH_PLUGIN_DIR=~/.local/share/wsh/plugins
```

### Plugin Metadata

Include a comment header:

```bash
#!/bin/bash
# wsh Plugin: Time Tracker
# Version: 1.0.0
# Author: Your Name
# Description: Advanced time tracking and reporting
# Dependencies: jq, sqlite3
# License: MIT

$WSH_BINARY args --register ...
```

---

## Getting Help

- Check plugin help: `wsh -Xh` (where X is your context letter)
- Test registration: `WSH_PLUGIN_SCRIPT=./plugins/test.sh wsh args --register ...`
- Debug environment: Add `env | grep -v ^_` to your plugin
- Read examples: Check `./plugins/` for working examples

Happy plugin development!
