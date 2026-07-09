package sentiment

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed lexicon.json
var afinnRaw []byte

var (
	afinnOnce sync.Once
	afinnData map[string]float64
)

// defaultLexicon parses the embedded AFINN-165 JSON (word -> int, -5..5)
// once, and rescales it to the -0.9..0.9 valence range used by scoreTokens.
func defaultLexicon() map[string]float64 {
	afinnOnce.Do(func() {
		var raw map[string]int
		if err := json.Unmarshal(afinnRaw, &raw); err != nil {
			panic("sentiment: invalid embedded lexicon.json: " + err.Error())
		}
		afinnData = make(map[string]float64, len(raw))
		for word, score := range raw {
			// Rescale AFINN's -5..5 range to this package's -0.9..0.9 range.
			afinnData[word] = (float64(score) / 5.0) * 0.9
		}
	})
	return afinnData
}
