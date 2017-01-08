package home

import (
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	loginURLFormat = "http://%s/login_sid.lua"
	invalidSid     = "0000000000000000"
)

func (c *homeCollector) sidValid() bool {
	return len(c.Sid) > 0 &&
		c.Sid != invalidSid &&
		time.Since(c.SidTimestamp) < 60*time.Minute
}

type sidResponse struct {
	Sid       string `xml:"SID"`
	Challenge string
	BlockTime int
}

func (c *homeCollector) authenticate() (string, error) {
	challenge, err := c.getChallenge()
	if err != nil {
		return "", fmt.Errorf("error getting challenge: %s", err)
	}

	response := fmt.Sprintf("response=%s", getResponse(challenge, c.Password))
	body := strings.NewReader(response)

	url := fmt.Sprintf(loginURLFormat, c.Hostname)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result, err := c.sidRequest(req)
	if err != nil {
		return "", err
	}

	return result.Sid, nil
}

func (c *homeCollector) getChallenge() (string, error) {
	url := fmt.Sprintf(loginURLFormat, c.Hostname)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	result, err := c.sidRequest(req)
	if err != nil {
		return "", err
	}

	return result.Challenge, nil
}

func (c *homeCollector) sidRequest(req *http.Request) (sidResponse, error) {
	var result sidResponse
	res, err := c.Client.Do(req)
	if err != nil {
		return result, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return result, fmt.Errorf("invalid status code: %d", res.StatusCode)
	}

	if err := xml.NewDecoder(res.Body).Decode(&result); err != nil {
		return result, err
	}

	if result.BlockTime > 0 {
		return result, fmt.Errorf("login blocked: %d seconds", result.BlockTime)
	}

	return result, nil
}

func getResponse(challenge, password string) string {
	response := fmt.Sprintf("%s-%s", challenge, password)
	bytes := getMangledUTF16Bytes(response)
	sum := md5.Sum(bytes)
	return fmt.Sprintf("%s-%x", challenge, sum)
}

func getMangledUTF16Bytes(input string) []byte {
	runes := []rune(input)
	bytes := make([]byte, len(runes)*2)
	for i, r := range runes {
		if r > 255 {
			bytes[2*i] = 0x2e
		} else {
			bytes[2*i] = byte(r)
		}
		bytes[2*i+1] = 0
	}
	return bytes
}
