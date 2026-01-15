// Package agent provides a minimal, pluggable tool execution layer for agentic loops.
// This file adds enhanced state tracking and callbacks for the agent loop.
package agent

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// AgentState represents the current state of an agent loop.
type AgentState string

const (
	// StateIdle means the agent is not running.
	StateIdle AgentState = "idle"

	// StateThinking means the agent is waiting for model response.
	StateThinking AgentState = "thinking"

	// StateExecutingTools means the agent is executing tool calls.
	StateExecutingTools AgentState = "executing_tools"

	// StateWaitingConfirmation means the agent is waiting for user confirmation.
	StateWaitingConfirmation AgentState = "waiting_confirmation"

	// StateComplete means the agent loop has completed.
	StateComplete AgentState = "complete"

	// StateError means the agent loop encountered an error.
	StateError AgentState = "error"

	// StateMaxIterations means the agent reached max iterations.
	StateMaxIterations AgentState = "max_iterations"
)

// Iteration represents a single iteration of the agent loop.
type Iteration struct {
	// Number is the 1-based iteration index.
	Number int `json:"number"`

	// State is the current state of this iteration.
	State AgentState `json:"state"`

	// StartTime is when this iteration started.
	StartTime time.Time `json:"start_time"`

	// EndTime is when this iteration ended.
	EndTime time.Time `json:"end_time,omitempty"`

	// ToolCalls are the tool calls made in this iteration.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// ToolResults are the results from tool execution.
	ToolResults []ToolResult `json:"tool_results,omitempty"`

	// Response is the raw model response.
	Response json.RawMessage `json:"response,omitempty"`

	// ThinkingContent is extracted thinking/reasoning content.
	ThinkingContent string `json:"thinking_content,omitempty"`

	// Error holds any error that occurred.
	Error string `json:"error,omitempty"`

	// TokensUsed tracks token usage for this iteration.
	TokensUsed TokenUsage `json:"tokens_used,omitempty"`
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	ThinkingTokens   int64 `json:"thinking_tokens,omitempty"`
	TotalTokens      int64 `json:"total_tokens"`
}

// IterationCallback is called after each iteration completes.
type IterationCallback func(iteration Iteration)

// ConfirmationCallback is called when the agent needs user confirmation.
// Return true to continue, false to stop.
type ConfirmationCallback func(iteration Iteration, toolCalls []ToolCall) bool

// LoopConfig configures the agent loop behavior.
type LoopConfig struct {
	// MaxIterations limits the number of loop iterations.
	MaxIterations int

	// ParallelToolCalls enables parallel tool execution.
	ParallelToolCalls bool

	// MaxConcurrency limits concurrent tool executions.
	MaxConcurrency int

	// ToolTimeout is the timeout for tool execution.
	ToolTimeout time.Duration

	// RequireConfirmation requires user confirmation before tool execution.
	RequireConfirmation bool

	// OnIteration is called after each iteration completes.
	OnIteration IterationCallback

	// OnConfirmation is called when confirmation is required.
	OnConfirmation ConfirmationCallback

	// StopSequences are strings that trigger loop termination.
	StopSequences []string
}

// DefaultLoopConfig returns sensible defaults.
func DefaultLoopConfig() LoopConfig {
	return LoopConfig{
		MaxIterations:     8,
		ParallelToolCalls: true,
		MaxConcurrency:    4,
		ToolTimeout:       30 * time.Second,
	}
}

// Loop manages the state of an agentic execution loop.
type Loop struct {
	config     LoopConfig
	registry   Registry
	iterations []Iteration
	state      AgentState
	mu         sync.RWMutex
}

// NewLoop creates a new agent loop with the given config.
func NewLoop(cfg LoopConfig, registry Registry) *Loop {
	if registry == nil {
		registry = DefaultRegistry()
	}
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 8
	}
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 4
	}
	if cfg.ToolTimeout <= 0 {
		cfg.ToolTimeout = 30 * time.Second
	}
	return &Loop{
		config:   cfg,
		registry: registry,
		state:    StateIdle,
	}
}

// State returns the current loop state.
func (l *Loop) State() AgentState {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.state
}

// Iterations returns a copy of all iterations.
func (l *Loop) Iterations() []Iteration {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]Iteration, len(l.iterations))
	copy(result, l.iterations)
	return result
}

// CurrentIteration returns the current (most recent) iteration.
func (l *Loop) CurrentIteration() *Iteration {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(l.iterations) == 0 {
		return nil
	}
	iter := l.iterations[len(l.iterations)-1]
	return &iter
}

// TotalTokensUsed returns the total tokens used across all iterations.
func (l *Loop) TotalTokensUsed() TokenUsage {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var total TokenUsage
	for _, iter := range l.iterations {
		total.PromptTokens += iter.TokensUsed.PromptTokens
		total.CompletionTokens += iter.TokensUsed.CompletionTokens
		total.ThinkingTokens += iter.TokensUsed.ThinkingTokens
		total.TotalTokens += iter.TokensUsed.TotalTokens
	}
	return total
}

// StartIteration begins a new iteration.
func (l *Loop) StartIteration() *Iteration {
	l.mu.Lock()
	defer l.mu.Unlock()

	iter := Iteration{
		Number:    len(l.iterations) + 1,
		State:     StateThinking,
		StartTime: time.Now(),
	}
	l.iterations = append(l.iterations, iter)
	l.state = StateThinking
	return &iter
}

// RecordModelResponse records the model's response for the current iteration.
func (l *Loop) RecordModelResponse(response []byte, toolCalls []ToolCall, thinking string, tokens TokenUsage) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.iterations) == 0 {
		return
	}

	idx := len(l.iterations) - 1
	l.iterations[idx].Response = response
	l.iterations[idx].ToolCalls = toolCalls
	l.iterations[idx].ThinkingContent = thinking
	l.iterations[idx].TokensUsed = tokens

	if len(toolCalls) > 0 {
		l.iterations[idx].State = StateExecutingTools
		l.state = StateExecutingTools
	} else {
		l.iterations[idx].State = StateComplete
		l.iterations[idx].EndTime = time.Now()
		l.state = StateComplete
	}
}

// RecordToolResults records tool execution results.
func (l *Loop) RecordToolResults(results []ToolResult) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.iterations) == 0 {
		return
	}

	idx := len(l.iterations) - 1
	l.iterations[idx].ToolResults = results
	l.iterations[idx].EndTime = time.Now()
}

// RecordError records an error in the current iteration.
func (l *Loop) RecordError(err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.iterations) == 0 {
		return
	}

	idx := len(l.iterations) - 1
	l.iterations[idx].State = StateError
	l.iterations[idx].Error = err.Error()
	l.iterations[idx].EndTime = time.Now()
	l.state = StateError
}

// ShouldContinue determines if the loop should continue.
func (l *Loop) ShouldContinue() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.iterations) >= l.config.MaxIterations {
		l.state = StateMaxIterations
		return false
	}

	if l.state == StateError || l.state == StateComplete || l.state == StateMaxIterations {
		return false
	}

	// Check if last iteration had tool calls (need to continue)
	if len(l.iterations) > 0 {
		lastIter := l.iterations[len(l.iterations)-1]
		return len(lastIter.ToolCalls) > 0
	}

	return true
}

// ExecuteTools executes the tool calls from the current iteration.
func (l *Loop) ExecuteTools(ctx context.Context) []ToolResult {
	l.mu.RLock()
	if len(l.iterations) == 0 {
		l.mu.RUnlock()
		return nil
	}
	idx := len(l.iterations) - 1
	toolCalls := l.iterations[idx].ToolCalls
	l.mu.RUnlock()

	if len(toolCalls) == 0 {
		return nil
	}

	// Check if confirmation is required
	if l.config.RequireConfirmation && l.config.OnConfirmation != nil {
		l.mu.Lock()
		l.state = StateWaitingConfirmation
		l.iterations[idx].State = StateWaitingConfirmation
		l.mu.Unlock()

		if !l.config.OnConfirmation(l.iterations[idx], toolCalls) {
			l.mu.Lock()
			l.state = StateComplete
			l.iterations[idx].State = StateComplete
			l.iterations[idx].EndTime = time.Now()
			l.mu.Unlock()
			return nil
		}
	}

	l.mu.Lock()
	l.state = StateExecutingTools
	l.iterations[idx].State = StateExecutingTools
	l.mu.Unlock()

	results := ExecuteToolCalls(ctx, toolCalls, ExecuteOptions{
		Parallel:       l.config.ParallelToolCalls,
		MaxConcurrency: l.config.MaxConcurrency,
		Timeout:        l.config.ToolTimeout,
	}, l.registry)

	l.RecordToolResults(results)

	// Call iteration callback if configured
	if l.config.OnIteration != nil {
		l.mu.RLock()
		iter := l.iterations[idx]
		l.mu.RUnlock()
		l.config.OnIteration(iter)
	}

	return results
}

// MarkComplete marks the loop as complete.
func (l *Loop) MarkComplete() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.state = StateComplete
}

// Reset resets the loop for reuse.
func (l *Loop) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.iterations = nil
	l.state = StateIdle
}

// Summary returns a summary of the loop execution.
type LoopSummary struct {
	State           AgentState   `json:"state"`
	TotalIterations int          `json:"total_iterations"`
	TotalDuration   string       `json:"total_duration"`
	TotalToolCalls  int          `json:"total_tool_calls"`
	TokensUsed      TokenUsage   `json:"tokens_used"`
	Iterations      []Iteration  `json:"iterations,omitempty"`
}

// Summary returns a summary of the loop execution.
func (l *Loop) Summary() LoopSummary {
	l.mu.RLock()
	defer l.mu.RUnlock()

	summary := LoopSummary{
		State:           l.state,
		TotalIterations: len(l.iterations),
		Iterations:      make([]Iteration, len(l.iterations)),
	}

	copy(summary.Iterations, l.iterations)

	if len(l.iterations) > 0 {
		start := l.iterations[0].StartTime
		end := l.iterations[len(l.iterations)-1].EndTime
		if end.IsZero() {
			end = time.Now()
		}
		summary.TotalDuration = end.Sub(start).String()

		for _, iter := range l.iterations {
			summary.TotalToolCalls += len(iter.ToolCalls)
			summary.TokensUsed.PromptTokens += iter.TokensUsed.PromptTokens
			summary.TokensUsed.CompletionTokens += iter.TokensUsed.CompletionTokens
			summary.TokensUsed.ThinkingTokens += iter.TokensUsed.ThinkingTokens
			summary.TokensUsed.TotalTokens += iter.TokensUsed.TotalTokens
		}
	}

	return summary
}
