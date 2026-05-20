package util

import "testing"

func TestTruncateCaption(t *testing.T) {
	cases := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"empty", "", 10, ""},
		{"under limit", "hello", 10, "hello"},
		{"exact limit", "hello", 5, "hello"},
		{"ascii over", "hello world", 5, "hello"},
		{"chinese under", "你好世界", 4, "你好世界"},
		{"chinese exact", "你好", 2, "你好"},
		{"chinese over", "你好世界", 3, "你好世"},
		// 单个 emoji 占 2 个 UTF-16 units
		{"emoji exact", "📋", 2, "📋"},
		{"emoji not enough room", "📋", 1, ""},
		// 模板前缀(占 5 units) + emoji(占 2 units) = 7
		{"mixed exact", "abcde📋", 7, "abcde📋"},
		{"mixed cuts emoji", "abcde📋", 6, "abcde"},
		{"zero max", "anything", 0, ""},
		{"negative max", "anything", -1, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := TruncateCaption(tc.in, tc.max)
			if got != tc.want {
				t.Errorf("TruncateCaption(%q, %d) = %q; want %q", tc.in, tc.max, got, tc.want)
			}
		})
	}
}

func TestCaptionUTF16Len(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"hello", 5},
		{"你好", 2},
		{"📋", 2},
		{"a📋b", 4},
	}
	for _, tc := range cases {
		got := CaptionUTF16Len(tc.in)
		if got != tc.want {
			t.Errorf("CaptionUTF16Len(%q) = %d; want %d", tc.in, got, tc.want)
		}
	}
}
