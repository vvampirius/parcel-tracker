package belpost

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var ApiUrl = `https://api.belpost.by/api/v1/tracking`

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


func GetApiResponse(number string) (ApiResponse, error) {
	apiResponse := ApiResponse{}
	body := bytes.NewBufferString(`{"number":"` + number + `"}`)
	request, err := http.NewRequest(http.MethodPost, ApiUrl, body)
	if err != nil { return apiResponse, err }
	request.Header.Add(`Accept`, `application/json`)
	request.Header.Add(`Content-Type`, `application/json`)
	request.Header.Add(`User-Agent`, UserAgent)
	client := http.Client{Timeout: Timeout}
	response, err := client.Do(request)
	if err != nil { return apiResponse, err }
	defer response.Body.Close()
	if response.StatusCode != 200 {
		data, _ := io.ReadAll(response.Body)
		return apiResponse, errors.New(fmt.Sprintf("%s: %s", response.Status, string(data)))
	}
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&apiResponse); err != nil { return apiResponse, err }
	if len(apiResponse.Errors) >0 {
		return apiResponse, errors.New(fmt.Sprintf("%v %s", apiResponse.Errors, apiResponse.Message))
	}
	return apiResponse, nil
}