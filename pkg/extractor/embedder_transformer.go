// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/nlpodyssey/cybertron/pkg/models/bert"
	"github.com/nlpodyssey/cybertron/pkg/tasks"
	"github.com/nlpodyssey/cybertron/pkg/tasks/textencoding"
)

// defaultModelName is the default sentence-transformer used when no model name
// is supplied via WithModelName.
const defaultModelName = "all-MiniLM-L6-v2"

// Per-role retrieval prefixes. Empty for symmetric models like
// all-MiniLM-L6-v2. Set to the model's required prefixes when the configured
// model is asymmetric (e.g. e5: "query: " / "passage: ").
const (
	queryPrefix    = ""
	documentPrefix = ""
)

// transformerDim is the default model's embedding width, used as the fallback
// until newTransformerEmbedder probes the loaded model for its actual dimension.
const transformerDim = 384

// transformerMaxTokens is the chunking window in *word* tokens. The model's
// hard limit is 512 *sub-word* (WordPiece) tokens, and dense text (code,
// markdown, long tokens) can expand to ~2.4 WordPieces per word, so we keep the
// word window conservative. The encode backstop truncates anything that still
// overflows, so this value only affects quality/throughput, never correctness.
const transformerMaxTokens = 200

// TransformerEmbedder is the package's semantic Embedder: a real, trained
// sentence-transformer run in-process via cybertron/spago. The model is loaded
// from the on-disk asset directory (downloaded+converted by Provision). Pure
// Go; no CGo and no external service. It is safe for concurrent use.
type TransformerEmbedder struct {
	model   textencoding.Interface
	dim     int
	modelID string

	mu sync.Mutex
}

// normalizeModelName ensures the HF "<org>/<model>" form. A bare model name is
// assumed to be a sentence-transformers model.
func normalizeModelName(name string) string {
	if strings.Contains(name, "/") {
		return name
	}

	return "sentence-transformers/" + name
}

// newTransformerEmbedder loads (downloading+converting if missing) the BERT
// model from modelDir and returns a ready embedder. Pure Go; network access
// happens only when the model is absent from modelDir.
func newTransformerEmbedder(ctx context.Context, modelDir, name string) (*TransformerEmbedder, error) {
	full := normalizeModelName(name)

	m, err := tasks.LoadModelForTextEncoding(&tasks.Config{
		ModelsDir:           modelDir,
		ModelName:           full,
		DownloadPolicy:      tasks.DownloadMissing,
		ConversionPolicy:    tasks.ConvertMissing,
		ConversionPrecision: tasks.F32,
	})
	if err != nil {
		return nil, fmt.Errorf("load model %q: %w", full, err)
	}

	e := &TransformerEmbedder{
		model:   m,
		dim:     transformerDim,
		modelID: "cybertron:" + full,
	}

	// Derive the true embedding dimension from the model with a one-off probe, so
	// Dim() is correct for non-default BERT models (e.g. 768-wide) as well. Falls
	// back to transformerDim if the probe fails.
	if vec, probeErr := e.encode(ctx, "dimension probe"); probeErr == nil && len(vec) > 0 {
		e.dim = len(vec)
	}

	return e, nil
}

// Embed implements Embedder. Returns one L2-normalized vector per input text
// (so the extractor's dot product equals cosine similarity).
func (e *TransformerEmbedder) Embed(ctx context.Context, texts []string, role EmbedRole) ([][]float32, error) {
	prefix := queryPrefix
	if role == RoleDocument {
		prefix = documentPrefix
	}

	out := make([][]float32, len(texts))

	for i, t := range texts {
		vec, err := e.encode(ctx, prefix+t)
		if err != nil {
			return nil, fmt.Errorf("encode text %d: %w", i, err)
		}

		out[i] = vec
	}

	return out, nil
}

func (e *TransformerEmbedder) encode(ctx context.Context, text string) ([]float32, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Backstop: the model rejects inputs over its sub-word token limit. Word
	// counts don't map exactly to WordPiece counts, so shrink and retry until
	// the input fits. Guarantees encode never fails on over-long text.
	for {
		res, err := e.model.Encode(ctx, text, int(bert.MeanPooling))
		if err != nil {
			if errors.Is(err, textencoding.ErrInputSequenceTooLong) {
				if shorter := shrink(text); shorter != text {
					text = shorter

					continue
				}
			}

			return nil, fmt.Errorf("failed to encode text: %w", err)
		}

		f64 := res.Vector.Data().F64()

		vec := make([]float32, len(f64))
		for i, v := range f64 {
			vec[i] = float32(v)
		}

		l2normalize(vec)

		return vec, nil
	}
}

// shrinkDropPct is the fraction of word tokens to DROP when shrinking: remove
// the trailing 15% of words on each retry so the input converges to the model's
// token limit.
const (
	shrinkDropPct     = 15
	shrinkDropDivisor = 100
)

// shrink returns a strictly shorter version of text: drop ~15% of the trailing
// words, or halve the runes when there is a single oversized token. Returns ""
// only when text cannot shrink further, which terminates the encode retry loop.
func shrink(text string) string {
	if words := strings.Fields(text); len(words) > 1 {
		keep := (shrinkDropDivisor - shrinkDropPct) * len(words) / shrinkDropDivisor
		n := max(keep, 1)

		return strings.Join(words[:n], " ")
	}

	r := []rune(strings.TrimSpace(text))
	if len(r) <= 1 {
		return ""
	}

	return string(r[:len(r)/2])
}

func (e *TransformerEmbedder) Dim() int       { return e.dim }
func (e *TransformerEmbedder) MaxTokens() int { return transformerMaxTokens }
func (e *TransformerEmbedder) ID() string     { return e.modelID }
