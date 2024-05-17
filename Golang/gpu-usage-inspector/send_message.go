package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func sendMarkdownToDingTalk(webhookURL string, title string, markdown string) error {
	type DingTalkMessage struct {
		MsgType  string `json:"msgtype"`
		Markdown struct {
			Title string `json:"title"`
			Text  string `json:"text"`
		} `json:"markdown"`
	}

	message := DingTalkMessage{
		MsgType: "markdown",
		Markdown: struct {
			Title string `json:"title"`
			Text  string `json:"text"`
		}{Title: title, Text: markdown},
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}

	webhookURL += "&timestamp=" + fmt.Sprintf("%d", time.Now().UnixNano()/int64(time.Millisecond))
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(messageJSON))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	fmt.Println(resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
