package controllers

import (
	"fmt"
	"net/http"

	"github.com/clwei/simple-enroll/models"
	"github.com/flosch/pongo2"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
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
	if val, ok = sess.Values["key"]; !ok {
		val = defval
	}
	return val
}

func requirePermission(c echo.Context, isStaff, isAdmin bool) (ok bool) {
	sess := getSession(c)
	tuser, ok := sess.Values["user"]
	if !ok {
		tuser = models.User{}
	}
	var user models.User
	if user, ok = tuser.(models.User); !ok {
		fmt.Println("**type assertion error**", ok)
		user = models.User{}
	}
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

/*
func importUserFromString(raw string) {
	fmt.Println("** importUserFromString **")
	lines := strings.Split(raw, "\n")
	sql1 := `
		INSERT INTO public.user(username, passwd, cno, seat, name, is_staff, is_admin)
			VALUES`
	sql2 := `ON CONFLICT(username)
				DO UPDATE
					SET (passwd, cno, seat, name) = (EXCLUDED.passwd, EXCLUDED.cno, EXCLUDED.seat, EXCLUDED.name)`
	values := []string{}
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) < 5 || fields[0] == "學號" {
			continue
		}
		// fmt.Println(fields)
		var hash []byte
		hash, _ = bcrypt.GenerateFromPassword([]byte(fields[4]), bcrypt.DefaultCost)
		values = append(values, fmt.Sprintf("('%s', '%s', %s, %s, '%s', false, false)", fields[0], string(hash), fields[1], fields[2], fields[3]))
	}
	sql := sql1 + strings.Join(values, ",") + sql2
	result, err := db.NamedExec(sql, models.User{})
	fmt.Println("\tdb.Exec err =", err, ", result =", result)
}
*/
