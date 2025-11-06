package main

import (
	"fmt"
	"os"
	"strings"
)

// ShowHelp displays context-aware help based on the context path
func ShowHelp(registry *PluginRegistry, contextPath []rune) {
	// If no context path, show top-level help
	if len(contextPath) == 0 {
		showTopLevelHelp(registry)
		return
	}

	// Look up the context
	ctx := registry.Lookup(contextPath)
	if ctx == nil {
		fmt.Fprintf(os.Stderr, "wsh: unknown context: %s\n", formatContextPath(contextPath))
		return
	}

	showContextHelp(ctx, contextPath)
}

// showTopLevelHelp displays help for all top-level contexts
func showTopLevelHelp(registry *PluginRegistry) {
	contexts := registry.GetAllContexts()

	fmt.Println("Usage: wsh [OPTIONS] [COMMAND]")
	fmt.Println()
	fmt.Println("A zsh wrapper with plugin support")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, --help              Show this help message")
	fmt.Println()
	fmt.Println("Contexts:")

	// Sort contexts for consistent output
	sortedContexts := sortContexts(contexts)

	for _, ctx := range sortedContexts {
		shortFlag := fmt.Sprintf("-%c", ctx.Context)
		longFlag := fmt.Sprintf("--%s", ctx.ContextLong)
		fmt.Printf("  %-6s %-20s %s\n", shortFlag, longFlag, ctx.Description)
	}

	fmt.Println()
	fmt.Println("Use 'wsh -<context>h' or 'wsh --<context> --help' for context-specific help")
}

// showContextHelp displays help for a specific context
func showContextHelp(ctx *PluginContext, contextPath []rune) {
	contextStr := formatContextPath(contextPath)

	fmt.Printf("Usage: wsh -%s [OPTIONS] [ARGS]\n", contextStr)
	fmt.Println()
	fmt.Printf("%s\n", ctx.Description)
	fmt.Println()

	// Show flags if any
	if len(ctx.Flags) > 0 {
		fmt.Println("Options:")
		fmt.Println("  -h, --help              Show this help message")

		for _, flag := range ctx.Flags {
			showFlagHelp(flag)
		}
		fmt.Println()
	}

	// Show sub-contexts if any
	if len(ctx.SubContexts) > 0 {
		fmt.Println("Sub-contexts:")

		sortedSubContexts := sortSubContexts(ctx.SubContexts)

		for _, subCtx := range sortedSubContexts {
			shortFlag := fmt.Sprintf("-%c", subCtx.Context)
			longFlag := fmt.Sprintf("--%s", subCtx.ContextLong)
			fmt.Printf("  %-6s %-20s %s\n", shortFlag, longFlag, subCtx.Description)
		}
		fmt.Println()
		fmt.Printf("Use 'wsh -%s<subcontext>h' for sub-context help\n", contextStr)
	}

	// Show script path if external plugin
	if ctx.Script != "" {
		fmt.Printf("Plugin script: %s\n", ctx.Script)
	}
}

// showFlagHelp displays help for a single flag
func showFlagHelp(flag Flag) {
	var shortFlag, longFlag, argPart string

	if flag.Short != 0 {
		shortFlag = fmt.Sprintf("-%c", flag.Short)
	} else {
		shortFlag = "  "
	}

	if flag.Long != "" {
		if flag.ArgName != "" {
			longFlag = fmt.Sprintf("--%s <%s>", flag.Long, flag.ArgName)
		} else {
			longFlag = fmt.Sprintf("--%s", flag.Long)
		}
	}

	if flag.Short != 0 && flag.Long != "" {
		if flag.ArgName != "" {
			fmt.Printf("  %s, %-20s %s\n", shortFlag, longFlag, flag.Description)
		} else {
			fmt.Printf("  %s, %-20s %s\n", shortFlag, longFlag, flag.Description)
		}
	} else if flag.Short != 0 {
		if flag.ArgName != "" {
			argPart = fmt.Sprintf(" <%s>", flag.ArgName)
			fmt.Printf("  %-24s %s\n", shortFlag+argPart, flag.Description)
		} else {
			fmt.Printf("  %-24s %s\n", shortFlag, flag.Description)
		}
	} else if flag.Long != "" {
		fmt.Printf("  %-24s %s\n", longFlag, flag.Description)
	}
}

// formatContextPath converts a context path to a string
func formatContextPath(path []rune) string {
	var sb strings.Builder
	for _, r := range path {
		sb.WriteRune(r)
	}
	return sb.String()
}

// sortContexts sorts contexts by their Context rune
func sortContexts(contexts []*PluginContext) []*PluginContext {
	// Create a copy to avoid modifying the original
	sorted := make([]*PluginContext, len(contexts))
	copy(sorted, contexts)

	// Simple insertion sort by Context rune
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j].Context > key.Context {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}

	return sorted
}

// sortSubContexts sorts sub-contexts by their Context rune
func sortSubContexts(subContexts map[rune]*PluginContext) []*PluginContext {
	// Convert map to slice
	contexts := make([]*PluginContext, 0, len(subContexts))
	for _, ctx := range subContexts {
		contexts = append(contexts, ctx)
	}

	return sortContexts(contexts)
}
