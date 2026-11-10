package calculator

import "testing"

func TestGetRatesUsesCurrentAnthropicPricing(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		wantFamily string
		wantInput  float64
		wantOutput float64
		wantRead   float64
		wantWrite  float64
	}{
		{
			name:       "fable 5",
			model:      "claude-fable-5",
			wantFamily: "claude-fable-5",
			wantInput:  10,
			wantOutput: 50,
			wantRead:   1,
			wantWrite:  12.5,
		},
		{
			name:       "opus 4.8",
			model:      "claude-opus-4-8",
			wantFamily: "claude-opus-4-8",
			wantInput:  5,
			wantOutput: 25,
			wantRead:   0.5,
			wantWrite:  6.25,
		},
		{
			name:       "opus 4.1 stays legacy priced",
			model:      "claude-opus-4-1-20250805",
			wantFamily: "claude-opus-4-1",
			wantInput:  15,
			wantOutput: 75,
			wantRead:   1.5,
			wantWrite:  18.75,
		},
		{
			name:       "opus 4 stays legacy priced",
			model:      "claude-opus-4-20250514",
			wantFamily: "claude-opus-4",
			wantInput:  15,
			wantOutput: 75,
			wantRead:   1.5,
			wantWrite:  18.75,
		},
		{
			name:       "sonnet 4.6",
			model:      "claude-sonnet-4-6",
			wantFamily: "claude-sonnet-4-6",
			wantInput:  3,
			wantOutput: 15,
			wantRead:   0.3,
			wantWrite:  3.75,
		},
		{
			name:       "haiku 4.5",
			model:      "claude-haiku-4-5-20251001",
			wantFamily: "claude-haiku-4-5",
			wantInput:  1,
			wantOutput: 5,
			wantRead:   0.1,
			wantWrite:  1.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRates(tt.model)
			if got.Family != tt.wantFamily {
				t.Fatalf("family = %q, want %q", got.Family, tt.wantFamily)
			}
			if got.InputPerMToken != tt.wantInput ||
				got.OutputPerMToken != tt.wantOutput ||
				got.CacheReadPerMToken != tt.wantRead ||
				got.CacheWritePerMToken != tt.wantWrite {
				t.Fatalf("rates = input %.2f output %.2f read %.2f write %.2f, want input %.2f output %.2f read %.2f write %.2f",
					got.InputPerMToken, got.OutputPerMToken, got.CacheReadPerMToken, got.CacheWritePerMToken,
					tt.wantInput, tt.wantOutput, tt.wantRead, tt.wantWrite)
			}
		})
	}
}
