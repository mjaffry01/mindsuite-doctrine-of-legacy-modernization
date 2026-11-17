package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

// LLM calls Claude for the model-based agents.
type LLM struct {
	client anthropic.Client
}

// NewLLM returns nil when no API key is available.
func NewLLM(apiKey string) *LLM {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	return &LLM{client: anthropic.NewClient(option.WithAPIKey(apiKey))}
}

// JSON asks a model for output that matches schema, and decodes it into out.
// It streams the response (long outputs) and accumulates the final message.
func (l *LLM) JSON(ctx context.Context, model, system, user string, schema map[string]any, maxTokens int64, out any) error {
	params := anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: maxTokens,
		System:    []anthropic.TextBlockParam{{Text: system}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(user))},
		OutputConfig: anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{Schema: schema},
		},
	}
	stream := l.client.Messages.NewStreaming(ctx, params)
	msg := anthropic.Message{}
	for stream.Next() {
		if err := msg.Accumulate(stream.Current()); err != nil {
			return err
		}
	}
	if err := stream.Err(); err != nil {
		return err
	}
	if msg.StopReason == anthropic.StopReasonRefusal {
		return errors.New("the model declined this request")
	}
	if msg.StopReason == anthropic.StopReasonMaxTokens {
		return errors.New("the answer was longer than allowed and was cut off")
	}
	var text strings.Builder
	for _, b := range msg.Content {
		if t, ok := b.AsAny().(anthropic.TextBlock); ok {
			text.WriteString(t.Text)
		}
	}
	if err := json.Unmarshal([]byte(text.String()), out); err != nil {
		return fmt.Errorf("could not read the model's answer: %w", err)
	}
	return nil
}

// Schema helpers: structured outputs need every property listed and no extra properties.
func obj(props map[string]any) map[string]any {
	req := make([]string, 0, len(props))
	for k := range props {
		req = append(req, k)
	}
	return map[string]any{"type": "object", "properties": props, "required": req, "additionalProperties": false}
}
func arr(items map[string]any) map[string]any { return map[string]any{"type": "array", "items": items} }
func str() map[string]any                     { return map[string]any{"type": "string"} }
func integer() map[string]any                 { return map[string]any{"type": "integer"} }
func enum(v ...string) map[string]any         { return map[string]any{"type": "string", "enum": v} }
