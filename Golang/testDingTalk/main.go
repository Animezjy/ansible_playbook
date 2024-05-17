package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func sendMarkdownToDingTalk(webhookURL string, secret string, title string, markdown string) error {
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

func main() {
	webhookURL := "https://oapi.dingtalk.com/robot/send?access_token=59e5648d22598b6108f662ad52a178df571c5656959cc875dc1004b9c58b01fe"
	secret := "SEC2cec4b82f0d9afb0461df97abdbdaf912fa29d38e391e252b4635ef6d87037c7"

	// 这里替换为你的 markdown 内容
	markdown := `
		hello world  
    golang 发送钉钉消息测试
    `

	err := sendMarkdownToDingTalk(webhookURL, secret, "GPU 使用情况", markdown)
	if err != nil {
		log.Fatal(err)
	}
}
