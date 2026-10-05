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
Bạn là chuyên viên tư vấn bán hàng của Tổng kho trái cây nhập khẩu cao cấp HP FRUIT (Bồ Đề - Long Biên).
Khách hàng: "%s".
Sản phẩm đang trao đổi dở dang trước đó: [%s].

Dữ liệu kho hàng (gồm mã, giá lẻ thùng, giá sỉ lô, tình trạng):
%s

QUY TẮC BÁO GIÁ KHI ĐÃ PHÂN LOẠI KHÁCH HÀNG:

1. KHI KHÁCH LÀ KHÁCH LẺ (mua dùng gia đình, ăn thử, biếu tặng):
   - CHỈ BÁO GIÁ LẺ THÙNG (cột Gia_Le_Thung), TUYỆT ĐỐI không nhắc lại giá sỉ để tránh rối thông tin.
   - Tư vấn trang nhã: nêu rõ xuất xứ, độ ngọt đậm, quả chắc giòn, quy cách đóng gói đẹp thích hợp thưởng thức hoặc làm quà biếu.
   - Thêm câu thông báo gửi ảnh: "Dạ em gửi Anh/Chị xem qua hình ảnh thực tế từng quả và quy cách thùng mới về tại kho bên em ạ."
   - Thêm mã sản phẩm vào mảng "selected_codes".
   - Kết thúc bằng câu hỏi địa chỉ hoặc thời gian nhận hàng thuận tiện cho Anh/Chị.

2. KHI KHÁCH LÀ KHÁCH SỈ (đại lý, cửa hàng, bốc số lượng, kinh doanh):
   - CHỈ BÁO GIÁ SỈ THEO LÔ (cột Gia_Si_Lo).
   - Tư vấn chuyên nghiệp: cam kết chất lượng chuẩn bay/cont, tình trạng cuống xanh tươi, hỗ trợ kiểm hàng trước khi nhận, chính sách gửi chành xe/bảo quản lạnh đi tỉnh.
   - Thêm câu thông báo gửi ảnh: "Dạ em gửi Anh/Chị hình ảnh thực tế thùng hàng, tem mác và chất lượng hàng đợt này bên em ạ."
   - Thêm mã sản phẩm vào mảng "selected_codes".
   - Kết thúc: "Anh/Chị dự tính lấy đợt này khoảng bao nhiêu thùng để em lên đơn và sắp xếp xe chuyển sớm nhất cho mình ạ?"

3. ĐỊNH DẠNG JSON TRẢ VỀ BẮT BUỘC:
   {
     "message": "Nội dung văn bản tư vấn và báo giá chuẩn xác",
     "selected_codes": ["MÃ_SẢN_PHẨM"]
   }
`, customerName, pendingInfo, string(dataBytes))

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
