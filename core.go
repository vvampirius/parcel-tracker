package main

import (
	"bytes"
	"fmt"
	"github.com/vvampirius/mygolibs/telegram"
	"github.com/vvampirius/parcel-tracker/belpost"
	"github.com/vvampirius/parcel-tracker/config"
	"math/rand"
	"time"
)

type Core struct {
	ConfigFile *config.ConfigFile
	Tracks *Tracks
	Telegram *telegram.Api
}

func (core *Core) GetTracksForScan() []string {
	tracks := make([]string, 0)
	for _, user := range core.ConfigFile.Config.Users {
		for _, userTrack := range user.Watch {
			if IsInSliceString(tracks, userTrack.Id) { continue }
			tracks = append(tracks, userTrack.Id)
		}
	}
	return tracks
}

func (core *Core) ScanRoutine() {
	rand.Seed(time.Now().Unix())
	for {
		for _, trackId := range core.GetTracksForScan() {
			track := core.Tracks.Get(trackId)
			if track.IsFinished() {
				DebugLog.Println(trackId, `is finished`)
				continue
			}
			DebugLog.Println(track)
			addedSteps := make([]TrackStep, 0)
			if belpostApiResponse, err := belpost.GetApiResponse(trackId); err == nil {
				DebugLog.Println(belpostApiResponse)
				trackSteps := BelpostSteps2TrackSteps(belpostApiResponse)
				addedSteps = append(addedSteps, track.AddSteps(trackSteps)...)
			} else { ErrorLog.Println(trackId, err.Error())}
			if len(addedSteps) > 0 {
				DebugLog.Println(addedSteps)
				core.Tracks.Save(track)
				core.NotifyUsers(trackId, addedSteps)
			}
			sleepTime := time.Duration(15 + rand.Intn(4)) * time.Second
			DebugLog.Printf(`Waiting %s before next request to Belpost API...`, sleepTime.String())
			time.Sleep(sleepTime)
		}
		time.Sleep(time.Hour)
	}
}

func (core *Core) GetUsersToNotify(trackId string) map[int]config.UserTrack {
	users := make(map[int]config.UserTrack)
	for _, user := range core.ConfigFile.Config.Users {
		for _, userTrack := range user.Watch {
			if userTrack.Id == trackId {
				users[user.Id] = userTrack
				break
			}
		}
	}
	return users
}

func (core *Core) NotifyUser(userId int, userTrack config.UserTrack, updateMessage []byte, isImportant bool) {
	message := bytes.NewBuffer(nil)
	fmt.Fprintf(message, "%s %s %s\n\n", userTrack.Id, userTrack.Description, userTrack.Url)
	message.Write(updateMessage)
	payload := telegram.SendMessageIntWithoutReplyMarkup{ DisableNotification: !isImportant }
	payload.Text = message.String()
	payload.ChatId = userId
	if statusCode, x, err := core.Telegram.Request(`sendMessage`, payload); err != nil || statusCode != 200 {
		ErrorLog.Println(err, statusCode, x)
		// TODO: add prometheus error
	}
}

func (core *Core) NotifyUsers(trackId string, steps []TrackStep) {
	updateMessage := bytes.NewBuffer(nil)
	isImportant := false
	for _, step := range steps {
		fmt.Fprintf(updateMessage, "%s %s %s\n", step.Time.Format(`02.01 15:04`), step.Place, step.Event)
		if step.Important { isImportant = true }
	}
	for userId, userTrack := range core.GetUsersToNotify(trackId) {
		DebugLog.Println(`Notify user`, userId)
		core.NotifyUser(userId, userTrack, updateMessage.Bytes(), isImportant)
	}
}


func NewCore(configFile *config.ConfigFile) (*Core, error) {
	tracks, err := NewTracks(configFile.Config.TracksPath)
	if err != nil { return nil, err }

	core := Core{
		ConfigFile: configFile,
		Tracks: tracks,
		Telegram: telegram.NewApi(configFile.Config.Telegram.Token),
	}
	return &core, nil
}