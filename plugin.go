package main

import (
	"fmt"
	"strings"
	"sync"
)

// Flag represents a command-line flag within a context
type Flag struct {
	Short       rune   // Short flag (e.g., 'o' for -o), 0 if not provided
	Long        string // Long flag (e.g., "offline" for --offline)
	ArgName     string // Argument name (e.g., "days", "time"), empty if no argument
	Description string // Flag description for help text
}

// PluginContext represents a context (like -T for time) with its flags and sub-contexts
type PluginContext struct {
	Context     rune                       // Context character (e.g., 'T')
	ContextLong string                     // Long context name (e.g., "time")
	Description string                     // Context description
	Script      string                     // Path to plugin script
	Flags       []Flag                     // Flags within this context
	SubContexts map[rune]*PluginContext    // Nested sub-contexts (recursive)
}

// PluginRegistry manages all registered plugins
type PluginRegistry struct {
	contexts map[rune]*PluginContext
	mu       sync.RWMutex
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		contexts: make(map[rune]*PluginContext),
	}
}

// Register adds a plugin context to the registry (idempotent)
func (r *PluginRegistry) Register(ctx *PluginContext) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already registered
	if existing, exists := r.contexts[ctx.Context]; exists {
		// If already registered with same details, it's a no-op (idempotent)
		if existing.Script == ctx.Script {
			return nil
		}
		// Different script wants same context - warn but keep first
		return fmt.Errorf("context -%c already registered by %s, ignoring %s",
			ctx.Context, existing.Script, ctx.Script)
	}

	r.contexts[ctx.Context] = ctx
	return nil
}

// Lookup finds a context by traversing the context path
// contextPath is a slice of context characters, e.g., ['T', 'O'] for -TO
func (r *PluginRegistry) Lookup(contextPath []rune) *PluginContext {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(contextPath) == 0 {
		return nil
	}

	// Start with top-level context
	ctx, exists := r.contexts[contextPath[0]]
	if !exists {
		return nil
	}

	// Traverse sub-contexts
	for i := 1; i < len(contextPath); i++ {
		if ctx.SubContexts == nil {
			return nil
		}
		subCtx, exists := ctx.SubContexts[contextPath[i]]
		if !exists {
			return nil
		}
		ctx = subCtx
	}

	return ctx
}

// GetAllContexts returns all top-level contexts (for help generation)
func (r *PluginRegistry) GetAllContexts() []*PluginContext {
	r.mu.RLock()
	defer r.mu.RUnlock()

	contexts := make([]*PluginContext, 0, len(r.contexts))
	for _, ctx := range r.contexts {
		contexts = append(contexts, ctx)
	}
	return contexts
}

// ParseResult contains the result of parsing command-line arguments
type ParseResult struct {
	ContextPath  []rune            // Path of contexts traversed (e.g., ['T', 'O'])
	Context      *PluginContext    // Final context
	Flags        map[string]string // Parsed flags (key is long or short flag name)
	Args         []string          // Remaining arguments
	ShowHelp     bool              // Whether help was requested
}

// Parse parses command-line arguments according to registered plugins
// Returns the context, parsed flags, and remaining arguments
func (r *PluginRegistry) Parse(args []string) (*ParseResult, error) {
	result := &ParseResult{
		ContextPath: []rune{},
		Flags:       make(map[string]string),
		Args:        []string{},
		ShowHelp:    false,
	}

	if len(args) == 0 {
		return result, nil
	}

	var currentCtx *PluginContext
	i := 0

	for i < len(args) {
		arg := args[i]

		// Handle long flags
		if strings.HasPrefix(arg, "--") {
			flagName := strings.TrimPrefix(arg, "--")

			// Check for help
			if flagName == "help" {
				result.ShowHelp = true
				i++
				continue
			}

			// If no context set and we encounter a flag, default to Shell context
			if currentCtx == nil {
				currentCtx = r.Lookup([]rune{'S'})
				if currentCtx != nil {
					result.Context = currentCtx
				}
			}

			// Try to match flag in current context or parent contexts
			matched, nextI := r.matchLongFlag(flagName, currentCtx, result, args, i)
			if matched {
				i = nextI
				continue
			}

			// Check if it's a context switch (long context name)
			ctx := r.findContextByLong(flagName, currentCtx)
			if ctx != nil {
				currentCtx = ctx
				result.ContextPath = append(result.ContextPath, ctx.Context)
				result.Context = ctx
				i++
				continue
			}

			return nil, fmt.Errorf("unknown flag: --%s", flagName)
		}

		// Handle short flags
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			flags := []rune(arg[1:]) // Strip the '-' and convert to runes
			flagIdx := 0

			for flagIdx < len(flags) {
				ch := flags[flagIdx]

				// Check for help
				if ch == 'h' {
					result.ShowHelp = true
					flagIdx++
					continue
				}

				// Check if it's a context (capital letter)
				if ch >= 'A' && ch <= 'Z' {
					// Check if it's a sub-context of current context
					if currentCtx != nil && currentCtx.SubContexts != nil {
						if subCtx, exists := currentCtx.SubContexts[ch]; exists {
							currentCtx = subCtx
							result.ContextPath = append(result.ContextPath, ch)
							result.Context = subCtx
							flagIdx++
							continue
						}
					}

					// Check if it's a top-level context
					ctx := r.Lookup([]rune{ch})
					if ctx != nil {
						currentCtx = ctx
						result.ContextPath = []rune{ch} // Reset context path
						result.Context = ctx
						flagIdx++
						continue
					}
					return nil, fmt.Errorf("unknown context: -%c", ch)
				}

				// If no context set and we encounter a lowercase flag, default to Shell context
				if currentCtx == nil {
					currentCtx = r.Lookup([]rune{'S'})
					if currentCtx != nil {
						result.Context = currentCtx
					}
				}

				// Try to match flag in current context
				// Check if this flag takes an argument
				flag := r.findFlagInContext(ch, currentCtx)
				if flag == nil {
					return nil, fmt.Errorf("unknown flag: -%c", ch)
				}

				key := flag.Long
				if key == "" {
					key = string(flag.Short)
				}

				if flag.ArgName != "" {
					// Flag takes an argument
					// Check if there are more chars in this arg (shouldn't be)
					if flagIdx < len(flags)-1 {
						// More flags after this one, but this flag needs an argument
						return nil, fmt.Errorf("flag -%c requires an argument", ch)
					}
					// Get argument from next item
					if i+1 < len(args) {
						result.Flags[key] = args[i+1]
						i++ // Skip the argument
						break
					}
					return nil, fmt.Errorf("flag -%c requires an argument", ch)
				}

				// Boolean flag
				result.Flags[key] = "true"
				flagIdx++
			}
			i++
			continue
		}

		// Not a flag, add to remaining args
		result.Args = append(result.Args, arg)
		i++
	}

	return result, nil
}

// matchLongFlag attempts to match a long flag in the current context or parent contexts
func (r *PluginRegistry) matchLongFlag(flagName string, ctx *PluginContext, result *ParseResult, args []string, i int) (bool, int) {
	if ctx == nil {
		return false, i
	}

	// Search in current context
	for _, flag := range ctx.Flags {
		if flag.Long == flagName {
			// Flag found, check if it takes an argument
			if flag.ArgName != "" {
				// Needs an argument
				if i+1 < len(args) {
					result.Flags[flag.Long] = args[i+1]
					return true, i + 2
				}
				return false, i
			}
			// Boolean flag
			result.Flags[flag.Long] = "true"
			return true, i + 1
		}
	}

	return false, i
}

// matchShortFlag attempts to match a short flag in the current context
func (r *PluginRegistry) matchShortFlag(ch rune, ctx *PluginContext, result *ParseResult, args []string, i int) (bool, int) {
	if ctx == nil {
		return false, i
	}

	// Search in current context
	for _, flag := range ctx.Flags {
		if flag.Short == ch {
			// Flag found, check if it takes an argument
			if flag.ArgName != "" {
				// Needs an argument - next arg
				if i+1 < len(args) {
					// Use long name if available, otherwise short
					key := flag.Long
					if key == "" {
						key = string(flag.Short)
					}
					result.Flags[key] = args[i+1]
					return true, i + 2
				}
				return false, i
			}
			// Boolean flag
			key := flag.Long
			if key == "" {
				key = string(flag.Short)
			}
			result.Flags[key] = "true"
			return true, i + 1
		}
	}

	return false, i
}

// findContextByLong finds a context by its long name, checking sub-contexts if currentCtx is provided
func (r *PluginRegistry) findContextByLong(longName string, currentCtx *PluginContext) *PluginContext {
	// First check sub-contexts if we're in a context
	if currentCtx != nil && currentCtx.SubContexts != nil {
		for _, subCtx := range currentCtx.SubContexts {
			if subCtx.ContextLong == longName {
				return subCtx
			}
		}
	}

	// Check top-level contexts
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, ctx := range r.contexts {
		if ctx.ContextLong == longName {
			return ctx
		}
	}
	return nil
}

// findFlagInContext finds a flag by its short name in the given context
func (r *PluginRegistry) findFlagInContext(ch rune, ctx *PluginContext) *Flag {
	if ctx == nil {
		return nil
	}

	for i := range ctx.Flags {
		if ctx.Flags[i].Short == ch {
			return &ctx.Flags[i]
		}
	}
	return nil
}
