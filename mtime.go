package mtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// ErrInvalidSols reports a NaN or infinite sols input.
	ErrInvalidSols = errors.New("mtime: sols value is NaN or infinite")
	// ErrInvalidMSD reports a NaN or infinite MSD input.
	ErrInvalidMSD = errors.New("mtime: invalid MSD value")
	// ErrOutOfRange reports values outside representable time range.
	ErrOutOfRange = errors.New("mtime: result is out of representable range")
	// ErrInvalidFormat reports parsing/formatting input that cannot be decoded.
	ErrInvalidFormat = errors.New("mtime: invalid time format")
)

const defaultStringLayout = "MY%04d-%02d-%02d S%03d %02d:%02d:%02d.%03d MTC"

const (
	// Version is the current package semantic version.
	Version = "v0.3.0"
	// SecondsPerSol is the length of one Martian sol in Earth seconds.
	SecondsPerSol = 88775.244147
	// MarsYearSols is the baseline number of sols in a Martian year.
	MarsYearSols = 668
)

const (
	julianUnixEpoch = 2440587.5
	// MSD epoch in Julian Date TT (Allison & McEwen 2000, NASA Mars24 convention).
	msdEpochJDTT       = 2405522.0028779
	leapSolNumerator   = 5921
	leapSolDenominator = 10000
)

var monthLengths = [...]int{28, 28, 28, 28, 28, 27, 28, 28, 28, 28, 28, 27, 28, 28, 28, 28, 28, 27, 28, 28, 28, 28, 28, 27}

const secondsPerSolNanos int64 = 88775244147000

var msdUnixOffsetNanos = int64(math.Round((julianUnixEpoch - msdEpochJDTT) * 86400.0 * float64(time.Second)))

var (
	bigOne               = big.NewInt(1)
	bigNanosPerSecond    = big.NewInt(int64(time.Second))
	bigSecondsPerSolNano = big.NewInt(secondsPerSolNanos)
	bigMSDUnixOffsetNano = big.NewInt(msdUnixOffsetNanos)
)

// TTMinusUTCProvider returns TT-UTC in seconds for a UTC instant.
type TTMinusUTCProvider func(at time.Time) float64

type leapSecondEntry struct {
	at      time.Time
	deltaAT float64
}

type preLeapDriftEntry struct {
	at     time.Time
	delta  float64
	refMJD float64
	drift  float64
}

var leapSeconds = [...]leapSecondEntry{
	{at: time.Date(1972, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 10},
	{at: time.Date(1972, 7, 1, 0, 0, 0, 0, time.UTC), deltaAT: 11},
	{at: time.Date(1973, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 12},
	{at: time.Date(1974, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 13},
	{at: time.Date(1975, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 14},
	{at: time.Date(1976, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 15},
	{at: time.Date(1977, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 16},
	{at: time.Date(1978, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 17},
	{at: time.Date(1979, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 18},
	{at: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 19},
	{at: time.Date(1981, 7, 1, 0, 0, 0, 0, time.UTC), deltaAT: 20},
	{at: time.Date(1982, 7, 1, 0, 0, 0, 0, time.UTC), deltaAT: 21},
	{at: time.Date(1983, 7, 1, 0, 0, 0, 0, time.UTC), deltaAT: 22},
	{at: time.Date(1985, 7, 1, 0, 0, 0, 0, time.UTC), deltaAT: 23},
	{at: time.Date(1988, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 24},
	{at: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 25},
	{at: time.Date(1991, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 26},
	{at: time.Date(1992, 7, 1, 0, 0, 0, 0, time.UTC), deltaAT: 27},
	{at: time.Date(1993, 7, 1, 0, 0, 0, 0, time.UTC), deltaAT: 28},
	{at: time.Date(1994, 7, 1, 0, 0, 0, 0, time.UTC), deltaAT: 29},
	{at: time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 30},
	{at: time.Date(1997, 7, 1, 0, 0, 0, 0, time.UTC), deltaAT: 31},
	{at: time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 32},
	{at: time.Date(2006, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 33},
	{at: time.Date(2009, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 34},
	{at: time.Date(2012, 7, 1, 0, 0, 0, 0, time.UTC), deltaAT: 35},
	{at: time.Date(2015, 7, 1, 0, 0, 0, 0, time.UTC), deltaAT: 36},
	{at: time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC), deltaAT: 37},
}

var preLeapDrift = [...]preLeapDriftEntry{
	{at: time.Date(1960, 1, 1, 0, 0, 0, 0, time.UTC), delta: 1.4178180, refMJD: 37300, drift: 0.001296},
	{at: time.Date(1961, 1, 1, 0, 0, 0, 0, time.UTC), delta: 1.4228180, refMJD: 37300, drift: 0.001296},
	{at: time.Date(1961, 8, 1, 0, 0, 0, 0, time.UTC), delta: 1.3728180, refMJD: 37300, drift: 0.001296},
	{at: time.Date(1962, 1, 1, 0, 0, 0, 0, time.UTC), delta: 1.8458580, refMJD: 37665, drift: 0.0011232},
	{at: time.Date(1963, 11, 1, 0, 0, 0, 0, time.UTC), delta: 1.9458580, refMJD: 37665, drift: 0.0011232},
	{at: time.Date(1964, 1, 1, 0, 0, 0, 0, time.UTC), delta: 3.2401300, refMJD: 38761, drift: 0.001296},
	{at: time.Date(1964, 4, 1, 0, 0, 0, 0, time.UTC), delta: 3.3401300, refMJD: 38761, drift: 0.001296},
	{at: time.Date(1964, 9, 1, 0, 0, 0, 0, time.UTC), delta: 3.4401300, refMJD: 38761, drift: 0.001296},
	{at: time.Date(1965, 1, 1, 0, 0, 0, 0, time.UTC), delta: 3.5401300, refMJD: 38761, drift: 0.001296},
	{at: time.Date(1965, 3, 1, 0, 0, 0, 0, time.UTC), delta: 3.6401300, refMJD: 38761, drift: 0.001296},
	{at: time.Date(1965, 7, 1, 0, 0, 0, 0, time.UTC), delta: 3.7401300, refMJD: 38761, drift: 0.001296},
	{at: time.Date(1965, 9, 1, 0, 0, 0, 0, time.UTC), delta: 3.8401300, refMJD: 38761, drift: 0.001296},
	{at: time.Date(1966, 1, 1, 0, 0, 0, 0, time.UTC), delta: 4.3131700, refMJD: 39126, drift: 0.002592},
	{at: time.Date(1968, 2, 1, 0, 0, 0, 0, time.UTC), delta: 4.2131700, refMJD: 39126, drift: 0.002592},
}

var (
	ttOffsetMu       sync.RWMutex
	ttOffsetProvider TTMinusUTCProvider = defaultTTMinusUTC
	leapWarnOnce     sync.Once
)

// LastLeapSecondDate is the latest date in the built-in leap-second table.
// Times at or after this date may be off if additional leap seconds exist.
var LastLeapSecondDate = leapSeconds[len(leapSeconds)-1].at

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

// TTMinusUTC returns the current TT-UTC offset in seconds for an instant.
func TTMinusUTC(at time.Time) float64 {
	ttOffsetMu.RLock()
	provider := ttOffsetProvider
	ttOffsetMu.RUnlock()
	return provider(at.UTC())
}

// SetTTMinusUTCProvider overrides the TT-UTC source used by conversions.
func SetTTMinusUTCProvider(provider TTMinusUTCProvider) {
	if provider == nil {
		provider = defaultTTMinusUTC
	}
	ttOffsetMu.Lock()
	ttOffsetProvider = provider
	ttOffsetMu.Unlock()
}

// SolsInYear reports how many sols exist in a given Martian year.
func SolsInYear(year int) int {
	if isLeapYear(year) {
		return MarsYearSols + 1
	}
	return MarsYearSols
}

// IsLeapYear reports whether a Martian year contains an intercalary leap sol.
func IsLeapYear(year int) bool {
	return isLeapYear(year)
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
	out, err := FromMSDSafe(msd)
	if err != nil {
		panic(err)
	}
	return out
}

// FromMSDSafe builds a Martian time from a Mars Sol Date value without panic.
func FromMSDSafe(msd float64) (Time, error) {
	if math.IsNaN(msd) || math.IsInf(msd, 0) {
		return Time{}, ErrInvalidMSD
	}
	adjustedNanos := roundFloatProductToInt(msd, bigSecondsPerSolNano)

	// Seed from the target MSD itself and iteratively account for TT-UTC.
	utcNanos := new(big.Int).Sub(new(big.Int).Set(adjustedNanos), bigMSDUnixOffsetNano)
	for range 5 {
		prev := new(big.Int).Set(utcNanos)
		sec, nsec, ok := splitUnixNanosBig(utcNanos)
		if !ok {
			return Time{}, ErrOutOfRange
		}
		instant := time.Unix(sec, nsec).UTC()
		ttNanos := int64(math.Round(TTMinusUTC(instant) * float64(time.Second)))
		next := new(big.Int).Sub(new(big.Int).Set(adjustedNanos), big.NewInt(ttNanos))
		next.Sub(next, bigMSDUnixOffsetNano)
		utcNanos = next
		if prev.Cmp(utcNanos) == 0 {
			break
		}
	}

	sec, nsec, ok := splitUnixNanosBig(utcNanos)
	if !ok {
		return Time{}, ErrOutOfRange
	}
	return FromUnix(sec, nsec), nil
}

// Earth returns the Earth UTC timestamp for this Martian instant.
func (t Time) Earth() time.Time {
	return t.earth
}

// MSD returns the Mars Sol Date for this instant.
func (t Time) MSD() float64 {
	sol, rem := splitMSD(t.earth)
	return float64(sol) + float64(rem)/float64(secondsPerSolNanos)
}

// MTC returns Mars coordinated time for this instant.
func (t Time) MTC() Clock {
	_, rem := splitMSD(t.earth)
	ms := int(math.Round(float64(rem) * 86400000.0 / float64(secondsPerSolNanos)))
	if ms >= 86400000 {
		ms = 0
	}

	hour := ms / 3600000
	ms -= hour * 3600000
	minute := ms / 60000
	ms -= minute * 60000
	second := ms / 1000
	millisecond := ms - second*1000

	return Clock{
		Hour:        hour,
		Minute:      minute,
		Second:      second,
		Millisecond: millisecond,
	}
}

// Date returns the compact Martian calendar date for this instant.
func (t Time) Date() Date {
	solNumber, _ := splitMSD(t.earth)
	year, solOfYear0 := splitYearAndSol(solNumber)
	month, day := splitMonthAndDay(solOfYear0, isLeapYear(year))
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
	out, err := t.AddSolsSafe(sols)
	if err != nil {
		panic(err)
	}
	return out
}

// AddSolsSafe returns a new Time after adding sols, without panicking on overflow.
func (t Time) AddSolsSafe(sols float64) (Time, error) {
	if math.IsNaN(sols) || math.IsInf(sols, 0) {
		return Time{}, ErrInvalidSols
	}

	deltaNanos := roundFloatProductToInt(sols, bigSecondsPerSolNano)
	deltaSec, deltaNsec, ok := splitUnixNanosBig(deltaNanos)
	if !ok {
		return Time{}, ErrOutOfRange
	}

	sec, secOK := addInt64Checked(t.earth.Unix(), deltaSec)
	if !secOK {
		return Time{}, ErrOutOfRange
	}
	nsec := int64(t.earth.Nanosecond()) + deltaNsec
	if nsec >= int64(time.Second) {
		sec, secOK = addInt64Checked(sec, 1)
		if !secOK {
			return Time{}, ErrOutOfRange
		}
		nsec -= int64(time.Second)
	}
	if nsec < 0 {
		sec, secOK = addInt64Checked(sec, -1)
		if !secOK {
			return Time{}, ErrOutOfRange
		}
		nsec += int64(time.Second)
	}

	return FromUnix(sec, nsec), nil
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
	return fmt.Sprintf(defaultStringLayout, d.Year, d.Month, d.Day, d.SolOfYear, c.Hour, c.Minute, c.Second, c.Millisecond)
}

// AppendFormat appends the formatted representation of t to b.
// Tokens: MY MM DD SSS hh mm ss fff.
func (t Time) AppendFormat(b []byte, layout string) []byte {
	return append(b, t.Format(layout)...)
}

// Format returns a string formatted with custom tokens.
// Tokens: MY MM DD SSS hh mm ss fff.
func (t Time) Format(layout string) string {
	d := t.Date()
	c := t.MTC()
	r := strings.NewReplacer(
		"SSS", fmt.Sprintf("%03d", d.SolOfYear),
		"fff", fmt.Sprintf("%03d", c.Millisecond),
		"MY", fmt.Sprintf("%04d", d.Year),
		"MM", fmt.Sprintf("%02d", d.Month),
		"DD", fmt.Sprintf("%02d", d.Day),
		"hh", fmt.Sprintf("%02d", c.Hour),
		"mm", fmt.Sprintf("%02d", c.Minute),
		"ss", fmt.Sprintf("%02d", c.Second),
	)
	return r.Replace(layout)
}

// Parse parses a Martian time from layout and value.
// Tokens: MY MM DD SSS hh mm ss fff.
func Parse(layout, value string) (Time, error) {
	tokens := []string{"SSS", "fff", "MY", "MM", "DD", "hh", "mm", "ss"}
	patterns := map[string]string{
		"MY":  `(-?[0-9]+)`,
		"MM":  `([0-9]{2})`,
		"DD":  `([0-9]{2})`,
		"SSS": `([0-9]{3})`,
		"hh":  `([0-9]{2})`,
		"mm":  `([0-9]{2})`,
		"ss":  `([0-9]{2})`,
		"fff": `([0-9]{3})`,
	}

	var order []string
	var sb strings.Builder
	sb.WriteString("^")
	for i := 0; i < len(layout); {
		matched := ""
		for _, token := range tokens {
			if strings.HasPrefix(layout[i:], token) {
				matched = token
				break
			}
		}
		if matched != "" {
			sb.WriteString(patterns[matched])
			order = append(order, matched)
			i += len(matched)
			continue
		}
		sb.WriteString(regexp.QuoteMeta(layout[i : i+1]))
		i++
	}
	sb.WriteString("$")

	re, err := regexp.Compile(sb.String())
	if err != nil {
		return Time{}, fmt.Errorf("%w: %v", ErrInvalidFormat, err)
	}
	matches := re.FindStringSubmatch(value)
	if matches == nil {
		return Time{}, fmt.Errorf("%w: value does not match layout", ErrInvalidFormat)
	}

	vals := map[string]int{}
	for i, token := range order {
		n, convErr := strconv.Atoi(matches[i+1])
		if convErr != nil {
			return Time{}, fmt.Errorf("%w: invalid %s", ErrInvalidFormat, token)
		}
		vals[token] = n
	}

	required := []string{"MY", "MM", "DD", "hh", "mm", "ss", "fff"}
	for _, token := range required {
		if _, ok := vals[token]; !ok {
			return Time{}, fmt.Errorf("%w: missing token %s", ErrInvalidFormat, token)
		}
	}

	return timeFromCalendar(vals["MY"], vals["MM"], vals["DD"], vals["hh"], vals["mm"], vals["ss"], vals["fff"], vals["SSS"])
}

// ParseDefault parses the fixed String() representation.
func ParseDefault(s string) (Time, error) {
	var year, month, day, solOfYear, hour, minute, second, millisecond int
	n, err := fmt.Sscanf(strings.TrimSpace(s), "MY%d-%d-%d S%d %d:%d:%d.%d MTC", &year, &month, &day, &solOfYear, &hour, &minute, &second, &millisecond)
	if err != nil || n != 8 {
		return Time{}, fmt.Errorf("%w: cannot parse default format", ErrInvalidFormat)
	}
	return timeFromCalendar(year, month, day, hour, minute, second, millisecond, solOfYear)
}

// MarshalText implements encoding.TextMarshaler.
func (t Time) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (t *Time) UnmarshalText(data []byte) error {
	parsed, err := ParseDefault(string(data))
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

// MarshalJSON encodes Time as UTC nanoseconds for stable round trips.
func (t Time) MarshalJSON() ([]byte, error) {
	payload := struct {
		UTCNS int64 `json:"utc_ns"`
	}{
		UTCNS: t.earth.UnixNano(),
	}
	return json.Marshal(payload)
}

// UnmarshalJSON decodes Time from UTC nanoseconds.
func (t *Time) UnmarshalJSON(data []byte) error {
	var payload struct {
		UTCNS int64 `json:"utc_ns"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	*t = FromEarth(time.Unix(0, payload.UTCNS).UTC())
	return nil
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
	year64 := floorDiv64(solNumber, int64(MarsYearSols)) + 1
	for solsBeforeYear(year64) > solNumber {
		year64--
	}
	for solsBeforeYear(year64+1) <= solNumber {
		year64++
	}
	solOfYear0 = int(solNumber - solsBeforeYear(year64))
	return int(year64), solOfYear0
}

func splitMonthAndDay(solOfYear0 int, leapYear bool) (month int, day int) {
	sol := solOfYear0
	for i, length := range monthLengths {
		if leapYear && i == len(monthLengths)-1 {
			length++
		}
		if sol < length {
			return i + 1, sol + 1
		}
		sol -= length
	}
	end := monthLengths[len(monthLengths)-1]
	if leapYear {
		end++
	}
	return len(monthLengths), end
}

func isLeapYear(year int) bool {
	if year <= 0 {
		return false
	}
	return leapSolsThroughYear(int64(year)) != leapSolsThroughYear(int64(year-1))
}

func leapSolsThroughYear(year int64) int64 {
	if year <= 0 {
		return 0
	}
	return floorDiv64(year*leapSolNumerator, leapSolDenominator)
}

func solsBeforeYear(year int64) int64 {
	if year <= 1 {
		return 0
	}
	y := year - 1
	return y*MarsYearSols + leapSolsThroughYear(y)
}

func splitMSD(utc time.Time) (sol int64, rem int64) {
	ttNanos := int64(math.Round(TTMinusUTC(utc) * float64(time.Second)))

	utcNanos := new(big.Int).Mul(big.NewInt(utc.Unix()), bigNanosPerSecond)
	utcNanos.Add(utcNanos, big.NewInt(int64(utc.Nanosecond())))
	utcNanos.Add(utcNanos, big.NewInt(ttNanos))
	utcNanos.Add(utcNanos, bigMSDUnixOffsetNano)

	q := new(big.Int)
	r := new(big.Int)
	q.QuoRem(utcNanos, bigSecondsPerSolNano, r)
	if r.Sign() < 0 {
		q.Sub(q, bigOne)
		r.Add(r, bigSecondsPerSolNano)
	}
	if !q.IsInt64() || !r.IsInt64() {
		panic("mtime: time out of supported range")
	}

	sol = q.Int64()
	rem = r.Int64()
	return sol, rem
}

func splitUnixNanosBig(total *big.Int) (sec int64, nsec int64, ok bool) {
	q := new(big.Int)
	r := new(big.Int)
	q.QuoRem(total, bigNanosPerSecond, r)
	if r.Sign() < 0 {
		q.Sub(q, bigOne)
		r.Add(r, bigNanosPerSecond)
	}
	if !q.IsInt64() || !r.IsInt64() {
		return 0, 0, false
	}
	return q.Int64(), r.Int64(), true
}

func floorDiv64(a int64, b int64) int64 {
	div := a / b
	rem := a % b
	if rem != 0 && ((rem > 0) != (b > 0)) {
		div--
	}
	return div
}

func defaultTTMinusUTC(at time.Time) float64 {
	at = at.UTC()
	if at.Before(preLeapDrift[0].at) {
		return 32.184
	}
	if at.Before(leapSeconds[0].at) {
		entry := preLeapDrift[0]
		for _, candidate := range preLeapDrift {
			if at.Before(candidate.at) {
				break
			}
			entry = candidate
		}
		mjd := modifiedJulianDate(at)
		deltaAT := entry.delta + (mjd-entry.refMJD)*entry.drift
		return deltaAT + 32.184
	}
	deltaAT := leapSeconds[0].deltaAT
	for _, entry := range leapSeconds {
		if at.Before(entry.at) {
			break
		}
		deltaAT = entry.deltaAT
	}
	if !at.Before(LastLeapSecondDate) {
		leapWarnOnce.Do(func() {
			fmt.Fprintf(os.Stderr, "mtime: time %s is past the built-in leap second table (%s); consider SetTTMinusUTCProvider\n", at.Format(time.RFC3339), LastLeapSecondDate.Format(time.RFC3339))
		})
	}
	return deltaAT + 32.184
}

func roundFloatProductToInt(v float64, factor *big.Int) *big.Int {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		panic("mtime: invalid floating value")
	}
	bf := new(big.Float).SetPrec(256).SetMode(big.ToNearestEven).SetFloat64(v)
	bFactor := new(big.Float).SetPrec(256).SetInt(factor)
	bf.Mul(bf, bFactor)
	out, _ := bf.Int(nil)
	frac := new(big.Float).Sub(bf, new(big.Float).SetInt(out))
	half := new(big.Float).SetFloat64(0.5)
	minusHalf := new(big.Float).SetFloat64(-0.5)
	if frac.Cmp(half) >= 0 {
		out.Add(out, bigOne)
	} else if frac.Cmp(minusHalf) <= 0 {
		out.Sub(out, bigOne)
	}
	return out
}

func modifiedJulianDate(at time.Time) float64 {
	unix := float64(at.Unix()) + float64(at.Nanosecond())/1e9
	return unix/86400.0 + 40587.0
}

func addInt64Checked(a int64, b int64) (int64, bool) {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		return 0, false
	}
	return a + b, true
}

func timeFromCalendar(year, month, day, hour, minute, second, millisecond, solOfYear int) (Time, error) {
	if year <= 0 {
		return Time{}, fmt.Errorf("%w: year must be positive", ErrInvalidFormat)
	}
	if month < 1 || month > len(monthLengths) {
		return Time{}, fmt.Errorf("%w: month out of range", ErrInvalidFormat)
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 || second < 0 || second > 59 || millisecond < 0 || millisecond > 999 {
		return Time{}, fmt.Errorf("%w: clock value out of range", ErrInvalidFormat)
	}

	leapYear := isLeapYear(year)
	maxDay := monthLengths[month-1]
	if leapYear && month == len(monthLengths) {
		maxDay++
	}
	if day < 1 || day > maxDay {
		return Time{}, fmt.Errorf("%w: day out of range", ErrInvalidFormat)
	}

	solOfYear0 := 0
	for i := 0; i < month-1; i++ {
		length := monthLengths[i]
		if leapYear && i == len(monthLengths)-1 {
			length++
		}
		solOfYear0 += length
	}
	solOfYear0 += day - 1
	computedSolOfYear := solOfYear0 + 1
	if solOfYear != 0 && solOfYear != computedSolOfYear {
		return Time{}, fmt.Errorf("%w: sol-of-year does not match date", ErrInvalidFormat)
	}

	millisOfDay := ((hour*60+minute)*60+second)*1000 + millisecond
	solNumber := solsBeforeYear(int64(year)) + int64(solOfYear0)
	msd := float64(solNumber) + float64(millisOfDay)/86400000.0
	return FromMSDSafe(msd)
}
