package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hitalos/send2hugo/api"
	"github.com/hitalos/send2hugo/auth"
	"github.com/hitalos/send2hugo/config"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func main() {
	configFile := flag.String("c", "send2hugo.json", "Configuration file")
	flag.Parse()
	configuration, err := config.New(*configFile)
	if err != nil {
		fmt.Printf("Error trying to load a configuration file: %q. Default values will be used.\n", err)
	}

	if err := os.Mkdir(configuration.ContentFolder, 0750); err != nil {
		if os.IsExist(err) {
			fileInfo, _ := os.Stat(configuration.ContentFolder)
			if !fileInfo.IsDir() {
				fmt.Printf("Error trying to create content folder. A file exists with name %q!", fileInfo.Name())
				os.Exit(1)
			}
		} else {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	fmt.Printf("The folder %q will be used for content\n", configuration.ContentFolder)

	e := echo.New()
	e.HideBanner = true
	e.Pre(middleware.RemoveTrailingSlash())
	e.Static("/", configuration.StaticFolder)

	g := e.Group("/api")
	if configuration.AuthEnabled() {
		e.POST("/login", auth.Login(configuration.Auth))
		g.Use(middleware.JWT([]byte(configuration.Auth.JwtSecret)))
	}
	api.Routes(g, configuration)

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", configuration.Port)))
}
