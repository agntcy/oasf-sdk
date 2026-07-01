// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import "strings"

// chunkText splits text into overlapping windows of at most maxTokens word
// tokens so that long inputs (for example a full SKILL.md) fit inside an
// embedding model's context window.
//
// Chunks overlap by `overlap` tokens so concepts spanning a boundary are not
// lost. Each chunk is embedded separately and a class's semantic score is the
// maximum similarity over all chunks (max-pooling), so a single highly relevant
// section is enough to surface a label.
func chunkText(text string, maxTokens, overlap int) []string {
	words := strings.Fields(text)

	if maxTokens <= 0 || len(words) <= maxTokens {
		// Short enough to embed in one pass; keep the original text so the
		// embedder sees the natural spacing/punctuation.
		if strings.TrimSpace(text) == "" {
			return []string{""}
		}

		return []string{text}
	}

	// fallbackOverlapDivisor: when overlap is out of range, use 1/4 of
	// maxTokens as the default overlap so the window still advances.
	const fallbackOverlapDivisor = 4

	if overlap < 0 || overlap >= maxTokens {
		overlap = maxTokens / fallbackOverlapDivisor
	}

	step := maxTokens - overlap

	var chunks []string

	for start := 0; start < len(words); start += step {
		end := min(start+maxTokens, len(words))

		chunks = append(chunks, strings.Join(words[start:end], " "))

		if end == len(words) {
			break
		}
	}

	return chunks
}
