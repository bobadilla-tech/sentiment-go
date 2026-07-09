package sentiment

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

// neutralityWeight is added to the denominator when computing the breakdown
// to ensure no class ever reaches a probability of exactly 1 and to model
// the inherent ambiguity of natural language.
const neutralityWeight = 0.1

// negationWords, when encountered, invert the valence of the next three tokens.
var negationWords = map[string]bool{
	"not": true, "never": true, "no": true, "nobody": true,
	"nothing": true, "neither": true, "nor": true, "nowhere": true,
	"hardly": true, "barely": true, "scarcely": true,
	"without": true, "cannot": true,
	// Contractions — tokenize handles stripping punctuation, so these
	// also need to be listed in their "clean" forms where applicable.
	"dont": true, "didnt": true, "wont": true, "cant": true,
	"couldnt": true, "wouldnt": true, "shouldnt": true,
	"isnt": true, "arent": true, "wasnt": true, "werent": true,
	"doesnt": true, "havent": true, "hasnt": true, "hadnt": true,
}

// intensifiers multiply the valence of the following word.
var intensifiers = map[string]float64{
	"very": 1.3, "extremely": 1.5, "really": 1.2, "absolutely": 1.5,
	"totally": 1.4, "completely": 1.4, "utterly": 1.5, "incredibly": 1.5,
	"especially": 1.3, "particularly": 1.3, "quite": 1.1, "highly": 1.3,
	"deeply": 1.3, "truly": 1.3, "genuinely": 1.2, "super": 1.4,
	"most": 1.3,
}

// Breakdown contains the proportional score for each sentiment class.
type Breakdown struct {
	Positive float64 `json:"positive"`
	Negative float64 `json:"negative"`
	Neutral  float64 `json:"neutral"`
}

// Result is the response payload for the sentiment
type Result struct {
	Sentiment string    `json:"sentiment"`
	Score     float64   `json:"score"`
	Breakdown Breakdown `json:"breakdown"`
}

type service struct {
	lexicon       map[string]float64
	negationWords map[string]bool
	intensifiers  map[string]float64
}

// Option configures a Service.
type Option func(*service)

// WithLexicon overrides the sentiment lexicon (word -> valence, range [-0.9, 0.9]).
func WithLexicon(lex map[string]float64) Option {
	return func(s *service) {
		s.lexicon = lex
	}
}

// WithNegationWords overrides the set of negation words.
func WithNegationWords(words map[string]bool) Option {
	return func(s *service) {
		s.negationWords = words
	}
}

// WithIntensifiers overrides the intensifier multipliers.
func WithIntensifiers(intensifiers map[string]float64) Option {
	return func(s *service) {
		s.intensifiers = intensifiers
	}
}

// ValidateLexicon checks that every value in lex falls within the
// [-0.9, 0.9] range expected by the scoring algorithm. Lexicons with a
// different native scale (e.g. AFINN's raw -5..5 integers) must be
// normalized before being passed to WithLexicon.
//
// It returns an error describing the first offending entry found, if any.
func ValidateLexicon(lex map[string]float64) error {
	if len(lex) == 0 {
		return fmt.Errorf("sentiment: lexicon must not be empty")
	}
	for word, val := range lex {
		// Reject NaN/Inf before the range check, since NaN would silently
		// pass a naive `< -0.9 || > 0.9` comparison (NaN compares false
		// against everything).
		if math.IsNaN(val) || math.IsInf(val, 0) {
			return fmt.Errorf("sentiment: lexicon word %q has non-finite value %v", word, val)
		}
		// Values outside [-0.9, 0.9] would unbalance the breakdown math in
		// scoreTokens/Analyze, which assumes this as the max valence per word.
		if val < -0.9 || val > 0.9 {
			return fmt.Errorf("sentiment: lexicon word %q has value %v, must be in [-0.9, 0.9]", word, val)
		}
	}
	return nil
}

func NewService(opts ...Option) (*service, error) {
	s := &service{
		lexicon:       defaultLexicon(),
		negationWords: negationWords,
		intensifiers:  intensifiers,
	}

	for _, opt := range opts {
		opt(s)
	}

	if err := ValidateLexicon(s.lexicon); err != nil {
		return nil, err
	}
	return s, nil
}

// Analyze computes the sentiment of the given text, returning a label
// ("positive", "negative", or "neutral"), a confidence score, and a
// probability breakdown across the three classes.
func (s *service) Analyze(text string) Result {
	tokens := tokenize(text)

	posScore, negScore := s.scoreTokens(tokens)
	total := posScore + negScore

	if total == 0 {
		return Result{
			Sentiment: "neutral",
			Score:     1.0,
			Breakdown: Breakdown{Neutral: 1.0},
		}
	}

	// Divide each class by (total + neutralityWeight) so that even a single
	// strongly positive word does not push the positive score all the way to 1.
	denom := total + neutralityWeight
	pos := round2(posScore / denom)
	neg := round2(negScore / denom)
	// Derive neutral as the remainder to guarantee the three values sum to 1.
	neu := round2(1 - pos - neg)

	sentiment := "neutral"
	score := neu
	if pos >= neg && pos > neu {
		sentiment = "positive"
		score = pos
	} else if neg > pos && neg > neu {
		sentiment = "negative"
		score = neg
	}

	return Result{
		Sentiment: sentiment,
		Score:     score,
		Breakdown: Breakdown{
			Positive: pos,
			Negative: neg,
			Neutral:  neu,
		},
	}
}

// tokenize lower-cases text and splits it into alphanumeric tokens,
// stripping punctuation and whitespace.
func tokenize(text string) []string {
	var tokens []string
	var buf strings.Builder

	for _, r := range strings.ToLower(text) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			buf.WriteRune(r)
		case r == '\'' || r == '\u2019':
			// skip apostrophes so contractions stay intact: "don’t" → "dont"
			continue
		case buf.Len() > 0:
			tokens = append(tokens, buf.String())
			buf.Reset()
		}
	}
	if buf.Len() > 0 {
		tokens = append(tokens, buf.String())
	}
	return tokens
}

// round2 rounds f to two decimal places.
func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

// scoreTokens iterates over tokens, applying negation and intensifier
// modifiers, and returns separate positive and negative valence sums.
func (s *service) scoreTokens(tokens []string) (posSum, negSum float64) {
	// negationLeft tracks how many subsequent tokens are still under a negation.
	negationLeft := 0
	// intensifierMult carries an intensifier multiplier into the next token.
	intensifierMult := 1.0

	for _, token := range tokens {
		if negationLeft > 0 {
			negationLeft--
		}

		// Update state for the current token before applying it as a valence.
		if s.negationWords[token] {
			negationLeft = 3
			intensifierMult = 1.0
			continue
		}

		if mult, ok := s.intensifiers[token]; ok {
			intensifierMult = mult
			continue
		}

		valence, ok := s.lexicon[token]
		if !ok {
			intensifierMult = 1.0
			continue
		}

		valence *= intensifierMult
		intensifierMult = 1.0

		if negationLeft > 0 {
			valence = -valence
		}

		if valence > 0 {
			posSum += valence
		} else {
			negSum += math.Abs(valence)
		}
	}

	return posSum, negSum
}
