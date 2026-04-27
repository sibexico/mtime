![Go Version](https://img.shields.io/badge/Go-1.26-blue?labelColor=gray&logo=go)
 [![Go Report Card](https://goreportcard.com/badge/github.com/sibexico/mtime)](https://goreportcard.com/report/github.com/sibexico/mtime)
 [![Support Me](https://img.shields.io/badge/Support-Me-darkgreen?labelColor=black&logo=data:image/svg%2Bxml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI%2BPHBhdGggZmlsbD0iI0ZGRiIgZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0xMiAxQzUuOTI1IDEgMSA1LjkyNSAxIDEyczQuOTI1IDExIDExIDExIDExLTQuOTI1IDExLTExUzE4LjA3NSAxIDEyIDF6bTAgNGwyLjUgNi41SDIxbC01LjUgNCAyIDYuNUwxMiAxNy41IDYgMjJsMi02LjUtNS41LTRoNi41TDEyIDV6Ii8%2BPC9zdmc%2B)](https://sibexi.co/support)

![Made for Mars](https://img.shields.io/badge/MCR-Made_For_Mars-E13800?logo=data:image/webp;base64,UklGRoYCAABXRUJQVlA4IHoCAACwDwCdASpAACkAPm0qkkYkIiGhLVZtsIANiWwAxQfofAfiB+KvSXcV98OABOpWA/qv2AfQDbAeZPz1ekA/pP9m6yX0APLK/Yj4Nv22ceOMDsBra+Fj0qfAA0FKZxJY2UQf1ngl0oZCLbEU8m6uikFtbtwEDp4aFtLQK/y9OTLuUBdhiNgwAAD+5xk6FByL97XaROTxptveSTqSY91/pLqPNQp//e7Lj9N/xaSMp6eb7sX3//7Riwq7QJ5x7UPTRUHc1u8fuK7kGBkCxsWeDAUZ6vZZirb9Cr/5LKsfgQgaBhuWmwMr1y3tyGlRt8AJXZKwy6d9qMsbYH800QZSnNdfj6bq808cPCQYP2UrEmn1V//HX/MjS+Tx2KYautjtTz+TUBcuPjJ1ahL625UtaaS6u9ijRnDiXikZbfg3BHi53zM6mcns1zrZlkc4NKeo1++deY1KdNE5RnXZHOMO4auDr5BTjUJxYEN5SMaT9ThY/23/01Zmgwy6oqQiPsrN7Q1OXfMFk7oqplBBwYwn8ZZcSMep3X50KJbQa0oa5PbVRuq+fTwvCSBE2+1+nnidGlzvpEHX9BFexjxA6p0SFAxRPdrIuq7flZAwvCLzDVOc4wNY0si7qClAA9j7lSS0sBP9ubZci/lkWQxI8Fr36eBIxsX+9Bumn980H+hsyfsm5qytesA2WupGLz+PQdRVmSgt6koXa5XLifO6+o/MjRdVnpTQWF9mLdxrIh2n6FuaqOzvm3FJ1fB248UkJc3B1aPmjfM3JUA7brdEBjV/ruu2qPm/d6qcbCU6GjSoF3nKwfghkPKXKnvSXVl3Jtu017n98n17hWCQwAAA)

[![Tests passed](https://img.shields.io/badge/Tests-Passed-green?labelColor=gray&logo=github)](https://github.com/sibexico/mtime/actions/runs/24825088774)
 [![Tests coverage](https://img.shields.io/badge/Tests%20Coverage-86.0%25-green?labelColor=gray&logo=gitextensions)](https://github.com/sibexico/mtime/actions/runs/24825088774)
 


# mtime

Compact Go package for working with Martian time and date.

It stores instants in Earth UTC and exposes Mars Sol Date (MSD), Mars Coordinated Time (MTC), and a compact Martian calendar date.

Use it when you want Earth/Mars conversions but still keep familiar time-style operations.

**Note about leap seconds:** the built-in leap-second table currently includes data through 2017. If a new leap second is announced, update the table in code or show your own via SetTTMinusUTCProvider.


**Install:**

go get github.com/sibexico/mtime@latest

**Import:**

import "github.com/sibexico/mtime"


## Examples

### Current Martian date and clock

```go
now := mtime.Now()
fmt.Println(now.String())
fmt.Println(now.Date())
fmt.Println(now.MTC())
fmt.Println(mtime.TTMinusUTC(now.Earth()))
```


### Convert an Earth timestamp to Mars time

```go
earth := time.Date(2026, 4, 20, 15, 30, 0, 0, time.UTC)
mt := mtime.FromEarth(earth)

fmt.Println("Earth:", mt.Earth())
fmt.Println("MSD:", mt.MSD())
fmt.Println("MTC:", mt.MTC())
fmt.Println("Date:", mt.Date())
```


### Work with a known MSD value (safe)

```go
msd := 54000.25
mt, err := mtime.FromMSDSafe(msd)
if err != nil {
	log.Fatalf("invalid MSD: %v", err)
}

fmt.Println("Earth:", mt.Earth())
fmt.Println("Mars:", mt)
```


### Add sols and compare instants

```go
start := mtime.Now()
landingWindow := start.AddSols(12.5)

fmt.Println("window:", landingWindow)
fmt.Println("sols from start:", landingWindow.DiffSols(start))
fmt.Println("duration:", landingWindow.Sub(start))
fmt.Println("after start:", landingWindow.After(start))
```

### Safe sol addition with typed error checks

```go
result, err := mtime.Now().AddSolsSafe(userInputSols)
if err != nil {
	if errors.Is(err, mtime.ErrInvalidSols) {
		log.Printf("invalid sols input: %v", err)
		return
	}
	if errors.Is(err, mtime.ErrOutOfRange) {
		log.Printf("sols result out of range: %v", err)
		return
	}
	log.Printf("unexpected error: %v", err)
	return
}
fmt.Println(result)
```

### Parse and format Martian timestamps

```go
// Use millisecond-aligned input because fff token has millisecond precision.
base := mtime.FromEarth(time.Date(2026, 4, 18, 12, 34, 56, 789000000, time.UTC))

// Fixed default format used by Time.String().
raw := base.String()
parsed, err := mtime.ParseDefault(raw)
if err != nil {
	log.Fatal(err)
}

// Custom format tokens: MY MM DD SSS hh mm ss fff
layout := "MY-MM-DD SSS hh:mm:ss.fff"
custom := base.Format(layout)
parsedCustom, err := mtime.Parse(layout, custom)
if err != nil {
	log.Fatal(err)
}

fmt.Println(raw)
fmt.Println(parsed.Equal(base), parsedCustom.Equal(base))
```

### JSON round-trip with UTC nanoseconds

```go
type payload struct {
	At mtime.Time `json:"at"`
}

in := payload{At: mtime.Now()}
b, err := json.Marshal(in)
if err != nil {
	log.Fatal(err)
}

var out payload
if err := json.Unmarshal(b, &out); err != nil {
	log.Fatal(err)
}

fmt.Println(string(b)) // {"at":{"utc_ns":...}}
fmt.Println(out.At.Equal(in.At))
```


### Custom TT-UTC data

```go
mtime.SetTTMinusUTCProvider(func(at time.Time) float64 {
	// Example: replace with your value.
	return 69.184
})

fmt.Println(mtime.TTMinusUTC(time.Now().UTC()))
```
