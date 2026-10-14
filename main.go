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

func sendProductPhotos(sender *MetaSender, pageID, recipientID, folderURL, productName string) {
	if strings.TrimSpace(folderURL) == "" {
		return
	}

	files := GlobalDriveHelper.GetImageFilesFromFolder(folderURL)
	if len(files) > 0 {
		_ = sender.SendTextMessage(pageID, recipientID, fmt.Sprintf("📸 Em gửi Anh/Chị ảnh thực tế lô %s tại kho bên em ạ:", productName))
		for _, img := range files {
			time.Sleep(250 * time.Millisecond)
			if err := sender.UploadAndSendImage(pageID, recipientID, img.Filename, img.Data); err != nil {
				log.Printf("[Meta] Lỗi upload ảnh %s: %v", img.Filename, err)
			}
		}
	} else {
		_ = sender.SendTextMessage(pageID, recipientID, fmt.Sprintf("📸 Anh/Chị bấm vào link sau để xem ảnh thực tế lô %s tại kho bên em nhé ạ:\n%s", productName, folderURL))
	}
}

func markAdminActive(sessionKey, text string) {
	if strings.Contains(text, "Chị đang ngắm món nào bên em thế") || strings.Contains(text, "Tổng kho") || strings.Contains(text, "cảm ơn bạn đã liên hệ") {
		return
	}

	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	sess, exists := userSessions[sessionKey]
	if !exists {
		sess = &UserSession{}
		userSessions[sessionKey] = sess
	}

	sess.LastAdminMessageTime = time.Now()
	// Hủy bỏ hẹn giờ và đánh dấu không gửi link nhóm Zalo làm phiền khách
	if sess.FollowupTimer != nil {
		sess.FollowupTimer.Stop()
	}
	sess.InvitedToGroup = true
	if isOrderConfirmation(text) {
		// Đặt cooldown dài 48 tiếng cho khách đã chốt đơn
		sess.LastAdminMessageTime = time.Now().Add(48 * time.Hour)
		log.Printf("[Order] Nhận diện Admin chốt đơn cho %s. Đã tắt bot 48h!", sessionKey)
	}
	if sess.FollowupTimer != nil {
		sess.FollowupTimer.Stop()
		sess.FollowupTimer = nil
	}
	sess.InvitedToGroup = false
	log.Printf("[Takeover] Admin chat tay: '%s' -> Tạm dừng bot trong %v cho %s", text, AdminCooldownDuration, sessionKey)
}

func isOrderConfirmation(msg string) bool {
	low := strings.ToLower(msg)
	if strings.Contains(low, "lên đơn") || strings.Contains(low, "len don") ||
		strings.Contains(low, "chốt đơn") || strings.Contains(low, "chot don") {
		return true
	}
	if strings.Contains(low, " x ") && (strings.Contains(low, "k") || strings.Contains(low, "ship")) {
		return true
	}
	return false
}

func isHumanTakeoverActive(sess *UserSession) bool {
	if sess == nil || sess.LastAdminMessageTime.IsZero() {
		return false
	}
	return time.Since(sess.LastAdminMessageTime) < AdminCooldownDuration
}

func getCustomerName(pageAccessToken, psid string) string {
	url := fmt.Sprintf("https://graph.facebook.com/v21.0/%s?fields=first_name,name&access_token=%s", psid, pageAccessToken)
	resp, err := httpClient.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var res struct {
		FirstName string `json:"first_name"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return ""
	}
	if res.FirstName != "" {
		return res.FirstName
	}
	return res.Name
}

func scheduleFollowup(sender *MetaSender, pageID, recipientID, sessionKey string, sess *UserSession) {
	groupURL := os.Getenv("WHOLESALE_GROUP_URL")
	if strings.TrimSpace(groupURL) == "" {
		groupURL = "https://zalo.me/g/8q8it61v5mpczevlrtkh"
	}

	sessionMutex.Lock()
	if sess.FollowupTimer != nil {
		sess.FollowupTimer.Stop()
	}

	// Nếu đã gửi hoặc admin đã tiếp quản/lên đơn thì tuyệt đối không hẹn giờ gửi Zalo
	if sess.InvitedToGroup || !sess.LastAdminMessageTime.IsZero() {
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
		_ = sender.SendTextMessage(pageID, recipientID, msg)
		log.Printf("[Followup] Đã gửi lời mời vào nhóm sỉ tự động cho khách %s", sessionKey)
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

	// Endpoint kiểm tra và giữ kết nối 24/7 cho UptimeRobot
	pingHandler := func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	}
	r.GET("/ping", pingHandler)
	r.HEAD("/ping", pingHandler)
	r.GET("/", pingHandler)
	r.HEAD("/", pingHandler)

	// Endpoint trang Chính sách quyền riêng tư để kích hoạt Live Mode trên Meta
	r.GET("/privacy", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, `<!DOCTYPE html>
<html lang="vi">
<head>
    <meta charset="utf-8">
    <title>Chính sách quyền riêng tư - HP FRUIT</title>
    <style>body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; line-height: 1.6; max-width: 800px; margin: 40px auto; padding: 0 20px; color: #333; }</style>
</head>
<body>
    <h1>Chính sách quyền riêng tư - HP FRUIT Bot</h1>
    <p>Chào mừng bạn đến với ứng dụng chăm sóc khách hàng của Tổng kho trái cây & nông sản HP FRUIT.</p>
    <p>Ứng dụng chatbot này được phát triển nhằm mục đích hỗ trợ tra cứu giá, gửi hình ảnh sản phẩm và hỗ trợ lên đơn hàng tự động trên Facebook Messenger.</p>
    <h3>1. Thu thập dữ liệu</h3>
    <p>Chúng tôi chỉ tiếp nhận các thông tin công khai do người dùng gửi trực tiếp qua tin nhắn (bao gồm nội dung chat, tên hiển thị Facebook) và thông tin nhận hàng (số điện thoại, địa chỉ) do bạn tự nguyện cung cấp để phục vụ đóng gói và gửi hàng.</p>
    <h3>2. Sử dụng thông tin</h3>
    <p>Thông tin thu thập chỉ được dùng cho mục đích tư vấn, xác nhận đơn hàng và điều phối vận chuyển nội bộ. Chúng tôi cam kết tuyệt đối không chia sẻ hoặc bán dữ liệu cho bất kỳ bên thứ ba nào.</p>
    <h3>3. Quyền của người dùng</h3>
    <p>Người dùng có toàn quyền yêu cầu xóa lịch sử trò chuyện hoặc thông tin cá nhân bằng cách liên hệ trực tiếp với quản trị viên qua Fanpage chính thức.</p>
</body>
</html>`)
	})

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
			entryPageID := entry.ID

			for _, event := range entry.Messaging {
				// Nhận diện khi Admin nhắn tay để tạm dừng bot
				if event.Message.IsEcho {
					pageID := event.Sender.ID
					customerID := event.Recipient.ID
					sessionKey := fmt.Sprintf("%s_%s", pageID, customerID)
					markAdminActive(sessionKey, event.Message.Text)
					continue
				}

				pageID := event.Recipient.ID
				if pageID == "" {
					pageID = entryPageID
				}
				customerID := event.Sender.ID
				userMsg := strings.TrimSpace(event.Message.Text)
				mid := event.Message.Mid

				if userMsg == "" || isDuplicateMessage(mid) {
					continue
				}

				sessionKey := fmt.Sprintf("%s_%s", pageID, customerID)

				sessionMutex.Lock()
				sess, exists := userSessions[sessionKey]
				if !exists {
					sess = &UserSession{}
					userSessions[sessionKey] = sess
				}
				sessionMutex.Unlock()

				// Nếu admin đang trực chat thì bot bỏ qua
				if isHumanTakeoverActive(sess) {
					log.Printf("[Bot Muted] Khách %s nhắn nhưng Admin đang trực -> Bỏ qua", sessionKey)
					continue
				}

				available := inventoryMgr.GetAvailableProducts()
				scheduleFollowup(metaSender, pageID, customerID, sessionKey, sess)

				go func(pID, uID string, sKey string, text string, s *UserSession) {

					// Xử lý gửi lời chào mở đầu kèm tên khách nếu là tin nhắn đầu tiên hoặc câu ngắn/dấu chấm
					cleanText := strings.TrimSpace(strings.ToLower(text))
					isShortGreeting := cleanText == "." || cleanText == "hi" || cleanText == "hello" || cleanText == "alo" || cleanText == "chao" || cleanText == "chào shop" || cleanText == "o"

					if !s.Greeted && (isShortGreeting || (reply == nil || strings.TrimSpace(reply.Message) == "")) {
						custName := getCustomerName(metaSender.PageAccessToken, uID)
						xungHo := "anh/chị"
						if custName != "" {
							xungHo = fmt.Sprintf("anh/chị %s", custName)
						}
						greetingMsg := fmt.Sprintf("Dạ em chào %s ạ! Mình đang cần tìm mã hoa quả nào hay cần nhập sỉ/lẻ loại nào để em gửi ảnh thực tế và báo giá chi tiết cho mình nhé ạ?", xungHo)
						_ = metaSender.SendTextMessage(pID, uID, greetingMsg)
						s.Greeted = true
						scheduleFollowup(metaSender, pID, uID, sessionKey, s)
						continue
					}
					s.Greeted = true

					reply := ProcessCustomerMessage(text, s, available)
					if reply != nil && strings.TrimSpace(reply.Message) != "" {
						_ = metaSender.SendTextMessage(pID, uID, reply.Message)
					}

					if reply.ShouldSendImg && reply.Product != nil && reply.Product.FolderAnhID != "" {
						sendProductPhotos(metaSender, pID, uID, reply.Product.FolderAnhID, reply.Product.TenSP)
					}
				}(pageID, customerID, sessionKey, userMsg, sess)
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
