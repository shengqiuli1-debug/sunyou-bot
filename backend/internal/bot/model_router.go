package bot

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"sunyou-bot/backend/internal/bot/character"
	"sunyou-bot/backend/internal/bot/llm"
	"sunyou-bot/backend/internal/models"
)

const (
	poolChatMain     = "chat_main_pool"
	poolPremium      = "premium_fallback_pool"
	poolReasoning    = "reasoning_pool"
	poolCode         = "code_pool"
	poolSpecialized  = "specialized_do_not_use_for_chat"
	defaultTryNormal = 3
	circuitModel     = "model"
	circuitProvider  = "provider"
)

func normalizeModelConfigs(opts Options) []ModelConfig {
	out := make([]ModelConfig, 0, 16)
	seen := map[string]bool{}
	push := func(cfg ModelConfig) {
		cfg.Name = strings.TrimSpace(cfg.Name)
		if cfg.Name == "" {
			return
		}
		if strings.TrimSpace(cfg.Provider) == "" {
			cfg.Provider = opts.LLMProviderName
		}
		if strings.TrimSpace(cfg.BaseURL) == "" {
			cfg.BaseURL = opts.LLMBaseURL
		}
		if strings.TrimSpace(cfg.Pool) == "" {
			cfg.Pool = poolChatMain
		}
		if cfg.TimeoutSeconds <= 0 {
			if opts.LLMPerModelTOsec > 0 {
				cfg.TimeoutSeconds = opts.LLMPerModelTOsec
			} else {
				cfg.TimeoutSeconds = opts.LLMTimeoutSeconds
			}
		}
		key := strings.ToLower(strings.TrimSpace(cfg.Pool) + "|" + cfg.Name)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, cfg)
	}

	if len(opts.LLMModels) > 0 {
		for _, item := range opts.LLMModels {
			push(item)
		}
	} else if strings.TrimSpace(opts.LLMModel) != "" {
		push(ModelConfig{
			Name:           opts.LLMModel,
			Provider:       opts.LLMProviderName,
			BaseURL:        opts.LLMBaseURL,
			APIKey:         "",
			Enabled:        true,
			Priority:       1,
			Pool:           poolChatMain,
			TimeoutSeconds: opts.LLMPerModelTOsec,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Pool != out[j].Pool {
			return out[i].Pool < out[j].Pool
		}
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (e *Engine) providerReady() bool {
	if !e.llmEnabled {
		return false
	}
	for _, cfg := range e.llmModels {
		if !cfg.Enabled {
			continue
		}
		if strings.TrimSpace(cfg.Name) == "" || strings.TrimSpace(cfg.BaseURL) == "" {
			continue
		}
		return true
	}
	return e.llmProvider != nil
}

func (e *Engine) listPools() (chatMain []string, premium []string, reasoning []string, code []string, specialized []string) {
	chatMain = make([]string, 0, 12)
	premium = make([]string, 0, 4)
	reasoning = make([]string, 0, 8)
	code = make([]string, 0, 4)
	specialized = make([]string, 0, 12)
	for _, cfg := range e.llmModels {
		if !cfg.Enabled {
			continue
		}
		switch cfg.Pool {
		case poolPremium:
			premium = append(premium, cfg.Name)
		case poolReasoning:
			reasoning = append(reasoning, cfg.Name)
		case poolCode:
			code = append(code, cfg.Name)
		case poolSpecialized:
			specialized = append(specialized, cfg.Name)
		default:
			chatMain = append(chatMain, cfg.Name)
		}
	}
	return chatMain, premium, reasoning, code, specialized
}

type poolPlan struct {
	Pool        string
	MaxAttempts int
}

func (e *Engine) selectPoolPlan(in Input, forceReply bool, spec character.Spec) []poolPlan {
	modelPolicy := spec.ModelPolicy
	normalAttempts := modelPolicy.NormalAttempts
	if normalAttempts <= 0 {
		normalAttempts = defaultTryNormal
	}
	codeAttempts := modelPolicy.CodeAttempts
	if codeAttempts <= 0 {
		codeAttempts = defaultTryNormal
	}
	forceMainAttempts := modelPolicy.ForceMainAttempts
	if forceMainAttempts <= 0 {
		forceMainAttempts = 2
	}
	forcePremiumAttempts := modelPolicy.ForcePremiumAttempt
	if forcePremiumAttempts <= 0 {
		forcePremiumAttempts = 1
	}

	preferred := make([]string, 0, len(modelPolicy.PreferredPools))
	for _, p := range modelPolicy.PreferredPools {
		pool := strings.TrimSpace(p)
		if pool == "" {
			continue
		}
		preferred = append(preferred, pool)
	}
	if len(preferred) == 0 {
		preferred = append(preferred, poolChatMain)
	}

	primaryPool := preferred[0]
	if forceReply {
		plan := []poolPlan{{Pool: primaryPool, MaxAttempts: forceMainAttempts}}
		if modelPolicy.UsePremiumOnForce {
			plan = append(plan, poolPlan{Pool: poolPremium, MaxAttempts: forcePremiumAttempts})
		}
		return plan
	}
	if isCodeLikeContent(in.Content, in.RecentMessages) {
		return []poolPlan{{Pool: poolCode, MaxAttempts: codeAttempts}}
	}

	plan := make([]poolPlan, 0, len(preferred))
	for _, pool := range preferred {
		plan = append(plan, poolPlan{Pool: pool, MaxAttempts: normalAttempts})
	}
	return plan
}

func isCodeLikeContent(content string, recent []models.ChatMessage) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	if c == "" {
		return false
	}
	markers := []string{
		"```", "traceback", "exception", "panic:", "stack trace", "error:", "fatal:", "segmentation fault",
		".go", ".ts", ".js", ".py", ".java", ".rs", "npm ", "pnpm ", "yarn ", "go test", "go build",
		"sqlstate", "syntaxerror", "undefined", "nullpointer",
	}
	for _, m := range markers {
		if strings.Contains(c, m) {
			return true
		}
	}
	if strings.Count(c, "{") > 1 && strings.Count(c, "}") > 1 {
		return true
	}
	for _, msg := range recent {
		if msg.SenderType != "user" {
			continue
		}
		lower := strings.ToLower(msg.Content)
		if strings.Contains(lower, "```") || strings.Contains(lower, "error") {
			return true
		}
	}
	return false
}

func (e *Engine) allPoolModels(pool string) []ModelConfig {
	all := make([]ModelConfig, 0, 12)
	for _, cfg := range e.llmModels {
		if !cfg.Enabled {
			continue
		}
		if cfg.Pool != pool {
			continue
		}
		all = append(all, cfg)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Priority != all[j].Priority {
			return all[i].Priority < all[j].Priority
		}
		return all[i].Name < all[j].Name
	})
	return all
}

func (e *Engine) availableModels(ctx context.Context, pool string) ([]ModelConfig, []string) {
	all := e.allPoolModels(pool)

	now := time.Now().Unix()
	out := make([]ModelConfig, 0, len(all))
	skipped := make([]string, 0, len(all))
	for _, cfg := range all {
		if until, ok := e.providerCoolingUntil(ctx, cfg, now); ok {
			skipped = append(skipped, fmt.Sprintf("%s:provider_cooling_until_%s", cfg.Name, time.Unix(until, 0).UTC().Format(time.RFC3339)))
			continue
		}
		if e.isModelDisabled(ctx, cfg) {
			skipped = append(skipped, fmt.Sprintf("%s:model_disabled", cfg.Name))
			continue
		}
		if until, ok := e.modelCoolingUntil(ctx, cfg, now); ok {
			skipped = append(skipped, fmt.Sprintf("%s:model_cooling_until_%s", cfg.Name, time.Unix(until, 0).UTC().Format(time.RFC3339)))
			continue
		}
		out = append(out, cfg)
	}
	return out, skipped
}

func (e *Engine) prioritizeHealthy(pool string, candidates []ModelConfig) []ModelConfig {
	if len(candidates) <= 1 {
		return candidates
	}
	healthy := e.getHealthyCandidates(pool)
	if len(healthy) == 0 {
		return candidates
	}
	byName := make(map[string]ModelConfig, len(candidates))
	for _, c := range candidates {
		byName[c.Name] = c
	}
	used := make(map[string]bool, len(candidates))
	out := make([]ModelConfig, 0, len(candidates))
	for _, name := range healthy {
		if c, ok := byName[name]; ok && !used[name] {
			out = append(out, c)
			used[name] = true
		}
	}
	for _, c := range candidates {
		if !used[c.Name] {
			out = append(out, c)
		}
	}
	return out
}

func (e *Engine) getHealthyCandidates(pool string) []string {
	e.healthyMu.RLock()
	defer e.healthyMu.RUnlock()
	items := e.healthyCandidates[pool]
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	out = append(out, items...)
	return out
}

func (e *Engine) setHealthyCandidates(pool string, models []string) {
	uniq := make([]string, 0, len(models))
	seen := map[string]bool{}
	for _, m := range models {
		name := strings.TrimSpace(m)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		uniq = append(uniq, name)
	}
	e.healthyMu.Lock()
	e.healthyCandidates[pool] = uniq
	e.healthyMu.Unlock()
	if e.logger != nil {
		e.logger.Info("[LLM] healthy_candidates_updated",
			"pool_name", pool,
			"healthy_candidates", joinList(uniq),
		)
	}
}

func (e *Engine) markHealthyCandidate(pool, model string, ok bool) {
	pool = strings.TrimSpace(pool)
	model = strings.TrimSpace(model)
	if pool == "" || model == "" {
		return
	}
	current := e.getHealthyCandidates(pool)
	seen := map[string]bool{}
	for _, v := range current {
		seen[v] = true
	}
	if ok {
		if seen[model] {
			return
		}
		current = append([]string{model}, current...)
		if len(current) > 12 {
			current = current[:12]
		}
		e.setHealthyCandidates(pool, current)
		return
	}
	if !seen[model] {
		return
	}
	next := make([]string, 0, len(current))
	for _, v := range current {
		if v != model {
			next = append(next, v)
		}
	}
	e.setHealthyCandidates(pool, next)
}

func (e *Engine) providerForModel(cfg ModelConfig) llm.Provider {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "", "openai_compatible":
		timeout := cfg.TimeoutSeconds
		if timeout <= 0 {
			timeout = e.llmPerModelTO
		}
		if timeout <= 0 {
			timeout = e.llmTimeoutSec
		}
		if timeout <= 0 {
			timeout = 10
		}
		return llm.NewOpenAICompatibleProvider(
			cfg.BaseURL,
			strings.TrimSpace(cfg.APIKey),
			cfg.Name,
			time.Duration(timeout)*time.Second,
			e.llmDebugRawResp,
		)
	default:
		return nil
	}
}

func (e *Engine) applyModelResult(ctx context.Context, cfg ModelConfig, failClass string, ok bool) (circuitUntil int64, circuitScope string, modelUntil int64, providerUntil int64) {
	if ok {
		_ = e.redis.Del(ctx, e.modelCooldownKey(cfg), e.modelFailCountKey(cfg), e.modelLastErrKey(cfg)).Err()
		_ = e.redis.Set(ctx, e.modelLastSuccessKey(cfg), strconv.FormatInt(time.Now().Unix(), 10), 24*time.Hour).Err()
		e.markHealthyCandidate(cfg.Pool, cfg.Name, true)
		return 0, "", 0, 0
	}

	_ = e.redis.Incr(ctx, e.modelFailCountKey(cfg)).Err()
	_ = e.redis.Set(ctx, e.modelLastErrKey(cfg), failClass, 24*time.Hour).Err()
	e.markHealthyCandidate(cfg.Pool, cfg.Name, false)
	circuitScope = circuitScopeForFailureClass(failClass)

	modelCD, providerCD, disableModel, disableProvider := cooldownByFailureClass(failClass)
	if disableModel {
		_ = e.redis.Set(ctx, e.modelDisabledKey(cfg), "1", 24*time.Hour).Err()
	}
	if modelCD > 0 {
		until := time.Now().Add(modelCD).Unix()
		_ = e.redis.Set(ctx, e.modelCooldownKey(cfg), strconv.FormatInt(until, 10), modelCD+time.Hour).Err()
		modelUntil = until
		circuitUntil = until
	}
	if circuitScope == circuitProvider && (disableProvider || providerCD > 0) {
		cd := providerCD
		if cd <= 0 {
			cd = 2 * time.Hour
		}
		until := time.Now().Add(cd).Unix()
		_ = e.redis.Set(ctx, e.providerCooldownKey(cfg), strconv.FormatInt(until, 10), cd+time.Hour).Err()
		providerUntil = until
		if until > circuitUntil {
			circuitUntil = until
		}
	}
	return circuitUntil, circuitScope, modelUntil, providerUntil
}

func (e *Engine) schedulePoolProbe(poolName, reason string) {
	poolName = strings.TrimSpace(poolName)
	if poolName == "" || !e.llmEnabled {
		return
	}
	e.probeMu.Lock()
	if e.probeRunning[poolName] {
		e.probeMu.Unlock()
		return
	}
	e.probeRunning[poolName] = true
	e.probeMu.Unlock()

	go func() {
		defer func() {
			e.probeMu.Lock()
			e.probeRunning[poolName] = false
			e.probeMu.Unlock()
		}()
		budget := e.llmProbeBudget
		if budget <= 0 {
			budget = 60
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(budget)*time.Second)
		defer cancel()
		e.runPoolProbe(ctx, poolName, reason)
	}()
}

func (e *Engine) runPoolProbe(ctx context.Context, poolName, reason string) {
	models := e.allPoolModels(poolName)
	total := len(models)
	started := time.Now()
	if e.logger != nil {
		e.logger.Info("[LLM] pool_probe_started",
			"pool_name", poolName,
			"reason", reason,
			"probe_total_models", total,
			"healthy_candidates", joinList(e.getHealthyCandidates(poolName)),
		)
	}
	if total == 0 {
		return
	}

	checked := 0
	failed := 0
	success := 0
	healthy := make([]string, 0, total)
	for _, candidate := range models {
		select {
		case <-ctx.Done():
			if e.logger != nil {
				e.logger.Warn("[LLM] pool_probe_finished",
					"pool_name", poolName,
					"reason", reason,
					"probe_total_models", total,
					"probe_checked_models", checked,
					"probe_failed_models", failed,
					"probe_success_models", success,
					"healthy_candidates", joinList(healthy),
					"request_total_model_time_ms", int(time.Since(started).Milliseconds()),
					"error", ctx.Err(),
				)
			}
			if len(healthy) > 0 {
				e.setHealthyCandidates(poolName, healthy)
			}
			return
		default:
		}

		checked++
		now := time.Now().Unix()
		if until, ok := e.providerCoolingUntil(ctx, candidate, now); ok {
			if e.logger != nil {
				e.logger.Info("[LLM] pool_probe_model_checked",
					"pool_name", poolName,
					"model_name", candidate.Name,
					"skipped", true,
					"skip_reason", "provider_cooling",
					"circuit_scope", circuitProvider,
					"circuit_key", e.providerScopeKey(candidate),
					"provider_circuit_open_until", time.Unix(until, 0).UTC().Format(time.RFC3339),
				)
			}
			continue
		}
		if e.isModelDisabled(ctx, candidate) {
			if e.logger != nil {
				e.logger.Info("[LLM] pool_probe_model_checked",
					"pool_name", poolName,
					"model_name", candidate.Name,
					"skipped", true,
					"skip_reason", "model_disabled",
				)
			}
			continue
		}
		if until, ok := e.modelCoolingUntil(ctx, candidate, now); ok {
			if e.logger != nil {
				e.logger.Info("[LLM] pool_probe_model_checked",
					"pool_name", poolName,
					"model_name", candidate.Name,
					"skipped", true,
					"skip_reason", "model_cooling",
					"circuit_scope", circuitModel,
					"circuit_key", e.modelScopeKey(candidate),
					"model_circuit_open_until", time.Unix(until, 0).UTC().Format(time.RFC3339),
				)
			}
			continue
		}

		timeoutSec := candidate.TimeoutSeconds
		if timeoutSec <= 0 {
			timeoutSec = e.llmProbeTO
		}
		if timeoutSec <= 0 {
			timeoutSec = 4
		}
		perModelCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		provider := e.providerForModel(candidate)
		if provider == nil {
			cancel()
			failClass := "provider_misconfig"
			circuitUntil, circuitScope, modelUntil, providerUntil := e.applyModelResult(context.Background(), candidate, failClass, false)
			failed++
			if e.logger != nil {
				modelUntilStr := ""
				if modelUntil > 0 {
					modelUntilStr = time.Unix(modelUntil, 0).UTC().Format(time.RFC3339)
				}
				providerUntilStr := ""
				if providerUntil > 0 {
					providerUntilStr = time.Unix(providerUntil, 0).UTC().Format(time.RFC3339)
				}
				e.logger.Warn("[LLM] pool_probe_model_failed",
					"pool_name", poolName,
					"model_name", candidate.Name,
					"last_error_type", failClass,
					"circuit_scope", circuitScope,
					"circuit_key", e.modelScopeKey(candidate),
					"provider_circuit_key", e.providerScopeKey(candidate),
					"circuit_open_until", time.Unix(circuitUntil, 0).UTC().Format(time.RFC3339),
					"model_circuit_open_until", modelUntilStr,
					"provider_circuit_open_until", providerUntilStr,
				)
			}
			continue
		}

		result, err := provider.GenerateReply(perModelCtx, llm.GenerateInput{
			SystemPrompt: "你是可用性探测助手，仅回复“好”。",
			UserPrompt:   "回复一个字：好",
			Messages: []llm.Message{
				{Role: "system", Content: "你是可用性探测助手，仅回复“好”。"},
				{Role: "user", Content: "回复一个字：好"},
			},
		})
		cancel()
		if err == nil {
			reply := strings.TrimSpace(result.Content)
			if reply != "" && !IsPlannerLeakText(reply) && result.DisplayableFound && !result.ReasoningOnly {
				_, _, _, _ = e.applyModelResult(context.Background(), candidate, "", true)
				healthy = append(healthy, candidate.Name)
				success++
				if e.logger != nil {
					e.logger.Info("[LLM] pool_probe_model_success",
						"pool_name", poolName,
						"model_name", candidate.Name,
						"http_status", result.HTTPStatus,
					)
				}
				continue
			}
			err = fmt.Errorf("probe no displayable content")
		}

		meta := generateMeta{
			HTTPStatus:         result.HTTPStatus,
			RawResponsePreview: result.RawBodyPreview,
			ErrorMessage:       shortErr(err),
			DisplayableFound:   result.DisplayableFound,
			ReasoningOnly:      result.ReasoningOnly,
		}
		failClass := classifyLLMFailure(err, meta)
		circuitUntil, circuitScope, modelUntil, providerUntil := e.applyModelResult(context.Background(), candidate, failClass, false)
		failed++
		if e.logger != nil {
			modelUntilStr := ""
			if modelUntil > 0 {
				modelUntilStr = time.Unix(modelUntil, 0).UTC().Format(time.RFC3339)
			}
			providerUntilStr := ""
			if providerUntil > 0 {
				providerUntilStr = time.Unix(providerUntil, 0).UTC().Format(time.RFC3339)
			}
			e.logger.Warn("[LLM] pool_probe_model_failed",
				"pool_name", poolName,
				"model_name", candidate.Name,
				"last_error_type", failClass,
				"circuit_scope", circuitScope,
				"circuit_key", e.modelScopeKey(candidate),
				"provider_circuit_key", e.providerScopeKey(candidate),
				"http_status", result.HTTPStatus,
				"circuit_open_until", time.Unix(circuitUntil, 0).UTC().Format(time.RFC3339),
				"model_circuit_open_until", modelUntilStr,
				"provider_circuit_open_until", providerUntilStr,
			)
		}
	}

	if len(healthy) > 0 {
		e.setHealthyCandidates(poolName, healthy)
	}
	if e.logger != nil {
		e.logger.Info("[LLM] pool_probe_finished",
			"pool_name", poolName,
			"reason", reason,
			"probe_total_models", total,
			"probe_checked_models", checked,
			"probe_failed_models", failed,
			"probe_success_models", success,
			"healthy_candidates", joinList(e.getHealthyCandidates(poolName)),
			"request_total_model_time_ms", int(time.Since(started).Milliseconds()),
		)
	}
}

func cooldownByFailureClass(failClass string) (modelCooldown time.Duration, providerCooldown time.Duration, disableModel bool, disableProvider bool) {
	switch failClass {
	case "quota_exceeded":
		return 6 * time.Hour, 0, false, false
	case "rate_limited":
		return 10 * time.Minute, 0, false, false
	case "timeout", "network_error", "server_error", "provider_unavailable":
		return 5 * time.Minute, 0, false, false
	case "model_not_found":
		return 1 * time.Hour, 0, false, false
	case "provider_auth", "provider_misconfig", "provider_unreachable":
		return 0, 1 * time.Hour, false, true
	case "no_displayable_content", "planner_leak", "reasoning_only_response", "robotic_tone", "robotic_repetition":
		return 15 * time.Minute, 0, false, false
	case "language_mismatch":
		return 5 * time.Minute, 0, false, false
	default:
		return 5 * time.Minute, 0, false, false
	}
}

func classifyLLMFailure(err error, meta generateMeta) string {
	msg := strings.ToLower(strings.TrimSpace(shortErr(err) + " " + meta.RawResponsePreview + " " + meta.ErrorMessage))
	status := meta.HTTPStatus

	if strings.Contains(msg, "planner leak") {
		return "planner_leak"
	}
	if strings.Contains(msg, "robotic tone") {
		return "robotic_tone"
	}
	if strings.Contains(msg, "robotic repetitive") || strings.Contains(msg, "robotic repetition") {
		return "robotic_repetition"
	}
	if strings.Contains(msg, "language mismatch") {
		return "language_mismatch"
	}
	if strings.Contains(msg, "reasoning only response") || meta.ReasoningOnly {
		return "reasoning_only_response"
	}
	if strings.Contains(msg, "no displayable content") || strings.Contains(msg, "displayable") {
		return "no_displayable_content"
	}
	if strings.Contains(msg, "today's quota") || strings.Contains(msg, "quota") {
		return "quota_exceeded"
	}
	if status == 429 || strings.Contains(msg, "rate limit") {
		return "rate_limited"
	}
	if status == 404 {
		if strings.Contains(msg, "model") {
			return "model_not_found"
		}
		return "provider_misconfig"
	}
	if status == 401 || status == 403 || strings.Contains(msg, "unauthorized") || strings.Contains(msg, "forbidden") {
		return "provider_auth"
	}
	if status >= 500 {
		return "server_error"
	}
	if strings.Contains(msg, "<html") || strings.Contains(msg, "docs.newapi.pro") && strings.Contains(msg, "not found") {
		return "provider_misconfig"
	}
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(msg, "timeout") {
		return "timeout"
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "timeout"
		}
		return "provider_unreachable"
	}
	if strings.Contains(msg, "connection refused") || strings.Contains(msg, "connection reset") || strings.Contains(msg, "no such host") || strings.Contains(msg, "server misbehaving") || strings.Contains(msg, "tls") || strings.Contains(msg, "x509") || strings.Contains(msg, "certificate") || strings.Contains(msg, "handshake") {
		return "provider_unreachable"
	}
	if strings.Contains(msg, "eof") {
		return "network_error"
	}
	if strings.Contains(msg, "provider not initialized") {
		return "provider_misconfig"
	}
	if status == 0 && strings.Contains(msg, "llm disabled") {
		return "provider_misconfig"
	}
	return classifyLLMError(err)
}

func shouldStopProviderForRound(failClass string) bool {
	switch failClass {
	case "provider_auth", "provider_misconfig", "provider_unreachable":
		return true
	default:
		return false
	}
}

func circuitScopeForFailureClass(failClass string) string {
	switch strings.TrimSpace(failClass) {
	case "provider_auth", "provider_misconfig", "provider_unreachable":
		return circuitProvider
	default:
		return circuitModel
	}
}

func (e *Engine) planAttemptLimit(limit int) int {
	if limit <= 0 {
		limit = defaultTryNormal
	}
	if e.llmMaxAttempts > 0 && limit > e.llmMaxAttempts {
		limit = e.llmMaxAttempts
	}
	if limit <= 0 {
		limit = 1
	}
	return limit
}

func (e *Engine) isModelDisabled(ctx context.Context, cfg ModelConfig) bool {
	v, err := e.redis.Get(ctx, e.modelDisabledKey(cfg)).Result()
	return err == nil && strings.TrimSpace(v) == "1"
}

func (e *Engine) modelCoolingUntil(ctx context.Context, cfg ModelConfig, now int64) (int64, bool) {
	v, err := e.redis.Get(ctx, e.modelCooldownKey(cfg)).Result()
	if err != nil {
		return 0, false
	}
	until, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	return until, until > now
}

func (e *Engine) providerCoolingUntil(ctx context.Context, cfg ModelConfig, now int64) (int64, bool) {
	v, err := e.redis.Get(ctx, e.providerCooldownKey(cfg)).Result()
	if err != nil {
		return 0, false
	}
	until, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	return until, until > now
}

func (e *Engine) modelScopeKey(cfg ModelConfig) string {
	raw := strings.ToLower(normalizeProviderEndpoint(cfg.Provider, cfg.BaseURL) + "|" + strings.TrimSpace(cfg.Name))
	return sanitizeKey(raw)
}

func (e *Engine) providerScopeKey(cfg ModelConfig) string {
	raw := strings.ToLower(normalizeProviderEndpoint(cfg.Provider, cfg.BaseURL))
	return sanitizeKey(raw)
}

func (e *Engine) modelCooldownKey(cfg ModelConfig) string {
	return "llm:model:" + e.modelScopeKey(cfg) + ":cooldown_until"
}
func (e *Engine) modelDisabledKey(cfg ModelConfig) string {
	return "llm:model:" + e.modelScopeKey(cfg) + ":disabled"
}
func (e *Engine) modelFailCountKey(cfg ModelConfig) string {
	return "llm:model:" + e.modelScopeKey(cfg) + ":consecutive_failures"
}
func (e *Engine) modelLastErrKey(cfg ModelConfig) string {
	return "llm:model:" + e.modelScopeKey(cfg) + ":last_error_class"
}
func (e *Engine) modelLastSuccessKey(cfg ModelConfig) string {
	return "llm:model:" + e.modelScopeKey(cfg) + ":last_success_at"
}
func (e *Engine) providerCooldownKey(cfg ModelConfig) string {
	return "llm:provider:" + e.providerScopeKey(cfg) + ":cooldown_until"
}

func sanitizeKey(v string) string {
	if v == "" {
		return "default"
	}
	b := strings.Builder{}
	b.Grow(len(v))
	for _, r := range v {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return strings.Trim(strings.Join(strings.Fields(strings.TrimSpace(b.String())), "_"), "_")
}

func normalizeProviderEndpoint(provider, baseURL string) string {
	base := normalizeBaseURL(baseURL)
	if base != "" {
		return base
	}
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "" {
		p = "unknown_provider"
	}
	return "provider://" + p
}

func normalizeBaseURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil {
		return strings.TrimRight(strings.ToLower(s), "/")
	}
	scheme := strings.ToLower(strings.TrimSpace(u.Scheme))
	host := strings.ToLower(strings.TrimSpace(u.Host))
	path := strings.TrimRight(strings.TrimSpace(u.Path), "/")
	normalized := ""
	if scheme != "" && host != "" {
		normalized = scheme + "://" + host
	} else {
		normalized = strings.TrimRight(strings.ToLower(s), "/")
	}
	if path != "" {
		normalized += path
	}
	return normalized
}

func joinList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.Join(items, ",")
}

func buildMessageForLLM(sys, usr string, recent []models.ChatMessage) []llm.Message {
	out := make([]llm.Message, 0, 14)
	if strings.TrimSpace(sys) != "" {
		out = append(out, llm.Message{Role: "system", Content: sys})
	}
	for _, msg := range takeRecent(recent, 8) {
		role := "user"
		switch msg.SenderType {
		case "bot":
			role = "assistant"
		case "system":
			role = "system"
		default:
			role = "user"
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		out = append(out, llm.Message{Role: role, Content: content})
	}
	if strings.TrimSpace(usr) != "" {
		out = append(out, llm.Message{Role: "user", Content: usr})
	}
	return out
}

func formatModelFailure(model, reason string, status int) string {
	if status > 0 {
		return fmt.Sprintf("%s:%s(status=%d)", model, reason, status)
	}
	return fmt.Sprintf("%s:%s", model, reason)
}
