package client

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Match scoring. A match at or above MatchCertainScore is reported as certain;
// below it the CLI says "uncertain match".
const (
	MatchCertainScore = 0.85
	// matchAmbiguityGap: a different product scoring within this gap of the
	// best one makes the match uncertain.
	matchAmbiguityGap = 0.05
	// Weights of the name score: share of Heureka name tokens found in the
	// Alza name, and share of Alza model codes (tokens with digits) found in
	// the Heureka name.
	matchPrecisionWeight = 0.6
	matchCodeWeight      = 0.4
)

// matchStopwords are Heureka name suffixes that carry no identity.
var matchStopwords = map[string]bool{"originalny": true}

// matchOtherKind marks a different kind of product (clone, refurbished, used).
// A Heureka name with such a word that the Alza name lacks scores lower.
var matchOtherKind = map[string]bool{
	"kompatibilny": true, "kompatibilni": true, "repasovany": true,
	"renovovany": true, "bazar": true, "pouzity": true,
}

const matchOtherKindFactor = 0.6

// unitSpaceRe joins a number and its unit ("0,6 mm" -> "0,6mm", "8 ks" -> "8ks"),
// so both spellings give the same tokens.
var unitSpaceRe = regexp.MustCompile(`(\d)\s+(mm|cm|m|ml|l|g|kg|mg|ks|w|mah|gb|tb|v)\b`)

var diacriticsFold = strings.NewReplacer(
	"á", "a", "ä", "a", "č", "c", "ď", "d", "é", "e", "ě", "e", "í", "i",
	"ĺ", "l", "ľ", "l", "ň", "n", "ó", "o", "ô", "o", "ö", "o", "ŕ", "r",
	"ř", "r", "š", "s", "ť", "t", "ú", "u", "ů", "u", "ü", "u", "ý", "y", "ž", "z",
)

// matchTokens lowercases, folds diacritics and splits on non-alphanumerics.
func matchTokens(s string) []string {
	s = diacriticsFold.Replace(strings.ToLower(s))
	s = unitSpaceRe.ReplaceAllString(s, "$1$2")
	return strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func compactName(s string) string {
	return strings.Join(matchTokens(s), "")
}

func tokenSet(tokens []string) map[string]bool {
	out := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		if !matchStopwords[t] {
			out[t] = true
		}
	}
	return out
}

func hasDigit(s string) bool {
	return strings.IndexFunc(s, unicode.IsDigit) >= 0
}

// ScoreHeurekaName scores how well a Heureka product name matches an Alza
// product (0..1) and says why.
func ScoreHeurekaName(alzaName, partNumber, heurekaName string) (float64, string) {
	if compactName(alzaName) == compactName(heurekaName) && compactName(heurekaName) != "" {
		return 1, "same name"
	}

	alza := tokenSet(matchTokens(alzaName))
	hk := tokenSet(matchTokens(heurekaName))
	if len(hk) == 0 || len(alza) == 0 {
		return 0, ""
	}

	// Model codes come from the Alza name only: the part number is often an
	// EAN that never appears in a Heureka name.
	codes, codesFound := 0, 0
	for t := range alza {
		if len(t) >= 2 && hasDigit(t) {
			codes++
			if hk[t] {
				codesFound++
			}
		}
	}

	part := compactName(partNumber)
	if part != "" {
		alza[part] = true
	}

	shared := 0
	for t := range hk {
		if alza[t] {
			shared++
		}
	}
	precision := float64(shared) / float64(len(hk))
	codeRecall := precision
	if codes > 0 {
		codeRecall = float64(codesFound) / float64(codes)
	}
	score := matchPrecisionWeight*precision + matchCodeWeight*codeRecall
	reason := "name tokens"

	if part != "" && len(part) >= 4 && (hk[part] || strings.Contains(compactName(heurekaName), part)) {
		if partScore := 0.8 + 0.2*precision; partScore > score {
			score, reason = partScore, "part number"
		}
	}
	for t := range hk {
		if matchOtherKind[t] && !alza[t] {
			score *= matchOtherKindFactor
			break
		}
	}
	return score, reason
}

// MatchHeureka scores candidates and returns the best one, or nil when there
// are none. Ties are broken by offer count (the main listing has most shops).
func MatchHeureka(alzaName, partNumber string, candidates []HeurekaCandidate) *HeurekaMatch {
	if len(candidates) == 0 {
		return nil
	}
	scored := make([]HeurekaCandidate, len(candidates))
	for i, c := range candidates {
		c.Score, c.Reason = ScoreHeurekaName(alzaName, partNumber, c.Name)
		c.Score = round2(c.Score)
		scored[i] = c
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		return scored[i].OfferCount > scored[j].OfferCount
	})

	best := scored[0]
	if best.Score <= 0 {
		return nil
	}
	m := &HeurekaMatch{Candidate: best, Certain: best.Score >= MatchCertainScore}
	bestName := compactName(best.Name)
	for _, c := range scored[1:] {
		if c.Score <= 0 {
			break
		}
		if len(m.Alternatives) < 2 {
			m.Alternatives = append(m.Alternatives, c)
		}
		if c.Score >= best.Score-matchAmbiguityGap && compactName(c.Name) != bestName {
			m.Certain = false
		}
	}
	return m
}
