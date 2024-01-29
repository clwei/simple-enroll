package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/clwei/simple-enroll/models"
	"github.com/flosch/pongo2/v6"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Renderer ...
type Renderer struct {
	Debug bool
}

var _viewPrefix = "views/"

// Render ...
func (r Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	var ctx pongo2.Context

	if data != nil {
		var ok bool
		ctx, ok = data.(pongo2.Context)

		if !ok {
			return errors.New("no pongo2.Context data was passed")
		}
	}

	var t *pongo2.Template
	var err error

	if r.Debug {
		t, err = pongo2.FromFile(_viewPrefix + name)
	} else {
		t, err = pongo2.FromCache(_viewPrefix + name)
	}

	// Add some static values
	// ctx["version_number"] = "v0.0.1-beta"
	if csrf, ok := c.Get(middleware.DefaultCSRFConfig.ContextKey).(string); ok {
		ctx["csrf_token"] = csrf
		ctx["csrf_token_input"] = fmt.Sprintf("<input type=\"hidden\" name=\"csrfmiddlewaretoken\" value=\"%s\" />", csrf)
	}

	sess, _ := session.Get("session", c)
	user, ok := sess.Values["user"]
	if !ok {
		user = models.User{}
		sess.Values["user"] = user
		sess.Save(c.Request(), c.Response())
	}
	// for _, talert := range talerts {
	// 	var alert controllers.Alert{}
	// 	if alert, ok := talert.(controllers.Alert); !ok {
	// 		alert.atype = controllers.AlertNormal
	// 		alert.msg = talert
	// 	}
	// 	alerts = append(alerts, alert)
	// }

	ctx["alerts"] = sess.Flashes()
	// sess.Save(c.Request(), c.Response())
	ctx["cuser"] = user

	ctx["static"] = "/static"

	if err != nil {
		return err
	}

	return t.ExecuteWriter(ctx, w)
}

func newRenderer(debug bool) Renderer {
	return Renderer{Debug: debug}
}
