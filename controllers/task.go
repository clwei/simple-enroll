package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/clwei/simple-enroll/models"
	"github.com/flosch/pongo2"
	"github.com/labstack/echo"
)

// TaskController 選課任務
type TaskController struct {
	ControllInterface
}

// RegisterRoutes ...
func (t *TaskController) RegisterRoutes(prefix string) {
	g := e.Group(prefix)
	g.GET("", t.taskList).Name = "taskList"
	g.GET(":tid/", t.taskLogin)
	g.POST(":tid/", t.taskSubmit)
	g.GET("create/", t.taskForm, requireStaff)
	g.POST("create/", t.taskFormSubmit, requireStaff)
	g.GET(":tid/edit/", t.taskForm, requireStaff)
	g.POST(":tid/edit/", t.taskFormSubmit, requireStaff)
	g.GET(":tid/delete/", t.taskDelete, requireStaff)
	g.POST(":tid/delete/", t.taskDelete, requireStaff)
}

//
func (t *TaskController) taskList(c echo.Context) error {
	tasks := []models.Task{}
	db.Select(&tasks, "SELECT * FROM task ORDER BY tstart desc")
	data := pongo2.Context{
		"tasks": tasks,
	}
	return c.Render(http.StatusOK, "task/index.html", data)
}

func (t *TaskController) getTaskByParam(c echo.Context, param string) (task models.Task) {
	tidParam := c.Param(param)
	if tid, err := strconv.Atoi(tidParam); err != nil {
		// fmt.Println("tidParam =", tidParam)
		if tidParam != "" {
			addAlertFlash(c, AlertDanger, "參數型態錯誤！")
		}
	} else if err = db.Get(&task, fmt.Sprintf("SELECT * FROM task WHERE id=%d", tid)); err != nil {
		addAlertFlash(c, AlertDanger, "無此選課任務！！")
	}
	return task
}

//
func (t *TaskController) taskLogin(c echo.Context) error {
	task := t.getTaskByParam(c, "tid")
	data := pongo2.Context{
		"task": task,
	}
	return c.Render(http.StatusOK, "task/login.html", data)
}

func (t *TaskController) taskSubmit(c echo.Context) error {
	task := t.getTaskByParam(c, "tid")
	username := strings.TrimSpace(c.FormValue("username"))
	selection := strings.TrimSpace(c.FormValue("selection"))
	// fmt.Println("** taskSubmit ** selection =", selection)
	if selection != "" {
		enrollment := models.Enrollment{
			Tid:       task.ID,
			Sid:       username,
			Selection: selection,
		}
		sql := `INSERT INTO public.enrollment(tid, sid, selection) 
					VALUES(:tid, :sid, :selection)
					ON CONFLICT(tid, sid) DO UPDATE
					SET selection = EXCLUDED.selection`
		if _, err := db.NamedExec(sql, enrollment); err != nil {
			// fmt.Println("** taskSubmit ** insert error =", err, "result =", result)
		}
		return c.Redirect(http.StatusSeeOther, "/task/")
	}
	passwd := strings.TrimSpace(c.FormValue("passwd"))

	lines := strings.Split(task.Students, "\r\n")
	found := false
	stu := map[string]interface{}{}
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if username == strings.TrimSpace(fields[0]) && passwd == strings.TrimSpace(fields[4]) {
			found = true
			stu["sid"] = fields[0]
			stu["cno"] = fields[1]
			stu["seat"] = fields[2]
			stu["name"] = fields[3]
			break
		}
	}
	data := pongo2.Context{
		"task": task,
	}
	if found {
		enrollment := models.Enrollment{}
		if err := db.Get(&enrollment, "SELECT * FROM enrollment WHERE tid = $1 AND sid = $2", task.ID, username); err != nil {
		}
		// scourses: 學生已選課程
		var scourses []string
		if enrollment.Selection == "" {
			scourses = []string{}
		} else {
			scourses = strings.Split(enrollment.Selection, ",")
		}
		smap := map[string]bool{}
		for _, course := range scourses {
			smap[course] = true
		}
		// courses: 學生可選課程(去除已選課程)
		lines = strings.Split(task.Courses, "\r\n")
		courses := []string{}
		for _, line := range lines {
			fields := strings.Split(line, "\t")
			if _, ok := smap[fields[0]]; !ok {
				courses = append(courses, fields[0])
			}
		}
		data["courses"] = courses
		data["stu"] = stu
		data["username"] = username
		data["selection"] = enrollment.Selection
		data["scourses"] = scourses
		return c.Render(http.StatusOK, "task/enroll.html", data)
	}
	addAlertFlash(c, AlertDanger, "學號或身分證號有誤！請重新登入！")
	return c.Render(http.StatusOK, "task/login.html", data)
}

//---------------------------------------------------------------------------
// 以下需行政人員或管理員權限
//---------------------------------------------------------------------------

func (t *TaskController) taskForm(c echo.Context) error {
	task := t.getTaskByParam(c, "tid")
	opTitle := "新增選課任務"
	if task.ID > 0 {
		opTitle = "修改選課任務"
	} else {
		// task.ID == 0 => 新增選課任務，預先填入選課起迄時間
		task.Tstart = time.Now()
		task.Tend = task.Tstart.AddDate(0, 0, 7)
	}
	data := pongo2.Context{
		"title": opTitle,
		"task":  task,
	}
	return c.Render(http.StatusOK, "task/form.html", data)
}

func trimHeaderAndSpace(src string, headerPrefix string) (dst string) {
	dst = strings.TrimSpace(src)
	lines := strings.Split(dst, "\r\n")
	if strings.HasPrefix(lines[0], headerPrefix) {
		dst = strings.Join(lines[1:], "\n")
	}
	return dst
}

func (t *TaskController) taskFormSubmit(c echo.Context) (err error) {
	task := new(models.Task)
	c.Bind(task)
	task.Tstart, _ = time.Parse("2006-01-02 15:04-0700", c.FormValue("tstart")+"+0800")
	task.Tend, _ = time.Parse("2006-01-02 15:04-0700", c.FormValue("tend")+"+0800")
	task.Students = trimHeaderAndSpace(task.Students, "學號")
	task.Courses = trimHeaderAndSpace(task.Courses, "課程名稱")
	// lines := strings.Split(task.Students, "\n")
	// if strings.HasPrefix(lines[0], "學號") {
	// 	task.Students = strings.Join(lines[1:], "\n")
	// }
	// lines = strings.Split(task.Courses, "\n")
	// if strings.HasPrefix(lines[0], "課程名稱") {
	// 	task.Courses = strings.Join(lines[1:], "\n")
	// }
	var sql string
	if task.ID > 0 {
		sql = `UPDATE task SET (title, vnum, tstart, tend, description, students, courses) = (:title, :vnum, :tstart, :tend, :description, :students, :courses) WHERE id=:id`
	} else {
		sql = `INSERT INTO task(title, vnum, tstart, tend, description, students, courses) VALUES(:title, :vnum, :tstart, :tend, :description, :students, :courses)`
	}
	if _, err := db.NamedExec(sql, task); err != nil {
		//
	}
	return c.Redirect(http.StatusSeeOther, "/task/")
}

func (t *TaskController) taskDelete(c echo.Context) (err error) {
	task := t.getTaskByParam(c, "tid")
	if c.Request().Method == http.MethodPost {

		return c.Redirect(http.StatusSeeOther, "/task/")
	}
	data := pongo2.Context{
		"task": task,
	}
	return c.Render(http.StatusOK, "task/confirm_delete.html", data)
}
