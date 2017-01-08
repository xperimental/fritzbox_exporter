package home

import "testing"

func TestGetResponse(t *testing.T) {
	for _, test := range []struct {
		challenge string
		password  string
		response  string
	}{
		{
			challenge: "1234567z",
			password:  "äbc",
			response:  "1234567z-9e224a41eeefa284df7bb0f26c2913e2",
		},
		{
			challenge: "1234567z",
			password:  ".bc",
			response:  "1234567z-4d422a0edeeded87635c6de7ff5857e2",
		},
		{
			challenge: "1234567z",
			password:  "€bc",
			response:  "1234567z-4d422a0edeeded87635c6de7ff5857e2",
		},
	} {
		response := getResponse(test.challenge, test.password)
		if response != test.response {
			t.Errorf("got %s, wanted %s", response, test.response)
		}
	}
}
