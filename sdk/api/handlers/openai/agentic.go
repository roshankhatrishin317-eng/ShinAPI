package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/agent"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/api/handlers"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type agenticConfig struct {
	Enabled           bool
	MaxSteps          int
	ParallelToolCalls bool
	MaxConcurrency    int
	ToolTimeout       time.Duration
}

const (
	defaultAgenticMaxSteps       = 8
	defaultAgenticMaxConcurrency = 4
	defaultAgenticToolTimeout    = 30 * time.Second
)

func parseAgenticConfig(rawJSON []byte) (agenticConfig, []byte) {
	cfg := agenticConfig{
		MaxSteps:          defaultAgenticMaxSteps,
		ParallelToolCalls: true,
		MaxConcurrency:    defaultAgenticMaxConcurrency,
		ToolTimeout:       defaultAgenticToolTimeout,
	}

	agentic := gjson.GetBytes(rawJSON, "agentic")
	if !agentic.Exists() {
		return cfg, rawJSON
	}

	cleaned, err := sjson.DeleteBytes(rawJSON, "agentic")
	if err == nil {
		rawJSON = cleaned
	}

	if agentic.Type == gjson.True {
		cfg.Enabled = true
		return cfg, rawJSON
	}

	if agentic.IsObject() {
		if enabled := agentic.Get("enabled"); enabled.Exists() {
			cfg.Enabled = enabled.Bool()
		} else {
			cfg.Enabled = true
		}

		if v := agentic.Get("max_steps"); v.Exists() {
			cfg.MaxSteps = int(v.Int())
		}
		if v := agentic.Get("parallel_tool_calls"); v.Exists() {
			cfg.ParallelToolCalls = v.Bool()
		}
		if v := agentic.Get("max_concurrency"); v.Exists() {
			cfg.MaxConcurrency = int(v.Int())
		}
		if v := agentic.Get("tool_timeout_ms"); v.Exists() {
			cfg.ToolTimeout = time.Duration(v.Int()) * time.Millisecond
		}
	}

	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = defaultAgenticMaxSteps
	}
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = defaultAgenticMaxConcurrency
	}
	if cfg.ToolTimeout <= 0 {
		cfg.ToolTimeout = defaultAgenticToolTimeout
	}

	return cfg, rawJSON
}

func (h *OpenAIAPIHandler) handleAgenticNonStreamingResponse(c *gin.Context, rawJSON []byte, cfg agenticConfig) {
	c.Header("Content-Type", "application/json")

	requestJSON, _ := sjson.SetBytes(rawJSON, "stream", false)
	alt := h.GetAlt(c)
	modelName := gjson.GetBytes(requestJSON, "model").String()

	// Initialize agent loop with config
	loopCfg := agent.LoopConfig{
		MaxIterations:     cfg.MaxSteps,
		ParallelToolCalls: cfg.ParallelToolCalls,
		MaxConcurrency:    cfg.MaxConcurrency,
		ToolTimeout:       cfg.ToolTimeout,
	}
	loop := agent.NewLoop(loopCfg, agent.DefaultRegistry())

	for loop.ShouldContinue() {
		loop.StartIteration()

		cliCtx, cliCancel := h.GetContextWithCancel(h, c, context.Background())
		resp, errMsg := h.ExecuteWithAuthManager(cliCtx, h.HandlerType(), modelName, requestJSON, alt)
		if errMsg != nil {
			h.WriteErrorResponse(c, errMsg)
			cliCancel(errMsg.Error)
			loop.RecordError(errMsg.Error)
			return
		}
		cliCancel(nil)

		assistantMsg, toolCalls, err := extractToolCallsFromChatResponse(resp)
		if err != nil {
			c.JSON(httpStatusBadRequest, handlers.ErrorResponse{
				Error: handlers.ErrorDetail{
					Message: err.Error(),
					Type:    "invalid_request_error",
				},
			})
			loop.RecordError(err)
			return
		}

		// Record model response with tool calls
		loop.RecordModelResponse(resp, toolCalls, "", agent.TokenUsage{})

		if len(toolCalls) == 0 {
			_, _ = c.Writer.Write(resp)
			loop.MarkComplete()
			return
		}

		// Execute tools through the loop
		results := loop.ExecuteTools(c.Request.Context())

		requestJSON, err = appendAgenticMessages(requestJSON, assistantMsg, results)
		if err != nil {
			c.JSON(httpStatusBadRequest, handlers.ErrorResponse{
				Error: handlers.ErrorDetail{
					Message: err.Error(),
					Type:    "invalid_request_error",
				},
			})
			loop.RecordError(err)
			return
		}
	}

	// Loop ended due to max iterations
	c.JSON(httpStatusBadRequest, handlers.ErrorResponse{
		Error: handlers.ErrorDetail{
			Message: fmt.Sprintf("agentic max_steps (%d) reached after %d iterations",
				cfg.MaxSteps, len(loop.Iterations())),
			Type: "invalid_request_error",
		},
	})
}

func extractToolCallsFromChatResponse(resp []byte) ([]byte, []agent.ToolCall, error) {
	root := gjson.ParseBytes(resp)
	choice := root.Get("choices.0")
	if !choice.Exists() {
		return nil, nil, nil
	}

	message := choice.Get("message")
	if !message.Exists() {
		return nil, nil, nil
	}

	toolCalls := message.Get("tool_calls")
	if toolCalls.Exists() && toolCalls.IsArray() {
		calls, assistantMsg, err := extractToolCallsFromArray(toolCalls, message)
		return assistantMsg, calls, err
	}

	legacy := message.Get("function_call")
	if legacy.Exists() {
		callID := "call_1"
		name := legacy.Get("name").String()
		args := legacy.Get("arguments").String()
		call := agent.ToolCall{
			ID:         callID,
			Name:       name,
			Arguments:  normalizeArguments(args),
			RawPayload: args,
		}

		msgJSON := `{"role":"assistant","tool_calls":[]}`
		if content := message.Get("content"); content.Exists() {
			msgJSON, _ = sjson.Set(msgJSON, "content", content.String())
		}
		toolCall := `{"id":"","type":"function","function":{"name":"","arguments":""}}`
		toolCall, _ = sjson.Set(toolCall, "id", callID)
		toolCall, _ = sjson.Set(toolCall, "function.name", name)
		toolCall, _ = sjson.Set(toolCall, "function.arguments", args)
		msgJSON, _ = sjson.SetRaw(msgJSON, "tool_calls.0", toolCall)

		return []byte(msgJSON), []agent.ToolCall{call}, nil
	}

	return nil, nil, nil
}

func extractToolCallsFromArray(toolCalls gjson.Result, message gjson.Result) ([]agent.ToolCall, []byte, error) {
	if !toolCalls.IsArray() {
		return nil, nil, nil
	}

	assistantMsg := message.Raw
	if assistantMsg == "" {
		return nil, nil, fmt.Errorf("assistant message missing")
	}
	if !message.Get("role").Exists() {
		assistantMsg, _ = sjson.Set(assistantMsg, "role", "assistant")
	}

	calls := make([]agent.ToolCall, 0, len(toolCalls.Array()))
	index := 0
	toolCalls.ForEach(func(_, call gjson.Result) bool {
		index++
		callID := strings.TrimSpace(call.Get("id").String())
		if callID == "" {
			callID = fmt.Sprintf("call_%d", index)
			assistantMsg, _ = sjson.Set(assistantMsg, fmt.Sprintf("tool_calls.%d.id", index-1), callID)
		}
		name := call.Get("function.name").String()
		args := call.Get("function.arguments").String()
		calls = append(calls, agent.ToolCall{
			ID:         callID,
			Name:       name,
			Arguments:  normalizeArguments(args),
			RawPayload: args,
		})
		return true
	})

	return calls, []byte(assistantMsg), nil
}

func normalizeArguments(raw string) json.RawMessage {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	rawBytes := []byte(trimmed)
	if json.Valid(rawBytes) {
		return rawBytes
	}
	encoded, _ := json.Marshal(trimmed)
	return encoded
}

func appendAgenticMessages(rawJSON []byte, assistantMsg []byte, results []agent.ToolResult) ([]byte, error) {
	messages := gjson.GetBytes(rawJSON, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return nil, fmt.Errorf("messages array missing")
	}

	messagesJSON := messages.Raw
	if messagesJSON == "" {
		messagesJSON = "[]"
	}

	if len(assistantMsg) > 0 {
		updated, err := sjson.SetRaw(messagesJSON, "-1", string(assistantMsg))
		if err != nil {
			return nil, err
		}
		messagesJSON = updated
	}

	for _, result := range results {
		msgJSON, err := buildToolMessage(result)
		if err != nil {
			return nil, err
		}
		updated, err := sjson.SetRaw(messagesJSON, "-1", msgJSON)
		if err != nil {
			return nil, err
		}
		messagesJSON = updated
	}

	updatedRaw, err := sjson.SetRawBytes(rawJSON, "messages", []byte(messagesJSON))
	if err != nil {
		return nil, err
	}
	return updatedRaw, nil
}

func buildToolMessage(result agent.ToolResult) (string, error) {
	msg := map[string]any{
		"role":         "tool",
		"tool_call_id": result.ID,
		"content":      result.Content,
	}
	encoded, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

const httpStatusBadRequest = 400

// handleAgenticStreamingResponse handles agentic loops with streaming responses.
// It streams each model response as SSE events, then executes tools, and continues the loop.
// Between iterations, it sends custom SSE events to notify the client of tool execution.
func (h *OpenAIAPIHandler) handleAgenticStreamingResponse(c *gin.Context, rawJSON []byte, cfg agenticConfig) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(interface{ Flush() })
	if !ok {
		c.JSON(httpStatusBadRequest, handlers.ErrorResponse{
			Error: handlers.ErrorDetail{
				Message: "streaming not supported",
				Type:    "server_error",
			},
		})
		return
	}

	alt := h.GetAlt(c)
	requestJSON := rawJSON

	for step := 0; step < cfg.MaxSteps; step++ {
		modelName := gjson.GetBytes(requestJSON, "model").String()

		// Set stream=true for the actual request
		streamReq, _ := sjson.SetBytes(requestJSON, "stream", true)

		cliCtx, cliCancel := h.GetContextWithCancel(h, c, context.Background())

		// Execute streaming request and accumulate tool calls
		resp, toolCalls, err := h.executeAgenticStreamingRequest(c, cliCtx, modelName, streamReq, alt, flusher)
		cliCancel(nil)

		if err != nil {
			// Send error as SSE event
			errJSON, _ := json.Marshal(map[string]any{
				"error": map[string]any{
					"message": err.Error(),
					"type":    "server_error",
				},
			})
			_, _ = c.Writer.Write([]byte("data: " + string(errJSON) + "\n\n"))
			flusher.Flush()
			return
		}

		// If no tool calls, we're done
		if len(toolCalls) == 0 {
			_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
			return
		}

		// Send tool execution notification event
		toolEvent := map[string]any{
			"type":  "agentic.tool_execution_start",
			"step":  step + 1,
			"tools": len(toolCalls),
		}
		toolEventJSON, _ := json.Marshal(toolEvent)
		_, _ = c.Writer.Write([]byte("data: " + string(toolEventJSON) + "\n\n"))
		flusher.Flush()

		// Execute tools
		results := agent.ExecuteToolCalls(c.Request.Context(), toolCalls, agent.ExecuteOptions{
			Parallel:       cfg.ParallelToolCalls,
			MaxConcurrency: cfg.MaxConcurrency,
			Timeout:        cfg.ToolTimeout,
		}, agent.DefaultRegistry())

		// Send tool results notification
		toolResultEvent := map[string]any{
			"type":    "agentic.tool_execution_complete",
			"step":    step + 1,
			"results": len(results),
		}
		toolResultEventJSON, _ := json.Marshal(toolResultEvent)
		_, _ = c.Writer.Write([]byte("data: " + string(toolResultEventJSON) + "\n\n"))
		flusher.Flush()

		// Append assistant message and tool results to messages
		requestJSON, err = appendAgenticMessages(requestJSON, resp, results)
		if err != nil {
			errJSON, _ := json.Marshal(map[string]any{
				"error": map[string]any{
					"message": err.Error(),
					"type":    "invalid_request_error",
				},
			})
			_, _ = c.Writer.Write([]byte("data: " + string(errJSON) + "\n\n"))
			flusher.Flush()
			return
		}
	}

	// Max steps reached
	maxStepsEvent := map[string]any{
		"type":    "agentic.max_steps_reached",
		"message": "agentic max_steps reached",
	}
	maxStepsJSON, _ := json.Marshal(maxStepsEvent)
	_, _ = c.Writer.Write([]byte("data: " + string(maxStepsJSON) + "\n\n"))
	_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}

// executeAgenticStreamingRequest executes a streaming request and returns the accumulated response.
func (h *OpenAIAPIHandler) executeAgenticStreamingRequest(
	c *gin.Context,
	ctx context.Context,
	modelName string,
	requestJSON []byte,
	alt string,
	flusher interface{ Flush() },
) ([]byte, []agent.ToolCall, error) {
	// Execute the streaming request
	respChan, errChan := h.ExecuteStreamingWithAuthManager(ctx, h.HandlerType(), modelName, requestJSON, alt)

	var assistantMsgBuilder strings.Builder
	var toolCalls []agent.ToolCall
	var lastChunk []byte

	assistantMsgBuilder.WriteString(`{"role":"assistant","content":"","tool_calls":[]}`)

	for {
		select {
		case chunk, ok := <-respChan:
			if !ok {
				// Channel closed, check for tool calls
				if len(toolCalls) > 0 {
					// Build assistant message with tool calls
					assistantMsg := assistantMsgBuilder.String()
					for i, tc := range toolCalls {
						toolCallJSON := fmt.Sprintf(`{"id":"%s","type":"function","function":{"name":"%s","arguments":%s}}`,
							tc.ID, tc.Name, tc.RawPayload)
						assistantMsg, _ = sjson.SetRaw(assistantMsg, fmt.Sprintf("tool_calls.%d", i), toolCallJSON)
					}
					return []byte(assistantMsg), toolCalls, nil
				}
				return lastChunk, nil, nil
			}

			// Forward chunk to client
			_, _ = c.Writer.Write(chunk)
			_, _ = c.Writer.Write([]byte("\n"))
			flusher.Flush()

			// Parse the SSE data
			if len(chunk) > 6 && string(chunk[:6]) == "data: " {
				data := chunk[6:]
				if string(data) == "[DONE]" {
					continue
				}

				lastChunk = data

				// Extract content delta
				contentDelta := gjson.GetBytes(data, "choices.0.delta.content")
				if contentDelta.Exists() {
					// Append to content
					currentContent := gjson.Get(assistantMsgBuilder.String(), "content").String()
					newContent := currentContent + contentDelta.String()
					newMsg, _ := sjson.Set(assistantMsgBuilder.String(), "content", newContent)
					assistantMsgBuilder.Reset()
					assistantMsgBuilder.WriteString(newMsg)
				}

				// Extract tool calls
				tcDelta := gjson.GetBytes(data, "choices.0.delta.tool_calls")
				if tcDelta.Exists() && tcDelta.IsArray() {
					for _, tc := range tcDelta.Array() {
						idx := int(tc.Get("index").Int())

						// Ensure we have enough slots
						for len(toolCalls) <= idx {
							toolCalls = append(toolCalls, agent.ToolCall{})
						}

						// Update ID if present
						if id := tc.Get("id"); id.Exists() && id.String() != "" {
							toolCalls[idx].ID = id.String()
						}

						// Update function name if present
						if name := tc.Get("function.name"); name.Exists() && name.String() != "" {
							toolCalls[idx].Name = name.String()
						}

						// Append to arguments
						if args := tc.Get("function.arguments"); args.Exists() {
							toolCalls[idx].RawPayload += args.String()
						}
					}
				}

				// Check finish reason
				finishReason := gjson.GetBytes(data, "choices.0.finish_reason")
				if finishReason.Exists() && finishReason.String() == "tool_calls" {
					// Finalize tool calls
					for i := range toolCalls {
						if toolCalls[i].ID == "" {
							toolCalls[i].ID = fmt.Sprintf("call_%d", i+1)
						}
						toolCalls[i].Arguments = normalizeArguments(toolCalls[i].RawPayload)
					}
				}
			}

		case err := <-errChan:
			return nil, nil, err

		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}
}

// ExecuteStreamingWithAuthManager is a placeholder interface method.
// The actual implementation should be in the handler's base.
func (h *OpenAIAPIHandler) ExecuteStreamingWithAuthManager(ctx context.Context, handlerType, modelName string, requestJSON []byte, alt string) (<-chan []byte, <-chan error) {
	respChan := make(chan []byte)
	errChan := make(chan error, 1)

	go func() {
		defer close(respChan)
		defer close(errChan)

		// This is a stub - the actual streaming implementation depends on the base handler
		// For now, fall back to non-streaming and convert
		resp, errMsg := h.ExecuteWithAuthManager(ctx, handlerType, modelName, requestJSON, alt)
		if errMsg != nil {
			errChan <- errMsg.Error
			return
		}

		// Convert non-streaming response to SSE format
		sseData := "data: " + string(resp) + "\n\n"
		respChan <- []byte(sseData)
		respChan <- []byte("data: [DONE]\n\n")
	}()

	return respChan, errChan
}
