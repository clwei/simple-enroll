package models

import (
	"time"
)

// Task 選課項目
type Task struct {
	ID       int       `form:"id" json:"id" db:"id"`
	Title    string    `form:"title" json:"title" db:"title"`
	Tstart   time.Time `form:"-" json:"tstart" db:"tstart"`
	Tend     time.Time `form:"-" json:"tend" db:"tend"`
	Vnum     int       `form:"vnum" json:"vnum" db:"vnum"`
	Desc     string    `form:"desc" json:"desc" db:"description"`
	Students string    `form:"students" json:"students" db:"students"`
	Courses  string    `form:"courses" json:"courses" db:"courses"`
}
