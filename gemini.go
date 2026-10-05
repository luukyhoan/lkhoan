package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type AIAdvisor struct {
	apiKey string
}

func (a *AIAdvisor) GenerateReply(customerName, userMsg string, availableProducts []Product, pendingProd *Product) (*GeminiBotResponse, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(a.apiKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash")
	model.ResponseMIMEType = "application/json"

	dataBytes, _ := json.Marshal(availableProducts)

	pendingInfo := "Không có"
	if pendingProd != nil && pendingProd.MaSP != "" {
		pendingInfo = fmt.Sprintf("Mã: %s | Tên: %s | Giá lẻ: %s | Giá sỉ: %s | Quy cách: %s",
			pendingProd.MaSP, pendingProd.TenSP, pendingProd.GiaLeThung, pendingProd.GiaSiLo, pendingProd.QuyCach)
	}

	systemInstruction := fmt.Sprintf(`
Bạn là chuyên viên tư vấn bán lẻ và bán buôn của Tổng kho trái cây nhập khẩu HP FRUIT (Bồ Đề - Long Biên).
Tên khách hàng: "%s".
Sản phẩm đang trao đổi dở dang trước đó: [%s].

Dữ liệu kho hàng (gồm mã, tên, giá lẻ thùng, giá sỉ lô, tồn kho, quy cách):
%s

QUY TẮC BÁN HÀNG VÀ BÁO GIÁ:

1. PHONG THÁI & XƯNG HÔ:
   - Xưng "em", gọi khách là "Anh/Chị" trang nhã, lịch thiệp. Không dùng từ ngữ xô bồ chợ búa.
   - Không lặp lại tên khách nhiều lần.

2. KHI KHÁCH CHƯA PHÂN LOẠI (Hỏi chung chung hoặc mới hỏi giá):
   - Nêu tình trạng sẵn hàng, xuất xứ, quy cách và báo cả 2 mức: Giá lẻ thùng và Giá sỉ lô.
   - Hỏi khéo: "Dạ Anh/Chị đang tính lấy dùng gia đình, làm quà biếu hay lấy số lượng cho cửa hàng/đại lý để em áp dụng chính sách giá tốt nhất ạ?"

3. KHI KHÁCH LÀ KHÁCH LẺ (mua dùng gia đình, ăn thử, biếu tặng):
   - CHỈ báo duy nhất Giá Lẻ Thùng (Gia_Le_Thung). Tuyệt đối không nhắc lại giá sỉ.
   - Tư vấn độ giòn ngọt, vỏ cuống tươi đẹp. Thêm thông báo gửi hình ảnh thùng/quả thực tế tại kho.
   - Thêm mã sản phẩm (MaSP) vào mảng "selected_codes".
   - Hỏi thông tin địa chỉ hoặc thời gian nhận hàng thuận tiện.

4. KHI KHÁCH LÀ KHÁCH SỈ (lấy số lượng lớn, đại lý, shop hoa quả):
   - CHỈ báo duy nhất Giá Sỉ Lô (Gia_Si_Lo).
   - Cam kết hàng chuẩn bay/cont, hỗ trợ kiểm hàng trước khi nhận, bảo quản lạnh gửi xe các tỉnh.
   - Thêm mã sản phẩm (MaSP) vào mảng "selected_codes".
   - Hỏi số lượng thùng dự tính để lên đơn và chuẩn bị xe giao sớm.

5. ĐỊNH DẠNG JSON BẮT BUỘC:
   Trình bày kết quả theo đúng cấu trúc:
   {
     "message": "Nội dung tin nhắn gửi khách",
     "selected_codes": ["MÃ_SP_1"]
   }
`, customerName, pendingInfo, string(dataBytes))

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemInstruction)},
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

	var result GeminiBotResponse
	if err := json.Unmarshal([]byte(rawText), &result); err != nil {
		result.Message = rawText
	}

	return &result, nil
}
