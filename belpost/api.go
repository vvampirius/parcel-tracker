package belpost

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type ApiStep struct {
	CreatedAt string	`json:"created_at"`
	Timestamp int
	Event string
	Place string
}

func (apiStep *ApiStep) Time() time.Time {
	return time.Unix(int64(apiStep.Timestamp), 0)
}

type ApiResponse struct {
	Data []struct{
		Redirectable bool
		Steps []ApiStep
	}
	Errors map[string][]string
	Message string
}

func (apiResponse *ApiResponse) IsFound() bool {
	if len(apiResponse.Data) > 0 { return true }
	return false
}


type Api struct {
	Url string
	RequestTimeout time.Duration
	RequestInterval func(time.Time)
	OnError func(string, error)
	UserAgent string
	requestMu sync.Mutex
	lastRequestAt time.Time
}

func (api *Api) get(trackId string) (ApiResponse, error) {
	apiResponse := ApiResponse{}
	request := bytes.NewBufferString(`{"number":"` + trackId + `"}`)
	response, err := api.httpRequest(request)
	if err != nil { return apiResponse, err }
	if err := json.Unmarshal(response, &apiResponse); err != nil { return apiResponse, err }
	if len(apiResponse.Errors) >0 {
		return apiResponse, errors.New(fmt.Sprintf("%v %s", apiResponse.Errors, apiResponse.Message))
	}
	return apiResponse, nil
}

func (api *Api) Get(trackId string) (ApiResponse, error) {
	response, err := api.get(trackId)
	if err != nil && api.OnError != nil {
		api.OnError(trackId, err)
	}
	return response, err
}

func (api *Api) httpRequest(body io.Reader) ([]byte, error) {
	api.requestMu.Lock()
	defer api.requestMu.Unlock()
	if api.RequestInterval != nil {
		api.RequestInterval(api.lastRequestAt)
	}
	defer api.touchLastRequestAt()
	request, err := http.NewRequest(http.MethodPost, api.Url, body)
	if err != nil { return nil, err }
	request.Header.Add(`Accept`, `application/json`)
	request.Header.Add(`Content-Type`, `application/json`)
	request.Header.Add(`User-Agent`, api.UserAgent)
	client := http.Client{Timeout: api.RequestTimeout}
	response, err := client.Do(request)
	if err != nil { return nil, err }
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil { return nil, err }
	if response.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("%s: %s", response.Status, string(data)))
	}
	return data, nil
}

func (api *Api) touchLastRequestAt() {
	api.lastRequestAt = time.Now()
}


func NewApi() *Api {
	api := Api{
		Url: `https://api.belpost.by/api/v1/tracking`,
		RequestTimeout: 30 * time.Second,
		UserAgent: `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36`,
		RequestInterval: RequestInterval,
	}
	return &api
}