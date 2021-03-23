// Copyright 2021 Harran Ali <harran.m@gmail.com>. All rights reserved.
// Use of this source code is governed by MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gincoat/gincoat/core/cache"
	"github.com/gincoat/gincoat/core/database"
	"github.com/gincoat/gincoat/core/middlewaresengine"
	"github.com/gincoat/gincoat/core/pkgintegrator"
	"github.com/gincoat/gincoat/core/routing"
	"github.com/unrolled/secure"
)

// App struct
type App struct {
	Features *Features
}

// GORM is a const represents gorm variable name
const GORM = "gorm"

// CACHE a cache engine variable
const CACHE = "cache"

// logs file path
const logsFilePath = "logs/app.log"

// logs file
var logsFile *os.File

// New initiates the app struct
func New() *App {
	return &App{
		Features: &Features{},
	}
}

// SetEnv sets environment varialbes
func (app *App) SetEnv(env map[string]string) {
	for key, val := range env {
		os.Setenv(strings.TrimSpace(key), strings.TrimSpace(val))
	}
}

//Bootstrap initiate app
func (app *App) Bootstrap() {
	//initiate package integrator
	pkgintegrator.New()

	//initiate middlewares engine
	middlewaresengine.New()

	//initiate routing engine
	routing.New()

	//initiate db connection
	database.New()

	// initiate the cache
	cache.New()
}

// Run execute the app
func (app *App) Run(portNumber string) {
	// fallback to port number to 80 if not set
	if portNumber == "" {
		portNumber = "80"
	}

	logsFile, err := os.OpenFile(logsFilePath, os.O_CREATE|os.O_APPEND|os.O_CREATE, 644)
	if err != nil {
		panic(err)
	}
	defer logsFile.Close()
	gin.DefaultWriter = io.MultiWriter(logsFile, os.Stdout)

	//initiate gin engines
	httpGinEngine := gin.Default()
	httpsGinEngine := gin.Default()

	httpsOn, _ := strconv.ParseBool(os.Getenv("APP_HTTPS_ON"))
	redirectToHTTPS, _ := strconv.ParseBool(os.Getenv("APP_REDIRECT_HTTP_TO_HTTPS"))

	if httpsOn {
		//serve the https
		httpsGinEngine = app.IntegratePackages(httpsGinEngine, pkgintegrator.Resolve().GetIntegrations())
		router := routing.ResolveRouter()
		httpsGinEngine = app.registerRoutes(router, httpsGinEngine)
		certFile := os.Getenv("APP_HTTPS_CERT_FILE_PATH")
		keyFile := os.Getenv("APP_HTTPS_KEY_FILE_PATH")
		host := app.getHTTPSHost() + ":443"
		go httpsGinEngine.RunTLS(host, certFile, keyFile)
	}

	//redirect http to https
	if httpsOn && redirectToHTTPS {
		secureFunc := func() gin.HandlerFunc {
			return func(c *gin.Context) {
				secureMiddleware := secure.New(secure.Options{
					SSLRedirect: true,
					SSLHost:     app.getHTTPSHost() + ":443",
				})
				err := secureMiddleware.Process(c.Writer, c.Request)
				if err != nil {
					return
				}
				c.Next()
			}
		}()
		redirectEngine := gin.New()
		redirectEngine.Use(secureFunc)
		host := fmt.Sprintf("%s:%s", app.getHTTPHost(), portNumber)
		redirectEngine.Run(host)
	}

	//serve the http version
	httpGinEngine = app.IntegratePackages(httpGinEngine, pkgintegrator.Resolve().GetIntegrations())
	router := routing.ResolveRouter()
	httpGinEngine = app.registerRoutes(router, httpGinEngine)
	host := fmt.Sprintf("%s:%s", app.getHTTPHost(), portNumber)
	httpGinEngine.Run(host)
}

func (app *App) handleRoute(route routing.Route, ginEngine *gin.Engine) {
	switch route.Method {
	case "get":
		ginEngine.GET(route.Path, route.Handlers...)
	case "post":
		ginEngine.POST(route.Path, route.Handlers...)
	case "delete":
		ginEngine.DELETE(route.Path, route.Handlers...)
	case "patch":
		ginEngine.PATCH(route.Path, route.Handlers...)
	case "put":
		ginEngine.PUT(route.Path, route.Handlers...)
	case "options":
		ginEngine.OPTIONS(route.Path, route.Handlers...)
	case "head":
		ginEngine.HEAD(route.Path, route.Handlers...)
	}
}

func (app *App) SetAppMode(mode string) {
	if mode == gin.ReleaseMode || mode == gin.TestMode || mode == gin.DebugMode {
		gin.SetMode(mode)
	} else {
		gin.SetMode(gin.TestMode)
	}
}

func (app *App) IntegratePackages(engine *gin.Engine, handlerFuncs []gin.HandlerFunc) *gin.Engine {
	for _, pkgIntegration := range handlerFuncs {
		engine.Use(pkgIntegration)
	}

	return engine
}

//FeaturesControl to control what features to turn on or off
func (app *App) FeaturesControl(features *Features) {
	app.Features = features
}

func (app *App) useMiddlewares(engine *gin.Engine) *gin.Engine {
	for _, middleware := range middlewaresengine.Resolve().GetMiddlewares() {
		engine.Use(middleware)
	}

	return engine
}

func (app *App) registerRoutes(router *routing.Router, engine *gin.Engine) *gin.Engine {
	for _, route := range router.GetRoutes() {
		app.handleRoute(route, engine)
	}

	return engine
}

func (app *App) getHTTPSHost() string {
	host := os.Getenv("APP_HTTPS_HOST")
	//if not set get http instead
	if host == "" {
		host = os.Getenv("APP_HTTP_HOST")
	}
	//if both not set use local host
	if host == "" {
		host = "localhost"
	}
	return host
}

func (app *App) getHTTPHost() string {
	host := os.Getenv("APP_HTTP_HOST")
	//if both not set use local host
	if host == "" {
		host = "localhost"
	}
	return host
}
