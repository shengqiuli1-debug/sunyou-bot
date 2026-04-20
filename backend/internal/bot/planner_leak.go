package bot

import "strings"

var plannerLeakPatterns = []string{
	"好的，我需要",
	"我需要生成",
	"我需要处理用户",
	"用户要求我",
	"先定性",
	"再放大",
	"然后反问",
	"最后尾刀",
	"根据角色设定",
	"当前发言人",
	"触发原因",
	"回复模式",
	"首先",
	"然后",
	"最后",
	"we need to",
	"we should",
	"let's tackle this",
	"the user wants",
	"the user is",
	"we need to produce",
	"start with a definition",
	"then amplify",
	"final sting",
	"first-person disparaging reply",
	"i need to",
	"i should respond",
	"i should",
	"user asked",
	"as an ai",
}

// IsPlannerLeakText checks whether text looks like internal planning/scratchpad
// instead of user-visible chat output.
func IsPlannerLeakText(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	hit := 0
	for _, p := range plannerLeakPatterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			hit++
		}
	}
	if hit >= 2 {
		return true
	}

	// Strong single-signature patterns.
	if strings.Contains(lower, "let's tackle this") ||
		strings.Contains(lower, "we need to") ||
		strings.Contains(lower, "the user wants") ||
		strings.Contains(lower, "first-person disparaging reply") ||
		strings.Contains(lower, "i need to") ||
		strings.Contains(lower, "我需要生成") ||
		strings.Contains(lower, "用户要求我") ||
		strings.Contains(lower, "当前发言人") ||
		strings.Contains(lower, "触发原因") {
		return true
	}
	return false
}
