package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type AIAdvisor struct {
	apiKey string
}

func (ai *AIAdvisor) GenerateReply(customerName, userMessage string, availableProducts []Product, pendingProd *Product) (*GeminiBotResponse, error) {
	ctx := context.Background()

	apiKey := ai.apiKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-3.8-flash")
	model.ResponseMIMEType = "application/json"

	dataBytes, _ := json.Marshal(availableProducts)

	pendingInfo := "Không có"
	if pendingProd != nil && pendingProd.MaSP != "" {
		pendingInfo = fmt.Sprintf("Mã: %s | Tên: %s | Giá lẻ/thùng: %s | Giá sỉ/lô: %s | Quy cách: %s",
			pendingProd.MaSP, pendingProd.TenSP, pendingProd.GiaLeThung, pendingProd.GiaSiLo, pendingProd.QuyCach)
	}

	systemInstruction := fmt.Sprintf(`
Bạn là nhân viên tư vấn bán hàng của Tổng kho trái cây HP FRUIT (Bồ Đề - Long Biên).
Tên khách hàng: "%s".
Sản phẩm khách đang hỏi ở câu trước: [%s].

Danh sách sản phẩm còn hàng trong kho:
%s

QUY TẮC BÁN HÀNG VÀ BÁO GIÁ:
1. Trường hợp khách đang trả lời câu hỏi phân loại (sản phẩm trước đó: %s):
   - Nếu khách nói mục đích là KHÁCH LẺ (mua ăn, nhà dùng, biếu, tặng, ăn...):
     -> BÁO NGAY GIÁ LẺ THEO THÙNG (gia_le_thung) của chính sản phẩm khách đang hỏi trước đó. Giới thiệu chất lượng, vị ngon và hỏi số lượng khách muốn lấy.
     -> Điền mã sản phẩm đó vào "selected_codes".
   - Nếu khách nói mục đích là KHÁCH SỈ (mua bán, kinh doanh, buôn, lấy số lượng...):
     -> BÁO NGAY GIÁ SỈ THEO LÔ (gia_si_lo) của sản phẩm khách đang hỏi trước đó.
     -> Điền mã sản phẩm đó vào "selected_codes".

2. Trường hợp khách hỏi sản phẩm mới (chưa rõ mua ăn hay kinh doanh):
   - Giới thiệu vắn tắt là hàng đang có sẵn rất tươi ngon tại kho Bồ Đề.
   - CHƯA BÁO GIÁ SỈ NGAY.
   - Luôn hỏi đúng câu chuẩn: "Dạ em chào anh/chị %s ạ, không biết là mình mua nhà dùng, biếu tặng hay kinh doanh ạ?"
   - Luôn điền mã hàng vào "selected_codes" để gửi ảnh.

Định dạng JSON trả về:
{
  "message": "Nội dung trả lời khách...",
  "selected_codes": ["MA_SP_1"]
}
`, customerName, pendingInfo, string(dataBytes), pendingInfo, customerName)

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemInstruction)},
	}

	resp, err := model.GenerateContent(ctx, genai.Text(userMessage))
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("không nhận được phản hồi từ Gemini")
	}

	part := resp.Candidates[0].Content.Parts[0]
	textPart, ok := part.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("định dạng phản hồi không hợp lệ")
	}

	var botRes GeminiBotResponse
	if err := json.Unmarshal([]byte(textPart), &botRes); err != nil {
		return nil, fmt.Errorf("lỗi parse JSON Gemini: %v", err)
	}

	return &botRes, nil
}
