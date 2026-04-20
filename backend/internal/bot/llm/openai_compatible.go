package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type OpenAICompatibleProvider struct {
	baseURL  string
	apiKey   string
	model    string
	client   *http.Client
	debugRaw bool
}

const (
	defaultBaseURL         = "https://docs.newapi.pro/v1"
	defaultModel           = "deepseek-ai/DeepSeek-R1-0528"
	chatCompletionsPath    = "/chat/completions"
	defaultProviderTimeout = 20 * time.Second
)

func NewOpenAICompatibleProvider(baseURL, apiKey, model string, timeout time.Duration, debugRaw bool) *OpenAICompatibleProvider {
	if timeout <= 0 {
		timeout = defaultProviderTimeout
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	if strings.TrimSpace(model) == "" {
		model = defaultModel
	}
	return &OpenAICompatibleProvider{
		baseURL:  strings.TrimSpace(baseURL),
		apiKey:   strings.TrimSpace(apiKey),
		model:    strings.TrimSpace(model),
		client:   &http.Client{Timeout: timeout},
		debugRaw: debugRaw,
	}
}

func (p *OpenAICompatibleProvider) GenerateReply(ctx context.Context, input GenerateInput) (GenerateResult, error) {
	endpoint := p.chatCompletionsEndpoint()
	messages := buildChatMessages(input)
	reqBody := map[string]any{
		"model":       p.model,
		"messages":    messages,
		"temperature": 0.9,
		"max_tokens":  160,
		"top_p":       0.9,
	}
	buf, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return GenerateResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return GenerateResult{}, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	baseResult := GenerateResult{
		HTTPStatus: resp.StatusCode,
	}
	if p.debugRaw {
		baseResult.RawBodyPreview = rawPreview(raw, 2000)
		baseResult.RawHeaders = normalizeHeaders(resp.Header)
	}

	if resp.StatusCode >= 400 {
		return baseResult, &ProviderError{
			StatusCode: resp.StatusCode,
			Message:    "llm non-2xx response: " + rawPreview(raw, 1200),
		}
	}

	parsed, parseErr := parseCompletionResponse(raw)
	baseResult.ChoicesCount = parsed.ChoicesCount
	baseResult.ChoiceSummary = parsed.ChoiceSummary
	baseResult.PromptTokens = parsed.PromptTokens
	baseResult.CompletionTokens = parsed.CompletionTokens
	baseResult.TotalTokens = parsed.TotalTokens
	baseResult.DisplayableFound = parsed.DisplayableFound
	baseResult.ReasoningOnly = parsed.ReasoningOnly

	if parseErr != nil {
		return baseResult, &ProviderError{
			StatusCode: resp.StatusCode,
			Message:    "llm parse response failed: " + parseErr.Error(),
		}
	}
	if !parsed.DisplayableFound {
		msg := "llm no displayable content"
		if parsed.ReasoningOnly {
			msg = "llm reasoning only response"
		}
		return baseResult, &ProviderError{
			StatusCode: resp.StatusCode,
			Message:    msg,
		}
	}

	content := normalizeContent(parsed.Content)
	baseResult.Content = content
	baseResult.ContentPreview = rawPreview([]byte(content), 220)
	if content == "" {
		return baseResult, &ProviderError{
			StatusCode: resp.StatusCode,
			Message:    "llm empty content",
		}
	}
	return baseResult, nil
}

func (p *OpenAICompatibleProvider) chatCompletionsEndpoint() string {
	return BuildChatCompletionsURL(p.baseURL)
}

func ChatCompletionsPath() string {
	return chatCompletionsPath
}

func BuildChatCompletionsURL(baseURL string) string {
	endpoint := strings.TrimSpace(baseURL)
	if endpoint == "" {
		endpoint = defaultBaseURL
	}
	if strings.HasSuffix(endpoint, chatCompletionsPath) {
		return endpoint
	}
	return strings.TrimRight(endpoint, "/") + chatCompletionsPath
}

func buildChatMessages(input GenerateInput) []map[string]string {
	if len(input.Messages) > 0 {
		out := make([]map[string]string, 0, len(input.Messages))
		for _, m := range input.Messages {
			role := strings.ToLower(strings.TrimSpace(m.Role))
			if role != "system" && role != "user" && role != "assistant" {
				continue
			}
			content := strings.TrimSpace(m.Content)
			if content == "" {
				continue
			}
			out = append(out, map[string]string{
				"role":    role,
				"content": content,
			})
		}
		if len(out) > 0 {
			return out
		}
	}

	systemPrompt := strings.TrimSpace(input.SystemPrompt)
	userPrompt := strings.TrimSpace(input.UserPrompt)
	out := make([]map[string]string, 0, 2)
	if systemPrompt != "" {
		out = append(out, map[string]string{"role": "system", "content": systemPrompt})
	}
	if userPrompt != "" {
		out = append(out, map[string]string{"role": "user", "content": userPrompt})
	}
	if len(out) == 0 {
		out = append(out, map[string]string{"role": "user", "content": "继续。"})
	}
	return out
}

type parsedCompletion struct {
	Content          string
	ChoicesCount     int
	ChoiceSummary    string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	DisplayableFound bool
	ReasoningOnly    bool
}

func parseCompletionResponse(raw []byte) (parsedCompletion, error) {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return parsedCompletion{}, err
	}
	out := parsedCompletion{}
	if usage, ok := root["usage"].(map[string]any); ok {
		out.PromptTokens = intValue(usage["prompt_tokens"])
		out.CompletionTokens = intValue(usage["completion_tokens"])
		out.TotalTokens = intValue(usage["total_tokens"])
	}

	if choices, ok := root["choices"].([]any); ok {
		out.ChoicesCount = len(choices)
		summaries := make([]string, 0, min(3, len(choices)))
		for i, item := range choices {
			choice, _ := item.(map[string]any)
			content, source, hasReasoning := extractChoiceContent(choice)
			if i < 3 {
				if strings.Contains(source, "len=") {
					summaries = append(summaries, fmt.Sprintf("choice[%d].%s", i, source))
				} else {
					summaries = append(summaries, fmt.Sprintf("choice[%d].%s(len=%d)", i, source, len([]rune(content))))
				}
			}
			if hasReasoning {
				out.ReasoningOnly = true
			}
			if out.Content == "" && strings.TrimSpace(content) != "" {
				out.Content = content
				out.DisplayableFound = true
			}
		}
		out.ChoiceSummary = strings.Join(summaries, "; ")
		if out.DisplayableFound && strings.TrimSpace(out.Content) != "" {
			return out, nil
		}
	}

	// Compatibility fallback for providers returning responses-style payload.
	if content, source := extractResponsesStyle(root); strings.TrimSpace(content) != "" {
		if out.ChoiceSummary == "" {
			out.ChoiceSummary = source
		} else {
			out.ChoiceSummary = out.ChoiceSummary + "; " + source
		}
		out.Content = content
		out.DisplayableFound = true
		return out, nil
	}

	if out.ChoicesCount == 0 {
		return out, fmt.Errorf("llm empty choices")
	}
	return out, nil
}

func extractChoiceContent(choice map[string]any) (string, string, bool) {
	if len(choice) == 0 {
		return "", "empty", false
	}
	if message, ok := choice["message"].(map[string]any); ok {
		if text := extractTextValue(message["content"]); text != "" {
			return text, "message.content", false
		}
		if text := extractTextValue(message["reasoning_content"]); text != "" {
			return "", fmt.Sprintf("message.reasoning_content(len=%d)", len([]rune(text))), true
		}
		return "", "message.empty", false
	}
	if text := extractTextValue(choice["text"]); text != "" {
		return text, "text", false
	}
	if delta, ok := choice["delta"].(map[string]any); ok {
		if text := extractTextValue(delta["content"]); text != "" {
			return text, "delta.content", false
		}
	}
	return "", "no_text", false
}

func extractResponsesStyle(root map[string]any) (string, string) {
	if text := extractTextValue(root["output_text"]); text != "" {
		return text, "output_text"
	}
	output, ok := root["output"].([]any)
	if !ok || len(output) == 0 {
		return "", ""
	}
	for _, item := range output {
		node, _ := item.(map[string]any)
		if text := extractTextValue(node["content"]); text != "" {
			return text, "output[].content"
		}
	}
	return "", ""
}

func extractTextValue(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			if text := extractTextValue(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, " "))
	case map[string]any:
		if text := extractTextValue(t["text"]); text != "" {
			return text
		}
		if text := extractTextValue(t["content"]); text != "" {
			return text
		}
		if text := extractTextValue(t["value"]); text != "" {
			return text
		}
	}
	return ""
}

func normalizeContent(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return strings.Join(strings.Fields(v), " ")
}

func normalizeHeaders(h http.Header) map[string]string {
	if len(h) == 0 {
		return map[string]string{}
	}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		if len(h[k]) == 0 {
			continue
		}
		out[strings.ToLower(k)] = strings.Join(h[k], ", ")
	}
	return out
}

func rawPreview(raw []byte, max int) string {
	if max <= 0 {
		max = 2000
	}
	v := strings.TrimSpace(string(raw))
	if len([]rune(v)) <= max {
		return v
	}
	r := []rune(v)
	return string(r[:max]) + "..."
}

func intValue(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return int(i)
		}
		if f, err := t.Float64(); err == nil {
			return int(f)
		}
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return i
		}
	}
	return 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
