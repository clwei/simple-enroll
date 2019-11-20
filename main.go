package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"

	"github.com/clwei/simple-enroll/controllers"
	"github.com/clwei/simple-enroll/models"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

func initDB() {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	viper.SetConfigName("config")
	viper.AddConfigPath(dir)
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		var defaultConfig = []byte(`
app:
	port: 1323
db:
	type: 'postgres'
	host: 'localhost'
	port: 5432
	user: 'simpleenroll'
	passwd: 'simpleenroll'
	database: 'enroll'
`)
		viper.ReadConfig(bytes.NewBuffer(defaultConfig))
	}
	connStr := fmt.Sprintf("%s://%s:%s@%s:%d/%s",
		viper.GetString("db.type"),
		viper.GetString("db.user"),
		viper.GetString("db.passwd"),
		viper.GetString("db.host"),
		viper.GetInt("db.port"),
		viper.GetString("db.database"),
	)
	controllers.InitDBConnection(connStr)
}

func main() {
	initDB()

	gob.Register(models.User{})

	e := echo.New()
	e.Renderer = newRenderer(true)
	e.HTTPErrorHandler = customHTTPErrorHandler
	//e.Use(middleware.Logger())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339} ${status} ${method} ${uri} - ${remote_ip}\n",
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
	}))
	e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup: "form:csrfmiddlewaretoken",
	}))
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("super-secret-secret"))))
	// Static Assets
	e.Static("/static", "static")

	registerControllerRoutes(e)
	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "task/")
	})
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", viper.GetInt("app.port"))))
}
