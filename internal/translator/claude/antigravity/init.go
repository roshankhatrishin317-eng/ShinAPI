// Package antigravity provides response translation for Antigravity -> Claude direction.
package antigravity

import (
	. "github.com/router-for-me/CLIProxyAPI/v6/internal/constant"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/interfaces"
	claude "github.com/router-for-me/CLIProxyAPI/v6/internal/translator/antigravity/claude"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/translator/translator"
)

func init() {
	translator.Register(
		Antigravity,
		Claude,
		nil,
		interfaces.TranslateResponse{
			Stream:     claude.ConvertAntigravityResponseToClaude,
			NonStream:  claude.ConvertAntigravityResponseToClaudeNonStream,
			TokenCount: claude.ClaudeTokenCount,
		},
	)
}
