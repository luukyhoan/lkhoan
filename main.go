package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

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
				senderID := event.Sender.ID
				userMsg := event.Message.Text

				if userMsg == "" {
					continue
				}

				customerName := metaSender.GetUserName(senderID)
				available := inventoryMgr.GetAvailableProducts()

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

				// 1. Tìm nhanh trong RAM (không tốn token, không lộ giá, tự gửi ảnh)
				if match := FindProductInMemory(userMsg); match != nil {
					sessionMutex.Lock()
					if match.LastProduct != nil {
						sess.LastProduct = *match.LastProduct
					}
					sessionMutex.Unlock()

					go metaSender.SendTextMessage(senderID, match.Message)

					if match.PhotoURL != "" {
						go func(link string, pName string) {
							caption := fmt.Sprintf("📸 Hình ảnh thực tế lô %s tại kho:\n%s", pName, link)
							metaSender.SendTextMessage(senderID, caption)
						}(match.PhotoURL, match.LastProduct.TenSP)
					}
					continue
				}

				// 2. Chuyển sang AI để xử lý báo giá riêng sỉ/lẻ khi khách phản hồi nhu cầu
				aiRes, err := aiAdvisor.GenerateReply(customerName, userMsg, available, pendingProd)
				if err != nil {
					log.Printf("Gemini Error: %v", err)
					continue
				}

				go metaSender.SendTextMessage(senderID, aiRes.Message)

				if len(aiRes.SelectedCodes) > 0 {
					go func(codes []string) {
						for _, code := range codes {
							for _, p := range available {
								if strings.EqualFold(p.MaSP, code) && p.FolderAnhID != "" {
									caption := fmt.Sprintf("📸 Hình ảnh thực tế %s:\n%s", p.TenSP, p.FolderAnhID)
									metaSender.SendTextMessage(senderID, caption)
									break
								}
							}
						}
					}(aiRes.SelectedCodes)
				}
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
