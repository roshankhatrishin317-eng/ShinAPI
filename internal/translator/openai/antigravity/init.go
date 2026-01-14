// Package antigravity provides response translation for Antigravity -> OpenAI direction.
// This registration enables proper response translation when the executor calls
// TranslateNonStream(antigravity, openai, ...) to convert Antigravity backend responses
// to OpenAI client format.
package antigravity

import (
	. "github.com/router-for-me/CLIProxyAPI/v6/internal/constant"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/interfaces"
	chat_completions "github.com/router-for-me/CLIProxyAPI/v6/internal/translator/antigravity/openai/chat-completions"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/translator/translator"
)

func init() {
	// Register Antigravity -> OpenAI translation.
	// This allows TranslateNonStream(antigravity, openai, ...) to find the correct response transformer.
	// The request transformer is nil since request translation goes OpenAI -> Antigravity,
	// not Antigravity -> OpenAI.
	translator.Register(
		Antigravity,
		OpenAI,
		nil, // No request transform needed for this direction
		interfaces.TranslateResponse{
			Stream:    chat_completions.ConvertAntigravityResponseToOpenAI,
			NonStream: chat_completions.ConvertAntigravityResponseToOpenAINonStream,
		},
	)
}
