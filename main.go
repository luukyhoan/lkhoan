package main

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	userSessions = make(map[string]*UserSession)
	sessionMutex sync.Mutex
)

// WebhookCallback cấu trúc nhận webhook từ Meta
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

func main() {
	_ = godotenv.Load()

	sheetID := os.Getenv("SPREADSHEET_ID")
	inventoryMgr := NewInventoryManager(sheetID, "credentials.json")

	aiAdvisor := &AIAdvisor{
		apiKey: os.Getenv("GEMINI_API_KEY"),
	}
	metaSender := NewMetaSender()

	verifyToken := os.Getenv("META_VERIFY_TOKEN")
	if verifyToken == "" {
		verifyToken = "hp_fruit_secret_2026"
	}

	r := gin.Default()

	// 1. Endpoint xác thực Webhook với Meta (GET)
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

	// 2. Endpoint tiếp nhận tin nhắn từ Facebook Messenger (POST)
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
				senderID := event.Sender.ID
				userMsg := event.Message.Text

				if userMsg == "" {
					continue
				}

				customerName := metaSender.GetUserName(senderID)
				available := inventoryMgr.GetAvailableProducts()

				// Lấy sản phẩm khách đang trao đổi dở dang trước đó
				sessionMutex.Lock()
				sess, exists := userSessions[senderID]
				if !exists {
					sess = &UserSession{}
					userSessions[senderID] = sess
				}
				var pendingProd *Product
				if sess.LastProduct.MaSP != "" {
					pendingProd = &sess.LastProduct
				}
				sessionMutex.Unlock()

				// Tối ưu: Kiểm tra nhanh trong RAM nếu khớp tên sản phẩm
				if quickReply := FindProductInMemory(userMsg); quickReply != "" {
					go metaSender.SendTextMessage(senderID, quickReply)
					continue
				}

				// Nếu không khớp từ khóa rõ ràng, gọi AI Gemini tư vấn
				aiRes, err := aiAdvisor.GenerateReply(customerName, userMsg, available, pendingProd)
				if err != nil {
					log.Printf("Gemini Error: %v", err)
					continue
				}

				// Phản hồi tin nhắn
				go metaSender.SendTextMessage(senderID, aiRes.Message)
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
