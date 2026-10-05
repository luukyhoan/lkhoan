package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// Cấu trúc 1 thẻ ảnh trong Album trượt
type CarouselElement struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	ImageURL string `json:"image_url"`
}

// Hàm gửi 1 lượt 5-7 ảnh dạng Album Carousel
func SendCarouselImages(recipientID string, elements []CarouselElement) error {
	token := os.Getenv("META_PAGE_ACCESS_TOKEN")
	url := fmt.Sprintf("https://graph.facebook.com/v26.0/me/messages?access_token=%s", token)

	// Giới hạn tối đa 10 phần tử theo tiêu chuẩn Meta API
	if len(elements) > 10 {
		elements = elements[:10]
	}

	payload := map[string]interface{}{
		"recipient": map[string]string{"id": recipientID},
		"message": map[string]interface{}{
			"attachment": map[string]interface{}{
				"type": "template",
				"payload": map[string]interface{}{
					"template_type": "generic",
					"elements":      elements,
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
		return fmt.Errorf("Meta API error: status %d", resp.StatusCode)
	}

	return nil
}
