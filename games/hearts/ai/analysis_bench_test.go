package ai

import (
	"testing"
)

// BenchmarkAnalyze measures analyze() per-call cost across the six
// benchmark fixtures covering the major code paths (opening lead, clean
// follow, follow with points on the table, void discard, and two
// late-game moon-threat scenarios).
//
// ReportAllocs ON — per-call B/op and allocs/op feed the PIMC sizing
// budget. Fixture is built once outside the loop because analyze is
// read-only and can safely share *hearts.Game across iterations.
func BenchmarkAnalyze(b *testing.B) {
	for _, tc := range benchFixtures() {
		b.Run(tc.name, func(b *testing.B) {
			g, seat := tc.build()
			b.ReportAllocs()
			for b.Loop() {
				_ = analyze(g, seat)
			}
		})
	}
}
