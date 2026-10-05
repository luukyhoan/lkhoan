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

	links := GlobalDriveHelper.GetImageLinksInFolder(folderURL)
	if len(links) > 0 {
		_ = sender.SendTextMessage(recipientID, fmt.Sprintf("📸 Em gửi Anh/Chị hình ảnh thực tế lô %s mới về tại kho bên em ạ:", productName))
		for _, imgURL := range links {
			// Thêm độ trễ nhỏ để Messenger hiển thị ảnh theo thứ tự mượt mà
			time.Sleep(300 * time.Millisecond)
			if err := sender.SendImageMessage(recipientID, imgURL); err != nil {
				log.Printf("[Meta] Gửi ảnh attachment thất bại (%s): %v", imgURL, err)
			}
		}
	} else {
		// Dự phòng nếu folder Drive chưa có ảnh hoặc chưa cấp quyền cho Service Account
		_ = sender.SendTextMessage(recipientID, fmt.Sprintf("📸 Anh/Chị có thể bấm xem trực tiếp trọn bộ album ảnh và video lô %s tại đây ạ:\n%s", productName, folderURL))
	}
}

func main() {
	_ = godotenv.Load()

	sheetID := os.Getenv("SPREADSHEET_ID")
	credFile := "credentials.json"

	// Khởi tạo quản lý kho và Drive Helper dùng chung credentials.json
	inventoryMgr := NewInventoryManager(sheetID, credFile)
	InitDriveHelper(credFile)

	aiAdvisor := NewMultiAIAdvisor(
		os.Getenv("GROQ_API_KEY"),
		os.Getenv("GEMINI_API_KEY"),
	)
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
				userMsg := strings.TrimSpace(event.Message.Text)

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

				// 1. Nhận diện từ RAM: phản hồi tức thì, bảo mật giá, tự động gửi cụm ảnh từ Drive
				if match := FindProductInMemory(userMsg); match != nil {
					sessionMutex.Lock()
					if match.LastProduct != nil {
						sess.LastProduct = *match.LastProduct
					}
					sessionMutex.Unlock()

					go metaSender.SendTextMessage(senderID, match.Message)

					if match.PhotoURL != "" {
						go sendProductPhotos(metaSender, senderID, match.PhotoURL, match.LastProduct.TenSP)
					}
					continue
				}

				// 2. Chuyển sang AI để xử lý phân loại và báo giá riêng biệt (sỉ/lẻ)
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
									sendProductPhotos(metaSender, senderID, p.FolderAnhID, p.TenSP)
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
