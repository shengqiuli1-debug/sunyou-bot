package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAICompatibleProvider_RequestAndParseStringContent(t *testing.T) {
	var seenPath string
	var seenAuth string
	var seenModel string
	var seenMsgCount int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenAuth = r.Header.Get("Authorization")
		var body struct {
			Model    string `json:"model"`
			Messages []any  `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		seenModel = body.Model
		seenMsgCount = len(body.Messages)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"  你好，已收到  "}}],"usage":{"prompt_tokens":12,"completion_tokens":8,"total_tokens":20}}`))
	}))
	defer srv.Close()

	p := NewOpenAICompatibleProvider(srv.URL+"/v1", "test-key", "nvidia/mock-model", 3*time.Second, true)
	got, err := p.GenerateReply(context.Background(), GenerateInput{
		SystemPrompt: "sys",
		UserPrompt:   "usr",
	})
	if err != nil {
		t.Fatalf("GenerateReply err = %v", err)
	}
	if seenPath != "/v1/chat/completions" {
		t.Fatalf("path mismatch, got %s", seenPath)
	}
	if !strings.HasPrefix(seenAuth, "Bearer ") {
		t.Fatalf("authorization header missing, got %q", seenAuth)
	}
	if seenModel != "nvidia/mock-model" {
		t.Fatalf("model mismatch, got %q", seenModel)
	}
	if seenMsgCount != 2 {
		t.Fatalf("message count mismatch, got %d", seenMsgCount)
	}
	if got.Content != "你好，已收到" {
		t.Fatalf("content mismatch, got %q", got.Content)
	}
	if got.HTTPStatus != 200 {
		t.Fatalf("http status mismatch, got %d", got.HTTPStatus)
	}
	if got.TotalTokens != 20 {
		t.Fatalf("token parse mismatch, got %d", got.TotalTokens)
	}
	if got.ChoicesCount != 1 {
		t.Fatalf("choices count mismatch, got %d", got.ChoicesCount)
	}
	if got.RawBodyPreview == "" {
		t.Fatalf("raw body preview should exist when debugRaw enabled")
	}
}

func TestOpenAICompatibleProvider_ParseContentArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "choices":[
		    {"message":{"content":[{"type":"text","text":"第一句"},{"type":"text","text":"第二句"}]}}
		  ],
		  "usage":{"prompt_tokens":"9","completion_tokens":"6","total_tokens":"15"}
		}`))
	}))
	defer srv.Close()

	p := NewOpenAICompatibleProvider(srv.URL+"/v1", "", "nvidia/mock-model", 3*time.Second, false)
	got, err := p.GenerateReply(context.Background(), GenerateInput{
		SystemPrompt: "sys",
		UserPrompt:   "usr",
	})
	if err != nil {
		t.Fatalf("GenerateReply err = %v", err)
	}
	if got.Content != "第一句 第二句" {
		t.Fatalf("content mismatch, got %q", got.Content)
	}
	if got.TotalTokens != 15 {
		t.Fatalf("token parse mismatch, got %d", got.TotalTokens)
	}
}

func TestOpenAICompatibleProvider_EmptyContentReturnsProviderErrorWithStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":""}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer srv.Close()

	p := NewOpenAICompatibleProvider(srv.URL, "", "m", 3*time.Second, true)
	got, err := p.GenerateReply(context.Background(), GenerateInput{
		SystemPrompt: "sys",
		UserPrompt:   "usr",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	pErr, ok := err.(*ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if pErr.StatusCode != 200 {
		t.Fatalf("status mismatch in error, got %d", pErr.StatusCode)
	}
	if got.HTTPStatus != 200 {
		t.Fatalf("result http status mismatch, got %d", got.HTTPStatus)
	}
	if got.ChoicesCount != 1 {
		t.Fatalf("choices count mismatch, got %d", got.ChoicesCount)
	}
}

func TestOpenAICompatibleProvider_ReasoningOnlyReturnsNoDisplayableError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "choices":[{"message":{"content":null,"reasoning_content":"这是 reasoning_content 里的文本"}}],
		  "usage":{"prompt_tokens":3,"completion_tokens":5,"total_tokens":8}
		}`))
	}))
	defer srv.Close()

	p := NewOpenAICompatibleProvider(srv.URL+"/v1", "", "nvidia/mock-model", 3*time.Second, true)
	got, err := p.GenerateReply(context.Background(), GenerateInput{
		SystemPrompt: "sys",
		UserPrompt:   "usr",
	})
	if err == nil {
		t.Fatal("expected no-displayable-content error, got nil")
	}
	pErr, ok := err.(*ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if !strings.Contains(strings.ToLower(pErr.Message), "reasoning only response") {
		t.Fatalf("expected reasoning-only message, got %q", pErr.Message)
	}
	if got.DisplayableFound {
		t.Fatalf("displayable content should be false")
	}
	if !got.ReasoningOnly {
		t.Fatalf("reasoning-only marker should be true")
	}
	if got.Content != "" {
		t.Fatalf("content should be empty, got %q", got.Content)
	}
	if !strings.Contains(got.ChoiceSummary, "reasoning_content") {
		t.Fatalf("choice summary mismatch, got %q", got.ChoiceSummary)
	}
}
