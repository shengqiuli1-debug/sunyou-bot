package bot

import (
	"strings"
	"unicode"
)

const (
	langZH      = "zh"
	langEN      = "en"
	langMixed   = "mixed"
	langUnknown = "unknown"
)

// DetectPrimaryLanguage returns zh/en/mixed/unknown for lightweight runtime gating.
func DetectPrimaryLanguage(text string) string {
	s := strings.TrimSpace(text)
	if s == "" {
		return langUnknown
	}
	var zhCount int
	var enCount int
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			zhCount++
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			enCount++
		}
	}

	if zhCount == 0 && enCount == 0 {
		return langUnknown
	}
	if zhCount > 0 && enCount == 0 {
		return langZH
	}
	if enCount > 0 && zhCount == 0 {
		return langEN
	}

	// Both exist: choose dominant one only when ratio is clearly higher.
	if zhCount*10 >= enCount*13 {
		return langZH
	}
	if enCount*10 >= zhCount*13 {
		return langEN
	}
	return langMixed
}

func isLanguageMismatch(inputLang, outputLang string) bool {
	in := strings.TrimSpace(strings.ToLower(inputLang))
	out := strings.TrimSpace(strings.ToLower(outputLang))
	if in == "" {
		in = langUnknown
	}
	if out == "" {
		out = langUnknown
	}

	// Mixed/unknown input should not hard block.
	if in == langMixed || in == langUnknown {
		return false
	}
	// If output cannot be detected, don't reject purely by language.
	if out == langUnknown || out == langMixed {
		return false
	}
	return in != out
}
