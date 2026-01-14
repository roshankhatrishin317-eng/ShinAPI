// Package antigravity provides response translation for Antigravity -> Gemini direction.
package antigravity

import (
	. "github.com/router-for-me/CLIProxyAPI/v6/internal/constant"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/interfaces"
	gemini "github.com/router-for-me/CLIProxyAPI/v6/internal/translator/antigravity/gemini"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/translator/translator"
)

func init() {
	translator.Register(
		Antigravity,
		Gemini,
		nil,
		interfaces.TranslateResponse{
			Stream:    gemini.ConvertAntigravityResponseToGemini,
			NonStream: gemini.ConvertAntigravityResponseToGeminiNonStream,
		},
	)
}
