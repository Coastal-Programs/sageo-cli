package llm

import (
	"fmt"
	"strings"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
)

// Factory builds a Provider from the given config. Registered by driver
// packages via Register.
type Factory func(cfg *config.Config) (Provider, error)

var registry = map[string]Factory{}

// Register associates a Factory with a provider name. Safe to call from
// package init.
func Register(name string, factory Factory) {
	registry[strings.ToLower(name)] = factory
}

// New returns a Provider for the requested name. If name is empty, the
// default from cfg.LLMProvider is used, falling back to "anthropic".
func New(name string, cfg *config.Config) (Provider, error) {
	resolved := strings.ToLower(strings.TrimSpace(name))
	if resolved == "" && cfg != nil {
		resolved = strings.ToLower(strings.TrimSpace(cfg.LLMProvider))
	}
	if resolved == "" {
		resolved = "anthropic"
	}
	factory, ok := registry[resolved]
	if !ok {
		return nil, fmt.Errorf("llm: unknown provider %q (available: %s)", resolved, availableNames())
	}
	return factory(cfg)
}

// Available returns the registered provider names.
func Available() []string {
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	return names
}

func availableNames() string {
	names := Available()
	if len(names) == 0 {
		return "<none registered>"
	}
	return strings.Join(names, ", ")
}
