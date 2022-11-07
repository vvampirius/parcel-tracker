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
	Unlimited bool
}

func (user *User) GetTrack(trackId string) *UserTrack {
	for i, userTrack := range user.Watch {
		if userTrack.Id == trackId {
			return &user.Watch[i]
		}
	}
	return nil
}


type Config struct {
	Listen string
	TracksPath string	`yaml:"tracks_path"`
	Users []User
	Telegram struct {
		Token   string
		Webhook string
	}
	StartResponse string `yaml:"start_response"`
}

func (config *Config) GetUser(userId int) *User {
	for i, user := range config.Users {
		if user.Id == userId {
			return &config.Users[i]
		}
	}
	return nil
}
