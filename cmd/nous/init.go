package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// hookSettings is the .claude/settings.json structure (partial — only hooks).
type hookSettings struct {
	Hooks map[string][]hookMatcher `json:"hooks"`
}

type hookMatcher struct {
	Matcher string     `json:"matcher"`
	Hooks   []hookSpec `json:"hooks"`
}

type hookSpec struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

func initCmd(args []string) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	claudeDir := filepath.Join(dir, ".claude")
	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Read existing settings if present
	var settings map[string]interface{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			fmt.Fprintf(os.Stderr, "error parsing existing %s: %v\n", settingsPath, err)
			os.Exit(1)
		}
	} else {
		settings = make(map[string]interface{})
	}

	// Build the SessionStart hook
	nousHook := hookMatcher{
		Matcher: "",
		Hooks: []hookSpec{
			{
				Type:    "command",
				Command: "nous guide observations && echo && nous guide scope",
			},
		},
	}

	// Merge into existing hooks (preserve other hook events)
	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = make(map[string]interface{})
	}

	// Replace any existing SessionStart hooks from nous, preserve others
	var existing []interface{}
	if ss, ok := hooks["SessionStart"]; ok {
		if arr, ok := ss.([]interface{}); ok {
			for _, entry := range arr {
				m, ok := entry.(map[string]interface{})
				if !ok {
					existing = append(existing, entry)
					continue
				}
				// Check if this is a nous hook by inspecting the command
				isNous := false
				if hs, ok := m["hooks"].([]interface{}); ok {
					for _, h := range hs {
						if hm, ok := h.(map[string]interface{}); ok {
							if cmd, ok := hm["command"].(string); ok {
								if len(cmd) >= 10 && cmd[:10] == "nous guide" {
									isNous = true
								}
							}
						}
					}
				}
				if !isNous {
					existing = append(existing, entry)
				}
			}
		}
	}
	existing = append(existing, nousHook)
	hooks["SessionStart"] = existing
	settings["hooks"] = hooks

	// Write
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating %s: %v\n", claudeDir, err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	data = append(data, '\n')

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", settingsPath, err)
		os.Exit(1)
	}

	fmt.Printf("nous: installed SessionStart hook in %s\n", settingsPath)
	fmt.Printf("nous: agents will see observation and scope guides on startup\n")
}
