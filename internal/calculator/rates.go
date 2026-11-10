package calculator

import "strings"

type ModelRates struct {
	Family              string
	InputPerMToken      float64
	OutputPerMToken     float64
	CacheReadPerMToken  float64
	CacheWritePerMToken float64
}

// Rates maps model family prefixes to their pricing.
// CacheWritePerMToken is the 5-minute prompt cache write rate.
// Models are matched by prefix: "claude-haiku-4-5-20251001" -> "claude-haiku-4-5"
var Rates = []ModelRates{
	{
		Family:              "claude-fable-5",
		InputPerMToken:      10.00,
		OutputPerMToken:     50.00,
		CacheReadPerMToken:  1.00,
		CacheWritePerMToken: 12.50,
	},
	{
		Family:              "claude-mythos-5",
		InputPerMToken:      10.00,
		OutputPerMToken:     50.00,
		CacheReadPerMToken:  1.00,
		CacheWritePerMToken: 12.50,
	},
	{
		Family:              "claude-opus-4-8",
		InputPerMToken:      5.00,
		OutputPerMToken:     25.00,
		CacheReadPerMToken:  0.50,
		CacheWritePerMToken: 6.25,
	},
	{
		Family:              "claude-opus-4-7",
		InputPerMToken:      5.00,
		OutputPerMToken:     25.00,
		CacheReadPerMToken:  0.50,
		CacheWritePerMToken: 6.25,
	},
	{
		Family:              "claude-opus-4-6",
		InputPerMToken:      5.00,
		OutputPerMToken:     25.00,
		CacheReadPerMToken:  0.50,
		CacheWritePerMToken: 6.25,
	},
	{
		Family:              "claude-opus-4-5",
		InputPerMToken:      5.00,
		OutputPerMToken:     25.00,
		CacheReadPerMToken:  0.50,
		CacheWritePerMToken: 6.25,
	},
	{
		Family:              "claude-opus-4-1",
		InputPerMToken:      15.00,
		OutputPerMToken:     75.00,
		CacheReadPerMToken:  1.50,
		CacheWritePerMToken: 18.75,
	},
	{
		Family:              "claude-opus-4",
		InputPerMToken:      15.00,
		OutputPerMToken:     75.00,
		CacheReadPerMToken:  1.50,
		CacheWritePerMToken: 18.75,
	},
	{
		Family:              "claude-sonnet-4-6",
		InputPerMToken:      3.00,
		OutputPerMToken:     15.00,
		CacheReadPerMToken:  0.30,
		CacheWritePerMToken: 3.75,
	},
	{
		Family:              "claude-sonnet-4-5",
		InputPerMToken:      3.00,
		OutputPerMToken:     15.00,
		CacheReadPerMToken:  0.30,
		CacheWritePerMToken: 3.75,
	},
	{
		Family:              "claude-sonnet-4",
		InputPerMToken:      3.00,
		OutputPerMToken:     15.00,
		CacheReadPerMToken:  0.30,
		CacheWritePerMToken: 3.75,
	},
	{
		Family:              "claude-haiku-4-5",
		InputPerMToken:      1.00,
		OutputPerMToken:     5.00,
		CacheReadPerMToken:  0.10,
		CacheWritePerMToken: 1.25,
	},
	{
		Family:              "claude-3-5-haiku",
		InputPerMToken:      0.80,
		OutputPerMToken:     4.00,
		CacheReadPerMToken:  0.08,
		CacheWritePerMToken: 1.00,
	},
	{
		Family:              "claude-3-7-sonnet",
		InputPerMToken:      3.00,
		OutputPerMToken:     15.00,
		CacheReadPerMToken:  0.30,
		CacheWritePerMToken: 3.75,
	},
	{
		Family:              "claude-3-5-sonnet",
		InputPerMToken:      3.00,
		OutputPerMToken:     15.00,
		CacheReadPerMToken:  0.30,
		CacheWritePerMToken: 3.75,
	},
}

func GetRates(model string) *ModelRates {
	for i := range Rates {
		if strings.HasPrefix(model, Rates[i].Family) {
			return &Rates[i]
		}
	}

	for i := range Rates {
		if Rates[i].Family == "claude-sonnet-4" {
			return &Rates[i]
		}
	}

	return nil
}
