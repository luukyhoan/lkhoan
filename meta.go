package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type MetaSender struct {
	PageAccessToken string
}

func NewMetaSender() *MetaSender {
	return &MetaSender{
		PageAccessToken: os.Getenv("META_PAGE_ACCESS_TOKEN"),
	}
}

// Lấy họ tên hiển thị của khách hàng từ Facebook Graph API
func (m *MetaSender) GetUserName(senderID string) string {
	if m.PageAccessToken == "" || senderID == "" {
		return "anh/chị"
	}
	url := fmt.Sprintf("https://graph.facebook.com/v26.0/%s?fields=first_name,last_name,name&access_token=%s", senderID, m.PageAccessToken)
	resp, err := http.Get(url)
	if err != nil {
		return "anh/chị"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "anh/chị"
	}

	var data struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "anh/chị"
	}

	if data.FirstName != "" {
		return data.FirstName
	}
	if data.Name != "" {
		return data.Name
	}
	return "anh/chị"
}

// Gửi tin nhắn văn bản thông thường
func (m *MetaSender) SendTextMessage(recipientID, text string) error {
	if m.PageAccessToken == "" {
		return fmt.Errorf("META_PAGE_ACCESS_TOKEN chưa được cấu hình")
	}

	url := fmt.Sprintf("https://graph.facebook.com/v26.0/me/messages?access_token=%s", m.PageAccessToken)
	payload := map[string]interface{}{
		"recipient": map[string]string{"id": recipientID},
		"message":   map[string]string{"text": text},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("gửi tin nhắn lỗi: status %d", resp.StatusCode)
	}
	return nil
}

// Quét tối đa 6 ảnh từ thư mục Google Drive
func (m *MetaSender) GetImageLinksFromDrive(folderID string) []string {
	if folderID == "" {
		return nil
	}

	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithCredentialsFile("credentials.json"), option.WithScopes(drive.DriveReadonlyScope))
	if err != nil {
		log.Printf("[Drive] Lỗi khởi tạo service: %v", err)
		return nil
	}

	q := fmt.Sprintf("'%s' in parents and mimeType contains 'image/' and trashed = false", folderID)
	r, err := srv.Files.List().Q(q).PageSize(6).Fields("files(id, name)").Do()
	if err != nil {
		log.Printf("[Drive] Lỗi duyệt thư mục %s: %v", folderID, err)
		return nil
	}

	var directLinks []string
	for _, f := range r.Files {
		link := fmt.Sprintf("https://lh3.googleusercontent.com/d/%s", f.Id)
		directLinks = append(directLinks, link)
	}

	return directLinks
}

// Gửi một ảnh đơn lẻ qua Messenger API
func (m *MetaSender) SendSingleImage(recipientID, imageURL string) error {
	if m.PageAccessToken == "" {
		return fmt.Errorf("META_PAGE_ACCESS_TOKEN chưa được cấu hình")
	}

	url := fmt.Sprintf("https://graph.facebook.com/v26.0/me/messages?access_token=%s", m.PageAccessToken)
	payload := map[string]interface{}{
		"recipient": map[string]string{"id": recipientID},
		"message": map[string]interface{}{
			"attachment": map[string]interface{}{
				"type": "image",
				"payload": map[string]interface{}{
					"url":         imageURL,
					"is_reusable": true,
				},
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Meta API gửi ảnh lỗi: status %d", resp.StatusCode)
	}
	return nil
}

// Bắn đồng thời các ảnh để Messenger hiển thị Photo Grid
func (m *MetaSender) SendPhotoGrid(recipientID string, imgLinks []string) {
	if len(imgLinks) == 0 {
		return
	}

	maxImgs := 6
	if len(imgLinks) < maxImgs {
		maxImgs = len(imgLinks)
	}

	var wg sync.WaitGroup
	for i := 0; i < maxImgs; i++ {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			if err := m.SendSingleImage(recipientID, url); err != nil {
				log.Printf("[Messenger] Lỗi gửi ảnh Photo Grid: %v", err)
			}
		}(imgLinks[i])
	}
	wg.Wait()
}
