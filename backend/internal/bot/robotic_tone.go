package bot

import "strings"

var roboticToneHardPatterns = []string{
	"这句话毫无价值",
	"这属于低质发言",
	"在阴阳裁判的标准下",
	"在这个标准下",
	"根据你的发言",
	"根据你这句",
	"当前发言人",
	"该用户",
	"此人",
	"你在群里发了一句",
	"你在群里时发了一句",
	"你留下了一抹污染",
	"这句话在本房间",
	"按标准来看",
	"我深感气愤",
	"我感到不悦",
	"让我感到",
	"仿佛你在",
	"挑战群里的整套秩序",
	"犹如战场般",
	"响彻群里",
	"缺乏品味",
	"令人不满",
}

var roboticToneSoftPatterns = []string{
	"低质发言",
	"毫无价值",
	"判定",
	"裁定",
	"归类",
	"性质",
	"当前房间",
	"根据发言",
	"标准下",
	"发言质量",
	"说明如下",
	"综上",
	"秩序",
	"战场",
	"震荡",
	"气愤",
}

// IsRoboticToneText detects report-like/system-like output that sounds like
// a judge report instead of an in-group real-time comeback.
func IsRoboticToneText(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}

	for _, p := range roboticToneHardPatterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}

	hits := 0
	for _, p := range roboticToneSoftPatterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			hits++
		}
	}
	if hits >= 2 {
		return true
	}

	// Report style: too many connector words with almost no direct "你" addressing.
	reportWords := 0
	for _, k := range []string{"因此", "所以", "首先", "其次", "最后", "综上", "结论", "标准"} {
		if strings.Contains(lower, k) {
			reportWords++
		}
	}
	if reportWords >= 2 && strings.Count(lower, "你") == 0 {
		return true
	}

	return false
}

// IsRoboticRepetitionText detects machine-like repetition where multiple
// sentences paraphrase the same conclusion or reuse the same structure.
func IsRoboticRepetitionText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	parts := splitSentences(trimmed)
	if len(parts) < 2 {
		return false
	}

	normalized := make([]string, 0, len(parts))
	for _, p := range parts {
		n := normalizeSentenceForRepeat(p)
		if n != "" {
			normalized = append(normalized, n)
		}
	}
	if len(normalized) < 2 {
		return false
	}

	seen := map[string]int{}
	for _, n := range normalized {
		seen[n]++
		if seen[n] >= 2 {
			return true
		}
	}

	// near-duplicate: one sentence almost contains another.
	for i := 0; i < len(normalized); i++ {
		for j := i + 1; j < len(normalized); j++ {
			a, b := normalized[i], normalized[j]
			if len([]rune(a)) < 6 || len([]rune(b)) < 6 {
				continue
			}
			if strings.Contains(a, b) || strings.Contains(b, a) {
				return true
			}
		}
	}

	// repeated structure markers.
	repeatMarkers := []string{
		"你这句", "不是聊天", "刷存在", "真觉得", "有必要", "别再", "你这是", "你不是",
	}
	repeatHits := 0
	for _, m := range repeatMarkers {
		if strings.Count(trimmed, m) >= 2 {
			repeatHits++
		}
	}
	return repeatHits >= 2
}

func splitSentences(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		switch r {
		case '。', '！', '!', '？', '?', ';', '；', '\n':
			return true
		default:
			return false
		}
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		s := strings.TrimSpace(f)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func normalizeSentenceForRepeat(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		" ", "", "，", "", ",", "", "。", "", ".", "", "！", "", "!", "", "？", "", "?", "",
		"：", "", ":", "", "；", "", ";", "", "“", "", "”", "", "\"", "", "'", "", "（", "", ")", "", "）", "",
	)
	s = replacer.Replace(s)
	if len([]rune(s)) > 80 {
		return string([]rune(s)[:80])
	}
	return s
}
