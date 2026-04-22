![Go Version](https://img.shields.io/badge/Go-1.26-blue?labelColor=gray&logo=go)
 [![Go Report Card](https://goreportcard.com/badge/github.com/sibexico/mtime)](https://goreportcard.com/report/github.com/sibexico/mtime)
 [![Support Me](https://img.shields.io/badge/Support-Me-darkgreen?labelColor=black&logo=data:image/svg%2Bxml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI%2BPHBhdGggZmlsbD0iI0ZGRiIgZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik0xMiAxQzUuOTI1IDEgMSA1LjkyNSAxIDEyczQuOTI1IDExIDExIDExIDExLTQuOTI1IDExLTExUzE4LjA3NSAxIDEyIDF6bTAgNGwyLjUgNi41SDIxbC01LjUgNCAyIDYuNUwxMiAxNy41IDYgMjJsMi02LjUtNS41LTRoNi41TDEyIDV6Ii8%2BPC9zdmc%2B)](https://sibexi.co/support)

[![Tests passed](https://img.shields.io/badge/Tests-Passed-green?labelColor=gray&logo=github)](https://github.com/sibexico/mtime/actions/runs/24762391838)
 [![Tests coverage](https://img.shields.io/badge/Tests%20Coverage-86.2%25-green?labelColor=gray&logo=gitextensions)](https://github.com/sibexico/mtime/actions/runs/24762391838)
 


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


### Work with a known MSD value

```go
mt := mtime.FromMSD(54000.25)

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

### Safe sol addition (no panic)

If input can be untrusted or large, use AddSolsSafe then.

```go
result, err := mtime.Now().AddSolsSafe(userInputSols)
if err != nil {
	log.Printf("bad sols value: %v", err)
	return
}
fmt.Println(result)
```


### Custom TT-UTC data

```go
mtime.SetTTMinusUTCProvider(func(at time.Time) float64 {
	// Example: replace with your value.
	return 69.184
})

fmt.Println(mtime.TTMinusUTC(time.Now().UTC()))
```
