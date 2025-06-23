package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	ginI18n "github.com/gin-contrib/i18n"
	"golang.org/x/text/language"
	"gopkg.in/natefinch/lumberjack.v2"
	"ired.com/callcenter/app"
	"ired.com/callcenter/controllers"
	"ired.com/callcenter/middlewares"

	"github.com/gin-gonic/gin"
)

func init() {
	app.LoadEnvVariables()
	app.InitDbPgsql()
	app.InitDbMysql()
	app.LoadCrontab()

	gin.SetMode(os.Getenv("GIN_MODE"))

	// create log file and handle logrotate also
	logfile := &lumberjack.Logger{
		Filename:   "logs/main.log",
		MaxSize:    100,  // megabytes
		MaxBackups: 30,   // Keep logs for a month
		Compress:   true, // disabled by default
	}
	log.SetOutput(logfile)

	//manual function to rotate logs daily
	go func() {
		for range time.Tick(24 * time.Hour) {
			logfile.Rotate()
		}
	}()
}

// @Title								CallCenter Service API
// @Version							1.0
// @Description 				service in Go using Gin framework
// @Host								127.0.0.1:7006
// @Contact.name   			Juan Acosta
// @Contact.url			    https://www.linkedin.com/in/juan-m-acosta-f-54219758/
// @Contact.email  			juan9acosta@gmail.com
// @securityDefinitions.basic BasicAuth
// @securityDefinitions.basic.description Basic Authentication
// @BasePath /
func main() {
	r := gin.Default()

	// aply startTimer middleware
	r.Use(middlewares.StartTimer())

	// apply custom logger middleware
	r.Use(middlewares.RequestLogger())

	// recover if panic and log the fail
	r.Use(gin.RecoveryWithWriter(log.Writer()))

	// apply i18n middleware
	r.Use(
		ginI18n.Localize(ginI18n.WithBundle(&ginI18n.BundleCfg{
			RootPath:         "./i18n",
			AcceptLanguage:   []language.Tag{language.English, language.Spanish},
			DefaultLanguage:  language.Spanish,
			UnmarshalFunc:    json.Unmarshal,
			FormatBundleFile: "json",
		})),
	)

	// load templates
	r.LoadHTMLGlob("templates/*")

	// load static files
	r.Static("/public", "./public")

	// manual routes
	controllers.CronRoutes(r)
	controllers.GrafanaRoutes(r)
	controllers.AmiRoutes(r)

	// load docs
	controllers.SwaggerRoutes(r)

	// handle 404 error with custom template
	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "404.tmpl", gin.H{
			"title": "Page Not Found",
		})
	})

	// run server default port 8080 or lookup .env file
	r.Run()

	// this is if we need to use https
	// if os.Getenv("PROTOCOL") == "https" {
	// 	err := r.RunTLS(":"+os.Getenv("PORT"), "server-cert.pem", "server-key.pem")
	// 	if err != nil {
	// 		utils.Fatalf("Failed to start HTTPS server: ", err)
	// 	}
	// } else {
	// 	r.Run()
	// }

	defer app.CloseDbPgsql()
	defer app.InitDbMysql()
}
