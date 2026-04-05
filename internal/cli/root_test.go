package cli

import "testing"

func TestNewRootCmdRegistersTopLevelCommands(t *testing.T) {
	cmd := newRootCmd("test-version")

	expected := map[string]bool{
		"version":  false,
		"config":   false,
		"crawl":    false,
		"audit":    false,
		"report":   false,
		"provider": false,
	}

	for _, sub := range cmd.Commands() {
		if _, ok := expected[sub.Name()]; ok {
			expected[sub.Name()] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Fatalf("expected top-level command %q to be registered", name)
		}
	}
}

func TestSubcommandStructure(t *testing.T) {
	cmd := newRootCmd("test-version")

	// Map of parent command -> expected subcommands
	expectedSubs := map[string][]string{
		"crawl":    {"run"},
		"audit":    {"run"},
		"report":   {"generate", "list"},
		"provider": {"list", "use"},
		"config":   {"show", "get", "set", "path"},
	}

	for _, sub := range cmd.Commands() {
		if subs, ok := expectedSubs[sub.Name()]; ok {
			found := map[string]bool{}
			for _, child := range sub.Commands() {
				found[child.Name()] = true
			}
			for _, expected := range subs {
				if !found[expected] {
					t.Errorf("expected %s to have subcommand %q", sub.Name(), expected)
				}
			}
		}
	}
}
