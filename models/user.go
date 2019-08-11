package models

// User ...
type User struct {
	ID       int    `form:"id" json:"id" db:"id"`
	Username string `form:"username" json:"username" db:"username"`
	Passwd   string `form:"passwd" json:"passwd" db:"passwd"`
	Cno      int16  `form:"cno" json:"cno" db:"cno"`
	Seat     int8   `form:"seat" json:"seat" db:"seat"`
	Name     string `form:"name" json:"name" db:"name"`
	IsStaff  bool   `form:"is_staff" json:"is_staff" db:"is_staff"`
	IsAdmin  bool   `form:"is_admin" json:"is_admin" db:"is_admin"`
}
