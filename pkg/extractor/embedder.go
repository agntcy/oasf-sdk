// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"context"
	"math"
)

// EmbedRole tells an Embedder whether a text is a search query or a stored
// document (an OASF label). Asymmetric retrieval models (e.g. E5) prepend a
// different prefix per role; symmetric models ignore it.
type EmbedRole int

const (
	RoleQuery EmbedRole = iota
	RoleDocument
)

// Embedder turns text into dense vectors whose cosine similarity approximates
// semantic similarity. Implementations must be safe for concurrent use.
//
// The package ships a single implementation, TransformerEmbedder (a real,
// in-process sentence-transformer), which New uses by default. The interface
// exists as an injection seam: tests substitute a lightweight fake, and callers
// may supply a different model via WithEmbedder.
type Embedder interface {
	// Embed returns one vector per input text. All vectors have length Dim().
	// role distinguishes query text from document/label text for asymmetric
	// models; symmetric models ignore it.
	Embed(ctx context.Context, texts []string, role EmbedRole) ([][]float32, error)
	// Dim is the dimensionality of the produced vectors.
	Dim() int
	// MaxTokens is the approximate number of word tokens the model handles well
	// in a single pass. Longer inputs are split into chunks of this size before
	// embedding (see chunkText).
	MaxTokens() int
	// ID uniquely identifies the embedder configuration. Useful as a cache key
	// and for reproducibility (pin it alongside the OASF version).
	ID() string
}

// l2normalize scales vec in place to unit length (no-op for a zero vector).
func l2normalize(vec []float32) {
	var sum float64
	for _, v := range vec {
		sum += float64(v) * float64(v)
	}

	if sum == 0 {
		return
	}

	inv := float32(1 / math.Sqrt(sum))
	for i := range vec {
		vec[i] *= inv
	}
}

// dot returns the dot product of two equal-length vectors. For L2-normalized
// vectors this equals their cosine similarity.
func dot(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}

	return sum
}
