package resemble

import "testing"

func TestResemble(t *testing.T) {
	data := []struct{ A, B, GCD int }{
		{252, 105, 21},
		{53, 6, 1},
		{1386, 3213, 63},
	}
	for _, c := range data {
		gc := Resemble(c.A, c.B)
		if gc != c.GCD {
			t.Errorf("GCD of %d and %d should be %d, not %d", c.A, c.B, c.GCD, gc)
		}
	}
}
