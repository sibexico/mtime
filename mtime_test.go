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

func TestTTMinusUTCLeapTable(t *testing.T) {
	before := TTMinusUTC(time.Date(2016, 12, 31, 23, 59, 59, 0, time.UTC))
	after := TTMinusUTC(time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC))
	if math.Abs((after-before)-1.0) > 1e-9 {
		t.Fatalf("expected +1 second TT-UTC change, got before=%v after=%v", before, after)
	}
}

func TestTTMinusUTCPreLeapDrift(t *testing.T) {
	a := TTMinusUTC(time.Date(1961, 1, 1, 0, 0, 0, 0, time.UTC))
	b := TTMinusUTC(time.Date(1961, 12, 1, 0, 0, 0, 0, time.UTC))
	if b <= a {
		t.Fatalf("expected drifting TT-UTC to increase in 1961, got start=%v end=%v", a, b)
	}
}

func TestSetTTMinusUTCProvider(t *testing.T) {
	SetTTMinusUTCProvider(nil)
	defer SetTTMinusUTCProvider(nil)

	SetTTMinusUTCProvider(func(time.Time) float64 { return 70.0 })
	if got := TTMinusUTC(time.Unix(0, 0).UTC()); got != 70.0 {
		t.Fatalf("unexpected custom TT-UTC: %v", got)
	}
}

func TestLeapYearRule(t *testing.T) {
	if SolsInYear(1) != 668 {
		t.Fatalf("unexpected year 1 length: %d", SolsInYear(1))
	}
	if SolsInYear(2) != 669 {
		t.Fatalf("unexpected year 2 length: %d", SolsInYear(2))
	}
	if !IsLeapYear(2) {
		t.Fatal("expected year 2 to be leap year")
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

func TestMSDRoundTripExtremeDates(t *testing.T) {
	tests := []time.Time{
		time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(1600, 6, 15, 9, 30, 15, 987654321, time.UTC),
		time.Date(9999, 12, 31, 23, 59, 59, 999000000, time.UTC),
	}
	for _, base := range tests {
		m := FromEarth(base)
		round := FromMSD(m.MSD()).Earth()
		delta := round.Sub(base)
		if delta < 0 {
			delta = -delta
		}
		if delta > 50*time.Millisecond {
			t.Fatalf("extreme round-trip drift too large for %v: %v", base, delta)
		}
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
		{1336, 2, 24, 28, 669},
		{1337, 3, 1, 1, 1},
	}

	for _, tc := range tests {
		d := FromMSD(tc.msd).Date()
		if d.Year != tc.year || d.Month != tc.month || d.Day != tc.day || d.SolOfYear != tc.solOfYear {
			t.Fatalf("msd=%v got %+v want year=%d month=%d day=%d sol=%d", tc.msd, d, tc.year, tc.month, tc.day, tc.solOfYear)
		}
	}
}

func TestDateNearBoundaryNoEpsilonHack(t *testing.T) {
	d := FromMSD(100.9999999999).Date()
	if d.SolOfYear != 101 {
		t.Fatalf("unexpected boundary day from MSD: %+v", d)
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

func TestAddSolsRounding(t *testing.T) {
	start := FromEarth(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	shift := 0.123456789
	got := start.AddSols(shift).Sub(start).Nanoseconds()
	want := int64(math.Round(shift * float64(secondsPerSolNanos)))
	if got != want {
		t.Fatalf("unexpected AddSols rounding: got=%d want=%d", got, want)
	}
}

func TestAddSolsLargeRange(t *testing.T) {
	start := FromEarth(time.Date(1800, 1, 1, 0, 0, 0, 0, time.UTC))
	forward := start.AddSols(200000)
	back := forward.AddSols(-200000)

	delta := back.Sub(start)
	if delta < 0 {
		delta = -delta
	}
	if delta > 2*time.Millisecond {
		t.Fatalf("unexpected AddSols large-range drift: %v", delta)
	}
}
