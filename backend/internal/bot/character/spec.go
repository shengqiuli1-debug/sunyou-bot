package character

import (
	"strings"

	"sunyou-bot/backend/internal/bot/scorer"
	"sunyou-bot/backend/internal/models"
)

const (
	poolChatMain = "chat_main_pool"
	poolPremium  = "premium_fallback_pool"
	poolCode     = "code_pool"
)

type TriggerPolicy struct {
	ScoreBias        int
	ThresholdOffset  int
	SmallTalkBonus   int
	HighRiskPenalty  int
	ForceReplyBonus  int
	DefaultReplyMode string
}

type PromptPolicy struct {
	Persona         string
	RoleInstruction string
	ToneHint        string
}

type ModelPolicy struct {
	PreferredPools      []string
	NormalAttempts      int
	CodeAttempts        int
	ForceMainAttempts   int
	ForcePremiumAttempt int
	UsePremiumOnForce   bool
}

type OutputPolicy struct {
	BlockPlannerLeak     bool
	BlockRoboticTone     bool
	EnforceLanguageMatch bool
	MaxReplyRunes        int
}

type UIMeta struct {
	DisplayName string
	Bio         string
	Identity    string
	Duty        string
	AvatarText  string
}

type TemplatePack struct {
	Classify []string
	Amplify  []string
	Question []string
	Final    []string
}

type FallbackPolicy struct {
	PacksByScenario map[string]TemplatePack
	DefaultPack     TemplatePack
}

type Spec struct {
	ID             models.BotRole
	TriggerPolicy  TriggerPolicy
	PromptPolicy   PromptPolicy
	ModelPolicy    ModelPolicy
	OutputPolicy   OutputPolicy
	FallbackPolicy FallbackPolicy
	UIMeta         UIMeta
}

type TriggerContext struct {
	Content         string
	ForceReply      bool
	SpeakerIdentity models.Identity
}

var specs = map[models.BotRole]Spec{
	models.RoleNpc: {
		ID: models.RoleNpc,
		TriggerPolicy: TriggerPolicy{
			ScoreBias:        6,
			ThresholdOffset:  -2,
			SmallTalkBonus:   10,
			HighRiskPenalty:  8,
			ForceReplyBonus:  15,
			DefaultReplyMode: "light_absurd",
		},
		PromptPolicy: PromptPolicy{
			Persona:         "损友 NPC：群里那个嘴碎又烦人的老油条，看到破绽就顺嘴接。第一人称，像真人群友，不像系统。",
			RoleInstruction: "别写评语，别解释质量，直接接对方动作。优先接问候废话、问号轰炸、装傻、嘴硬、找借口。",
			ToneHint:        "短句口语、当场顶回去、像在群里连招。",
		},
		ModelPolicy: ModelPolicy{
			PreferredPools:      []string{poolChatMain},
			NormalAttempts:      3,
			CodeAttempts:        2,
			ForceMainAttempts:   2,
			ForcePremiumAttempt: 1,
			UsePremiumOnForce:   true,
		},
		OutputPolicy: OutputPolicy{
			BlockPlannerLeak:     true,
			BlockRoboticTone:     true,
			EnforceLanguageMatch: true,
			MaxReplyRunes:        96,
		},
		FallbackPolicy: FallbackPolicy{
			PacksByScenario: npcScenarioPacks(),
			DefaultPack:     npcDefaultPack(),
		},
		UIMeta: UIMeta{
			DisplayName: "损友 NPC",
			Bio:         "低质发言巡查员",
			Identity:    "房间角色 / 特殊成员",
			Duty:        "负责高频接话、抓破绽、现场补刀。",
			AvatarText:  "损",
		},
	},
	models.RoleJudge: {
		ID: models.RoleJudge,
		TriggerPolicy: TriggerPolicy{
			ScoreBias:        2,
			ThresholdOffset:  1,
			SmallTalkBonus:   6,
			HighRiskPenalty:  12,
			ForceReplyBonus:  12,
			DefaultReplyMode: "judgement",
		},
		PromptPolicy: PromptPolicy{
			Persona:         "阴阳裁判：冷脸、高位、阴阳怪气，像群里说话很稳但很压人的那个人，不是公告员。",
			RoleInstruction: "少解释原则，多当场回怼。保持克制但压迫感强，别写“根据标准”“该用户”。",
			ToneHint:        "短句、冷嘲、我懒得多说你的感觉。",
		},
		ModelPolicy: ModelPolicy{
			PreferredPools:      []string{poolChatMain},
			NormalAttempts:      2,
			CodeAttempts:        2,
			ForceMainAttempts:   2,
			ForcePremiumAttempt: 1,
			UsePremiumOnForce:   true,
		},
		OutputPolicy: OutputPolicy{
			BlockPlannerLeak:     true,
			BlockRoboticTone:     true,
			EnforceLanguageMatch: true,
			MaxReplyRunes:        88,
		},
		FallbackPolicy: FallbackPolicy{
			PacksByScenario: judgeScenarioPacks(),
			DefaultPack:     judgeDefaultPack(),
		},
		UIMeta: UIMeta{
			DisplayName: "阴阳裁判",
			Bio:         "房间秩序鉴定员",
			Identity:    "房间角色 / 特殊成员",
			Duty:        "负责定性、收尾和秩序压舱。",
			AvatarText:  "判",
		},
	},
	models.RoleNarrator: {
		ID: models.RoleNarrator,
		TriggerPolicy: TriggerPolicy{
			ScoreBias:        -1,
			ThresholdOffset:  2,
			SmallTalkBonus:   3,
			HighRiskPenalty:  4,
			ForceReplyBonus:  10,
			DefaultReplyMode: "cold_narration",
		},
		PromptPolicy: PromptPolicy{
			Persona:         "冷面旁白：像群里冷冷看戏的人，观察式补刀，但还是人话，不是播报员。",
			RoleInstruction: "别写总结报告，别当第三方讲解。只做一两句冷观察，顺手补刀。",
			ToneHint:        "冷、淡、短、像围观群众顺嘴一句。",
		},
		ModelPolicy: ModelPolicy{
			PreferredPools:      []string{poolChatMain},
			NormalAttempts:      2,
			CodeAttempts:        2,
			ForceMainAttempts:   2,
			ForcePremiumAttempt: 1,
			UsePremiumOnForce:   true,
		},
		OutputPolicy: OutputPolicy{
			BlockPlannerLeak:     true,
			BlockRoboticTone:     true,
			EnforceLanguageMatch: true,
			MaxReplyRunes:        90,
		},
		FallbackPolicy: FallbackPolicy{
			PacksByScenario: narratorScenarioPacks(),
			DefaultPack:     narratorDefaultPack(),
		},
		UIMeta: UIMeta{
			DisplayName: "冷面旁白",
			Bio:         "现场事故记录员",
			Identity:    "房间角色 / 特殊成员",
			Duty:        "负责冷观察补刀和局势总结。",
			AvatarText:  "述",
		},
	},
}

func Get(role models.BotRole) Spec {
	r := NormalizeRole(role)
	if spec, ok := specs[r]; ok {
		return spec
	}
	return specs[models.RoleNpc]
}

func NormalizeRole(role models.BotRole) models.BotRole {
	switch role {
	case models.RoleJudge, models.RoleNpc, models.RoleNarrator:
		return role
	default:
		return models.RoleNpc
	}
}

func DisplayName(role models.BotRole) string {
	return Get(role).UIMeta.DisplayName
}

func InferRoleFromNickname(nickname string) models.BotRole {
	lower := strings.ToLower(strings.TrimSpace(nickname))
	if strings.Contains(lower, "阴阳裁判") || strings.Contains(lower, "judge") {
		return models.RoleJudge
	}
	if strings.Contains(lower, "冷面旁白") || strings.Contains(lower, "narrator") {
		return models.RoleNarrator
	}
	return models.RoleNpc
}

func (s Spec) FallbackPack(scenario string) TemplatePack {
	k := strings.TrimSpace(strings.ToLower(scenario))
	if pack, ok := s.FallbackPolicy.PacksByScenario[k]; ok {
		return pack
	}
	return s.FallbackPolicy.DefaultPack
}

func ApplyTriggerPolicy(spec Spec, base scorer.Result, ctx TriggerContext) scorer.Result {
	res := base
	p := spec.TriggerPolicy
	res.Score += p.ScoreBias
	res.Threshold += p.ThresholdOffset
	if res.Threshold < 1 {
		res.Threshold = 1
	}
	if hasTag(res.Tags, "greeting") || hasTag(res.Tags, "small_talk") || hasTag(res.Tags, "low_value_noise") || hasTag(res.Tags, "chat_pollution") {
		res.Score += p.SmallTalkBonus
	}
	if hasTag(res.Tags, "risk_high") || hasTag(res.Tags, "heated_argument") || hasTag(res.Tags, "high_tension") {
		res.Score -= p.HighRiskPenalty
	}
	if res.ForceReply || ctx.ForceReply {
		res.Score += p.ForceReplyBonus
		res.ForceReply = true
	}
	if res.Score < 0 {
		res.Score = 0
	}

	// Keep global anti-overheat baseline, then re-evaluate role threshold.
	if res.ForceReply {
		res.ShouldReply = true
	} else if res.RiskScore >= 86 && res.AbsurdityScore < 58 {
		res.ShouldReply = false
	} else if res.PresenceBoost && res.AbsurdityScore >= 44 {
		res.ShouldReply = true
	} else {
		res.ShouldReply = res.Score >= res.Threshold
	}

	if spec.ID == models.RoleNarrator && (hasTag(res.Tags, "risk_high") || hasTag(res.Tags, "argument")) {
		res.ReplyMode = "cold_narration"
	}
	if spec.ID == models.RoleNpc && (hasTag(res.Tags, "small_talk") || hasTag(res.Tags, "greeting") || hasTag(res.Tags, "chat_pollution")) {
		res.ReplyMode = "light_absurd"
	}
	if strings.TrimSpace(res.ReplyMode) == "" {
		res.ReplyMode = p.DefaultReplyMode
	}
	return res
}

func hasTag(tags []string, target string) bool {
	for _, t := range tags {
		if strings.EqualFold(strings.TrimSpace(t), target) {
			return true
		}
	}
	return false
}
