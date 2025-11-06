package main

import (
	"testing"
)

func TestPluginRegistry_Register(t *testing.T) {
	registry := NewPluginRegistry()

	ctx := &PluginContext{
		Context:     'T',
		ContextLong: "time",
		Description: "Time tracking",
		Script:      "./plugins/time.sh",
		Flags: []Flag{
			{Short: 'o', Long: "offline", Description: "Offline mode"},
			{Short: 'f', Long: "from", ArgName: "days", Description: "Days ago"},
		},
	}

	err := registry.Register(ctx)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	// Verify registration
	found := registry.Lookup([]rune{'T'})
	if found == nil {
		t.Error("Expected to find registered context")
	}
	if found.Context != 'T' {
		t.Errorf("Context = %c, want T", found.Context)
	}
}

func TestPluginRegistry_RegisterIdempotent(t *testing.T) {
	registry := NewPluginRegistry()

	ctx := &PluginContext{
		Context:     'T',
		ContextLong: "time",
		Description: "Time tracking",
		Script:      "./plugins/time.sh",
	}

	// Register twice
	err1 := registry.Register(ctx)
	err2 := registry.Register(ctx)

	if err1 != nil {
		t.Errorf("First Register() error = %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second Register() should be idempotent, got error = %v", err2)
	}
}

func TestPluginRegistry_RegisterConflict(t *testing.T) {
	registry := NewPluginRegistry()

	ctx1 := &PluginContext{
		Context:     'T',
		ContextLong: "time",
		Script:      "./plugins/time.sh",
	}

	ctx2 := &PluginContext{
		Context:     'T',
		ContextLong: "test",
		Script:      "./plugins/test.sh",
	}

	err1 := registry.Register(ctx1)
	err2 := registry.Register(ctx2)

	if err1 != nil {
		t.Errorf("First Register() error = %v", err1)
	}
	if err2 == nil {
		t.Error("Second Register() with different script should return error")
	}

	// First registration should win
	found := registry.Lookup([]rune{'T'})
	if found.Script != "./plugins/time.sh" {
		t.Errorf("Script = %s, want ./plugins/time.sh", found.Script)
	}
}

func TestPluginRegistry_Lookup(t *testing.T) {
	registry := NewPluginRegistry()

	ctx := &PluginContext{
		Context:     'T',
		ContextLong: "time",
		Script:      "./plugins/time.sh",
	}

	registry.Register(ctx)

	tests := []struct {
		name        string
		contextPath []rune
		wantNil     bool
	}{
		{"found", []rune{'T'}, false},
		{"not found", []rune{'X'}, true},
		{"empty path", []rune{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.Lookup(tt.contextPath)
			if tt.wantNil && result != nil {
				t.Error("Expected nil, got context")
			}
			if !tt.wantNil && result == nil {
				t.Error("Expected context, got nil")
			}
		})
	}
}

func TestPluginRegistry_LookupNested(t *testing.T) {
	registry := NewPluginRegistry()

	overtimeCtx := &PluginContext{
		Context:     'O',
		ContextLong: "overtime",
		Script:      "./plugins/time.sh",
		Flags: []Flag{
			{Short: 's', Long: "start", ArgName: "time", Description: "Start time"},
		},
	}

	timeCtx := &PluginContext{
		Context:     'T',
		ContextLong: "time",
		Script:      "./plugins/time.sh",
		Flags: []Flag{
			{Short: 'o', Long: "offline", Description: "Offline mode"},
		},
		SubContexts: map[rune]*PluginContext{
			'O': overtimeCtx,
		},
	}

	registry.Register(timeCtx)

	// Test nested lookup
	found := registry.Lookup([]rune{'T', 'O'})
	if found == nil {
		t.Fatal("Expected to find nested context")
	}
	if found.Context != 'O' {
		t.Errorf("Context = %c, want O", found.Context)
	}
	if len(found.Flags) != 1 || found.Flags[0].Short != 's' {
		t.Error("Expected to find 'start' flag in overtime context")
	}
}

func TestPluginRegistry_Parse_SimpleFlags(t *testing.T) {
	registry := NewPluginRegistry()

	ctx := &PluginContext{
		Context:     'T',
		ContextLong: "time",
		Script:      "./plugins/time.sh",
		Flags: []Flag{
			{Short: 'o', Long: "offline", Description: "Offline mode"},
			{Short: 'f', Long: "from", ArgName: "days", Description: "Days ago"},
		},
	}

	registry.Register(ctx)

	tests := []struct {
		name      string
		args      []string
		wantCtx   rune
		wantFlags map[string]string
		wantArgs  []string
		wantHelp  bool
		wantErr   bool
	}{
		{
			name:      "short flags",
			args:      []string{"-Tof", "5"},
			wantCtx:   'T',
			wantFlags: map[string]string{"offline": "true", "from": "5"},
			wantArgs:  []string{},
			wantHelp:  false,
			wantErr:   false,
		},
		{
			name:      "long flags",
			args:      []string{"--time", "--offline", "--from", "5"},
			wantCtx:   'T',
			wantFlags: map[string]string{"offline": "true", "from": "5"},
			wantArgs:  []string{},
			wantHelp:  false,
			wantErr:   false,
		},
		{
			name:      "mixed flags",
			args:      []string{"-T", "--offline", "-f", "5"},
			wantCtx:   'T',
			wantFlags: map[string]string{"offline": "true", "from": "5"},
			wantArgs:  []string{},
			wantHelp:  false,
			wantErr:   false,
		},
		{
			name:      "with remaining args",
			args:      []string{"-To", "extra", "args"},
			wantCtx:   'T',
			wantFlags: map[string]string{"offline": "true"},
			wantArgs:  []string{"extra", "args"},
			wantHelp:  false,
			wantErr:   false,
		},
		{
			name:      "help flag short",
			args:      []string{"-Th"},
			wantCtx:   'T',
			wantFlags: map[string]string{},
			wantArgs:  []string{},
			wantHelp:  true,
			wantErr:   false,
		},
		{
			name:      "help flag long",
			args:      []string{"-T", "--help"},
			wantCtx:   'T',
			wantFlags: map[string]string{},
			wantArgs:  []string{},
			wantHelp:  true,
			wantErr:   false,
		},
		{
			name:      "unknown context",
			args:      []string{"-X"},
			wantCtx:   0,
			wantFlags: nil,
			wantArgs:  nil,
			wantHelp:  false,
			wantErr:   true,
		},
		{
			name:      "unknown flag",
			args:      []string{"-T", "-z"},
			wantCtx:   'T',
			wantFlags: nil,
			wantArgs:  nil,
			wantHelp:  false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.Parse(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}

			if result.ShowHelp != tt.wantHelp {
				t.Errorf("ShowHelp = %v, want %v", result.ShowHelp, tt.wantHelp)
			}

			if result.Context != nil && result.Context.Context != tt.wantCtx {
				t.Errorf("Context = %c, want %c", result.Context.Context, tt.wantCtx)
			}

			if len(result.Flags) != len(tt.wantFlags) {
				t.Errorf("Flags count = %d, want %d", len(result.Flags), len(tt.wantFlags))
			}

			for key, want := range tt.wantFlags {
				if got, ok := result.Flags[key]; !ok || got != want {
					t.Errorf("Flag %s = %s, want %s", key, got, want)
				}
			}

			if len(result.Args) != len(tt.wantArgs) {
				t.Errorf("Args = %v, want %v", result.Args, tt.wantArgs)
			}
		})
	}
}

func TestPluginRegistry_Parse_NestedContexts(t *testing.T) {
	registry := NewPluginRegistry()

	overtimeCtx := &PluginContext{
		Context:     'O',
		ContextLong: "overtime",
		Script:      "./plugins/time.sh",
		Flags: []Flag{
			{Short: 's', Long: "start", ArgName: "time", Description: "Start time"},
			{Short: 'e', Long: "end", ArgName: "time", Description: "End time"},
		},
	}

	timeCtx := &PluginContext{
		Context:     'T',
		ContextLong: "time",
		Script:      "./plugins/time.sh",
		Flags: []Flag{
			{Short: 'o', Long: "offline", Description: "Offline mode"},
		},
		SubContexts: map[rune]*PluginContext{
			'O': overtimeCtx,
		},
	}

	registry.Register(timeCtx)

	tests := []struct {
		name        string
		args        []string
		wantCtxPath []rune
		wantFlags   map[string]string
	}{
		{
			name:        "nested context short",
			args:        []string{"-TOs", "09:00"},
			wantCtxPath: []rune{'T', 'O'},
			wantFlags:   map[string]string{"start": "09:00"},
		},
		{
			name:        "nested context long",
			args:        []string{"--time", "--overtime", "--start", "09:00"},
			wantCtxPath: []rune{'T', 'O'},
			wantFlags:   map[string]string{"start": "09:00"},
		},
		{
			name:        "nested with separate args",
			args:        []string{"-TOs", "09:00", "-e", "17:00"},
			wantCtxPath: []rune{'T', 'O'},
			wantFlags:   map[string]string{"start": "09:00", "end": "17:00"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.Parse(tt.args)
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}

			if len(result.ContextPath) != len(tt.wantCtxPath) {
				t.Errorf("ContextPath length = %d, want %d", len(result.ContextPath), len(tt.wantCtxPath))
			}

			for key, want := range tt.wantFlags {
				if got, ok := result.Flags[key]; !ok || got != want {
					t.Errorf("Flag %s = %s, want %s", key, got, want)
				}
			}
		})
	}
}

func TestPluginRegistry_GetAllContexts(t *testing.T) {
	registry := NewPluginRegistry()

	ctx1 := &PluginContext{Context: 'T', ContextLong: "time", Script: "./plugins/time.sh"}
	ctx2 := &PluginContext{Context: 'S', ContextLong: "shell", Script: "./plugins/shell.sh"}

	registry.Register(ctx1)
	registry.Register(ctx2)

	contexts := registry.GetAllContexts()

	if len(contexts) != 2 {
		t.Errorf("GetAllContexts() returned %d contexts, want 2", len(contexts))
	}
}

func TestFlag_NoLongName(t *testing.T) {
	registry := NewPluginRegistry()

	ctx := &PluginContext{
		Context: 'T',
		Script:  "./plugins/test.sh",
		Flags: []Flag{
			{Short: 'x', Description: "X flag (no long name)"},
		},
	}

	registry.Register(ctx)

	result, err := registry.Parse([]string{"-Tx"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Should use short flag as key
	if _, ok := result.Flags["x"]; !ok {
		t.Error("Expected flag 'x' to be set")
	}
}
