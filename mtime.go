package mtime

import (
	"fmt"
	"math"
	"time"
)

const (
	// Version is the current package semantic version.
	Version = "v0.1.0"
	// SecondsPerSol is the length of one Martian sol in Earth seconds.
	SecondsPerSol = 88775.244147
	// MarsYearSols is the number of sols in a compact Martian calendar year.
	MarsYearSols = 668
)

const (
	julianUnixEpoch = 2440587.5
	msdEpochJDTT    = 2405522.0028779
	msdRatio        = 1.0274912517
	ttMinusUTC      = 69.184 // seconds
	msdEpsilon      = 1e-9
)

var monthLengths = [...]int{28, 28, 28, 28, 28, 27, 28, 28, 28, 28, 28, 27, 28, 28, 28, 28, 28, 27, 28, 28, 28, 28, 28, 27}

// Time is a Mars-aware instant backed by an Earth UTC timestamp.
type Time struct {
	earth time.Time
}

// Date is a compact Martian calendar date.
type Date struct {
	Year      int
	Month     int
	Day       int
	SolOfYear int
}

// Clock is a Mars coordinated time (MTC) wall clock.
type Clock struct {
	Hour        int
	Minute      int
	Second      int
	Millisecond int
}

// Now returns the current moment as Martian time.
func Now() Time {
	return FromEarth(time.Now())
}

// FromEarth builds a Martian time from an Earth timestamp.
func FromEarth(t time.Time) Time {
	return Time{earth: t.UTC()}
}

// FromUnix builds a Martian time from Unix seconds and nanoseconds.
func FromUnix(sec int64, nsec int64) Time {
	return FromEarth(time.Unix(sec, nsec))
}

// FromMSD builds a Martian time from a Mars Sol Date value.
func FromMSD(msd float64) Time {
	jdTT := msd*msdRatio + msdEpochJDTT
	jdUTC := jdTT - ttMinusUTC/86400.0
	unix := (jdUTC - julianUnixEpoch) * 86400.0
	sec, frac := math.Modf(unix)
	nsec := int64(math.Round(frac * 1e9))
	if nsec == 1e9 {
		sec++
		nsec = 0
	}
	return FromUnix(int64(sec), nsec)
}

// Earth returns the Earth UTC timestamp for this Martian instant.
func (t Time) Earth() time.Time {
	return t.earth
}

// MSD returns the Mars Sol Date for this instant.
func (t Time) MSD() float64 {
	unix := float64(t.earth.Unix()) + float64(t.earth.Nanosecond())/1e9
	jdUTC := unix/86400.0 + julianUnixEpoch
	jdTT := jdUTC + ttMinusUTC/86400.0
	return (jdTT - msdEpochJDTT) / msdRatio
}

// MTC returns Mars coordinated time for this instant.
func (t Time) MTC() Clock {
	msd := t.MSD() + msdEpsilon
	frac := msd - math.Floor(msd)
	hoursTotal := frac * 24.0
	hour := int(hoursTotal)
	minuteTotal := (hoursTotal - float64(hour)) * 60.0
	minute := int(minuteTotal)
	secondTotal := (minuteTotal - float64(minute)) * 60.0
	second := int(secondTotal)
	millisecond := int(math.Round((secondTotal - float64(second)) * 1000.0))

	if millisecond == 1000 {
		millisecond = 0
		second++
	}
	if second == 60 {
		second = 0
		minute++
	}
	if minute == 60 {
		minute = 0
		hour++
	}
	if hour == 24 {
		hour = 0
	}

	return Clock{
		Hour:        hour,
		Minute:      minute,
		Second:      second,
		Millisecond: millisecond,
	}
}

// Date returns the compact Martian calendar date for this instant.
func (t Time) Date() Date {
	solNumber := int64(math.Floor(t.MSD() + msdEpsilon))
	year, solOfYear0 := splitYearAndSol(solNumber)
	month, day := splitMonthAndDay(solOfYear0)
	return Date{
		Year:      year,
		Month:     month,
		Day:       day,
		SolOfYear: solOfYear0 + 1,
	}
}

// Add returns a new Time after adding an Earth duration.
func (t Time) Add(d time.Duration) Time {
	return FromEarth(t.earth.Add(d))
}

// AddSols returns a new Time after adding Martian sols.
func (t Time) AddSols(sols float64) Time {
	seconds := sols * SecondsPerSol
	return t.Add(time.Duration(seconds * float64(time.Second)))
}

// Sub returns the Earth duration between two Martian instants.
func (t Time) Sub(u Time) time.Duration {
	return t.earth.Sub(u.earth)
}

// DiffSols returns the time difference between two instants in sols.
func (t Time) DiffSols(u Time) float64 {
	return t.Sub(u).Seconds() / SecondsPerSol
}

// Before reports whether t happens before u.
func (t Time) Before(u Time) bool {
	return t.earth.Before(u.earth)
}

// After reports whether t happens after u.
func (t Time) After(u Time) bool {
	return t.earth.After(u.earth)
}

// Equal reports whether t and u are the same instant.
func (t Time) Equal(u Time) bool {
	return t.earth.Equal(u.earth)
}

// String returns a readable Martian date-time string.
func (t Time) String() string {
	d := t.Date()
	c := t.MTC()
	return fmt.Sprintf("MY%04d-%02d-%02d S%03d %02d:%02d:%02d.%03d MTC", d.Year, d.Month, d.Day, d.SolOfYear, c.Hour, c.Minute, c.Second, c.Millisecond)
}

// String formats the Martian date.
func (d Date) String() string {
	return fmt.Sprintf("MY%04d-%02d-%02d (S%03d)", d.Year, d.Month, d.Day, d.SolOfYear)
}

// String formats Mars coordinated time.
func (c Clock) String() string {
	return fmt.Sprintf("%02d:%02d:%02d.%03d", c.Hour, c.Minute, c.Second, c.Millisecond)
}

// Since returns the Earth duration since t.
func Since(t Time) time.Duration {
	return Now().Sub(t)
}

// Until returns the Earth duration until t.
func Until(t Time) time.Duration {
	return t.Sub(Now())
}

func splitYearAndSol(solNumber int64) (year int, solOfYear0 int) {
	year0 := floorDiv(solNumber, MarsYearSols)
	solOfYear0 = int(solNumber - int64(year0*MarsYearSols))
	return year0 + 1, solOfYear0
}

func splitMonthAndDay(solOfYear0 int) (month int, day int) {
	sol := solOfYear0
	for i, length := range monthLengths {
		if sol < length {
			return i + 1, sol + 1
		}
		sol -= length
	}
	return len(monthLengths), monthLengths[len(monthLengths)-1]
}

func floorDiv(a int64, b int) int {
	div := int(a / int64(b))
	rem := int(a % int64(b))
	if rem != 0 && ((rem > 0) != (b > 0)) {
		div--
	}
	return div
}
