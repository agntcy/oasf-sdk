// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestPoolScores(t *testing.T) {
	// One strong chunk among many weak ones: top-3 mean damps the single spike.
	if g := poolScores([]float64{0.9, 0.1, 0.1, 0.1}); !approx(g, (0.9+0.1+0.1)/3) {
		t.Errorf("top-3 mean = %v, want %v", g, (0.9+0.1+0.1)/3)
	}
	// A single chunk (short query) behaves like max — no regression.
	if g := poolScores([]float64{0.42}); !approx(g, 0.42) {
		t.Errorf("single chunk = %v, want 0.42", g)
	}
	// n larger than len clamps to the mean of all.
	if g := poolScores([]float64{0.2, 0.4}); !approx(g, 0.3) {
		t.Errorf("clamped mean = %v, want 0.3", g)
	}
	// Empty -> 0.
	if g := poolScores(nil); g != 0 {
		t.Errorf("empty = %v, want 0", g)
	}
	// Consistently-strong label beats one-spike label of equal max.
	spike := poolScores([]float64{0.8, 0.05, 0.05})

	broad := poolScores([]float64{0.8, 0.7, 0.6})
	if !(broad > spike) {
		t.Errorf("broad (%v) should beat spike (%v)", broad, spike)
	}
}
