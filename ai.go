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

type CompactProduct struct {
	MaSP       string `json:"ma"`
	TenSP      string `json:"ten"`
	QuyCach    string `json:"qc"`
	GiaLeThung string `json:"gia_le"`
	GiaSiLo    string `json:"gia_si"`
}

func (m *MultiAIAdvisor) GenerateReply(customerName, userMsg string, availableProducts []Product, pendingProd *Product) (*GeminiBotResponse, error) {
	var compactList []CompactProduct
	if pendingProd != nil && pendingProd.MaSP != "" {
		compactList = append(compactList, CompactProduct{
			MaSP:       pendingProd.MaSP,
			TenSP:      pendingProd.TenSP,
			QuyCach:    pendingProd.QuyCach,
			GiaLeThung: pendingProd.GiaLeThung,
			GiaSiLo:    pendingProd.GiaSiLo,
		})
	} else {
		limit := len(availableProducts)
		if limit > 8 {
			limit = 8
		}
		for i := 0; i < limit; i++ {
			p := availableProducts[i]
			compactList = append(compactList, CompactProduct{
				MaSP:       p.MaSP,
				TenSP:      p.TenSP,
				QuyCach:    p.QuyCach,
				GiaLeThung: p.GiaLeThung,
				GiaSiLo:    p.GiaSiLo,
			})
		}
	}

	dataBytes, _ := json.Marshal(compactList)

	pendingInfo := "Chưa có"
	if pendingProd != nil && pendingProd.MaSP != "" {
		pendingInfo = fmt.Sprintf("Mã: %s | Tên: %s | Giá lẻ: %s | Giá sỉ: %s",
			pendingProd.MaSP, pendingProd.TenSP, pendingProd.GiaLeThung, pendingProd.GiaSiLo)
	}

	systemInstruction := fmt.Sprintf(`Bạn là tư vấn viên HP FRUIT (Tổng kho hoa quả nhập khẩu Bồ Đề - Long Biên).
Khách hàng: "%s".
Sản phẩm đang trao đổi: [%s].
Dữ liệu: %s

QUY TẮC BÁO GIÁ VÀ ĐỊNH DẠNG:

1. KHI CHƯA BIẾT NHU CẦU CỦA KHÁCH:
   - TUYỆT ĐỐI KHÔNG BÁO GIÁ LẺ HOẶC GIÁ SỈ.
   - Giới thiệu ngắn gọn độ tươi ngon và hỏi:
     "Dạ bên em sẵn hàng tươi mới chuẩn loại 1 ạ. Anh/Chị dự tính lấy số lượng dùng thử ăn gia đình hay lấy cho shop để em báo giá tốt nhất cho mình ạ?"

2. KHI KHÁCH LÀ KHÁCH MUA LẺ (ăn gia đình, mua thử, biếu tặng):
   BẮT BUỘC TRÌNH BÀY CHÍNH XÁC THEO MẪU SAU (KHÔNG nhắc giá sỉ, đưa mã SP vào selected_codes):

Dạ em gửi Anh/Chị thông tin lô hàng chuẩn ngon bên em ạ:
✨ Sản phẩm: [Tên SP]
📦 Quy cách: [Quy cách net kg]
💰 Giá lẻ thùng: [Giá gia_le]
🍇 Hương vị / Chất ăn: [Mô tả ngắn gọn 1 câu: ngọt đậm, giòn tan, tép mọng nước, chuẩn air...]
Anh/Chị lấy mấy thùng để em lên đơn giao sớm cho mình ạ?

3. KHI KHÁCH LÀ KHÁCH MUA SỈ (kinh doanh, mở shop, đại lý, mua số lượng):
   BẮT BUỘC TRÌNH BÀY CHÍNH XÁC THEO MẪU SAU (KHÔNG nhắc giá lẻ, đưa mã SP vào selected_codes):

Dạ em gửi Anh/Chị chính sách giá sỉ ưu đãi cho đại lý/shop bên em:
✨ Sản phẩm: [Tên SP]
📦 Quy cách: [Quy cách net kg]
💰 Giá sỉ lô: [Giá gia_si]
🍇 Hương vị / Chất ăn: [Mô tả ngắn gọn chất lượng: hàng cont/bay tươi cứng, bao đẹp từng thùng...]
Anh/Chị dự tính vào số lượng bao nhiêu thùng để em chuẩn bị gửi xe ạ?

4. ĐỊNH DẠNG JSON TRẢ VỀ:
{"message": "nội dung tin nhắn", "selected_codes": ["MÃ"]}`, customerName, pendingInfo, string(dataBytes))

	// 1. Thử gọi Groq LPU
	if m.groqKey != "" {
		res, err := m.callGroq(systemInstruction, userMsg)
		if err == nil {
			return res, nil
		}
		log.Printf("[AI] Groq bận (%v), chuyển sang Gemini...", err)
	}

	// 2. Chuyển sang Gemini dự phòng
	if m.geminiKey != "" {
		res, err := m.callGemini(systemInstruction, userMsg)
		if err == nil {
			return res, nil
		}
		log.Printf("[AI] Gemini bận: %v", err)
	}

	return nil, fmt.Errorf("hệ thống AI đang bận")
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
		"temperature":    0.1,
		"max_tokens":     300,
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
		return nil, fmt.Errorf("lỗi đọc JSON từ Groq")
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

	model := client.GenerativeModel("gemini-2.5-flash")
	model.ResponseMIMEType = "application/json"
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(sysInst)},
	}

	resp, err := model.GenerateContent(ctx, genai.Text(userMsg))
	if err != nil {
		return nil, err
	}
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("không có phản hồi từ Gemini")
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
