package bot

import "testing"

func TestIsPlannerLeakText(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"好的，我需要处理用户发来的消息。首先分析触发原因。", true},
		{"Let's tackle this step by step. I need to respond carefully.", true},
		{"We need to produce a first-person disparaging reply, then amplify, final sting.", true},
		{"我需要生成一条回复：先定性，再放大，然后反问，最后尾刀。", true},
		{"你这句发言像空气加戏，收一收。", false},
		{"You typed two question marks like that was supposed to mean something.", false},
		{"本轮认定：低质开场，下一句给内容。", false},
	}
	for _, tc := range cases {
		if got := IsPlannerLeakText(tc.in); got != tc.want {
			t.Fatalf("IsPlannerLeakText(%q)=%v, want %v", tc.in, got, tc.want)
		}
	}
}
