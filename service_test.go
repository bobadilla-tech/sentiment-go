package sentiment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyze_Negation(t *testing.T) {
	t.Parallel()
	svc, err := NewService()
	assert.NoError(t, err)

	tests := []struct {
		name string
		text string
		want string
	}{
		{"negated positive word becomes negative", "not good", "negative"},
		{"negated strongly positive word becomes negative", "not amazing", "negative"},
		{"never + positive flips", "I never liked it", "negative"},
		{"no + positive flips", "no fun at all", "negative"},
		{"contraction negation flips", "dont like it", "negative"},
		{"negation only affects next 3 tokens", "not really really really good", "positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := svc.Analyze(tt.text)

			assert.Equal(t, tt.want, got.Sentiment,
				"Analyze(%q) returned breakdown %+v", tt.text, got.Breakdown)
		})
	}
}

func TestAnalyze_EmptyInput(t *testing.T) {
	t.Parallel()
	svc, err := NewService()
	assert.NoError(t, err)

	got := svc.Analyze("")

	assert.Equal(t, "neutral", got.Sentiment)
	assert.Equal(t, 1.0, got.Score)
	assert.Equal(t, Breakdown{
		Positive: 0,
		Negative: 0,
		Neutral:  1,
	}, got.Breakdown)
}

func TestAnalyze_UnknownWords(t *testing.T) {
	t.Parallel()
	svc, err := NewService()
	assert.NoError(t, err)

	tests := []struct {
		name string
		text string
	}{
		{"single unknown word", "asdfgh"},
		{"multiple unknown words", "foo bar baz"},
		{"numbers and unknown words", "123 xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := svc.Analyze(tt.text)

			assert.Equal(t, "neutral", got.Sentiment)
			assert.Equal(t, 1.0, got.Score)
			assert.Equal(t, Breakdown{
				Positive: 0,
				Negative: 0,
				Neutral:  1,
			}, got.Breakdown)
		})
	}
}

func TestAnalyze_Intensifiers(t *testing.T) {
	t.Parallel()
	svc, err := NewService()
	assert.NoError(t, err)

	tests := []struct {
		name string
		base string
		more string
	}{
		{"very", "good", "very good"},
		{"really", "good", "really good"},
		{"extremely", "good", "extremely good"},
		{"absolutely", "good", "absolutely good"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := svc.Analyze(tt.base)
			intense := svc.Analyze(tt.more)

			assert.Equal(t, "positive", intense.Sentiment)
			assert.Greater(t,
				intense.Breakdown.Positive,
				base.Breakdown.Positive,
			)
		})
	}
}

func TestValidateLexicon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lex  map[string]float64
		ok   bool
	}{
		{
			name: "valid",
			lex: map[string]float64{
				"good": 0.5,
			},
			ok: true,
		},
		{
			name: "too high",
			lex: map[string]float64{
				"good": 1.2,
			},
			ok: false,
		},
		{
			name: "too low",
			lex: map[string]float64{
				"bad": -1.1,
			},
			ok: false,
		},
		{
			name: "empty",
			lex:  map[string]float64{},
			ok:   false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateLexicon(tt.lex)

			if tt.ok {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
