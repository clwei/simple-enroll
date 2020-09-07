package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/clwei/simple-enroll/models"
	"github.com/flosch/pongo2"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo-contrib/session"
)

var (
	db *sqlx.DB
	e  *echo.Echo
)

type (
	// ControllInterface ...
	ControllInterface interface {
		RegisterRoutes(prefix string)
	}

	// Alert ...
	Alert struct {
		atype string
		msg   string
	}
)

// Alert Type
const (
	AlertNormal  = "uk-alert"
	AlertInfo    = "uk-alert-info"
	AlertDanger  = "uk-alert-danger"
	AlertSuccess = "uk-alert-success"
	AlertWarning = "uk-alert-warning"
)

// InitEchoInstance ...
func InitEchoInstance(ei *echo.Echo) {
	e = ei
}

// InitDBConnection ...
func InitDBConnection(connStr string) *sqlx.DB {
	var dbType string
	var err error
	fmt.Sscanf(connStr, "%s://", &dbType)
	db, err = sqlx.Connect("postgres", connStr)
	if err != nil {
		panic("Can not connect to database ==> " + connStr)
	}
	return db
}

func getSession(c echo.Context) (sess *sessions.Session) {
	sess, _ = session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}
	return sess
}

func addAlertFlash(c echo.Context, atype, msg string) error {
	sess := getSession(c)
	sess.AddFlash(Alert{atype, msg})
	return sess.Save(c.Request(), c.Response())
}

// AddAlertFlash ...
func AddAlertFlash(c echo.Context, atype, msg string) {
	addAlertFlash(c, atype, msg)
}

func setSessionValue(c echo.Context, key string, val interface{}) error {
	sess := getSession(c)
	sess.Values[key] = val
	return sess.Save(c.Request(), c.Response())
}

func getSessionValue(c echo.Context, key string, defval interface{}) (val interface{}) {
	sess := getSession(c)
	var ok bool
	if val, ok = sess.Values[key]; !ok {
		val = defval
	}
	return val
}

func getCurrUser(c echo.Context) (user models.User) {
	return getSessionValue(c, "user", models.User{}).(models.User)
}

func requirePermission(c echo.Context, isStaff, isAdmin bool) (ok bool) {
	user := getCurrUser(c)
	if user.IsAdmin {
		return true
	}
	ok = true
	if isAdmin {
		ok = user.IsAdmin
	}
	if isStaff {
		ok = ok && user.IsStaff
	}
	if !ok {
		addAlertFlash(c, AlertDanger, "權限不足！")
	}
	return ok
}

func requireStaff(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !requirePermission(c, true, false) {
			return c.Render(http.StatusOK, "base.html", pongo2.Context{})
		}
		if c.Get("task") == nil {
			c.Set("task", models.Task{})
		}
		return next(c)
	}
}

func requireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !requirePermission(c, false, true) {
			return c.Render(http.StatusOK, "base.html", pongo2.Context{})
		}
		return next(c)
	}
}

func requireValidTaskID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		task := (&TaskController{}).getTaskByParam(c, "tid")
		if task.ID == 0 {
			return c.Render(http.StatusNotFound, "base.html", pongo2.Context{})
		}
		c.Set("task", task)
		return next(c)
	}
}

func requireValidTime(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := getSessionValue(c, "user", models.User{}).(models.User)
		task := c.Get("task").(models.Task)
		currTime := time.Now()
		if !user.IsAdmin && !user.IsStaff && (currTime.Before(task.Tstart) || currTime.After(task.Tend)) {
			addAlertFlash(c, AlertDanger, "目前非選課時間！！")
			return c.Render(http.StatusForbidden, "base.html", pongo2.Context{})
		}
		return next(c)
	}
}
