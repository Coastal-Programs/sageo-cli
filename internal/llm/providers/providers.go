// Package providers blank-imports every concrete LLM driver so that their
// init() functions register with internal/llm. Import this package for side
// effects wherever llm.New is called.
package providers

import (
	_ "github.com/jakeschepis/sageo-cli/internal/llm/anthropic"
	_ "github.com/jakeschepis/sageo-cli/internal/llm/openai"
)
