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

	processedMIDs = make(map[string]time.Time)
	midMutex      sync.Mutex
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
				Mid    string `json:"mid"`
				Text   string `json:"text"`
				IsEcho bool   `json:"is_echo"`
			} `json:"message"`
		} `json:"messaging"`
	} `json:"entry"`
}

func isDuplicateMessage(mid string) bool {
	if mid == "" {
		return false
	}
	midMutex.Lock()
	defer midMutex.Unlock()

	now := time.Now()
	for k, t := range processedMIDs {
		if now.Sub(t) > 5*time.Minute {
			delete(processedMIDs, k)
		}
	}

	if _, exists := processedMIDs[mid]; exists {
		return true
	}
	processedMIDs[mid] = now
	return false
}

func sendProductPhotos(sender *MetaSender, recipientID, folderURL, productName string) {
	if strings.TrimSpace(folderURL) == "" {
		return
	}

	files := GlobalDriveHelper.GetImageFilesFromFolder(folderURL)
	if len(files) > 0 {
		_ = sender.SendTextMessage(recipientID, fmt.Sprintf("📸 Em gửi Anh/Chị ảnh thực tế lô %s tại kho bên em ạ:", productName))
		for _, img := range files {
			time.Sleep(250 * time.Millisecond)
			if err := sender.UploadAndSendImage(recipientID, img.Filename, img.Data); err != nil {
				log.Printf("[Meta] Lỗi upload ảnh %s: %v", img.Filename, err)
			}
		}
	} else {
		_ = sender.SendTextMessage(recipientID, fmt.Sprintf("📸 Anh/Chị bấm vào đây để xem trực tiếp album ảnh thực tế lô %s bên em nhé ạ:\n%s", productName, folderURL))
	}
}

// scheduleFollowup quản lý bộ đếm 15 phút không phản hồi
func scheduleFollowup(sender *MetaSender, recipientID string, sess *UserSession) {
	groupURL := os.Getenv("WHOLESALE_GROUP_URL")
	if strings.TrimSpace(groupURL) == "" {
		return
	}

	sessionMutex.Lock()
	// Hủy bỏ bộ hẹn giờ cũ nếu khách vừa nhắn tin lại
	if sess.FollowupTimer != nil {
		sess.FollowupTimer.Stop()
	}

	// Nếu đã gửi lời mời vào nhóm rồi thì không gửi lại
	if sess.InvitedToGroup {
		sessionMutex.Unlock()
		return
	}

	// Đặt lịch 15 phút sau
	sess.FollowupTimer = time.AfterFunc(15*time.Minute, func() {
		sessionMutex.Lock()
		if sess.InvitedToGroup {
			sessionMutex.Unlock()
			return
		}
		sess.InvitedToGroup = true
		sessionMutex.Unlock()

		msg := fmt.Sprintf("Dạ em thấy mình đang bận chưa kịp phản hồi. Anh/Chị có thể bấm vào link tham gia nhóm cập nhật bảng giá sỉ & theo dõi các cont hàng mới về mỗi ngày của Tổng kho HP FRUIT tại đây nhé ạ:\n👉 %s\n\nCần hỗ trợ gấp hoặc lên đơn gửi xe đi các tỉnh, Anh/Chị cứ nhắn tin trực tiếp tại đây bên em hỗ trợ mình ngay nhé ạ!", groupURL)
		_ = sender.SendTextMessage(recipientID, msg)
		log.Printf("[Followup] Đã gửi link nhóm sỉ tự động sau 15p cho khách %s", recipientID)
	})
	sessionMutex.Unlock()
}

func main() {
	_ = godotenv.Load()

	sheetID := os.Getenv("SPREADSHEET_ID")
	credFile := "credentials.json"

	inventoryMgr := NewInventoryManager(sheetID, credFile)
	InitDriveHelper(credFile)
	metaSender := NewMetaSender()

	verifyToken := os.Getenv("META_VERIFY_TOKEN")
	if verifyToken == "" {
		verifyToken = "hp_fruit_secret_2026"
	}

	r := gin.Default()

	r.GET("/webhook", func(c *gin.Context) {
		mode := c.Query("hub.mode")
		token := c.Query("hub.verify_token")
		challenge := c.Query("hub.challenge")

		if mode == "subscribe" && token == verifyToken {
			c.String(http.StatusOK, challenge)
			return
		}
		c.String(http.StatusForbidden, "Forbidden")
	})

	r.POST("/webhook", func(c *gin.Context) {
		var callback WebhookCallback
		if err := c.ShouldBindJSON(&callback); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "EVENT_RECEIVED"})

		if callback.Object != "page" {
			return
		}

		for _, entry := range callback.Entry {
			for _, event := range entry.Messaging {
				if event.Message.IsEcho {
					continue
				}

				senderID := event.Sender.ID
				userMsg := strings.TrimSpace(event.Message.Text)
				mid := event.Message.Mid

				if userMsg == "" {
					continue
				}

				if isDuplicateMessage(mid) {
					continue
				}

				available := inventoryMgr.GetAvailableProducts()

				sessionMutex.Lock()
				sess, exists := userSessions[senderID]
				if !exists {
					sess = &UserSession{}
					userSessions[senderID] = sess
				}
				sessionMutex.Unlock()

				// Kích hoạt/Gia hạn bộ đếm hẹn giờ 15 phút cho khách
				scheduleFollowup(metaSender, senderID, sess)

				go func(uid, text string, s *UserSession) {
					reply := ProcessCustomerMessage(text, s, available)
					_ = metaSender.SendTextMessage(uid, reply.Message)

					if reply.ShouldSendImg && reply.Product != nil && reply.Product.FolderAnhID != "" {
						sendProductPhotos(metaSender, uid, reply.Product.FolderAnhID, reply.Product.TenSP)
					}
				}(senderID, userMsg, sess)
			}
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server đang chạy trên cổng %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Không thể khởi động server: %v", err)
	}
}
