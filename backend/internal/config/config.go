package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                 string
	Host                   string
	Port                   string
	LogDir                 string
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	GracefulTimeout        time.Duration
	AllowedOrigin          string
	AIProvider             string
	SensitiveWords         []string
	RoomColdSeconds        int
	RoomBotCooldownSeconds int
	RiskMaxMsgPer10s       int

	LLMEnabled            bool
	LLMProvider           string
	LLMBaseURL            string
	LLMAPIKey             string
	LLMModel              string
	LLMTimeoutSeconds     int
	LLMMaxAttempts        int
	LLMTotalBudgetSec     int
	LLMPerModelTOsec      int
	LLMProbeTOsec         int
	LLMProbeBudgetSec     int
	LLMHealthyMinCnt      int
	LLMChatMainPool       []string
	LLMPremiumPool        []string
	LLMReasoningPool      []string
	LLMCodePool           []string
	LLMSpecializedPool    []string
	LLMReasoningEnabled   bool
	LLMSpecializedEnabled bool
	LLMModelsJSON         string
	LLMModels             []LLMModelConfig
	LLMDebugRawResp       bool
	BotDebugForceLLM      bool
	BotDebugMinReply      bool
	BotMinPresence        bool

	PostgresDSN string
	RedisAddr   string
	RedisPass   string
	RedisDB     int
}

type LLMModelConfig struct {
	Name           string `json:"name"`
	Provider       string `json:"provider"`
	BaseURL        string `json:"base_url"`
	APIKeyEnv      string `json:"api_key_env"`
	Enabled        bool   `json:"enabled"`
	Priority       int    `json:"priority"`
	Pool           string `json:"pool"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

func Load() (*Config, error) {
	_ = godotenv.Load(".env", "../.env", "../../.env")

	cfg := &Config{
		AppEnv:                 getenv("APP_ENV", "development"),
		Host:                   getenv("BACKEND_HOST", "0.0.0.0"),
		Port:                   getenv("BACKEND_PORT", "8080"),
		LogDir:                 getenv("LOG_DIR", "../runtime/logs/backend"),
		ReadTimeout:            mustDuration(getenv("BACKEND_READ_TIMEOUT", "10s")),
		WriteTimeout:           mustDuration(getenv("BACKEND_WRITE_TIMEOUT", "10s")),
		GracefulTimeout:        mustDuration(getenv("BACKEND_GRACEFUL_TIMEOUT", "15s")),
		AllowedOrigin:          getenv("BACKEND_ALLOWED_ORIGIN", "http://localhost:5173"),
		AIProvider:             getenv("BOT_AI_PROVIDER", "none"),
		SensitiveWords:         splitCSV(getenv("SENSITIVE_WORDS", "")),
		RoomColdSeconds:        mustInt(getenv("ROOM_COLD_SECONDS", "18")),
		RoomBotCooldownSeconds: mustInt(getenv("ROOM_BOT_COOLDOWN_SECONDS", "6")),
		RiskMaxMsgPer10s:       mustInt(getenv("RISK_MAX_MSG_PER_10S", "8")),
		LLMEnabled:             mustBool(getenv("LLM_ENABLED", "true")),
		LLMProvider:            getenv("LLM_PROVIDER", "openai_compatible"),
		LLMBaseURL:             getenv("LLM_BASE_URL", "https://docs.newapi.pro/v1"),
		LLMAPIKey:              getenv("LLM_API_KEY", ""),
		LLMModel:               getenv("LLM_MODEL", "deepseek-ai/DeepSeek-R1-0528"),
		LLMTimeoutSeconds:      mustInt(getenv("LLM_TIMEOUT_SECONDS", "20")),
		LLMMaxAttempts:         mustInt(getenv("LLM_MAX_ATTEMPTS", "3")),
		LLMTotalBudgetSec:      mustInt(getenv("LLM_TOTAL_BUDGET_SECONDS", "30")),
		LLMPerModelTOsec:       mustInt(getenv("LLM_PER_MODEL_TIMEOUT_SECONDS", "10")),
		LLMProbeTOsec:          mustInt(getenv("LLM_POOL_PROBE_TIMEOUT_SECONDS", "4")),
		LLMProbeBudgetSec:      mustInt(getenv("LLM_POOL_PROBE_TOTAL_BUDGET_SECONDS", "60")),
		LLMHealthyMinCnt:       mustInt(getenv("LLM_HEALTHY_MIN_CANDIDATES", "2")),
		LLMChatMainPool: splitCSV(getenv(
			"LLM_CHAT_MAIN_POOL",
			"glm-4.7-flash,glm-4.5-air,XiaomiMiMo/MiMo-V2-Flash,z-ai/glm4.7,ZhipuAI/GLM-4.7-Flash,Qwen/Qwen3-8B,Qwen/Qwen3-4B,google/gemma-3-4b-it,google/gemma-2-2b-it,openai/gpt-oss-20b,mistralai/mixtral-8x7b-instruct,nvidia/nemotron-mini-4b-instruct,nvidia/nemotron-3-nano-30b-a3b,Qwen/Qwen3-30B-A3B,Qwen/Qwen3-32B,Qwen/Qwen3.5-27B,Qwen/Qwen3.5-35B-A3B,google/gemma-3-12b-it,google/gemma-3-27b-it,moonshotai/kimi-k2-instruct,moonshotai/Kimi-K2.5,moonshotai/kimi-k2.5,glm-5,z-ai/glm5,ZhipuAI/GLM-5,ZhipuAI/GLM-5.1,minimaxai/minimax-m2.7,meta/llama-3.3-70b-instruct,meta/llama2-70b,deepseek-ai/DeepSeek-V3.2,nv-mistralai/mistral-nemo-1,Qwen/Qwen3-Next-80B-A3B-Instruct,Qwen/Qwen3-235B-A22B-Instruct-2507,Qwen/Qwen3-235B-A22B-Instruct,Qwen/Qwen3.5-122B-A10B,qwen/qwen3.5-122b-a10b,gpt-oss-120b,nvidia/nemotron-3-super-120b-a12b,nvidia/nemotron-4-340b-instruct,qwen/qwen3.5-397b-a17b,Qwen/Qwen3.5-397B-A17B",
		)),
		LLMPremiumPool: splitCSV(getenv(
			"LLM_PREMIUM_FALLBACK_POOL",
			"gpt-5.4,claude-opus-4.7",
		)),
		LLMReasoningPool: splitCSV(getenv(
			"LLM_REASONING_POOL",
			"deepseek-ai/DeepSeek-R1-Distill-Llama-70B,Qwen/QwQ-32B,Qwen/QwQ-32B-Preview,moonshotai/kimi-k2-thinking,Qwen/Qwen3-30B-A3B-Thinking,Qwen/Qwen3-235B-A22B-Think,nvidia/cosmos-reason2-8b",
		)),
		LLMCodePool: splitCSV(getenv(
			"LLM_CODE_POOL",
			"Qwen/Qwen3-Coder-480B-A35B-Instruct,Qwen/Qwen3-Coder-30B-A3B,meta/codellama-70b,mistralai/devstral-2-123b-instruct-2512,google/codegemma-7b,google/codegemma-1.1-7b",
		)),
		LLMSpecializedPool: splitCSV(getenv(
			"LLM_SPECIALIZED_DO_NOT_USE_FOR_CHAT",
			"writer/palmyra-creative-122b,writer/palmyra-fin-70b-32k,writer/palmyra-med-70b,writer/palmyra-med-70b-32k,XGenerationLab/XiYanSQL-*,nvidia/llama-3.2-nemoretriever-1b-vlm-embed-v1,nvidia/nemotron-content-safety-*,nvidia/nemotron-4-340b-reward-*,nvidia/nemotron-parse,nvidia/nv-embed-v1,nvidia/nv-embedcode-7b-v1,nvidia/nv-embedqa-e5-v5,nvidia/nv-embedqa-mistral-*,opencompass/CompassJud*,meta/llama-3.2-90b-vision-instruct,nvidia/neva-22b,nvidia/vila,nvidia/nvclip,MusePublic/Qwen-Image-Edit,google/deplot,nvidia/riva-translate-*,Shanghai_AI_Laboratory/Int*,sarvamai/sarvam-m",
		)),
		LLMReasoningEnabled:   mustBool(getenv("LLM_ENABLE_REASONING_POOL", "false")),
		LLMSpecializedEnabled: mustBool(getenv("LLM_ENABLE_SPECIALIZED_POOL", "false")),
		LLMModelsJSON:         strings.TrimSpace(getenv("LLM_MODELS_JSON", "")),
		LLMDebugRawResp:       mustBool(getenv("LLM_DEBUG_RAW_RESPONSE", "false")),
		BotDebugForceLLM:      mustBool(getenv("BOT_DEBUG_FORCE_LLM", "false")),
		BotDebugMinReply:      mustBool(getenv("BOT_DEBUG_MIN_REPLY", "false")),
		BotMinPresence:        mustBool(getenv("BOT_MIN_PRESENCE", "true")),
		RedisAddr:             getenv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPass:             getenv("REDIS_PASSWORD", ""),
		RedisDB:               mustInt(getenv("REDIS_DB", "0")),
	}

	pgHost := getenv("POSTGRES_HOST", "127.0.0.1")
	pgPort := getenv("POSTGRES_PORT", "5432")
	pgDB := getenv("POSTGRES_DB", "sunyou_bot")
	pgUser := getenv("POSTGRES_USER", "sunyou")
	pgPass := getenv("POSTGRES_PASSWORD", "sunyou123")
	pgSSL := getenv("POSTGRES_SSLMODE", "disable")
	cfg.PostgresDSN = fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s", pgHost, pgPort, pgDB, pgUser, pgPass, pgSSL)

	if cfg.LLMMaxAttempts <= 0 {
		cfg.LLMMaxAttempts = 3
	}
	if cfg.LLMTotalBudgetSec <= 0 {
		cfg.LLMTotalBudgetSec = 30
	}
	if cfg.LLMPerModelTOsec <= 0 {
		cfg.LLMPerModelTOsec = 10
	}
	if cfg.LLMProbeTOsec <= 0 {
		cfg.LLMProbeTOsec = 4
	}
	if cfg.LLMProbeBudgetSec <= 0 {
		cfg.LLMProbeBudgetSec = 60
	}
	if cfg.LLMHealthyMinCnt <= 0 {
		cfg.LLMHealthyMinCnt = 2
	}

	if cfg.LLMModelsJSON != "" {
		items := make([]LLMModelConfig, 0)
		if err := json.Unmarshal([]byte(cfg.LLMModelsJSON), &items); err != nil {
			return nil, fmt.Errorf("parse LLM_MODELS_JSON failed: %w", err)
		}
		cfg.LLMModels = items
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustInt(v string) int {
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return i
}

func mustBool(v string) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return false
	}
	return b
}

func mustDuration(v string) time.Duration {
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0
	}
	return d
}

func splitCSV(v string) []string {
	if strings.TrimSpace(v) == "" {
		return []string{}
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
