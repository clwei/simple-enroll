package controllers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
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
	g.GET("create/", t.taskForm, requireStaff)
	g.POST("create/", t.taskFormSubmit, requireStaff)
	gt := g.Group(":tid/", requireValidTaskID)
	gt.GET("", t.taskLogin)
	gt.POST("", t.taskSubmit)
	gt.GET("edit/", t.taskForm, requireStaff)
	gt.POST("edit/", t.taskFormSubmit, requireStaff)
	gt.GET("delete/", t.taskDelete, requireStaff)
	gt.POST("delete/", t.taskDelete, requireStaff)
	gt.GET("view/", t.taskView, requireStaff)
	gt.GET("view/enroll/", t.taskViewEnroll, requireStaff)
	gt.GET("view/dispatch/", t.taskViewDispatch, requireStaff)
	gt.POST("view/dispatch/", t.taskViewDispatch, requireStaff)
	gt.GET("view/dispatch/:did/", t.taskViewDispatchItem, requireStaff)
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
	task := c.Get("task").(models.Task)
	data := pongo2.Context{
		"task": task,
	}
	return c.Render(http.StatusOK, "task/login.html", data)
}

// TaskCourse ...
type TaskCourse struct {
	Name       string
	lowerbound int
	upperbound int
}

// getTaskCourseList 取得選課任務的課程清單
// 		task: 選課任務
//		filter: 要過濾掉的課程名稱
func getTaskCourseList(task models.Task, filter []string) (courses []TaskCourse) {
	smap := map[string]bool{}
	for _, course := range filter {
		smap[course] = true
	}
	// courses: 學生可選課程(去除已選課程)
	lines := strings.Split(task.Courses, "\r\n")
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if _, ok := smap[fields[0]]; !ok {
			lowerbound, _ := strconv.Atoi(fields[1])
			upperbound, _ := strconv.Atoi(fields[2])
			courses = append(courses, TaskCourse{fields[0], lowerbound, upperbound})
		}
	}
	return courses
}

func (t *TaskController) taskSubmit(c echo.Context) error {
	task := c.Get("task").(models.Task)
	username := strings.TrimSpace(c.FormValue("username"))
	selection := strings.TrimSpace(c.FormValue("selection"))
	// fmt.Println("** taskSubmit ** selection =", selection)
	if selection != "" {
		enrollment := models.Enrollment{
			Tid:       task.ID,
			Sid:       username,
			Selection: selection,
		}
		fmt.Println("Tid =", task.ID, "Sid =", username, "Selection =", selection)
		sql := `INSERT INTO public.enrollment(tid, sid, selection) 
					VALUES(:tid, :sid, :selection)
					ON CONFLICT(tid, sid) DO UPDATE
					SET selection = EXCLUDED.selection`
		if _, err := db.NamedExec(sql, enrollment); err != nil {
			fmt.Println("** taskSubmit ** insert error =", err)
		}
		return c.Redirect(http.StatusSeeOther, "/task/")
	}
	passwd := strings.TrimSpace(c.FormValue("passwd"))
	found := false

	stumap := parseStudent(task.Students)
	var stu Student
	var ok bool
	if stu, ok = stumap[username]; ok && passwd == stu.IDno {
		found = true
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
		courses := getTaskCourseList(task, scourses)
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
	task := c.Get("task").(models.Task)
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
	task := c.Get("task").(models.Task)
	c.Bind(task)
	task.Tstart, _ = time.Parse("2006-01-02 15:04-0700", c.FormValue("tstart")+"+0800")
	task.Tend, _ = time.Parse("2006-01-02 15:04-0700", c.FormValue("tend")+"+0800")
	task.Students = trimHeaderAndSpace(task.Students, "學號")
	task.Courses = trimHeaderAndSpace(task.Courses, "課程名稱")

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
	task := c.Get("task").(models.Task)
	if c.Request().Method == http.MethodPost {
		db.NamedExec(`DELETE FROM enrollment WHERE id = :id`, task)
		db.NamedExec(`DELETE FROM enrollment WHERE tid = :id`, task)
		return c.Redirect(http.StatusSeeOther, "/task/")
	}
	data := pongo2.Context{
		"task": task,
	}
	return c.Render(http.StatusOK, "task/confirm_delete.html", data)
}

// Student ...
type Student struct {
	Sid  string
	Cno  string
	Seat string
	Name string
	IDno string
}

// StuMap ...
type StuMap map[string]Student

func parseStudent(raw string) StuMap {
	smap := StuMap{}
	lines := strings.Split(raw, "\r\n")
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		stu := Student{fields[0], fields[1], fields[2], fields[3], fields[4]}
		smap[stu.Sid] = stu
	}
	return smap
}

// StudentEnroll ...
type StudentEnroll struct {
	Student
	Selection []string
}

func getStudentEnrollments(task models.Task) (pool []StudentEnroll, total int) {
	enrollments := []models.Enrollment{}
	db.Select(&enrollments, `SELECT * FROM enrollment WHERE tid = $1`, task.ID)
	smap := parseStudent(task.Students)
	for _, enroll := range enrollments {
		pool = append(pool, StudentEnroll{smap[enroll.Sid], strings.Split(enroll.Selection, ",")})
	}
	return pool, len(smap)
}

func (t *TaskController) taskView(c echo.Context) (err error) {
	task := c.Get("task").(models.Task)
	pool, total := getStudentEnrollments(task)
	courses := getTaskCourseList(task, []string{})
	courseStat := map[string][]int{}
	for _, c := range courses {
		courseStat[c.Name] = make([]int, task.Vnum)
	}
	fmt.Println(courseStat)
	for _, e := range pool {
		fmt.Println(e)
		for i, c := range e.Selection {
			fmt.Println(i, c)
			courseStat[c][i]++
			fmt.Println(courseStat)
		}
	}
	fmt.Println(courseStat)
	data := pongo2.Context{
		"task":       task,
		"seq":        "123456789"[:task.Vnum],
		"pool":       pool,
		"total":      total,
		"courses":    courses,
		"courseStat": courseStat,
	}
	return c.Render(http.StatusOK, "task/view.html", data)
}

func (t *TaskController) taskViewEnroll(c echo.Context) (err error) {
	task := c.Get("task").(models.Task)
	pool, total := getStudentEnrollments(task)
	sort.SliceStable(pool, func(i, j int) bool {
		if pool[i].Cno == pool[j].Cno {
			return pool[i].Seat < pool[j].Seat
		}
		return pool[i].Cno < pool[j].Cno
	})
	data := pongo2.Context{
		"task":  task,
		"seq":   "123456789"[:task.Vnum],
		"pool":  pool,
		"total": total,
	}
	return c.Render(http.StatusOK, "task/view_enroll.html", data)
}

//---------------------------------------------------------------------------
// 選課分發
//---------------------------------------------------------------------------

// CourseDispatchNode ...
type CourseDispatchNode struct {
	TaskCourse
	Fixed []Student // 確認選修名單
	Susp  []Student // 考慮選修名單
}

func (t *TaskController) taskViewDispatch(c echo.Context) (err error) {
	task := c.Get("task").(models.Task)
	if c.Request().Method == http.MethodPost {
		courses := getTaskCourseList(task, []string{})
		enrolls, _ := getStudentEnrollments(task)
		smap := parseStudent(task.Students)
		result := []CourseDispatchNode{}
		cdm := map[string]*CourseDispatchNode{}
		waitingQueue := map[string]StudentEnroll{}
		for _, enroll := range enrolls {
			waitingQueue[enroll.Sid] = enroll
		}
		for _, course := range courses {
			cdm[course.Name] = &CourseDispatchNode{course, []Student{}, []Student{}}
		}
		rand.Seed(time.Now().UnixNano())
		//
		// 分發: 依第 1 志願, 第 2 志願, ... 處理
		//
		for vIndex := 0; vIndex < task.Vnum; vIndex++ {
			// 階段1：先將所有未分發學生依目前處理的志願序，先分發到課程的考慮選修名單
			for sid, enroll := range waitingQueue {
				vCourseName := enroll.Selection[vIndex]
				if len(cdm[vCourseName].Fixed) < cdm[vCourseName].upperbound {
					cdm[vCourseName].Susp = append(cdm[vCourseName].Susp, smap[sid])
				}
			}
			// 階段2：檢查每一門課程，若確認選修人數與考慮選修人數合計超過課程上限，則由考慮選修名單中隨機剔除多餘人選
			for _, cdn := range cdm {
				if len(cdn.Susp) > 0 {
					// 加上考慮名單的人數會爆班 => 隨機從考慮名單中剔除(洗牌，再取前面需要的個數就好)
					if len(cdn.Susp) > cdn.upperbound-len(cdn.Fixed) {
						for k := 0; k < 7; k++ {
							rand.Shuffle(len(cdn.Susp), func(i, j int) { cdn.Susp[i], cdn.Susp[j] = cdn.Susp[j], cdn.Susp[i] })
						}
					}
					// 取出需要的名單，加進確認選修清單，並將其移出未分發學生(waitingQueue)清單
					end := len(cdn.Susp)
					if len(cdn.Susp)+len(cdn.Fixed) > cdn.upperbound {
						end -= len(cdn.Susp) + len(cdn.Fixed) - cdn.upperbound
					}
					for _, stu := range cdn.Susp[:end] {
						cdn.Fixed = append(cdn.Fixed, stu)
						delete(waitingQueue, stu.Sid)
					}
					// 處理完了，清空考慮選修名單，待下一志願序使用
					cdn.Susp = []Student{}
				}
			}
		}
		// 將 map 轉為 slice 以便在 template 中使用
		for _, p := range cdm {
			// 順便依班級座號將選修名單排序
			sort.SliceStable(p.Fixed, func(i, j int) bool {
				return (p.Fixed[i].Cno < p.Fixed[j].Cno) || ((p.Fixed[i].Cno == p.Fixed[j].Cno) && p.Fixed[i].Seat < p.Fixed[j].Seat)
			})
			result = append(result, *p)
		}
		// 未分發名單 map 轉 slice
		waiting := []StudentEnroll{}
		for _, wq := range waitingQueue {
			waiting = append(waiting, wq)
		}
		// 將結果轉為 JSON 字串，以便儲存在資料庫中
		dt := map[string]interface{}{
			"waiting": waiting,
			"result":  result,
		}
		dd, _ := json.Marshal(dt)
		ee := map[string]interface{}{}
		json.Unmarshal(dd, &ee)
		sql := `INSERT INTO dispatch(tid, data) VALUES (:tid, :data)`
		if _, err := db.NamedExec(sql, map[string]interface{}{"tid": task.ID, "data": string(dd)}); err != nil {
			fmt.Println("Result insert error = ", err)
		}
	}
	// 取出分發紀錄列表
	dispatches := []models.Dispatch{}
	if err := db.Select(&dispatches, `SELECT * FROM dispatch WHERE tid = $1 ORDER BY created DESC`, task.ID); err != nil {
		//
	}
	//cnt := len(dispatches)
	result := []map[string]interface{}{}
	/*
		for i := 0; i < cnt; i++ {
			ee := map[string]interface{}{}
			json.Unmarshal([]byte(dispatches[i].Data), &ee)
			ee["ID"] = dispatches[i].ID
			ee["Created"] = dispatches[i].Created
			result = append(result, ee)
		}
	*/
	for _, di := range dispatches {
		ee := map[string]interface{}{}
		json.Unmarshal([]byte(di.Data), &ee)
		ee["ID"] = di.ID
		ee["Created"] = di.Created
		result = append(result, ee)
	}

	data := pongo2.Context{
		"task":   task,
		"result": result,
	}
	return c.Render(http.StatusOK, "task/view_dispatch.html", data)
}

func (t *TaskController) taskViewDispatchItem(c echo.Context) (err error) {
	task := c.Get("task")
	didParam := c.Param("did")
	dispatch := models.Dispatch{}
	if did, err := strconv.Atoi(didParam); err != nil {
		addAlertFlash(c, AlertDanger, "參數型態錯誤！")
	} else if err = db.Get(&dispatch, `SELECT * FROM dispatch WHERE id = $1`, did); err != nil {
		addAlertFlash(c, AlertDanger, "無此分發結果！！")
	}
	result := map[string]interface{}{}
	json.Unmarshal([]byte(dispatch.Data), &result)
	result["Created"] = dispatch.Created
	data := pongo2.Context{
		"task":   task,
		"result": result,
	}
	return c.Render(http.StatusOK, "task/view_dispatch_item.html", data)
}
