# retelimit by leaky bucket

## Install
> go get -u github.com/nelsonken/ratelimit-go

## Ability

```go

package limit

import "time"

type Limiter interface {
	// Take return can access or not immediately
	Take() bool

	// SpinTake spin while can't access with timeout
	SpinTake(timeout time.Duration) bool
}

```

## Naked Usage

```go
package main

import (
	"fmt"
	"github.com/nelsonken/ratelimit-go"
	"time"
)

func main() {
	rate := 8
	maxBurst := 2
	duration := 10 * time.Second

	// every 10 s only allow 8 times access with evenly divided time
	// after long time no access, The earliest 2 requests will be allowed at the same time
	limiter := ratelimit.New(rate, maxBurst, duration)
	
	// Take
	if limiter.Take() {
		fmt.Println("allowed")
	}else {
		fmt.Println("deny")
	}

	if limiter.SpinTake(1*time.Second) {
		fmt.Println("allowed")
	}else {
		fmt.Println("deny")
	}
}
```

## Usage in labstack echo middleware
```go
package main

import (
	"net/http"
	"time"
	"github.com/nelsonken/ratelimit-go"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	// 10 qps
	m := ratelimit.NewRateLimitMiddleware(10, 2, time.Second)
	e.Use(m)
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.Logger.Fatal(e.Start(":1323"))
}
```
