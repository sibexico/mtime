package mtime

import (
	"math"
	"testing"
	"time"
)

func TestVersion(t *testing.T) {
	if Version != "v0.1.0" {
		t.Fatalf("unexpected version: %q", Version)
	}
}

func TestNowIsCloseToEarthNow(t *testing.T) {
	n := Now().Earth()
	d := time.Since(n)
	if d < 0 {
		d = -d
	}
	if d > time.Second {
		t.Fatalf("Now() too far from time.Now(): %v", d)
	}
}

func TestMSDRoundTrip(t *testing.T) {
	base := time.Date(2026, 4, 18, 12, 34, 56, 123456000, time.UTC)
	m := FromEarth(base)
	round := FromMSD(m.MSD()).Earth()
	delta := round.Sub(base)
	if delta < 0 {
		delta = -delta
	}
	if delta > 2*time.Millisecond {
		t.Fatalf("MSD round-trip drift too large: %v", delta)
	}
}

func TestAddSols(t *testing.T) {
	start := FromEarth(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	next := start.AddSols(1)
	delta := next.Sub(start).Seconds()
	if math.Abs(delta-SecondsPerSol) > 1e-6 {
		t.Fatalf("expected one sol=%f, got %f", SecondsPerSol, delta)
	}
}

func TestDateBoundaries(t *testing.T) {
	tests := []struct {
		msd       float64
		year      int
		month     int
		day       int
		solOfYear int
	}{
		{0, 1, 1, 1, 1},
		{27, 1, 1, 28, 28},
		{28, 1, 2, 1, 29},
		{166, 1, 6, 27, 167},
		{167, 1, 7, 1, 168},
		{668, 2, 1, 1, 1},
	}

	for _, tc := range tests {
		d := FromMSD(tc.msd).Date()
		if d.Year != tc.year || d.Month != tc.month || d.Day != tc.day || d.SolOfYear != tc.solOfYear {
			t.Fatalf("msd=%v got %+v want year=%d month=%d day=%d sol=%d", tc.msd, d, tc.year, tc.month, tc.day, tc.solOfYear)
		}
	}
}

func TestCompareAndDiff(t *testing.T) {
	a := FromEarth(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	b := a.Add(2 * time.Hour)

	if !b.After(a) {
		t.Fatal("expected b after a")
	}
	if !a.Before(b) {
		t.Fatal("expected a before b")
	}
	if a.Equal(b) {
		t.Fatal("did not expect equality")
	}

	if b.Sub(a) != 2*time.Hour {
		t.Fatalf("unexpected sub value: %v", b.Sub(a))
	}

	wantSols := (2 * time.Hour).Seconds() / SecondsPerSol
	if math.Abs(b.DiffSols(a)-wantSols) > 1e-12 {
		t.Fatalf("unexpected diff sols: got=%v want=%v", b.DiffSols(a), wantSols)
	}
}

func TestStringFormats(t *testing.T) {
	m := FromMSD(0)
	if got := m.Date().String(); got != "MY0001-01-01 (S001)" {
		t.Fatalf("unexpected date string: %q", got)
	}
	if got := m.MTC().String(); got != "00:00:00.000" {
		t.Fatalf("unexpected clock string: %q", got)
	}
	if got := m.String(); got == "" {
		t.Fatal("unexpected empty Time string")
	}
}
