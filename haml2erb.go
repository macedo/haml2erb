package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
)

const (
	baseURL     = "https://haml2erb.org/api/convert"
	contentType = "application/json"
)

type ResponseData struct {
	ERB     string `json:"erb"`
	Error   string `json:"error"`
	Success bool   `json:"success"`
}

type ErrUnprocessableEntity struct {
	error string
}

func (e *ErrUnprocessableEntity) Error() string {
	return e.error
}

func haml2erb(haml string) (string, error) {
	reqBody, err := json.Marshal(map[string]string{
		"haml": string(haml),
	})
	if err != nil {
		return "", err
	}

	response, err := http.Post(baseURL, contentType, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	data := &ResponseData{}
	if err := json.NewDecoder(response.Body).Decode(data); err != nil {
		return "", err
	}

	if data.Success {
		matched, err := regexp.Match(`unexpected`, []byte(data.ERB))
		if err != nil {
			return "", err
		}

		if matched {
			return "", &ErrUnprocessableEntity{error: data.ERB}
		}
		return data.ERB, nil
	}

	return "", &ErrUnprocessableEntity{error: data.Error}
}
