# mtime

Compact Go package for working with Martian time and date.

It stores instants in Earth UTC and exposes Mars Sol Date (MSD), Mars Coordinated Time (MTC), and a compact Martian calendar date.


**Install:**

go get github.com/sibexico/mtime@latest

**Import:**

import "github.com/sibexico/mtime"


## Example

```go
now := mtime.Now()
fmt.Println(now.String())
fmt.Println(now.Date())
fmt.Println(now.MTC())
```
