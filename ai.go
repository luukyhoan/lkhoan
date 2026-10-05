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

// compactProduct rút gọn thông tin chỉ giữ lại các trường quan trọng để tiết kiệm token
type CompactProduct struct {
	MaSP       string `json:"ma"`
	TenSP      string `json:"ten"`
	QuyCach    string `json:"qc"`
	GiaLeThung string `json:"gia_le"`
	GiaSiLo    string `json:"gia_si"`
}

func (m *MultiAIAdvisor) GenerateReply(customerName, userMsg string, availableProducts []Product, pendingProd *Product) (*GeminiBotResponse, error) {
	// Lọc gọn danh mục: nếu có sản phẩm đang nói dở, ưu tiên tập trung vào nó để tiết kiệm token
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
		// Nếu chưa có, chỉ lấy tối đa 10 sản phẩm tiêu biểu
		limit := len(availableProducts)
		if limit > 10 {
			limit = 10
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

	systemInstruction := fmt.Sprintf(`Bạn là nhân viên tư vấn bán hàng của Tổng kho trái cây nhập khẩu HP FRUIT (Bồ Đề - Long Biên).
Khách hàng: "%s". Sản phẩm quan tâm: [%s].
Bảng giá: %s

QUY TẮC:
1. NẾU CHƯA BIẾT NHU CẦU: Tuyệt đối KHÔNG báo giá lẻ/sỉ. Giới thiệu chất lượng và hỏi khách mua dùng gia đình hay kinh doanh shop để áp dụng giá tốt.
2. NẾU KHÁCH LẺ (mua ăn/biếu): Chỉ báo GIÁ LẺ THÙNG (gia_le). Không nhắc giá sỉ. Đưa mã SP vào selected_codes.
3. NẾU KHÁCH SỈ (lấy số lượng/kinh doanh): Chỉ báo GIÁ SỈ LÔ (gia_si). Không nhắc giá lẻ. Đưa mã SP vào selected_codes.
4. Trả về JSON: {"message": "nội dung trả lời", "selected_codes": ["MÃ"]}`, customerName, pendingInfo, string(dataBytes))

	// 1. Chạy Groq với max_tokens=300 để không vượt giới hạn TPM
	if m.groqKey != "" {
		res, err := m.callGroq(systemInstruction, userMsg)
		if err == nil {
			return res, nil
		}
		log.Printf("[AI] Groq gặp sự cố (%v), chuyển sang Gemini...", err)
	}

	// 2. Chuyển sang Gemini dự phòng nếu Groq lỗi
	if m.geminiKey != "" {
		res, err := m.callGemini(systemInstruction, userMsg)
		if err == nil {
			return res, nil
		}
		log.Printf("[AI] Gemini cũng gặp sự cố: %v", err)
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
		"temperature":    0.2,
		"max_tokens":     350, // Giới hạn token đầu ra để kiểm soát mức TPM
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
