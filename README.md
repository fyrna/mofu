# mofu
> simple net/http wrapper

mofu is a simple http micro-framework, basically it is an experiment to learn and create my own framework.

```go
package main

import "github.com/fyrna/mofu"

func main() {
  m := mofu.Miaw()

  m.Use(mofu.MwHug(func(c *mofu.C) error {
    c.SetHeader("X-Powered-By", "Mofu")
    return c.Next()
  }))

  m.GET("/", func(c *mofu.C) error {
    return c.String(200, "Hello World")
  })

  m.GET("/users/:id", func(c *mofu.C) error {
    id := c.Param("id")
    return c.OK(map[string]string{"user_id": id})
  })

  api := r.Group("/api")
  api.Use(authMiddleware)

  api.GET("/data", func(c *mofu.C) error {
    return c.OK("Protected data")
  })

  m.Start(":8080")
}

func authMiddleware() mofu.Middleware {
  return mofu.MwHug(func(c *mofu.C) error {
    if c.GetHeader("Authorization") == "" {
      return c.Error(401, "Unauthorized")
    }
    return c.Next()
  })
}
```
