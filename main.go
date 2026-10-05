package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	userSessions = make(map[string]*UserSession)
	sessionMutex sync.Mutex
)

type WebhookCallback struct {
	Object string `json:"object"`
	Entry  []struct {
		ID        string `json:"id"`
		Time      int64  `json:"time"`
		Messaging []struct {
			Sender struct {
				ID string `json:"id"`
			} `json:"sender"`
			Recipient struct {
				ID string `json:"id"`
			} `json:"recipient"`
			Message struct {
				Mid  string `json:"mid"`
				Text string `json:"text"`
			} `json:"message"`
		} `json:"messaging"`
	} `json:"entry"`
}

// sendProductPhotos quét toàn bộ ảnh trong Folder Drive và gửi bung trực tiếp ra Messenger
func sendProductPhotos(sender *MetaSender, recipientID, folderURL, productName string) {
	if folderURL == "" {
		return
	}

	files := GlobalDriveHelper.GetImageFilesFromFolder(folderURL)
	if len(files) > 0 {
		_ = sender.SendTextMessage(recipientID, fmt.Sprintf("📸 Em gửi Anh/Chị hình ảnh thực tế lô %s mới về tại kho bên em ạ:", productName))
		for _, img := range files {
			time.Sleep(300 * time.Millisecond)
			if err := sender.UploadAndSendImage(recipientID, img.Filename, img.Data); err != nil {
				log.Printf("[Meta] Lỗi upload ảnh %s: %v", img.Filename, err)
			}
		}
	} else {
		_ = sender.SendTextMessage(recipientID, fmt.Sprintf("📸 Anh/Chị có thể bấm xem album ảnh thực tế %s tại đây ạ:\n%s", productName, folderURL))
	}
}
