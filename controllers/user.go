package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/clwei/simple-enroll/models"
	"github.com/flosch/pongo2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo-contrib/session"
	"golang.org/x/crypto/bcrypt"
)

// UserController 使用者
type UserController struct {
	ControllInterface
}

// RegisterRoutes ...
func (u *UserController) RegisterRoutes(prefix string) {
	g := e.Group(prefix)
	g.GET("", u.userList, requireAdmin)
	g.GET("create/", u.userForm, requireAdmin)
	g.POST("create/", u.userFormSubmit, requireAdmin)
	g.GET(":uid/edit/", u.userForm, requireAdmin)
	g.POST(":uid/edit/", u.userFormSubmit, requireAdmin)
	g.GET("login/", u.userLogin)
	g.POST("login/", u.userLogin)
	g.GET("logout/", u.userLogout)
}

func (u *UserController) userList(c echo.Context) error {
	sess, err := session.Get("session", c)
	tuser, ok := sess.Values["user"]
	if !ok {
		tuser = models.User{}
	}
	fmt.Println("!! session !! user =", tuser)
	users := []models.User{}
	if err = db.Select(&users, "SELECT * FROM public.user ORDER BY id"); err != nil {
		fmt.Println(err)
	}
	data := pongo2.Context{
		"users": users,
	}
	return c.Render(http.StatusOK, "user/index.html", data)
}

func (u *UserController) userForm(c echo.Context) error {
	classOptions := []int{
		0, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110,
		201, 202, 203, 204, 205, 206, 207, 208, 209, 210,
		301, 302, 303, 304, 305, 306, 307, 308, 309, 310,
	}
	seatOptions := []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
		21, 22, 23, 24, 25, 26, 27, 28, 29, 30,
		31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
		41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
	}
	user := models.User{}
	uidParam := c.Param("uid")
	opTitle := "新增使用者"
	if uid, err := strconv.Atoi(uidParam); err != nil {

	} else if err = db.Get(&user, "SELECT * FROM public.user WHERE id = $1", uid); err != nil {
		fmt.Println("**Select Error**", err)
	} else {
		opTitle = "修改使用者"
	}
	fmt.Println(user)
	data := pongo2.Context{
		"title":   opTitle,
		"user":    user,
		"classes": classOptions,
		"seats":   seatOptions,
	}
	return c.Render(http.StatusOK, "user/form.html", data)
}

func (u *UserController) userFormSubmit(c echo.Context) (err error) {
	user := new(models.User)
	c.Bind(user)
	var hash []byte
	if user.Passwd != "" {
		hash, _ = bcrypt.GenerateFromPassword([]byte(user.Passwd), bcrypt.DefaultCost)
		user.Passwd = string(hash)
	}
	var sql string
	if user.ID > 0 {
		if len(hash) > 0 {
			sql = `UPDATE public.user SET(username, passwd, cno, seat, name, is_staff, is_admin) = (:username, :passwd, :cno, :seat, :name, :is_staff, :is_admin) WHERE id = :id`
		} else {
			sql = `UPDATE public.user SET(username, cno, seat, name, is_staff, is_admin) = (:username, :cno, :seat, :name, :is_staff, :is_admin) WHERE id = :id`
		}
	} else {
		sql = `INSERT INTO public.user(username, passwd, cno, seat, name, is_staff, is_admin) VALUES(:username, :passwd, :cno, :seat, :name, :is_staff, :is_admin)`
	}
	if _, err := db.NamedExec(sql, user); err != nil {
		fmt.Println("**db error**", err)
	}
	fmt.Println("**userFormSubmit**", user)
	return c.Redirect(http.StatusSeeOther, "/user/")
}

func (u *UserController) userLogin(c echo.Context) (err error) {
	req := c.Request()
	if req.Method == http.MethodGet {
		return c.Render(http.StatusOK, "user/login.html", pongo2.Context{})
	}
	tuser := models.User{}
	user := models.User{}
	c.Bind(&tuser)
	if err = db.Get(&user, "SELECT * FROM public.user WHERE username = $1 LIMIT 1", tuser.Username); err != nil {
	} else if err = bcrypt.CompareHashAndPassword([]byte(user.Passwd), []byte(tuser.Passwd)); err != nil {
	}
	if err != nil {
		addAlertFlash(c, AlertDanger, "帳號或密碼錯誤！請重新登入！")
	} else if user.IsStaff == false && user.IsAdmin == false {
		addAlertFlash(c, AlertDanger, "權限不足！非行政人員與管理員無法登入後臺！")
	} else {
		setSessionValue(c, "user", user)
		return c.Redirect(http.StatusSeeOther, "/task/")
	}
	return c.Render(http.StatusOK, "user/login.html", pongo2.Context{})
}

func (u *UserController) userLogout(c echo.Context) (err error) {
	setSessionValue(c, "user", models.User{})
	return c.Redirect(http.StatusSeeOther, "/task/")
}
