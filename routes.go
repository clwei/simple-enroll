package main

import (
	"github.com/clwei/simple-enroll/controllers"
	"github.com/labstack/echo/v4"
)

type routeGroup struct {
	c controllers.ControllInterface
	p string
}

var routeGroups = []routeGroup{
	{&controllers.TaskController{}, "/task/"},
	{&controllers.UserController{}, "/user/"},
}

func registerControllerRoutes(e *echo.Echo) {
	controllers.InitEchoInstance(e)
	for _, group := range routeGroups {
		group.c.RegisterRoutes(group.p)
	}
}
