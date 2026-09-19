// Package dedupe finds wire reprints: near-identical articles run by several
// outlets. It uses MinHash over character shingles, which needs no word
// segmenter and so works for Chinese as well as English.
package dedupe

import (
	"hash/fnv"
	"unicode"
)

const (
	// SignatureSize is the number of hash functions. With 128, the estimate's
	// standard error near the threshold is about 3 points.
	SignatureSize = 128
	shingleSize   = 5
	// ReprintThreshold is the estimated Jaccard similarity above which two
	// articles are treated as the same wire story. Outlets trim, retitle and
	// append their own credit lines, so an exact match is not expected.
	ReprintThreshold = 0.8
	// minShingles guards against calling two very short texts "identical".
	minShingles = 40
)

// Signature is a MinHash signature. Values are stored as int32 because that
// is what a Postgres integer[] holds; only equality between positions matters.
type Signature []int32

// Sign returns the signature of text, or nil when the text is too short to
// compare meaningfully.
func Sign(text string) Signature {
	runes := normalize(text)
	if len(runes)-shingleSize+1 < minShingles {
		return nil
	}

	mins := make([]uint32, SignatureSize)
	for i := range mins {
		mins[i] = ^uint32(0)
	}
	for i := 0; i+shingleSize <= len(runes); i++ {
		h := fnv.New64a()
		for _, r := range runes[i : i+shingleSize] {
			h.Write([]byte(string(r)))
		}
		base := h.Sum64()
		// Derive the k hash functions from one 64-bit hash (double hashing).
		h1, h2 := uint32(base), uint32(base>>32)|1
		for k := range mins {
			if v := h1 + uint32(k)*h2; v < mins[k] {
				mins[k] = v
			}
		}
	}

	sig := make(Signature, SignatureSize)
	for i, v := range mins {
		sig[i] = int32(v)
	}
	return sig
}

// Similarity estimates the Jaccard similarity of the two signed texts.
func Similarity(a, b Signature) float64 {
	if len(a) != SignatureSize || len(b) != SignatureSize {
		return 0
	}
	same := 0
	for i := range a {
		if a[i] == b[i] {
			same++
		}
	}
	return float64(same) / SignatureSize
}

// normalize keeps letters and digits only, lower-cased, so that spacing,
// punctuation style and full-width/half-width marks don't affect the result.
func normalize(text string) []rune {
	out := make([]rune, 0, len(text))
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, unicode.ToLower(r))
		}
	}
	return out
}
