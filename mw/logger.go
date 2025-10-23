package mw

import (
	"fmt"
	"time"

	"github.com/fyrna/mofu"
)

func Logger() mofu.Middleware {
	return mofu.MwFunc(func(c *mofu.C) error {
		start := time.Now()
		dur := time.Since(start)

		status := 200
		if rw, ok := c.Writer.(interface{ Status() int }); ok {
			status = rw.Status()
		}

		fmt.Printf("[RAWR] %d %s %s (%s nyaa~)\n",
			status,
			c.Request.Method,
			c.Request.URL.Path,
			dur.Round(time.Millisecond))

		return c.Next()
	})
}
