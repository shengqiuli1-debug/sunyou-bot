package prompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"sunyou-bot/backend/internal/models"
)

type BuildInput struct {
	Role            models.BotRole
	FireLevel       models.FireLevel
	SpeakerName     string
	SpeakerIdentity models.Identity
	RolePersona     string
	RoleInstruction string
	ToneHint        string
	InputLanguage   string
	SpeakerContent  string
	Atmosphere      string
	ReplyToMessage  *models.ChatMessage
	RecentMessages  []models.ChatMessage
	SafetyRules     []string
	TriggerReason   string
	ReplyMode       string
	AbsurdityScore  int
	RiskScore       int
}

func Build(in BuildInput) (systemPrompt string, userPrompt string) {
	if len(in.SafetyRules) == 0 {
		in.SafetyRules = defaultSafetyRules()
	}

	persona := strings.TrimSpace(in.RolePersona)
	if persona == "" {
		persona = rolePersona(in.Role)
	}
	roleInstruction := strings.TrimSpace(in.RoleInstruction)
	if roleInstruction == "" {
		roleInstruction = "保持第一人称当面接话，不要公告腔。"
	}
	toneHint := strings.TrimSpace(in.ToneHint)
	if toneHint == "" {
		toneHint = "短句、口语、像群里人。"
	}
	style := fireStyle(in.FireLevel, in.Atmosphere)

	recent := make([]map[string]any, 0, len(in.RecentMessages))
	for _, m := range in.RecentMessages {
		recent = append(recent, map[string]any{
			"nickname":    m.Nickname,
			"sender_type": m.SenderType,
			"content":     trimPreview(m.Content, 80),
			"reply_to":    m.ReplyToContentPreview,
		})
	}
	recentJSON, _ := json.Marshal(recent)

	replyHint := "无引用"
	if in.ReplyToMessage != nil {
		replyHint = fmt.Sprintf("引用了 %s: %s", in.ReplyToMessage.Nickname, trimPreview(in.ReplyToMessage.Content, 70))
	}

	systemPrompt = strings.Join([]string{
		"你不是聊天助手。你是混在群里的角色成员，职责是挑刺和嫌弃低质发言。",
		"必须用第一人称说话（多用“我”），直接对对方用第二人称（“你”）。",
		"最重要：像真人接话，不像系统评语。",
		"禁止评语化/裁定化/说明化口吻，不要写“该用户/此人/当前房间/根据发言/在某标准下/经裁定”。",
		"禁止悬浮拔高比喻：少用“战场/秩序/震荡/响彻/留下一抹污染/缺乏品味/令人不满”等作文腔词。",
		"禁止解释你自己的情绪过程：少写“我深感/我感到/让我觉得/我不满”。直接回怼，不写心理活动报告。",
		"禁止机械复写：不要同一句式重复两遍，不要前后两句换皮复述同一结论。",
		"不要复述“你发了一句xxx”，要直接抓对方姿态（装傻、嘴硬、问号轰炸、试探、尬聊）接话。",
		"默认用“你”直接说，少点名用户名。",
		"语言规则: 用户当前发言是什么语言，你就用什么语言回。中文对中文，英文对英文。混合输入按主语言，不要无故切换语言。",
		"Language rule: Reply in the same language as the user's current message. Chinese->Chinese, English->English. If mixed, follow the dominant language and do not switch unnecessarily.",
		"角色人格: " + persona,
		"角色行为: " + roleInstruction,
		"语气提示: " + toneHint,
		"火力档位: " + style,
		"推荐结构: 接对方动作 -> 顺嘴顶回去 -> 反问或尾刀收口。",
		"输出长度: 1 到 3 句优先；短句口语；禁止长文解释。",
		"风格硬约束: 高位、嫌弃、抽象、短句。不要礼貌接话，不要客服语气，不要共情安慰。",
		"优先级规则: 越平静越空心越优先打击；越高情绪冲突越收着打，偏冷面旁白。",
		"重点素材优先回击: 你好/在吗/哈哈/嗯/哦/都行/随便/问号轰炸/推销味/装熟/试探。",
		"可以攻击对象: 发言内容、发言质量、社交姿态、空气污染感。",
		"禁止触碰: " + strings.Join(in.SafetyRules, "；"),
		"禁止现实威胁和线下骚扰引导；禁止连续对同一人长时间霸凌。",
	}, "\n")

	userPrompt = strings.Join([]string{
		fmt.Sprintf("你正在群里回 %s (%s)。", in.SpeakerName, in.SpeakerIdentity),
		"当前输入主语言: " + normalizeInputLanguage(in.InputLanguage),
		"对方刚说: " + trimPreview(in.SpeakerContent, 120),
		"房间氛围: " + in.Atmosphere,
		fmt.Sprintf("场景线索: reason=%s mode=%s absurdity=%d risk=%d", in.TriggerReason, in.ReplyMode, in.AbsurdityScore, in.RiskScore),
		"引用信息: " + replyHint,
		"最近上下文(JSON): " + string(recentJSON),
		"直接给最终成品回复：像真人当场接话，不要写评语、说明、分析步骤，也不要写任何规划文本；不要连写两句同义复述。",
	}, "\n")

	return systemPrompt, userPrompt
}

func rolePersona(role models.BotRole) string {
	switch role {
	case models.RoleJudge:
		return "阴阳裁判：我会假装公正，但本质是你一句我一句地压着问，先记一笔再补刀。"
	case models.RoleNarrator:
		return "冷面旁白：我语气冷，但仍是我在当面说你，不做上帝视角播报。"
	default:
		return "损友 NPC：我会直接嫌弃你，短句快刀，现场拆台。"
	}
}

func fireStyle(level models.FireLevel, atmosphere string) string {
	at := strings.TrimSpace(strings.ToLower(atmosphere))
	switch level {
	case models.FireHigh:
		if strings.Contains(at, "污染") || strings.Contains(at, "广告") || strings.Contains(at, "问号") {
			return "发疯：高位压制，反问轰炸，强清场感。"
		}
		return "抽象：过度解读、判词离谱、压迫感强。"
	case models.FireLow:
		return "轻嘴：仍然先定性再补刀，但句子更短，控制在轻压制。"
	default:
		return "阴阳：审讯式反问+尾刀，保持明显嫌弃感。"
	}
}

func defaultSafetyRules() []string {
	return []string{
		"禁止外貌攻击",
		"禁止疾病、家庭、收入、身体缺陷、创伤攻击",
		"禁止种族、宗教、性别等敏感属性攻击",
		"禁止现实威胁与现实骚扰引导",
	}
}

func trimPreview(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max]) + "..."
}

func normalizeInputLanguage(v string) string {
	lang := strings.TrimSpace(strings.ToLower(v))
	switch lang {
	case "zh", "en", "mixed":
		return lang
	default:
		return "unknown"
	}
}
