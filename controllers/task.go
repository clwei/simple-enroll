package controllers

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
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
	gt.GET("", t.taskLogin, requireValidTime)
	gt.POST("", t.taskSubmit, requireValidTime)
	gt.GET("edit/", t.taskForm, requireStaff)
	gt.POST("edit/", t.taskFormSubmit, requireStaff)
	gt.GET("delete/", t.taskDelete, requireStaff)
	gt.POST("delete/", t.taskDelete, requireStaff)
	gt.GET("view/", t.taskView, requireStaff)
	gt.GET("view/enroll/", t.taskViewEnroll, requireStaff)
	gt.GET("view/dispatch/", t.taskViewDispatch, requireStaff).Name = "dispatchList"
	gt.POST("view/dispatch/", t.taskViewDispatch, requireStaff)
	gt.GET("view/dispatch/:did/", t.taskViewDispatchItem, requireStaff)
	gt.GET("view/dispatch/:did/delete/", t.taskViewDispatchItemDelete, requireStaff)
	gt.POST("view/dispatch/:did/delete/", t.taskViewDispatchItemDelete, requireStaff)
	gt.GET("view/dispatch/:did/download/", t.taskViewDispatchItemDownload, requireStaff)
}

//
func (t *TaskController) taskList(c echo.Context) error {
	tasks := []models.Task{}
	user := getCurrUser(c)
	sql := "SELECT * FROM task"
	if !user.IsStaff && !user.IsAdmin {
		sql += " WHERE tstart <= now() AND now() <= tend"
	}
	sql += " ORDER BY tstart desc"
	if err := db.Select(&tasks, sql); err != nil {
		fmt.Println(sql, err)
	}
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
	Name       string `json:Name`
	lowerbound int    `json:lowerbound`
	upperbound int    `json:upperbound`
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
	// 學生送出選課結果
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
			fmt.Println("\n\n** taskSubmit ** insert error =", err, "\n")
		}
		return c.Redirect(http.StatusSeeOther, "/task/")
	}
	// 學生登入
	data := pongo2.Context{
		"task": task,
	}
	passwd := strings.TrimSpace(c.FormValue("passwd"))
	stumap := parseStudent(task.Students)
	forbid := parseStudent(task.Forbidden)
	if stu, ok := stumap[username]; ok && passwd == stu.IDno {
		// 是否在禁止選課名單中？
		if fstu, ok := forbid[username]; ok {
			addAlertFlash(c, AlertDanger, "抱歉！您被禁止參與此項選課/選社任務，禁止原因為「"+fstu.IDno+"」")
			return c.Render(http.StatusOK, "task/login.html", data)
		}
		// 通過帳號密碼驗設證
		enrollment := models.Enrollment{}
		if err := db.Get(&enrollment, "SELECT * FROM enrollment WHERE tid = $1 AND sid = $2", task.ID, username); err != nil {
		}
		// scourses: 學生已選課程
		var scourses []string
		if enrollment.Selection != "" {
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
	// 登入失敗!!
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
	c.Bind(&task)
	task.Tstart, _ = time.Parse("2006-01-02 15:04-0700", c.FormValue("tstart")+"+0800")
	task.Tend, _ = time.Parse("2006-01-02 15:04-0700", c.FormValue("tend")+"+0800")
	task.Students = trimHeaderAndSpace(task.Students, "學號")
	task.Courses = trimHeaderAndSpace(task.Courses, "課程名稱")
	task.Forbidden = trimHeaderAndSpace(task.Forbidden, "學號")

	var sql string
	if task.ID > 0 {
		sql = `UPDATE task SET (title, vnum, tstart, tend, description, students, courses, forbidden) = (:title, :vnum, :tstart, :tend, :description, :students, :courses, :forbidden) WHERE id=:id`
	} else {
		sql = `INSERT INTO task(title, vnum, tstart, tend, description, students, courses, forbidden) VALUES(:title, :vnum, :tstart, :tend, :description, :students, :courses, :forbidden)`
	}
	if _, err := db.NamedExec(sql, task); err != nil {
		//
	}
	return c.Redirect(http.StatusSeeOther, "/task/")
}

func (t *TaskController) taskDelete(c echo.Context) (err error) {
	task := c.Get("task").(models.Task)
	if c.Request().Method == http.MethodPost {
		db.NamedExec(`DELETE FROM task WHERE id = :id`, task)
		db.NamedExec(`DELETE FROM enrollment WHERE tid = :id`, task)
		db.NamedExec(`DELETE FROM dispatch WHERE tid = :tid`, task)
		return c.Redirect(http.StatusSeeOther, "/task/")
	}
	data := pongo2.Context{
		"task": task,
	}
	return c.Render(http.StatusOK, "task/confirm_delete.html", data)
}

// Student ...
type Student struct {
	Sid    string // 學號
	Cno    string // 班級
	Seat   string // 座號
	Name   string // 姓名
	IDno   string // 身分證號
	VIndex int    // 選課分發志願
}

// StuMap ...
type StuMap map[string]*Student

func parseStudent(raw string) StuMap {
	smap := StuMap{}
	lines := strings.Split(raw, "\r\n")
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		stu := Student{fields[0], fields[1], fields[2], fields[3], fields[4], 0}
		smap[stu.Sid] = &stu
	}
	return smap
}

// StudentEnroll ...
type StudentEnroll struct {
	Student
	Selection []string
}

func getStudentEnrollments(task models.Task) (pool []StudentEnroll, estu []Student) {
	enrollments := []models.Enrollment{}
	db.Select(&enrollments, `SELECT * FROM enrollment WHERE tid = $1`, task.ID)
	smap := parseStudent(task.Students)
	forbid := parseStudent(task.Forbidden)
	for sid := range forbid {
		if _, ok := smap[sid]; ok {
			delete(smap, sid)
		}
	}
	for _, enroll := range enrollments {
		// 若已先選課，事後被排入排除名單，則略過其選課資料
		if _, ok := forbid[enroll.Sid]; ok {

			continue
		}
		selection := strings.Split(enroll.Selection, ",")
		if len(selection) > task.Vnum {
			selection = selection[:task.Vnum]
		}
		pool = append(pool, StudentEnroll{*smap[enroll.Sid], selection})
		delete(smap, enroll.Sid)
	}
	for _, stu := range smap {
		estu = append(estu, *stu)
	}
	sort.SliceStable(estu, func(i, j int) bool {
		if estu[i].Cno == estu[j].Cno {
			return estu[i].Seat < estu[j].Seat
		}
		return estu[i].Cno < estu[j].Cno
	})
	return pool, estu
}

func (t *TaskController) taskView(c echo.Context) (err error) {
	task := c.Get("task").(models.Task)
	pool, _ := getStudentEnrollments(task)
	courses := getTaskCourseList(task, []string{})
	courseStat := map[string][]int{}
	for _, c := range courses {
		courseStat[c.Name] = make([]int, task.Vnum)
	}
	for _, e := range pool {
		for i, c := range e.Selection {
			if i < task.Vnum {
				courseStat[c][i]++
			}
		}
	}
	data := pongo2.Context{
		"task": task,
		"seq":  "123456789"[:task.Vnum],
		//"pool":       pool,
		//"total":      total,
		//"courses":    courses,
		"courseStat": courseStat,
	}
	return c.Render(http.StatusOK, "task/view.html", data)
}

func (t *TaskController) taskViewEnroll(c echo.Context) (err error) {
	task := c.Get("task").(models.Task)
	pool, estu := getStudentEnrollments(task)
	sort.SliceStable(pool, func(i, j int) bool {
		if pool[i].Cno == pool[j].Cno {
			return pool[i].Seat < pool[j].Seat
		}
		return pool[i].Cno < pool[j].Cno
	})
	total := len(pool) + len(estu)
	data := pongo2.Context{
		"task":  task,
		"seq":   "123456789"[:task.Vnum],
		"pool":  pool,
		"estu":  estu,
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
	Fixed      []Student // 確認選修名單
	Susp       []Student // 考慮選修名單
	EmptySlots int       // 剩餘空缺
}

// EVScore ...
type EVScore struct {
	Count    []int   // 每個志願的分發人數
	Score    int     // 總分發志願權重(志願序*分發人數的加總)
	AvgScore float32 // 平均分發志願序
	Success  int     // 成功分發人數
	Failed   int     // 分發失敗人數
}

// CourseDispatchResult ...
type CourseDispatchResult struct {
	ev      EVScore
	result  []CourseDispatchNode
	waiting []StudentEnroll
}

func (t *TaskController) taskViewDispatch(c echo.Context) (err error) {
	task := c.Get("task").(models.Task)
	if c.Request().Method == http.MethodPost {
		var dispatch models.Dispatch
		c.Bind(&dispatch)
		courses := getTaskCourseList(task, []string{})
		enrolls, _ := getStudentEnrollments(task)
		smap := parseStudent(task.Students)
		forbid := parseStudent(task.Forbidden)
		// 排除選修名單
		for sid := range forbid {
			if _, ok := smap[sid]; ok {
				delete(smap, sid)
			}
		}
		result := []CourseDispatchNode{}
		cdm := map[string]*CourseDispatchNode{}
		waitingQueue := map[string]StudentEnroll{}
		for sid, stu := range smap {
			waitingQueue[sid] = StudentEnroll{*stu, []string{}}
		}
		for _, enroll := range enrolls {
			waitingQueue[enroll.Sid] = enroll
		}
		for _, course := range courses {
			cdm[course.Name] = &CourseDispatchNode{course, []Student{}, []Student{}, course.upperbound}
		}
		rand.Seed(time.Now().UnixNano())
		//
		// 分發: 依第 1 志願, 第 2 志願, ... 處理
		//
		ev := EVScore{}
		ev.Count = make([]int, task.Vnum+1)
		for vIndex := 0; vIndex < task.Vnum; vIndex++ {
			// 階段1：先將所有未分發學生依目前處理的志願序，先分發到課程的考慮選修名單
			for sid, enroll := range waitingQueue {
				if len(enroll.Selection) <= vIndex {
					continue
				}
				vCourseName := enroll.Selection[vIndex]
				if len(cdm[vCourseName].Fixed) < cdm[vCourseName].upperbound {
					cdm[vCourseName].Susp = append(cdm[vCourseName].Susp, *smap[sid])
				}
			}

			// 統計課程缺額
			for cid, cdn := range cdm {
				cdm[cid].EmptySlots = int(math.Max(0, float64(cdn.upperbound-len(cdn.Fixed)-len(cdn.Susp))))
			}

			// 計算學生下一志額的缺額來當作排序依據
			priority := map[string]int{}
			fmt.Println("#######", vIndex, task.Vnum)
			for sid, enroll := range waitingQueue {
				fmt.Printf("\t%s%s%s: ", enroll.Cno, enroll.Seat, enroll.Name)
				fmt.Println(enroll.Selection)
				if vIndex >= (len(enroll.Selection)-1) || vIndex >= task.Vnum-1 {
					priority[sid] = 0
				} else {
					priority[sid] = cdm[enroll.Selection[vIndex+1]].EmptySlots*10 + (rand.Int() % 4)
				}
			}

			// 階段2：檢查每一門課程，若確認選修人數與考慮選修人數合計超過課程上限，則由考慮選修名單中剔除多餘人選
			for cid, cdn := range cdm {
				if len(cdn.Susp) > 0 {
					// 加上考慮名單的人數會爆班 => 依學生下一個志願的缺額數多的優先從考慮名單中剔除(取前面需要的個數就好)
					if len(cdn.Susp) > cdn.upperbound-len(cdn.Fixed) {
						sort.Slice(cdm[cid].Susp, func(i, j int) bool {
							return priority[cdm[cid].Susp[i].Sid] < priority[cdm[cid].Susp[j].Sid]
						})
					}
					fmt.Println(cdn.Name, cdn.upperbound, len(cdn.Fixed), len(cdn.Susp))
					for _, stu := range cdm[cid].Susp {
						fmt.Printf("\t%s%s%s: %d\n", stu.Cno, stu.Seat, stu.Name, priority[stu.Sid])
					}
					// 取出需要的名單，加進確認選修清單，並將其移出未分發學生(waitingQueue)清單
					end := len(cdn.Susp)
					if len(cdn.Susp)+len(cdn.Fixed) > cdn.upperbound {
						end -= len(cdn.Susp) + len(cdn.Fixed) - cdn.upperbound
					}
					for _, stu := range cdn.Susp[:end] {
						stu.VIndex = vIndex + 1            // 記錄該生的分發志願序
						cdn.Fixed = append(cdn.Fixed, stu) // 將該生加入課程的正式選修名單
						delete(waitingQueue, stu.Sid)
					}
					// 處理完了，清空考慮選修名單，待下一志願序使用
					cdn.Susp = []Student{}
					// 評估用
					ev.Count[vIndex] += end
					ev.Success += end
					ev.Score += (vIndex + 1) * end
				}
			}
		}
		// 啟用強制分發
		if dispatch.Forced {
			// 先找出尚未額滿的課程
			availableCourses := []string{}
			for _, cdn := range cdm {
				if len(cdn.Fixed) < cdn.upperbound {
					availableCourses = append(availableCourses, cdn.Name)
				}
			}
			// 尚未分發的人，優先分發至未達開課人數的課程，或人數較少的課程
			for sid := range waitingQueue {
				// 每處理完一人，重新排序課程
				sort.SliceStable(availableCourses, func(i, j int) bool {
					ni, nj := cdm[availableCourses[i]], cdm[availableCourses[j]]
					if (len(ni.Fixed) < ni.upperbound) != (len(nj.Fixed) < nj.upperbound) {
						return len(ni.Fixed) < ni.upperbound
					}
					return len(ni.Fixed)-ni.lowerbound < len(nj.Fixed)-nj.lowerbound
				})
				smap[sid].VIndex = task.Vnum + 1
				cdm[availableCourses[0]].Fixed = append(cdm[availableCourses[0]].Fixed, *smap[sid])
			}
			// 統一加總評估資料
			ev.Count[task.Vnum] += len(waitingQueue)
			ev.Success += len(waitingQueue)
			ev.Score += (task.Vnum + 1) * len(waitingQueue)
			waitingQueue = map[string]StudentEnroll{}
		}
		ev.Failed = len(waitingQueue)
		ev.AvgScore = float32(ev.Score) / float32(ev.Success)

		// 將 map 轉為 slice 以便在 template 中使用
		for _, p := range cdm {
			// 依班級座號將選修名單排序
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
		sort.SliceStable(waiting, func(i, j int) bool {
			return (waiting[i].Cno < waiting[j].Cno) || ((waiting[i].Cno == waiting[j].Cno) && (waiting[i].Seat < waiting[j].Seat))
		})
		// 將結果轉為 JSON 字串，以便儲存在資料庫中
		dt := map[string]interface{}{
			"waiting": waiting,
			"result":  result,
			"ev":      ev,
		}
		dd, _ := json.Marshal(dt)
		dispatch.Data = string(dd)
		sql := `INSERT INTO dispatch(tid, data, forced) VALUES (:tid, :data, :forced)`
		if _, err := db.NamedExec(sql, dispatch); err != nil {
			fmt.Println("Result insert error = ", err)
		}
	}
	// 取出分發紀錄列表
	dispatches := []models.Dispatch{}
	if err := db.Select(&dispatches, `SELECT * FROM dispatch WHERE tid = $1 ORDER BY created DESC`, task.ID); err != nil {
		//
	}
	result := []map[string]interface{}{}
	// 將紀錄中的 JSON 字串轉回資料結構
	for _, di := range dispatches {
		ee := map[string]interface{}{}
		json.Unmarshal([]byte(di.Data), &ee)
		ee["ID"] = di.ID
		ee["Created"] = di.Created
		ee["Forced"] = di.Forced
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
		"task":    task,
		"result":  result,
		"created": dispatch.Created,
	}
	return c.Render(http.StatusOK, "task/view_dispatch_item.html", data)
}

func (t *TaskController) taskViewDispatchItemDownload(c echo.Context) (err error) {
	task := c.Get("task").(models.Task)
	didParam := c.Param("did")
	dispatch := models.Dispatch{}
	if did, err := strconv.Atoi(didParam); err != nil {
		addAlertFlash(c, AlertDanger, "參數型態錯誤！")
	} else if err = db.Get(&dispatch, `SELECT * FROM dispatch WHERE id = $1`, did); err != nil {
		addAlertFlash(c, AlertDanger, "無此分發結果！！")
	}
	result := map[string]interface{}{}
	json.Unmarshal([]byte(dispatch.Data), &result)
	tmp, _ := json.Marshal(result["result"])
	r := []CourseDispatchNode{}
	json.Unmarshal(tmp, &r)
	tmp, _ = json.Marshal(result["waiting"])
	w := []StudentEnroll{}
	json.Unmarshal(tmp, &w)
	f := excelize.NewFile()
	failSheet := "分發失敗名單"
	f.SetSheetName(f.GetSheetName(1), failSheet)
	f.SetSheetRow(failSheet, "A1", &[]string{"班級", "座號", "學號", "姓名"})
	for rid, stu := range w {
		f.SetSheetRow(failSheet, fmt.Sprintf("A%d", rid+2), &[]interface{}{stu.Cno, stu.Seat, stu.Sid, stu.Name})
	}
	for _, cr := range r {
		f.NewSheet(cr.Name)
		f.SetSheetRow(cr.Name, "A1", &[]string{"班級", "座號", "學號", "姓名", "分發志願序"})
		for di, stu := range cr.Fixed {
			f.SetSheetRow(cr.Name, fmt.Sprintf("A%d", di+2), &[]interface{}{stu.Cno, stu.Seat, stu.Sid, stu.Name, stu.VIndex})
		}
	}
	fname := fmt.Sprintf("%s@%s.xlsx", task.Title, dispatch.Created.Format("2006-01-02_15-04"))
	f.SaveAs("static/" + fname)
	err = c.Attachment("static/"+fname, fname)
	os.Remove("static/" + fname)
	return err
}

func (t *TaskController) taskViewDispatchItemDelete(c echo.Context) (err error) {
	task := c.Get("task").(models.Task)
	didParam := c.Param("did")
	dispatch := models.Dispatch{}
	if did, err := strconv.Atoi(didParam); err != nil {
		addAlertFlash(c, AlertDanger, "參數型態錯誤！")
	} else if err = db.Get(&dispatch, `SELECT * FROM dispatch WHERE id = $1`, did); err != nil {
		addAlertFlash(c, AlertDanger, "無此分發結果！！")
	} else {
		if c.Request().Method == http.MethodPost {
			db.NamedExec(`DELETE FROM dispatch WHERE id = :id`, dispatch)
			return c.Redirect(http.StatusSeeOther, e.Reverse("dispatchList", task.ID, dispatch.ID))
		}
		data := pongo2.Context{
			"task":     task,
			"dispatch": dispatch,
		}
		return c.Render(http.StatusOK, "task/confirm_dispatch_delete.html", data)
	}
	return c.Render(http.StatusNotFound, "base.html", pongo2.Context{})
}
