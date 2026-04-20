package bot

import "testing"

func TestDetectPrimaryLanguage(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"你好，在吗", langZH},
		{"hello are you there", langEN},
		{"你好 hello", langEN},
		{"你好 hello hello hello", langEN},
		{"你好你好 hi", langZH},
		{"？？？", langUnknown},
		{"", langUnknown},
	}
	for _, tc := range cases {
		if got := DetectPrimaryLanguage(tc.in); got != tc.want {
			t.Fatalf("DetectPrimaryLanguage(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsLanguageMismatch(t *testing.T) {
	cases := []struct {
		inLang  string
		outLang string
		want    bool
	}{
		{langZH, langEN, true},
		{langEN, langZH, true},
		{langZH, langZH, false},
		{langEN, langEN, false},
		{langMixed, langEN, false},
		{langUnknown, langZH, false},
		{langZH, langUnknown, false},
		{langEN, langMixed, false},
	}
	for _, tc := range cases {
		if got := isLanguageMismatch(tc.inLang, tc.outLang); got != tc.want {
			t.Fatalf("isLanguageMismatch(%q,%q)=%v, want %v", tc.inLang, tc.outLang, got, tc.want)
		}
	}
}
