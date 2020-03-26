package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type respPlatform struct {
	Access string `json: "access"`
}

type auth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Session struct {
	Jwt       string
}

type Config struct {
	Session	Session
	Host	string
	Timeout int
}

func GetSession(platformURL string, usename string, password string) (*Session, error) {
	var bodyData = auth{usename, password}
	body, err := json.Marshal(&bodyData)
	if err != nil {
		return nil, err
	}

	resp, err := PostRequest(nil, platformURL, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsedResp = respPlatform{}
	err = json.Unmarshal([]byte(responseData), &parsedResp)
	if err != nil {
		return nil, err
	}
	if parsedResp.Access == "" {
		return nil, fmt.Errorf("An empty access field in the platform respomse.")
	}
	return &Session{
		Jwt:       parsedResp.Access,
	}, nil
}
