package service

import (
	"math"
	"testing"
)

func TestHaversineMeters(t *testing.T) {
	if d := haversineMeters(48.8566, 2.3522, 48.8566, 2.3522); d > 1 {
		t.Errorf("same point: expected ~0, got %.2f m", d)
	}
	d := haversineMeters(48.8566, 2.3522, 45.7640, 4.8357)
	if math.Abs(d-392000) > 10000 {
		t.Errorf("Paris-Lyon: expected ~392000 m, got %.0f m", d)
	}
}
