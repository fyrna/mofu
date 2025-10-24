# mofu
> simple net/http wrapper

mofu is a simple http micro-framework, basically it is an experiment to learn and create my own framework.

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/fyrna/mofu"
	"github.com/fyrna/mofu/mw"
)

func main() {
	m := mofu.Miaw()

	m.Use(mw.Logger())

	m.GET("/", func(c *mofu.C) error {
		return c.String(http.StatusOK, "Nyaa~ welcome to Mofu server 💞")
	})

	m.GET("/about", func(c *mofu.C) error {
		return c.JSON(http.StatusOK, map[string]any{
			"name":    "Fyrna",
			"project": "Mofu Framework",
			"version": "0.1.0",
			"cute":    true,
		})
	})

	api := m.Group("api")

	api.GET("/", func(c *mofu.C) error {
		return c.HTML(http.StatusOK, `<html>
		<head><title>Mofu API</title></head>
		<body>
			<h1>Hello nya~ 🐾</h1>
			<p>This API has /api/about and /api/hello/:name</p>
		</body>
		</html>`)
	})

	api.GET("/about", func(c *mofu.C) error {
		return c.JSON(http.StatusOK, map[string]any{
			"library": "Mofu",
			"author":  "Fyrna",
			"desc":    "A cute and simple http micro-framework for Go",
		})
	})

	api.GET("/hello/:name", func(c *mofu.C) error {
		name := c.Param("name")
		if name == "" {
			name = "stranger"
		}
		return c.JSON(http.StatusOK, map[string]any{
			"message": fmt.Sprintf("Nyaa~ hello %s!", name),
			"status":  "ok",
		})
	})

	if err := m.Start(":8000"); err != nil {
		fmt.Println("Error:", err)
	}
}
```
