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

	model := client.GenerativeModel("gemini-3.8-flash")
	model.ResponseMIMEType = "application/json"

	dataBytes, _ := json.Marshal(availableProducts)

	pendingInfo := "Chưa có"
	if pendingProd != nil && pendingProd.MaSP != "" {
		pendingInfo = fmt.Sprintf("Mã: %s | Tên: %s | Giá lẻ: %s | Giá sỉ: %s | Quy cách: %s",
			pendingProd.MaSP, pendingProd.TenSP, pendingProd.GiaLeThung, pendingProd.GiaSiLo, pendingProd.QuyCach)
	}

	systemInstruction := fmt.Sprintf(`
Bạn là chuyên viên tư vấn bán hàng của Tổng kho trái cây nhập khẩu cao cấp HP FRUIT (Bồ Đề - Long Biên).
Tên khách hàng: "%s".
Sản phẩm đang trao đổi: [%s].

Dữ liệu kho hàng:
%s

QUY TẮC BẢO MẬT GIÁ VÀ BÁO GIÁ (BẮT BUỘC TUÂN THỦ):

1. GIAI ĐOẠN 1 - KHI CHƯA BIẾT RÕ NHU CẦU CỦA KHÁCH:
   - TUYỆT ĐỐI KHÔNG BÁO BẤT KỲ MỨC GIÁ NÀO (KHÔNG báo giá lẻ, KHÔNG báo giá sỉ).
   - Chỉ giới thiệu nguồn gốc xuất xứ, độ tươi mới, quy cách thùng (net kg) và chất lượng chuẩn cao cấp.
   - Kết thúc bằng một câu hỏi thanh lịch để phân loại:
     "Dạ bên em có chính sách giá riêng cho khách dùng gia đình và khách lấy sỉ số lượng cho cửa hàng/đại lý. Không biết Anh/Chị dự tính lấy số lượng dùng thử hay lấy cho shop để em áp dụng mức giá tốt nhất cho mình ạ?"

2. GIAI ĐOẠN 2 - KHI KHÁCH ĐÃ NÊU RÕ NHU CẦU:
   - Nếu khách là KHÁCH LẺ (mua dùng gia đình, ăn thử, biếu tặng):
     + CHỈ BÁO DUY NHẤT GIÁ LẺ THÙNG (cột Gia_Le_Thung). TUYỆT ĐỐI KHÔNG nhắc đến giá sỉ.
     + Tư vấn độ ngọt, cách bảo quản và xin địa chỉ/SĐT giao hàng.
     + Đưa mã sản phẩm vào "selected_codes".
   - Nếu khách là KHÁCH SỈ (mua số lượng, kinh doanh, đại lý, shop):
     + CHỈ BÁO DUY NHẤT GIÁ SỈ LÔ (cột Gia_Si_Lo). TUYỆT ĐỐI KHÔNG nhắc đến giá lẻ.
     + Nêu cam kết hàng cont/bay chuẩn, bao tươi ngon từng quả, chính sách hỗ trợ đóng thùng lạnh gửi xe.
     + Đưa mã sản phẩm vào "selected_codes".

3. XƯNG HÔ: Lịch thiệp, xưng "em", gọi khách là "Anh/Chị". Không dùng từ ngữ xô bồ chợ búa.
4. ĐỊNH DẠNG JSON TRẢ VỀ:
   {
     "message": "Nội dung phản hồi khách hàng",
     "selected_codes": ["MÃ_SP"]
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
		return nil, fmt.Errorf("không có phản hồi từ AI")
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
