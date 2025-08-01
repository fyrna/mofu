package main

import (
	"net/http"

	"github.com/fyrna/mofu"
)

func main() {
	r := mofu.Miaw()

	r.GET("/", func(c *mofu.C) error {
		return c.SendText(200, "Hello World")
	})

	r.GET("/hello", func(c *mofu.C) error {
		return c.SendText(200, "hello")
	})

	r.GET("/h/:name", func(c *mofu.C) error {
		n := c.Param("name")

		if n == "fyrna" {
			data := map[string]string{
				"name":  "fyrna",
				"loves": "cute anime girl",
			}
			return c.SendJSON(200, data)
		}

		return c.SendText(200, "sorry, who is that?")
	})

	http.ListenAndServe(":8080", r)
}
