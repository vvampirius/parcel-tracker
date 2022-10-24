package config

import "time"

type UserTrack struct {
	Id string
	Description string
	Url string
	Added time.Time
}

type User struct {
	Id int
	Name string
	Watch []UserTrack
}

type Config struct {
	Listen string
	TracksPath string	`yaml:"tracks_path"`
	Users []User
	Telegram struct {
		Token   string
		Webhook string
	}
}
