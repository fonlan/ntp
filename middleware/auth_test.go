package middleware

import (
	"testing"
	"time"
)

func TestParseSessionTTL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want time.Duration
	}{
		{name: "默认一年", raw: "", want: 365 * 24 * time.Hour},
		{name: "解析天数", raw: "365d", want: 365 * 24 * time.Hour},
		{name: "解析标准时长", raw: "24h", want: 24 * time.Hour},
		{name: "无效值使用默认值", raw: "invalid", want: 365 * 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseSessionTTL(tt.raw); got != tt.want {
				t.Fatalf("parseSessionTTL(%q) = %s, want %s", tt.raw, got, tt.want)
			}
		})
	}
}
