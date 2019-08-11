package models

// Enrollment ...
type Enrollment struct {
	ID        int    `form:"id" json:"id" db:"id"`
	Tid       int    `form:"tid" json:"tid" db:"tid"`
	Sid       string `form:"sid" json:"sid" db:"sid"`
	Selection string `form:"selection" json:"selection" db:"selection"`
}
