// Package agent provides a minimal, pluggable tool execution layer for agentic loops.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ToolCall describes a single tool invocation requested by a model.
type ToolCall struct {
	ID         string
	Name       string
	Arguments  json.RawMessage
	RawPayload string
}

// ToolResult is the output returned from a tool execution.
type ToolResult struct {
	ID      string
	Name    string
	Content string
}

// ToolHandler executes a single tool call.
type ToolHandler func(ctx context.Context, call ToolCall) (ToolResult, error)

// Registry provides lookup for tool handlers.
type Registry interface {
	Get(name string) (ToolHandler, bool)
}

// RegistryMap stores tool handlers in memory.
type RegistryMap struct {
	mu    sync.RWMutex
	tools map[string]ToolHandler
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *RegistryMap {
	return &RegistryMap{tools: make(map[string]ToolHandler)}
}

// Register stores a tool handler in the registry.
func (r *RegistryMap) Register(name string, handler ToolHandler) {
	if r == nil || handler == nil {
		return
	}
	key := strings.TrimSpace(name)
	if key == "" {
		return
	}
	r.mu.Lock()
	r.tools[key] = handler
	r.mu.Unlock()
}

// Get returns a tool handler by name.
func (r *RegistryMap) Get(name string) (ToolHandler, bool) {
	if r == nil {
		return nil, false
	}
	key := strings.TrimSpace(name)
	if key == "" {
		return nil, false
	}
	r.mu.RLock()
	handler, ok := r.tools[key]
	r.mu.RUnlock()
	return handler, ok
}

var defaultRegistry = NewRegistry()

// RegisterTool registers a tool handler in the default registry.
func RegisterTool(name string, handler ToolHandler) {
	defaultRegistry.Register(name, handler)
}

// DefaultRegistry returns the default tool registry.
func DefaultRegistry() *RegistryMap {
	return defaultRegistry
}

// ExecuteOptions controls tool execution behavior.
type ExecuteOptions struct {
	Parallel       bool
	MaxConcurrency int
	Timeout        time.Duration
}

// ExecuteToolCalls runs tool calls through the registry and returns ordered results.
func ExecuteToolCalls(ctx context.Context, calls []ToolCall, opts ExecuteOptions, registry Registry) []ToolResult {
	if registry == nil {
		registry = defaultRegistry
	}
	if len(calls) == 0 {
		return nil
	}

	results := make([]ToolResult, len(calls))
	if !opts.Parallel || len(calls) == 1 {
		for i, call := range calls {
			results[i] = executeTool(ctx, call, opts, registry)
		}
		return results
	}

	maxConc := opts.MaxConcurrency
	if maxConc <= 0 {
		maxConc = len(calls)
	}
	sema := make(chan struct{}, maxConc)
	var wg sync.WaitGroup

	for i, call := range calls {
		wg.Add(1)
		i := i
		call := call
		go func() {
			defer wg.Done()
			sema <- struct{}{}
			results[i] = executeTool(ctx, call, opts, registry)
			<-sema
		}()
	}
	wg.Wait()

	return results
}

func executeTool(ctx context.Context, call ToolCall, opts ExecuteOptions, registry Registry) ToolResult {
	handler, ok := registry.Get(call.Name)
	if !ok {
		return ToolResult{
			ID:      call.ID,
			Name:    call.Name,
			Content: fmt.Sprintf(`{"error":"tool_not_found","tool":"%s"}`, call.Name),
		}
	}

	ctxCall := ctx
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctxCall, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	result, err := safeInvoke(ctxCall, call, handler)
	if err != nil {
		return ToolResult{
			ID:      call.ID,
			Name:    call.Name,
			Content: fmt.Sprintf(`{"error":%q}`, err.Error()),
		}
	}
	if result.ID == "" {
		result.ID = call.ID
	}
	if result.Name == "" {
		result.Name = call.Name
	}
	return result
}

func safeInvoke(ctx context.Context, call ToolCall, handler ToolHandler) (result ToolResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tool panic: %v", r)
		}
	}()
	result, err = handler(ctx, call)
	return result, err
}
