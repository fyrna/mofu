package mw

import (
	"fmt"
	"net/http"
	"time"

	"github.com/fyrna/mofu"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func Logger() mofu.Middleware {
	return mofu.MwHug(func(c *mofu.C) error {
		start := time.Now()

		recorder := &statusRecorder{ResponseWriter: c.Writer, status: 200}
		c.Writer = recorder

		err := c.Next()
		dur := time.Since(start)

		var duration string
		switch {
		case dur < time.Microsecond:
			duration = fmt.Sprintf("%dns", dur.Nanoseconds())
		case dur < time.Millisecond:
			duration = fmt.Sprintf("%.2fµs", float64(dur.Microseconds()))
		case dur < time.Second:
			duration = fmt.Sprintf("%.2fms", float64(dur.Milliseconds()))
		default:
			duration = fmt.Sprintf("%.2fs", dur.Seconds())
		}

		fmt.Printf("[MOFU] %d %s %s (%s nyaa~)\n",
			recorder.status,
			c.Request.Method,
			c.Request.URL.Path,
			duration)

		return err
	})
}
