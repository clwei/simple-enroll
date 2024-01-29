package main

import (
	"fmt"
	"net/http"

	"github.com/clwei/simple-enroll/controllers"
	"github.com/flosch/pongo2/v6"
	"github.com/labstack/echo/v4"
)

func customHTTPErrorHandler(err error, c echo.Context) {
	he, ok := err.(*echo.HTTPError)
	if ok {
		if he.Internal != nil {
			err = fmt.Errorf("%v, %v", err, he.Internal)
		}
	} else {
		he = &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: http.StatusText(http.StatusInternalServerError),
		}
	}

	// Send response
	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead { // Issue #608
			err = c.NoContent(he.Code)
		} else {
			message, _ := he.Message.(string)
			msg := fmt.Sprintf("噢噢！系統發生了一點錯誤！\n代碼：%d\n訊息：%s", he.Code, message)
			controllers.AddAlertFlash(c, controllers.AlertDanger, msg)
			err = c.Render(he.Code, "error.html", pongo2.Context{})
		}
	}
}
