package models

import (
	"time"
)

type (
	// Dispatch 選課分發結果
	Dispatch struct {
		ID      int       `form:"id" json:"id" db:"id"`
		Tid     int       `form:"tid" json:"tid" db:"tid"`
		Created time.Time `form:"created" json:"created" db:"created"`
		Data    string    `form:"data" json:"data" db:"data"`
		Forced  bool      `form:"forced" json:"forced" db:"forced"`
	}
)
