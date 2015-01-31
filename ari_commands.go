package ari

import (
	"fmt"
	"bytes"
)
/*
{
"channelId" : "<channelID>",
"media"		: "<mediaURI>",
"lang"		: "<options[0]",
"offsetms"	: "<options[1]",
"skipms"	: "<options[2]",
"playbackId": "<options[3]"
}

*/

func buildJSON(params map[string]string) string {
	mapsize := len(params)
	var counter int = 1
	body := bytes.NewBufferString("{")
	for key, value := range params {
		var s string
		if counter < mapsize {
			s = fmt.Sprintf("\"%s\":\"%s\",", key, value)
		} else {
			s = fmt.Sprintf("\"%s\":\"%s\"", key, value)
		}
		body.WriteString(s)
		counter++
	}
	body.WriteString("}")
	return body.String()
}

func ChannelPlay (channelID string, mediaURI string, options...string) {
	paramMap := make(map[string]string)
	paramMap["channelId"] = channelID
	paramMap["media"] = mediaURI
	url := fmt.Sprintf("\n/channels/%s/play", channelID)
	for index, value := range options {
		switch index {
		case 0:
			if len(value) > 0 {
				paramMap["lang"] = value
			}
		case 1:
			if len(value) > 0 {
				paramMap["offsetms"] = value
			}
		case 2:
			if len(value) > 0 {
				paramMap["skipms"] = value
			}
		case 3:
			if len(value) > 0 {
				paramMap["playbackId"] = value
			}
		}
	}

	body := buildJSON(paramMap)

	fmt.Println(url)
	fmt.Println(body)
}