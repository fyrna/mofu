package mw

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/fyrna/mofu"
)

func Recover() mofu.Middleware {
	return mofu.MwHug(func(c *mofu.C) (err error) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("mofu panic recovered nya~: %v\n%s\n", r, debug.Stack())
				c.String(http.StatusInternalServerError,
					"Internal Server Error 💥 meow~")
			}
		}()
		return c.Next()
	})
}
