package model

import "time"

type User struct {
	Id         string
	Username   string
	Password   string
	Created_at time.Time
	Updated_at time.Time
}
