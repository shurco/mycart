package security

import "testing"

func TestRandomString_HighCollisionResistance(t *testing.T) {
	t.Parallel()

	const n = 5000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		s := RandomString()
		if _, dup := seen[s]; dup {
			// 36^15 ≈ 2.2e23 combinations; a duplicate in 5000 samples is a
			// statistical near-impossibility, so it signals a regression.
			t.Fatalf("RandomString collision at iteration %d: %q", i, s)
		}
		seen[s] = struct{}{}
	}
}
