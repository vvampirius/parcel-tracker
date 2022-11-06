package main

import (
	"bytes"
	"fmt"
	"github.com/vvampirius/mygolibs/telegram"
	"github.com/vvampirius/parcel-tracker/belpost"
	"github.com/vvampirius/parcel-tracker/config"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	RegexpTrackDescription = regexp.MustCompile(`^\s*([A-Z]{2}\d{9}[A-z]{2})\s*(.*)$`)
	RegexpDescriptionUrl = regexp.MustCompile(`^(.*)\s*(https?://\S+)`)
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
			core.ScanTrack(trackId)
			sleepTime := time.Duration(15 + rand.Intn(4)) * time.Second
			DebugLog.Printf(`Waiting %s before next request to Belpost API...`, sleepTime.String())
			time.Sleep(sleepTime)
		}
		time.Sleep(time.Hour)
	}
}

// ScanTrack get info from external APIs for track, notify users, and returns added steps count
func (core *Core) ScanTrack(trackId string) int {
	track := core.Tracks.Get(trackId)
	DebugLog.Printf("Track ID: '%s', is finished: %t, steps count: %d, last step: %v", trackId, track.IsFinished(), len(track.Steps), track.GetLastStepsTime())
	if track.IsFinished() { return 0 }
	addedSteps := make([]TrackStep, 0)
	if belpostApiResponse, err := belpost.GetApiResponse(trackId); err == nil {
		DebugLog.Println(belpostApiResponse)
		trackSteps := BelpostSteps2TrackSteps(belpostApiResponse)
		addedSteps = append(addedSteps, track.AddSteps(trackSteps)...)
	} else { ErrorLog.Println(trackId, err.Error())}
	addedStepsCount := len(addedSteps)
	DebugLog.Printf("Track ID: '%s', added steps: %d", trackId, addedStepsCount)
	if len(addedSteps) > 0 {
		DebugLog.Println(addedSteps)
		core.Tracks.Save(track)
		core.NotifyUsers(trackId, addedSteps)
	}
	return addedStepsCount
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
	isImportant, _ := WriteSteps(updateMessage, steps)
	for userId, userTrack := range core.GetUsersToNotify(trackId) {
		DebugLog.Println(`Notify user`, userId)
		core.NotifyUser(userId, userTrack, updateMessage.Bytes(), isImportant)
	}
}

func (core *Core) TelegramSend(method string, payload interface{}) {
	if method == `` { method = `sendMessage` }
	statusCode, response, err := core.Telegram.Request(`sendMessage`, payload)
	if err != nil {
		ErrorLog.Println(err.Error())
		// TODO: prometheus counter
		return
	}
	if statusCode != 200 {
		ErrorLog.Println(statusCode, response)
		// TODO: prometheus counter
		return
	}
}

func (core *Core) TelegramHttpHandler(w http.ResponseWriter, r *http.Request) {
	DebugLog.Printf("%s : %s : %s : %s\n", r.Header.Get(`X-Real-IP`), r.Method, r.RequestURI, r.UserAgent())
	if r.Method != http.MethodPost {
		ErrorLog.Println(r.Method, r.Header, r.RequestURI)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		ErrorLog.Println(err.Error())
		return
	}
	DebugLog.Println(string(body))
	update, err := telegram.UnmarshalUpdate(body)
	if err != nil {
		ErrorLog.Println(string(body), err.Error())
		//core.Prometheus.Errors.With(prometheus.Labels{`situation`: `unmarshal_update`}).Inc()
		return
	}
	if update.IsMessage() {
		go core.TelegramMessage(update)
		return
	}
	if update.IsCallbackQuery() {
		go core.TelegramCallback(update)
		return
	}
}

func (core *Core) TelegramMessage(update telegram.Update) {
	DebugLog.Println(update.Message.From)
	DebugLog.Println(update.Message.Text)
	if update.Message.Text == `/list` {
		core.ListCommand(update.Message.Chat.Id)
		return
	}
	if trackId, description, url := core.StringToTrack(update.Message.Text); trackId != `` {
		core.AddTrackMessage(update, trackId, description, url)
		return
	}
	payload := telegram.SendMessageIntWithoutReplyMarkup{}
	payload.Text = `Команда не ясна`
	payload.ChatId = update.Message.Chat.Id
	core.TelegramSend(``, payload)
}

func (core *Core) TelegramCallback(update telegram.Update) {
	DebugLog.Println(update.CallbackQuery.Data)
	data := strings.Split(update.CallbackQuery.Data, `:`)
	if data[0] == `detail` {
		core.DetailCallback(update.CallbackQuery.Message.Chat.Id, data[1])
		return
	}
	if data[0] == `remove` {
		core.RemoveCallback(update.CallbackQuery.Message.Chat.Id, data[1])
		return
	}
}

// StringToTrack returns from string: track number, description, url
func (core *Core) StringToTrack(s string) (string, string, string) {
	match := RegexpTrackDescription.FindStringSubmatch(s)
	if len(match) != 3 { return "", "", "" }
	track, description := match[1], match[2]
	match = RegexpDescriptionUrl.FindStringSubmatch(description)
	if len(match) != 3 { return track, description, `` }
	description = strings.TrimSuffix(match[1], ` `)
	return track, description, match[2]
}

func (core *Core) AddTrackMessage(update telegram.Update, trackId, description, url string) {
	DebugLog.Printf("Track ID: '%s', Description: '%s', URL: '%s'", trackId, description, url)
	user := core.ConfigFile.Config.GetUser(update.Message.From.Id)
	if user == nil {
		ErrorLog.Println(`user is nil`)
		return
	}
	// TODO: check for user access
	track := user.GetTrack(trackId)
	if track != nil {
		// TODO: send with buttons
		payload := telegram.SendMessageIntWithoutReplyMarkup{}
		payload.Text = fmt.Sprintf("Трэк %s уже в списке слежения", trackId)
		payload.ChatId = update.Message.Chat.Id
		core.TelegramSend(``, payload)
		return
	}
	newTrack := config.UserTrack{
		 Id: trackId,
		 Description: description,
		 Url: url,
		 Added: time.Now(),
	}
	user.Watch = append(user.Watch, newTrack)
	if err := core.ConfigFile.Save(); err != nil {
		 ErrorLog.Println(err.Error())
		 return
	}
	if added := core.ScanTrack(trackId); added == 0 {
		// TODO: just get info about track
		payload := telegram.SendMessageIntWithoutReplyMarkup{}
		payload.Text = fmt.Sprintf("Трэк %s добавлен в списке слежения, но новых данный по нему нет", trackId)
		payload.ChatId = update.Message.Chat.Id
		core.TelegramSend(``, payload)
	}
}

func (core *Core) ListCommand(userId int) {
	user := core.ConfigFile.Config.GetUser(userId)
	if user == nil {
		ErrorLog.Println(userId, `is nil`)
		return
	}
	watchCount := len(user.Watch)
	if watchCount == 0 {
		payload := telegram.SendMessageIntWithoutReplyMarkup{}
		payload.Text = `Нет треков для отслеживания`
		payload.ChatId = userId
		core.TelegramSend(``, payload)
		return
	}
	trackButtons := make([][]telegram.InlineKeyboardButton, watchCount)
	for i, track := range user.Watch {
		trackButtons[i] = []telegram.InlineKeyboardButton{{
			Text: fmt.Sprintf("%s %s", track.Id, track.Description),
			CallbackData: `detail:` + track.Id,
		}}
	}
	payload := telegram.SendMessageIntWithInlineKeyboardMarkup{
		ReplyMarkup: telegram.InlineKeyboardMarkup{
			InlineKeyboard: trackButtons,
		},
	}
	payload.Text = fmt.Sprintf("У вас %d треков в списке слежения:", watchCount)
	payload.ChatId = userId
	core.TelegramSend(``, payload)
}

func (core *Core) DetailCallback(userId int, trackId string) {
	DebugLog.Println(userId, trackId)
	user := core.ConfigFile.Config.GetUser(userId)
	track := user.GetTrack(trackId)
	message := bytes.NewBufferString(fmt.Sprintln(trackId))
	if track.Description != `` { fmt.Fprintln(message, track.Description) }
	if track.Url != `` { fmt.Fprintln(message, track.Url) }
	if !track.Added.IsZero() { fmt.Fprintf(message, "Добавлен: %s\n", track.Added.Add(3 * time.Hour).Format("2 Jan 15:04"))}
	t := core.Tracks.Get(trackId)
	if len(t.Steps) > 0 {
		fmt.Fprintln(message, ``)
		WriteSteps(message, t.Steps)
	}
	payload := telegram.SendMessageIntWithInlineKeyboardMarkup{
		ReplyMarkup: telegram.InlineKeyboardMarkup{
			InlineKeyboard: [][]telegram.InlineKeyboardButton{{
				//telegram.InlineKeyboardButton{Text: `Описание`, CallbackData: `description:` + trackId},
				//telegram.InlineKeyboardButton{Text: `URL`, CallbackData: `url:` + trackId},
				telegram.InlineKeyboardButton{Text: `Удалить`, CallbackData: `remove:` + trackId},
			}},
		},
	}
	payload.Text = message.String()
	payload.ChatId = userId
	core.TelegramSend(``, payload)
}

func (core *Core) RemoveCallback(userId int, trackId string) {
	DebugLog.Println(userId, trackId)
	user := core.ConfigFile.Config.GetUser(userId)
	newWatch := make([]config.UserTrack, 0)
	for _, track := range user.Watch {
		if track.Id == trackId { continue }
		newWatch = append(newWatch, track)
	}
	user.Watch = newWatch
	if err := core.ConfigFile.Save(); err != nil {
		payload := telegram.SendMessageIntWithoutReplyMarkup{}
		payload.ChatId = userId
		ErrorLog.Println(err.Error())
		payload.Text = err.Error()
		core.TelegramSend(``, payload)
		return
	}
	core.ListCommand(userId)
}


func NewCore(configFile *config.ConfigFile) (*Core, error) {
	tracks, err := NewTracks(configFile.Config.TracksPath)
	if err != nil { return nil, err }

	me, err := telegram.GetMe(configFile.Config.Telegram.Token)
	if err != nil {
		ErrorLog.Println(err.Error())
		return nil, err
	}
	DebugLog.Printf("Got info from Telegram API: @%s with ID:%d and name '%s'\n", me.Username, me.Id, me.FirstName)

	if err := telegram.SetWebHook(configFile.Config.Telegram.Token, configFile.Config.Telegram.Webhook); err != nil {
		ErrorLog.Println(err.Error())
		return nil, err
	}
	DebugLog.Printf("Callback URL set to '%s'\n", configFile.Config.Telegram.Webhook)

	core := Core{
		ConfigFile: configFile,
		Tracks: tracks,
		Telegram: telegram.NewApi(configFile.Config.Telegram.Token),
	}
	return &core, nil
}