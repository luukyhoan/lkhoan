package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type MultiAIAdvisor struct {
	groqKey   string
	geminiKey string
}

func NewMultiAIAdvisor(groqKey, geminiKey string) *MultiAIAdvisor {
	return &MultiAIAdvisor{
		groqKey:   groqKey,
		geminiKey: geminiKey,
	}
}

func (m *MultiAIAdvisor) GenerateReply(customerName, userMsg string, availableProducts []Product, pendingProd *Product) (*GeminiBotResponse, error) {
	dataBytes, _ := json.Marshal(availableProducts)
	pendingInfo := "Chưa có"
	if pendingProd != nil && pendingProd.MaSP != "" {
		pendingInfo = fmt.Sprintf("Mã: %s | Tên: %s | Giá lẻ: %s | Giá sỉ: %s | Quy cách: %s",
			pendingProd.MaSP, pendingProd.TenSP, pendingProd.GiaLeThung, pendingProd.GiaSiLo, pendingProd.QuyCach)
	}

	systemInstruction := fmt.Sprintf(`
Bạn là chuyên viên tư vấn bán hàng của Tổng kho trái cây nhập khẩu cao cấp HP FRUIT (Bồ Đề - Long Biên).
Khách hàng: "%s".
Sản phẩm đang trao đổi: [%s].

Dữ liệu kho hàng:
%s

QUY TẮC BẢO MẬT GIÁ VÀ BÁO GIÁ:

1. GIAI ĐOẠN 1 - KHI CHƯA BIẾT RÕ NHU CẦU:
   - TUYỆT ĐỐI KHÔNG BÁO BẤT KỲ MỨC GIÁ NÀO (KHÔNG báo giá lẻ, KHÔNG báo giá sỉ).
   - Chỉ giới thiệu nguồn gốc, độ tươi ngon, quy cách thùng và chất lượng chuẩn cao cấp.
   - Kết thúc bằng một câu hỏi gợi mở thanh lịch để phân loại nhu cầu:
     "Dạ bên em có chính sách giá riêng cho khách dùng gia đình và khách lấy sỉ số lượng cho cửa hàng/đại lý. Không biết Anh/Chị dự tính lấy số lượng dùng thử hay lấy cho shop để em áp dụng mức giá tốt nhất cho mình ạ?"

2. GIAI ĐOẠN 2 - KHI KHÁCH ĐÃ NÊU RÕ NHU CẦU:
   - Khách lẻ (mua gia đình, ăn thử, biếu tặng):
     + CHỈ BÁO DUY NHẤT GIÁ LẺ THÙNG (cột Gia_Le_Thung). TUYỆT ĐỐI KHÔNG nhắc đến giá sỉ.
     + Đưa mã sản phẩm vào "selected_codes".
   - Khách sỉ (lấy số lượng lớn, đại lý, shop):
     + CHỈ BÁO DUY NHẤT GIÁ SỈ LÔ (cột Gia_Si_Lo). TUYỆT ĐỐI KHÔNG nhắc đến giá lẻ.
     + Cam kết chất lượng cont/bay, bao cuống tươi, hỗ trợ gửi chành xe.
     + Đưa mã sản phẩm vào "selected_codes".

3. XƯNG HÔ: Lịch thiệp, xưng "em", gọi khách là "Anh/Chị".
4. ĐỊNH DẠNG JSON TRẢ VỀ:
   {
     "message": "Nội dung phản hồi khách hàng",
     "selected_codes": ["MÃ_SP"]
   }
`, customerName, pendingInfo, string(dataBytes))

	// 1. Thử gọi Groq qua model openai/gpt-oss-20b (tốc độ cao, LPU 1000 TPS)
	if m.groqKey != "" {
		res, err := m.callGroq(systemInstruction, userMsg)
		if err == nil {
			return res, nil
		}
		log.Printf("[AI] Groq gặp sự cố (%v), chuyển sang Gemini 3.5 Flash-Lite...", err)
	}

	// 2. Chuyển sang Gemini thế hệ 3.5 Flash-Lite (hạn mức Free cao)
	if m.geminiKey != "" {
		res, err := m.callGemini(systemInstruction, userMsg)
		if err == nil {
			return res, nil
		}
		log.Printf("[AI] Gemini fallback lỗi: %v", err)
	}

	return nil, fmt.Errorf("tất cả hệ thống AI đều bận")
}

func (m *MultiAIAdvisor) callGroq(sysInst, userMsg string) (*GeminiBotResponse, error) {
	type GroqMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	payload := map[string]interface{}{
		"model": "openai/gpt-oss-20b",
		"messages": []GroqMsg{
			{Role: "system", Content: sysInst},
			{Role: "user", Content: userMsg},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":    0.2,
	}

	b, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+m.groqKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &data); err != nil || len(data.Choices) == 0 {
		return nil, fmt.Errorf("lỗi parse JSON từ Groq")
	}

	var res GeminiBotResponse
	_ = json.Unmarshal([]byte(data.Choices[0].Message.Content), &res)
	if res.Message == "" {
		res.Message = data.Choices[0].Message.Content
	}
	return &res, nil
}

func (m *MultiAIAdvisor) callGemini(sysInst, userMsg string) (*GeminiBotResponse, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(m.geminiKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// Sử dụng gemini-3.5-flash-lite thay thế các model 2.x cũ
	model := client.GenerativeModel("gemini-3.5-flash-lite")
	model.ResponseMIMEType = "application/json"
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(sysInst)},
	}

	resp, err := model.GenerateContent(ctx, genai.Text(userMsg))
	if err != nil {
		return nil, err
	}
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("không có nội dung từ Gemini")
	}

	rawText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			rawText += string(txt)
		}
	}

	rawText = strings.TrimSpace(rawText)
	rawText = strings.TrimPrefix(rawText, "```json")
	rawText = strings.TrimPrefix(rawText, "```")
	rawText = strings.TrimSuffix(rawText, "```")
	rawText = strings.TrimSpace(rawText)

	var res GeminiBotResponse
	if err := json.Unmarshal([]byte(rawText), &res); err != nil {
		res.Message = rawText
	}
	return &res, nil
}
