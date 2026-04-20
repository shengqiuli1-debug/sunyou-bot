package scorer

import (
	"strings"
	"unicode"

	"sunyou-bot/backend/internal/models"
)

type Input struct {
	SpeakerID        string
	SpeakerIdentity  models.Identity
	Content          string
	FireLevel        models.FireLevel
	IsColdScene      bool
	RecentMsgCount   int
	ConsecutiveHitID string
	ReplyToMessage   *models.ChatMessage
	RecentMessages   []models.ChatMessage
	MinPresence      bool
}

type Result struct {
	ForceReply     bool
	ShouldReply    bool
	Score          int
	AbsurdityScore int
	RiskScore      int
	Threshold      int
	Tags           []string
	Atmosphere     string
	ReplyMode      string
	PresenceBoost  bool
}

func Evaluate(in Input) Result {
	res := Result{
		Threshold:  thresholdByFire(in.FireLevel),
		Tags:       make([]string, 0, 16),
		Atmosphere: "平稳",
		ReplyMode:  "judgement",
	}
	content := normalize(in.Content)

	greeting := isGreeting(content)
	smallTalk := isSmallTalk(content)
	lowNoise := isLowValueNoise(content)
	adSmell := isAdSmell(content)
	hardMouth := isHardMouth(content)
	pretendExpert := isPretendExpert(content)
	flag := isFlag(content)
	hotConflict := isHighTension(content)

	if in.ReplyToMessage != nil {
		res.AbsurdityScore += 14
		res.Tags = append(res.Tags, "quoted_message")
		if in.ReplyToMessage.IsBotMessage {
			res.ForceReply = true
			res.AbsurdityScore += 120
			res.Tags = append(res.Tags, "quote_bot")
			res.ReplyMode = "force_quote_reply"
		}
	}

	if in.SpeakerIdentity == models.IdentityTarget {
		res.AbsurdityScore += 30
		res.Tags = append(res.Tags, "target_user")
	}
	if in.SpeakerIdentity == models.IdentityImmune {
		res.AbsurdityScore -= 18
		res.RiskScore += 6
		res.Tags = append(res.Tags, "immune_downweight")
	}

	if in.IsColdScene {
		res.AbsurdityScore += 22
		res.Tags = append(res.Tags, "cold_start")
		res.Atmosphere = "冷场"
	}

	if greeting {
		res.AbsurdityScore += 40
		res.Tags = append(res.Tags, "greeting", "small_talk", "chat_pollution")
		res.Atmosphere = "低质社交"
	}
	if smallTalk {
		res.AbsurdityScore += 28
		res.Tags = append(res.Tags, "small_talk")
		if res.Atmosphere == "平稳" {
			res.Atmosphere = "低质社交"
		}
	}
	if lowNoise {
		res.AbsurdityScore += 32
		res.Tags = append(res.Tags, "low_value_noise", "chat_pollution")
		if res.Atmosphere == "平稳" {
			res.Atmosphere = "污染"
		}
	}
	if adSmell {
		res.AbsurdityScore += 36
		res.RiskScore += 10
		res.Tags = append(res.Tags, "ad_smell", "chat_pollution")
		res.Atmosphere = "广告味"
	}

	if hardMouth {
		res.AbsurdityScore += 21
		res.Tags = append(res.Tags, "hard_mouth")
		if res.Atmosphere == "平稳" {
			res.Atmosphere = "嘴硬"
		}
	}
	if pretendExpert {
		res.AbsurdityScore += 18
		res.Tags = append(res.Tags, "pretend_expert")
	}
	if flag {
		res.AbsurdityScore += 17
		res.Tags = append(res.Tags, "flag")
		if res.Atmosphere == "平稳" {
			res.Atmosphere = "立flag"
		}
	}

	if questionBomb(content) {
		res.AbsurdityScore += 14
		res.RiskScore += 16
		res.Tags = append(res.Tags, "question_bomb")
		if res.Atmosphere == "平稳" {
			res.Atmosphere = "问号轰炸"
		}
	}

	if hotConflict {
		res.RiskScore += 38
		res.Tags = append(res.Tags, "high_tension", "heated_argument", "risk_high")
		if res.Atmosphere == "平稳" {
			res.Atmosphere = "高情绪"
		}
	}

	if hasRecentDebate(in.RecentMessages) {
		res.RiskScore += 22
		res.AbsurdityScore += 8
		res.Tags = append(res.Tags, "argument")
		if res.Atmosphere == "平稳" {
			res.Atmosphere = "争论"
		}
	}

	if isVeryShort(content) {
		res.AbsurdityScore += 10
		res.Tags = append(res.Tags, "short_ping")
	}

	if in.MinPresence && in.RecentMsgCount <= 3 && (greeting || smallTalk || lowNoise) && res.RiskScore < 48 {
		res.PresenceBoost = true
		res.AbsurdityScore += 16
		res.Tags = append(res.Tags, "presence_boost")
	}

	if in.RecentMsgCount < 5 && !(greeting || smallTalk || lowNoise || adSmell) {
		res.AbsurdityScore -= 6
		res.Tags = append(res.Tags, "low_activity_downweight")
	}

	if in.ConsecutiveHitID == in.SpeakerID && in.SpeakerIdentity != models.IdentityTarget && !res.ForceReply {
		res.AbsurdityScore -= 8
		res.RiskScore += 6
		res.Tags = append(res.Tags, "anti_streak")
	}

	if res.AbsurdityScore < 0 {
		res.AbsurdityScore = 0
	}
	if res.RiskScore < 0 {
		res.RiskScore = 0
	}

	res.Score = finalScore(res.AbsurdityScore, res.RiskScore)
	res.ReplyMode = chooseReplyMode(res, in)

	if res.ForceReply {
		res.ShouldReply = true
	} else if res.RiskScore >= 86 && res.AbsurdityScore < 58 {
		res.ShouldReply = false
	} else if res.PresenceBoost && res.AbsurdityScore >= 44 {
		res.ShouldReply = true
	} else {
		res.ShouldReply = res.Score >= res.Threshold
	}

	return res
}

func thresholdByFire(level models.FireLevel) int {
	switch level {
	case models.FireHigh:
		return 16
	case models.FireLow:
		return 28
	default:
		return 22
	}
}

func finalScore(absurdity, risk int) int {
	// Core rule: prioritize absurdity, penalize risk.
	return absurdity - risk/2
}

func chooseReplyMode(res Result, in Input) string {
	if res.ForceReply {
		return "force_quote_reply"
	}
	if res.RiskScore >= 60 {
		return "cold_narration"
	}
	if res.AbsurdityScore >= 56 {
		return "light_absurd"
	}
	if in.FireLevel == models.FireHigh {
		return "judgement"
	}
	return "judgement"
}

func normalize(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func containsAny(content string, keys []string) bool {
	for _, key := range keys {
		if strings.Contains(content, strings.ToLower(key)) {
			return true
		}
	}
	return false
}

func isGreeting(content string) bool {
	if content == "" {
		return false
	}
	return containsAny(content, []string{"你好", "哈喽", "hello", "hi", "嗨", "在吗", "有人吗"})
}

func isSmallTalk(content string) bool {
	if content == "" {
		return false
	}
	keys := []string{"哈哈", "哈", "哦", "啊", "嗯", "行吧", "都行", "随便", "干嘛", "在不在", "有空吗", "?", "？"}
	return containsAny(content, keys)
}

func isLowValueNoise(content string) bool {
	if content == "" {
		return false
	}
	if containsAny(content, []string{"。。。", "...", "??", "？？", "emmm", "emm", "ok", "okk", "哈哈哈", "在吗"}) {
		return true
	}
	if isOnlyPunctuation(content) {
		return true
	}
	if isVeryShort(content) {
		return true
	}
	return false
}

func isAdSmell(content string) bool {
	if content == "" {
		return false
	}
	keys := []string{
		"加v", "加vx", "加微", "私聊", "代发", "推广", "合作", "返利", "代理", "下单",
		"接单", "进群", "拉群", "福利", "优惠", "拼单", "联系方式", "带货", "广告",
	}
	return containsAny(content, keys)
}

func isHardMouth(content string) bool {
	return containsAny(content, []string{"都行", "随便", "行吧", "懂了", "你说得对", "呵", "无所谓", "我没错", "我又没说错"})
}

func isPretendExpert(content string) bool {
	return containsAny(content, []string{"我早就知道", "这不就很简单", "肯定是这样", "一眼就看出来", "不用想都知道"})
}

func isFlag(content string) bool {
	return containsAny(content, []string{"立flag", "立个flag", "稳了", "不可能翻车", "必赢", "包赢", "明天一定开始"})
}

func isHighTension(content string) bool {
	keys := []string{
		"滚", "闭嘴", "有病", "神经", "烦死", "气死", "你有完没完", "别逼我", "吵", "闹",
		"凭什么", "不服", "你再说", "算你狠", "你等着", "受够了", "离谱", "恶心",
	}
	if containsAny(content, keys) {
		return true
	}
	exclaim := 0
	for _, r := range content {
		if r == '!' || r == '！' {
			exclaim++
		}
	}
	return exclaim >= 3
}

func questionBomb(content string) bool {
	count := 0
	for _, r := range content {
		if r == '?' || r == '？' {
			count++
		}
	}
	return count >= 2
}

func isOnlyPunctuation(content string) bool {
	if content == "" {
		return false
	}
	for _, r := range content {
		if unicode.IsSpace(r) {
			continue
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.In(r, unicode.Han) {
			return false
		}
	}
	return true
}

func isVeryShort(content string) bool {
	r := []rune(strings.TrimSpace(content))
	return len(r) > 0 && len(r) <= 3
}

func hasRecentDebate(messages []models.ChatMessage) bool {
	if len(messages) < 4 {
		return false
	}
	count := 0
	for i := len(messages) - 1; i >= 0 && i >= len(messages)-10; i-- {
		msg := normalize(messages[i].Content)
		if strings.Contains(msg, "?") || strings.Contains(msg, "？") || strings.Contains(msg, "凭什么") || strings.Contains(msg, "不服") || strings.Contains(msg, "你确定") || strings.Contains(msg, "离谱") {
			count++
		}
	}
	return count >= 2
}
