package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

type MetaSender struct {
	PageAccessToken string
}

func NewMetaSender() *MetaSender {
	return &MetaSender{
		PageAccessToken: os.Getenv("META_PAGE_ACCESS_TOKEN"),
	}
}

func (m *MetaSender) GetUserName(senderID string) string {
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s?fields=first_name,last_name,name&access_token=%s", senderID, m.PageAccessToken)
	resp, err := http.Get(url)
	if err != nil {
		return "Anh/Chị"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "Anh/Chị"
	}

	var data struct {
		FirstName string `json:"first_name"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
		if data.FirstName != "" {
			return data.FirstName
		}
		if data.Name != "" {
			return data.Name
		}
	}
	return "Anh/Chị"
}

func (m *MetaSender) SendTextMessage(recipientID, messageText string) error {
	payload := map[string]interface{}{
		"recipient": map[string]string{"id": recipientID},
		"message":   map[string]string{"text": messageText},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://graph.facebook.com/v19.0/me/messages?access_token=%s", m.PageAccessToken)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		resBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Meta API Error: %s", string(resBody))
	}
	return nil
}

func (m *MetaSender) UploadAndSendImage(recipientID string, filename string, data []byte) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	recipientField, err := writer.CreateFormField("recipient")
	if err != nil {
		return err
	}
	_, _ = recipientField.Write([]byte(fmt.Sprintf(`{"id":"%s"}`, recipientID)))

	messageField, err := writer.CreateFormField("message")
	if err != nil {
		return err
	}
	_, _ = messageField.Write([]byte(`{"attachment":{"type":"image", "payload":{"is_reusable":true}}}`))

	part, err := writer.CreateFormFile("filedata", filename)
	if err != nil {
		return err
	}
	_, _ = part.Write(data)

	if err := writer.Close(); err != nil {
		return err
	}

	url := fmt.Sprintf("https://graph.facebook.com/v19.0/me/messages?access_token=%s", m.PageAccessToken)
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		resBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Meta Upload Image Error: %s", string(resBody))
	}
	return nil
}
