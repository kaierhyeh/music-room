package service

import "testing"

func TestComputePosition(t *testing.T) {
	cases := []struct {
		name      string
		positions []float64
		afterIdx  int
		want      float64
		renorm    bool
	}{
		{"empty, head", nil, -1, 1.0, false},
		{"head", []float64{1, 2, 3}, -1, 0.0, false},
		{"tail", []float64{1, 2, 3}, 2, 4.0, false},
		{"middle", []float64{1, 2, 3}, 0, 1.5, false},
		{"tight gap -> renorm", []float64{1.0, 1.0000000001}, 0, 1.00000000005, true},
	}
	for _, c := range cases {
		got, renorm := computePosition(c.positions, c.afterIdx)
		if renorm != c.renorm {
			t.Errorf("%s: renorm = %v, want %v", c.name, renorm, c.renorm)
		}
		if !c.renorm && got != c.want {
			t.Errorf("%s: pos = %v, want %v", c.name, got, c.want)
		}
	}
}
