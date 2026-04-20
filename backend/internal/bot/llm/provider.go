package llm

import "context"

type GenerateInput struct {
	SystemPrompt string
	UserPrompt   string
	Messages     []Message
}

type Message struct {
	Role    string
	Content string
}

type GenerateResult struct {
	Content          string
	HTTPStatus       int
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	RawBodyPreview   string
	RawHeaders       map[string]string
	ChoicesCount     int
	ChoiceSummary    string
	ContentPreview   string
	DisplayableFound bool
	ReasoningOnly    bool
}

type ProviderError struct {
	StatusCode int
	Message    string
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	if e.StatusCode <= 0 {
		return e.Message
	}
	return "llm status=" + itoa(e.StatusCode) + " body=" + e.Message
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	buf := make([]byte, 0, 16)
	for v > 0 {
		buf = append(buf, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return sign + string(buf)
}

type Provider interface {
	GenerateReply(ctx context.Context, input GenerateInput) (GenerateResult, error)
}
