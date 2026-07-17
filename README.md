# Sentiment Go

A lightweight Go package for sentiment analysis using an embedded, AFINN-165-based lexicon.

[![Go Reference](https://pkg.go.dev/badge/github.com/bobadilla-tech/sentiment-go.svg)](https://pkg.go.dev/github.com/bobadilla-tech/sentiment-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/bobadilla-tech/sentiment-go)](https://goreportcard.com/report/github.com/bobadilla-tech/sentiment-go)
[![License](https://img.shields.io/github/license/bobadilla-tech/sentiment-go)](LICENSE)

## Features

- 🚀 **Zero runtime dependencies** — pure Go, standard library only
- 📦 **Embedded lexicon** — AFINN-165 wordlist compiled into the binary via `go:embed`, no external files needed at runtime
- 🔁 **Negation-aware** — "not good" is scored as negative, not positive
- 📈 **Intensifier-aware** — "very good" scores higher than "good"
- 🔧 **Configurable lexicon** — swap in your own lexicon via `WithLexicon` to tune scoring for a specific domain (e.g. product reviews vs. social media). Negation words and intensifiers are currently English-only and not configurable.
- 💻 **Simple API** — `NewService()` + `Analyze(text)`, nothing else to wire up
- 🧪 **Well tested** — negation, intensifiers, edge cases (empty input, unknown words), and lexicon validation all covered

## Installation

```bash
go get github.com/bobadilla-tech/sentiment-go
```

## Usage

```go
package main

import (
	"fmt"
	"log"

	sentiment "github.com/bobadilla-tech/sentiment-go"
)

func main() {
	svc, err := sentiment.NewService()
	if err != nil {
		log.Fatal(err)
	}

	result := svc.Analyze("I really love this product, it's amazing!")

	fmt.Printf("Sentiment: %s (score: %.2f)\n", result.Sentiment, result.Score)
	fmt.Printf("Breakdown: %+v\n", result.Breakdown)
}
```

Output:
```
Sentiment: positive (score: 0.94)
Breakdown: {Positive:0.94 Negative:0 Neutral:0.06}
```

### Using a custom lexicon

By default, `NewService()` uses the embedded AFINN-165 lexicon. To tune scoring for a specific domain, pass your own lexicon via `WithLexicon`. Values must fall within `[-0.9, 0.9]`:

```go
myLexicon := map[string]float64{
	"delayed":  -0.6,
	"refunded": 0.5,
}

svc, err := sentiment.NewService(sentiment.WithLexicon(myLexicon))
```

## How It Works

1. **Tokenize** — the input text is lowercased and split into alphanumeric tokens, stripping punctuation.
2. **Score** — each token is looked up in the lexicon. Negation words (`not`, `never`, `don't`, ...) invert the valence of the following three tokens; intensifiers (`very`, `extremely`, ...) multiply the valence of the next token.
3. **Normalize** — positive and negative valence sums are converted into a `Positive` / `Negative` / `Neutral` breakdown that always adds up to 1, with a small smoothing factor so no class ever reaches exactly 100% confidence from a single word.
4. **Source** — the default lexicon comes from [AFINN-165](https://github.com/fnielsen/afinn), by Finn Årup Nielsen, rescaled from its native `-5..5` integer range to `-0.9..0.9`.

## Testing

Run the tests:

```bash
go test -v ./...
```

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

## Credits

- Default lexicon based on [AFINN-165](https://github.com/fnielsen/afinn), by Finn Årup Nielsen.
