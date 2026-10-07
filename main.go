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

// Thời gian bot tạm dừng nhường quyền cho nhân viên (10 phút)
const AdminCooldownDuration = 10 * time.Minute

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
				AppID  int64  `json:"app_id"`
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
		_ = sender.SendTextMessage(recipientID, fmt.Sprintf("📸 Anh/Chị bấm vào link sau để xem ảnh thực tế lô %s tại kho bên em nhé ạ:\n%s", productName, folderURL))
	}
}

// markAdminActive ghi nhận thời điểm admin chat tay và hủy timer mời sỉ
func markAdminActive(customerID string) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	sess, exists := userSessions[customerID]
	if !exists {
		sess = &UserSession{}
		userSessions[customerID] = sess
	}

	sess.LastAdminMessageTime = time.Now()
	if sess.FollowupTimer != nil {
		sess.FollowupTimer.Stop()
		sess.FollowupTimer = nil
	}
	sess.InvitedToGroup = true
	log.Printf("[Takeover] Admin vừa nhắn tin cho khách %s -> Bot sẽ tạm dừng trong %v", customerID, AdminCooldownDuration)
}

// isHumanTakeoverActive kiểm tra xem admin có đang trực chat không
func isHumanTakeoverActive(sess *UserSession) bool {
	if sess == nil || sess.LastAdminMessageTime.IsZero() {
		return false
	}
	return time.Since(sess.LastAdminMessageTime) < AdminCooldownDuration
}

// scheduleFollowup tự động hẹn giờ 15 phút mời khách vào nhóm Zalo sỉ nếu im lặng
func scheduleFollowup(sender *MetaSender, recipientID string, sess *UserSession) {
	// Nếu admin đang trực, không đặt timer tự động mời nhóm
	if isHumanTakeoverActive(sess) {
		return
	}

	groupURL := os.Getenv("WHOLESALE_GROUP_URL")
	if strings.TrimSpace(groupURL) == "" {
		groupURL = "https://zalo.me/g/8q8it61v5mpczevlrtkh"
	}

	sessionMutex.Lock()
	if sess.FollowupTimer != nil {
		sess.FollowupTimer.Stop()
	}

	if sess.InvitedToGroup {
		sessionMutex.Unlock()
		return
	}

	sess.FollowupTimer = time.AfterFunc(15*time.Minute, func() {
		sessionMutex.Lock()
		if sess.InvitedToGroup || isHumanTakeoverActive(sess) {
			sessionMutex.Unlock()
			return
		}
		sess.InvitedToGroup = true
		sessionMutex.Unlock()

		msg := fmt.Sprintf("Dạ em thấy mình đang bận chưa kịp phản hồi. Anh/Chị có thể bấm vào link tham gia nhóm Zalo cập nhật bảng giá sỉ & theo dõi các cont hàng mới về mỗi ngày của Tổng kho HP FRUIT tại đây nhé ạ:\n👉 %s\n\nCần hỗ trợ gấp hoặc lên đơn gửi xe đi các tỉnh, Anh/Chị cứ nhắn tin trực tiếp tại đây bên em hỗ trợ mình ngay nhé ạ!", groupURL)
		_ = sender.SendTextMessage(recipientID, msg)
		log.Printf("[Followup] Đã gửi lời mời vào nhóm sỉ tự động cho khách %s", recipientID)
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

	// Endpoint giữ server 24/24 cho UptimeRobot
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	// Webhook xác thực với Meta
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

	// Webhook tiếp nhận tin nhắn từ Facebook Messenger
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
				// KHI ADMIN HOẶC NHÂN VIÊN GỬI TIN BẰNG TAY (IS_ECHO == TRUE)
				if event.Message.IsEcho {
					// AppID == 0 là tin nhắn gửi trực tiếp từ Page inbox (Facebook app/trình duyệt)
					if event.Message.AppID == 0 {
						customerID := event.Recipient.ID
						markAdminActive(customerID)
					}
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

				sessionMutex.Lock()
				sess, exists := userSessions[senderID]
				if !exists {
					sess = &UserSession{}
					userSessions[senderID] = sess
				}
				sessionMutex.Unlock()

				// KIỂM TRA QUYỀN TRỰC CHAT CỦA ADMIN:
				// Nếu admin vừa nhắn trong vòng 10 phút, bot tuyệt đối im lặng để nhường admin
				if isHumanTakeoverActive(sess) {
					log.Printf("[Bot Muted] Khách %s nhắn '%s' nhưng Admin đang trực chat (còn trong 10 phút) -> Bot không can thiệp", senderID, userMsg)
					continue
				}

				available := inventoryMgr.GetAvailableProducts()

				// Kích hoạt/Gia hạn bộ đếm 15 phút gửi link nhóm Zalo
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

	log.Printf("HP FRUIT Bot đang chạy trên cổng %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Lỗi server: %v", err)
	}
}
