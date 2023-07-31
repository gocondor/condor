// Copyright 2023 Harran Ali <harran.m@gmail.com>. All rights reserved.
// Use of this source code is governed by MIT-style
// license that can be found in the LICENSE file.

package middlewares

import (
	"fmt"

	"github.com/gocondor/core"
)

// Example middleware
var ExampleMiddleware core.Handler = func(c *core.Context) *core.Response {
	fmt.Println("example middleware!")
	c.Next()
	return nil
}
