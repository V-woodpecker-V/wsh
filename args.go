package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// RegisterArgsPlugin registers the args plugin (-A/--args) as an internal plugin
func RegisterArgsPlugin(registry *PluginRegistry) error {
	argsCtx := &PluginContext{
		Context:     'A',
		ContextLong: "args",
		Description: "Argument parser operations",
		Script:      "", // Internal plugin, no script
		Flags: []Flag{
			{Short: 'r', Long: "register", Description: "Register plugin flags"},
		},
	}

	return registry.Register(argsCtx)
}

// HandleArgs processes the args subcommand
// Returns exit code
func HandleArgs(registry *PluginRegistry, args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "wsh args: no arguments provided\n")
		return 1
	}

	// Check if this is a registration call
	if args[0] == "--register" {
		return handleRegister(registry, args[1:])
	}

	// Otherwise, parse and output env vars
	return handleParse(registry, args)
}

// handleRegister processes plugin registration
// Expected format: --register -T --time "desc" -o --offline "desc" -f --from days "desc" ...
func handleRegister(registry *PluginRegistry, args []string) int {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "wsh args --register: insufficient arguments\n")
		fmt.Fprintf(os.Stderr, "usage: wsh args --register -T --time \"description\" [flags...]\n")
		return 1
	}

	// Parse context definition
	ctx, err := parsePluginDefinition(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wsh args --register: %v\n", err)
		return 1
	}

	// Register plugin (idempotent)
	err = registry.Register(ctx)
	if err != nil {
		// Already registered by different script - warn but continue
		fmt.Fprintf(os.Stderr, "wsh args --register: warning: %v\n", err)
	}

	// Output the registered context as JSON for parent process to parse
	jsonData, err := json.Marshal(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wsh args --register: failed to marshal context: %v\n", err)
		return 1
	}

	fmt.Println(string(jsonData))
	return 0
}

// parsePluginDefinition parses plugin definition from registration args
// Accepts: -X --name "desc", --name "desc", or -X "desc" (at least one of short/long required)
func parsePluginDefinition(args []string) (*PluginContext, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("need at least: --name \"description\" or -X --name \"description\"")
	}

	var contextShort rune
	var contextLong string
	var description string
	var flagsStart int

	// Parse context flags - check if first arg is short or long
	if strings.HasPrefix(args[0], "--") {
		// Long flag only: --name "description"
		contextLong = strings.TrimPrefix(args[0], "--")
		if len(args) < 2 {
			return nil, fmt.Errorf("missing description after %s", args[0])
		}
		description = args[1]
		flagsStart = 2
		// Generate a short flag from first letter of long name
		if len(contextLong) > 0 {
			contextShort = rune(strings.ToUpper(contextLong)[0])
		}
	} else if strings.HasPrefix(args[0], "-") && len(args[0]) == 2 {
		// Short flag: -X ...
		contextShort = rune(args[0][1])
		if contextShort < 'A' || contextShort > 'Z' {
			return nil, fmt.Errorf("context short flag must be a capital letter, got: -%c", contextShort)
		}

		// Check if next arg is long flag or description
		if len(args) < 2 {
			return nil, fmt.Errorf("missing long flag or description after %s", args[0])
		}

		if strings.HasPrefix(args[1], "--") {
			// Both short and long: -X --name "description"
			contextLong = strings.TrimPrefix(args[1], "--")
			if len(args) < 3 {
				return nil, fmt.Errorf("missing description after %s %s", args[0], args[1])
			}
			description = args[2]
			flagsStart = 3
		} else {
			// Short only: -X "description"
			description = args[1]
			flagsStart = 2
			// Generate long name from short (T -> time would need mapping, so just use lowercase)
			contextLong = strings.ToLower(string(contextShort))
		}
	} else {
		return nil, fmt.Errorf("expected context flag (e.g., -T or --time), got: %s", args[0])
	}

	// Validate we have at least one flag
	if contextShort == 0 && contextLong == "" {
		return nil, fmt.Errorf("must specify at least one of short (-X) or long (--name) context flag")
	}

	// Parse flags and sub-contexts
	flags, subContexts, err := parseFlagsAndSubContexts(args[flagsStart:])
	if err != nil {
		return nil, err
	}

	ctx := &PluginContext{
		Context:     contextShort,
		ContextLong: contextLong,
		Description: description,
		Script:      os.Getenv("WSH_PLUGIN_SCRIPT"), // Set by plugin loader
		Flags:       flags,
		SubContexts: subContexts,
	}

	return ctx, nil
}

// parseFlagsAndSubContexts parses flags and nested sub-contexts from args
func parseFlagsAndSubContexts(args []string) ([]Flag, map[rune]*PluginContext, error) {
	var flags []Flag
	subContexts := make(map[rune]*PluginContext)

	i := 0
	for i < len(args) {
		arg := args[i]

		// Check if this is a sub-context (capital letter)
		if strings.HasPrefix(arg, "-") && len(arg) == 2 {
			ch := rune(arg[1])
			if ch >= 'A' && ch <= 'Z' {
				// This is a sub-context definition
				subCtx, consumed, err := parseSubContext(args[i:])
				if err != nil {
					return nil, nil, fmt.Errorf("error parsing sub-context: %w", err)
				}
				subContexts[subCtx.Context] = subCtx
				i += consumed
				continue
			}
		}

		// Otherwise, it's a flag
		flag, consumed, err := parseFlag(args[i:])
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing flag: %w", err)
		}
		flags = append(flags, flag)
		i += consumed
	}

	return flags, subContexts, nil
}

// parseSubContext parses a sub-context definition
// Returns the context, number of args consumed, and error
func parseSubContext(args []string) (*PluginContext, int, error) {
	if len(args) < 3 {
		return nil, 0, fmt.Errorf("sub-context needs at least: -X --name \"description\"")
	}

	// Extract context short
	contextShort := rune(args[0][1])

	// Extract context long
	if !strings.HasPrefix(args[1], "--") {
		return nil, 0, fmt.Errorf("expected long flag for sub-context, got: %s", args[1])
	}
	contextLong := strings.TrimPrefix(args[1], "--")

	// Extract description
	description := args[2]

	// Parse nested flags (stop at next capital letter or end)
	consumed := 3
	var flags []Flag

	for consumed < len(args) {
		arg := args[consumed]

		// Check if this is another context (capital letter)
		if strings.HasPrefix(arg, "-") && len(arg) == 2 {
			ch := rune(arg[1])
			if ch >= 'A' && ch <= 'Z' {
				break // Stop, this is the next context
			}
		}

		// Parse flag
		flag, flagConsumed, err := parseFlag(args[consumed:])
		if err != nil {
			return nil, 0, err
		}
		flags = append(flags, flag)
		consumed += flagConsumed
	}

	subCtx := &PluginContext{
		Context:     contextShort,
		ContextLong: contextLong,
		Description: description,
		Script:      os.Getenv("WSH_PLUGIN_SCRIPT"),
		Flags:       flags,
	}

	return subCtx, consumed, nil
}

// parseFlag parses a single flag definition
// Format: -s --long [argname] "description"
// Returns flag, number of args consumed, and error
func parseFlag(args []string) (Flag, int, error) {
	if len(args) < 2 {
		return Flag{}, 0, fmt.Errorf("flag needs at least description")
	}

	flag := Flag{}
	consumed := 0

	// Parse short flag (optional)
	if strings.HasPrefix(args[0], "-") && len(args[0]) == 2 && args[0][1] >= 'a' && args[0][1] <= 'z' {
		flag.Short = rune(args[0][1])
		consumed++
	}

	// Parse long flag (optional but at least one of short/long required)
	if consumed < len(args) && strings.HasPrefix(args[consumed], "--") {
		flag.Long = strings.TrimPrefix(args[consumed], "--")
		consumed++
	}

	if flag.Short == 0 && flag.Long == "" {
		return Flag{}, 0, fmt.Errorf("flag must have at least short or long name")
	}

	// Determine if there's an argName
	// If there are 2+ non-flag args remaining, first is argName, second is description
	// If there's only 1 non-flag arg remaining, it's the description
	remainingArgs := 0
	for i := consumed; i < len(args) && !strings.HasPrefix(args[i], "-"); i++ {
		remainingArgs++
	}

	if remainingArgs >= 2 {
		// First arg is argName, second is description
		flag.ArgName = args[consumed]
		consumed++
		flag.Description = args[consumed]
		consumed++
	} else if remainingArgs == 1 {
		// Only description
		flag.Description = args[consumed]
		consumed++
	} else {
		return Flag{}, 0, fmt.Errorf("flag missing description")
	}

	return flag, consumed, nil
}

// handleParse parses command-line arguments and outputs environment variables
func handleParse(registry *PluginRegistry, args []string) int {
	result, err := registry.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wsh args: %v\n", err)
		return 1
	}

	// Output environment variables
	for key, value := range result.Flags {
		fmt.Printf("%s=%s\n", key, value)
	}

	// Output remaining args if any
	if len(result.Args) > 0 {
		fmt.Printf("WSH_ARGS=%s\n", strings.Join(result.Args, " "))
	}

	return 0
}
