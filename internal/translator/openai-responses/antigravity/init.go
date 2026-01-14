// Package antigravity provides response translation for Antigravity -> OpenaiResponse direction.
package antigravity

import (
	. "github.com/router-for-me/CLIProxyAPI/v6/internal/constant"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/interfaces"
	responses "github.com/router-for-me/CLIProxyAPI/v6/internal/translator/antigravity/openai/responses"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/translator/translator"
)

func init() {
	translator.Register(
		Antigravity,
		OpenaiResponse,
		nil,
		interfaces.TranslateResponse{
			Stream:    responses.ConvertAntigravityResponseToOpenAIResponses,
			NonStream: responses.ConvertAntigravityResponseToOpenAIResponsesNonStream,
		},
	)
}
