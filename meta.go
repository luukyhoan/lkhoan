package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"sync"
)

type MetaSender struct {
	tokens map[string]string // pageID -> token
	mutex  sync.RWMutex
}

var GlobalMetaSender *MetaSender

func NewMetaSender() *MetaSender {
	sender := &MetaSender{
		tokens: make(map[string]string),
	}

	// 1. Token mặc định từ biến cũ
	defaultToken := os.Getenv("META_PAGE_ACCESS_TOKEN")
	defaultPageID := os.Getenv("META_PAGE_ID")
	if defaultPageID != "" && defaultToken != "" {
		sender.tokens[defaultPageID] = defaultToken
	}

	// 2. Định dạng đa page: PAGE_TOKENS="PAGE_ID_1:TOKEN_1,PAGE_ID_2:TOKEN_2"
	extraTokens := os.Getenv("PAGE_TOKENS")
	if extraTokens != "" {
		pairs := strings.Split(extraTokens, ",")
		for _, pair := range pairs {
			parts := strings.Split(strings.TrimSpace(pair), ":")
			if len(parts) == 2 {
				pID := strings.TrimSpace(parts[0])
				tok := strings.TrimSpace(parts[1])
				sender.tokens[pID] = tok
				log.Printf("[Meta] Đã tải Page Access Token cho Page ID: %s", pID)
			}
		}
	}

	GlobalMetaSender = sender
	return sender
}

func (m *MetaSender) getToken(pageID string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if tok, ok := m.tokens[pageID]; ok && tok != "" {
		return tok
	}
	// Dự phòng lấy token mặc định nếu không tìm thấy ID cụ thể
	return os.Getenv("META_PAGE_ACCESS_TOKEN")
}

func (m *MetaSender) SendTextMessage(pageID, recipientID, text string) error {
	token := m.getToken(pageID)
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/me/messages?access_token=%s", token)

	payload := map[string]interface{}{
		"recipient": map[string]string{"id": recipientID},
		"message":   map[string]string{"text": text},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Meta Send Error: %s", string(respBytes))
	}
	return nil
}

func (m *MetaSender) UploadAndSendImage(pageID, recipientID, filename string, data []byte) error {
	token := m.getToken(pageID)
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/me/messages?access_token=%s", token)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	_ = writer.WriteField("recipient", fmt.Sprintf(`{"id":"%s"}`, recipientID))
	_ = writer.WriteField("message", `{"attachment":{"type":"image", "payload":{"is_reusable":true}}}`)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="filedata"; filename="%s"`, filename))
	h.Set("Content-Type", "image/jpeg")

	part, err := writer.CreatePart(h)
	if err != nil {
		return err
	}
	if _, err = part.Write(data); err != nil {
		return err
	}

	if err = writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, &buf)
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
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Meta Image Upload Error: %s", string(respBytes))
	}
	return nil
}
