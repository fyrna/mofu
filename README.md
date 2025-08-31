# mofu
> simple net/http wrapper

mofu looks similiar to other framework:

```go
package main

import (
	"log"
	"net/http"

	"github.com/fyrna/mofu"
)

func main() {
	r := mofu.Miaw()

	r.GET("/", func(c *mofu.C) error {
		return c.SendText(http.StatusOK, "Hello World")
	})

	r.GET("/h/:name", func(c *mofu.C) error {
		n := c.Param("name")
		if n == "fyrna" {
			data := map[string]string{
				"name":  n,
				"loves": "cute anime girl",
			}
			return c.SendJSON(http.StatusOK, data)
		}
		return c.SendText(http.StatusOK, n+"? who are you? ")
	})

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal(err)
	}
}
```
