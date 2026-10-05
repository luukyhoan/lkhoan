package main

import (
	"fmt"
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

	r.GET("/webhook", func(c *gin.Context) {
		mode := c.Query("hub.mode")
		token := c.Query("hub.verify_token")
		challenge := c.Query("hub.challenge")

		if mode == "subscribe" && token == verifyToken {
			c.String(http.StatusOK, challenge)
			return
		}
		c.String(http.StatusForbidden, "Xác minh thất bại")
	})

	r.POST("/webhook", func(c *gin.Context) {
		var req MetaCallback
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		c.Status(http.StatusOK)

		for _, entry := range req.Entry {
			for _, messaging := range entry.Messaging {
				senderID := messaging.Sender.ID
				userMsg := messaging.Message.Text

				if userMsg == "" {
					continue
				}

				customerName := metaSender.GetUserName(senderID)
				available := inventoryMgr.GetAvailableProducts()

				// Lấy sản phẩm khách đang trao đổi dở dang trước đó
				sessionMutex.Lock()
				sess, exists := userSessions[senderID]
				var pendingProd *Product
				if exists && sess.LastProduct.MaSP != "" {
					pendingProd = &sess.LastProduct
				}
				sessionMutex.Unlock()

				// Gọi AI với đầy đủ ngữ cảnh câu trước
				aiRes, err := aiAdvisor.GenerateReply(customerName, userMsg, available, pendingProd)
				if err != nil {
					log.Printf("Gemini Error: %v", err)
					continue
				}

				// Phản hồi tin nhắn
				go metaSender.SendTextMessage(senderID, aiRes.Message)

				// Cập nhật lại sản phẩm vào phiên làm việc của khách
				for _, code := range aiRes.SelectedCodes {
					for _, p := range available {
						if p.MaSP == code {
							sessionMutex.Lock()
							userSessions[senderID] = &UserSession{
								LastProduct: p,
							}
							sessionMutex.Unlock()

							// Chỉ gửi album ảnh nếu lần đầu nhắc tới sản phẩm
							if pendingProd == nil || pendingProd.MaSP != p.MaSP {
								if p.FolderAnhID != "" {
									go func(prod Product) {
										imgLinks := metaSender.GetImageLinksFromDrive(prod.FolderAnhID)
										if len(imgLinks) > 0 {
											metaSender.SendPhotoGrid(senderID, imgLinks)
										}
									}(p)
								}
							}
						}
					}
				}
			}
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server đang chạy trên cổng %s...\n", port)
	r.Run(":" + port)
}
