package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type BotReply struct {
	Message       string
	Product       *Product
	ShouldSendImg bool
}

func normalizeText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(
		"à", "a", "á", "a", "ả", "a", "ã", "a", "ạ", "a",
		"ă", "a", "ằ", "a", "ắ", "a", "ẳ", "a", "ẵ", "a", "ặ", "a",
		"â", "a", "ầ", "a", "ấ", "a", "ẩ", "a", "ẫ", "a", "ậ", "a",
		"è", "e", "é", "e", "ẻ", "e", "ẽ", "e", "ẹ", "e",
		"ê", "e", "ề", "e", "ế", "e", "ể", "e", "ễ", "e", "ệ", "e",
		"ì", "i", "í", "i", "ỉ", "i", "ĩ", "i", "ị", "i",
		"ò", "o", "ó", "o", "ỏ", "o", "õ", "o", "ọ", "o",
		"ô", "o", "ồ", "o", "ố", "o", "ổ", "o", "ỗ", "o", "ộ", "o",
		"ơ", "o", "ờ", "o", "ớ", "o", "ở", "o", "ỡ", "o", "ợ", "o",
		"ù", "u", "ú", "u", "ủ", "u", "ũ", "u", "ụ", "u",
		"ư", "u", "ừ", "u", "ứ", "u", "ử", "u", "ữ", "u", "ự", "u",
		"ỳ", "y", "ý", "y", "ỷ", "y", "ỹ", "y", "ỵ", "y",
		"đ", "d",
	)
	return replacer.Replace(s)
}

// resolveTasteAndTag ưu tiên 100% cột Chat_An (J) và Hinh_Thuc (I) từ Google Sheet
func resolveTasteAndTag(p *Product) (string, string) {
	tag := strings.TrimSpace(p.HinhThuc)
	if tag == "" {
		n := normalizeText(p.TenSP)
		if strings.Contains(n, "bay") || strings.Contains(n, "air") {
			tag = "Hàng Bay (Air Cargo)"
		} else if strings.Contains(n, "viet") || strings.Contains(n, "da lat") || strings.Contains(n, "son la") {
			tag = "Nông sản Việt"
		} else {
			tag = "Hàng Cont lạnh"
		}
	}

	taste := strings.TrimSpace(p.ChatAn)
	// Nếu cột Chat_An trên sheet chưa điền thì mới dùng mô tả dự phòng
	if taste == "" {
		dm := strings.ToLower(p.DanhMuc)
		n := normalizeText(p.TenSP)
		switch {
		case dm == "hong_tao" || strings.Contains(n, "hong tao"):
			taste = "Quả đanh giòn, vị ngọt thanh mát, cắn xốp nhẹ giòn tan."
		case strings.Contains(n, "dazz"):
			taste = "Cơm giòn đanh, ngọt đậm sâu, cắn ngập miệng bao giòn không xốp."
		case strings.Contains(n, "queen"):
			taste = "Cơm giòn thơm nức, vị ngọt đậm đặc trưng dòng Queen New Zealand."
		case strings.Contains(n, "rock"):
			taste = "Quả chắc giòn rụm, ngọt lịm đậm vị chuẩn hàng ống NZL."
		case dm == "tao" || strings.Contains(n, "tao"):
			taste = "Cơm dày giòn đanh, độ đường cao, ăn rất mát và thơm."
		case dm == "quyt" || strings.Contains(n, "quyt"):
			taste = "Tép căng mọng nước, vị ngọt đậm thơm lừng chuẩn hàng tuyển."
		case strings.Contains(dm, "nho") || strings.Contains(n, "nho"):
			taste = "Trái đanh cứng, chùm khít cuống xanh, vị ngọt thơm giòn tan."
		default:
			taste = "Hàng tuyển chuẩn ngon loại 1, bao tươi ngon ngọt từng trái."
		}
	}

	return tag, taste
}

func pickRandom(slice []string) string {
	if len(slice) == 0 {
		return ""
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return slice[r.Intn(len(slice))]
}

func buildCategorySummary(categoryName, label, icon string, available []Product) string {
	var items []string
	for _, p := range available {
		if strings.EqualFold(p.DanhMuc, categoryName) && p.SoLuong > 0 {
			items = append(items, fmt.Sprintf("%s %s (%s)", icon, p.TenSP, p.QuyCach))
		}
	}

	if len(items) == 0 {
		return fmt.Sprintf("Dạ hiện tại kho HP FRUIT đang tạm hết các mã %s mới, hàng đợt tới về em sẽ báo mình ngay nhé ạ!", label)
	}

	return fmt.Sprintf("Dạ hiện tại kho HP FRUIT đang sẵn các dòng %s nhập khẩu tươi ngon sau ạ:\n\n%s\n\nDạ không biết mình đang cần tìm mã %s nào em gửi ảnh chi tiết sản phẩm ạ?",
		label, strings.Join(items, "\n"), strings.ToLower(label))
}

// findSpecificProduct hỗ trợ cả từ viết tắt như dazz, rock, queen, mizuki...
func findSpecificProduct(userMsg string, available []Product) *Product {
	norm := normalizeText(userMsg)

	// Xử lý từ viết tắt phổ biến
	if strings.Contains(norm, "dazz") && !strings.Contains(norm, "dazzle") {
		norm += " dazzle"
	}
	if strings.Contains(norm, "rock") && !strings.Contains(norm, "rockit") {
		norm += " rockit"
	}

	var bestProd *Product
	highestScore := 0

	for i := range available {
		p := &available[i]
		pNorm := normalizeText(p.TenSP)
		pCode := strings.ToLower(p.MaSP)
		pCat := strings.ToLower(p.DanhMuc)

		score := 0

		// Khớp mã sản phẩm
		if pCode != "" && strings.Contains(norm, pCode) {
			score += 100
		}

		// Khớp trọn vẹn tên
		if strings.Contains(norm, pNorm) {
			score += 60
		}

		// Khớp danh mục
		if pCat != "" && strings.Contains(norm, pCat) {
			score += 15
		}

		// Tách từ khóa
		words := strings.Fields(norm)
		for _, w := range words {
			if len([]rune(w)) <= 1 {
				continue
			}
			if strings.Contains(pNorm, w) {
				if w != "tao" && w != "nho" && w != "cam" && w != "quyt" && w != "co" && w != "khong" && w != "gia" && w != "bao" && w != "nhieu" {
					score += 35 // Điểm rất cao cho các từ định danh như dazz, dazzle, queen, koru, fresh...
				} else {
					score += 5
				}
			}
		}

		// Tránh lẫn táo với hồng táo
		if strings.Contains(norm, "hong tao") && pCat != "hong_tao" {
			score -= 50
		}
		if !strings.Contains(norm, "hong tao") && strings.Contains(norm, "tao") && pCat == "hong_tao" {
			score -= 50
		}

		if score > highestScore {
			highestScore = score
			bestProd = p
		}
	}

	if highestScore >= 25 {
		return bestProd
	}
	return nil
}

func ProcessCustomerMessage(userMsg string, sess *UserSession, available []Product) *BotReply {
	norm := normalizeText(userMsg)

	retailKeywords := []string{"le", "mua an", "an thu", "dung thu", "gia dinh", "1 thung", "may can", "an", "le thung", "hop", "mua an thu"}
	wholesaleKeywords := []string{"si", "gia si", "lay si", "buon", "dai ly", "shop", "kinh doanh", "so luong", "vao so luong", "lo", "tuyen si", "xe hang"}

	isRetail := false
	for _, kw := range retailKeywords {
		if strings.Contains(norm, kw) {
			isRetail = true
			break
		}
	}

	isWholesale := false
	for _, kw := range wholesaleKeywords {
		if strings.Contains(norm, kw) {
			isWholesale = true
			break
		}
	}

	// TRƯỜNG HỢP 1: Khách đang xác nhận nhu cầu (sỉ hay lẻ) cho mã quả đã lưu trong phiên
	if sess != nil && sess.LastProduct.MaSP != "" && (isRetail || isWholesale) {
		p := sess.LastProduct
		tag, taste := resolveTasteAndTag(&p)

		if isRetail {
			msg := fmt.Sprintf("Dạ em gửi Anh/Chị thông tin lô hàng chuẩn ngon bên em ạ:\n"+
				"✨ Sản phẩm: %s\n"+
				"🏷️ Hình thức: %s\n"+
				"📦 Quy cách: %s\n"+
				"💰 Giá lẻ thùng: %s\n"+
				"🍇 Hương vị / Chất ăn: %s\n\n"+
				"Anh/Chị lấy mấy thùng để em lên đơn giao sớm cho mình ạ?",
				p.TenSP, tag, p.QuyCach, p.GiaLeThung, taste)

			return &BotReply{Message: msg, Product: &p, ShouldSendImg: false}
		}

		if isWholesale {
			msg := fmt.Sprintf("Dạ em gửi Anh/Chị chính sách giá sỉ ưu đãi cho đại lý/shop bên em:\n"+
				"✨ Sản phẩm: %s\n"+
				"🏷️ Tiêu chuẩn: %s\n"+
				"📦 Quy cách: %s\n"+
				"💰 Giá sỉ lô: %s\n"+
				"🍇 Hương vị / Chất ăn: %s\n\n"+
				"Anh/Chị dự tính vào số lượng bao nhiêu thùng để em chuẩn bị gửi xe ạ?",
				p.TenSP, tag, p.QuyCach, p.GiaSiLo, taste)

			return &BotReply{Message: msg, Product: &p, ShouldSendImg: false}
		}
	}

	// TRƯỜNG HỢP 2: Khách hỏi tên quả cụ thể (kể cả hỏi kèm "giá bao nhiêu", viết tắt "dazz", "queen"...)
	matchedProd := findSpecificProduct(userMsg, available)

	if matchedProd != nil {
		sess.LastProduct = *matchedProd
		tag, taste := resolveTasteAndTag(matchedProd)

		// Nếu trong câu hỏi khách đã nói luôn là mua lẻ
		if isRetail {
			msg := fmt.Sprintf("Dạ em gửi Anh/Chị thông tin lô hàng chuẩn ngon bên em ạ:\n"+
				"✨ Sản phẩm: %s\n"+
				"🏷️ Hình thức: %s\n"+
				"📦 Quy cách: %s\n"+
				"💰 Giá lẻ thùng: %s\n"+
				"🍇 Hương vị / Chất ăn: %s\n\n"+
				"Anh/Chị lấy mấy thùng để em lên đơn giao sớm cho mình ạ?",
				matchedProd.TenSP, tag, matchedProd.QuyCach, matchedProd.GiaLeThung, taste)
			return &BotReply{Message: msg, Product: matchedProd, ShouldSendImg: true}
		}

		// Nếu trong câu hỏi khách đã nói luôn là lấy sỉ
		if isWholesale {
			msg := fmt.Sprintf("Dạ em gửi Anh/Chị chính sách giá sỉ ưu đãi cho đại lý/shop bên em:\n"+
				"✨ Sản phẩm: %s\n"+
				"🏷️ Tiêu chuẩn: %s\n"+
				"📦 Quy cách: %s\n"+
				"💰 Giá sỉ lô: %s\n"+
				"🍇 Hương vị / Chất ăn: %s\n\n"+
				"Anh/Chị dự tính vào số lượng bao nhiêu thùng để em chuẩn bị gửi xe ạ?",
				matchedProd.TenSP, tag, matchedProd.QuyCach, matchedProd.GiaSiLo, taste)
			return &BotReply{Message: msg, Product: matchedProd, ShouldSendImg: true}
		}

		// Khách chỉ hỏi giá chung ("dazz giá bao nhiêu"): Đưa chất ăn và tập trung phân loại nhu cầu
		msg := fmt.Sprintf("Dạ bên em sẵn %s (%s) hàng về tươi mới chuẩn loại 1 ạ.\n"+
			"🍇 Hương vị / Chất ăn: %s\n\n"+
			"Bên em có chính sách giá riêng cho khách ăn gia đình và khách lấy sỉ cho shop/đại lý. Không biết Anh/Chị dự tính lấy dùng gia đình hay lấy cho shop để em báo giá tốt nhất cho mình ạ?",
			matchedProd.TenSP, tag, taste)

		return &BotReply{Message: msg, Product: matchedProd, ShouldSendImg: true}
	}

	// TRƯỜNG HỢP 3: Khách chỉ hỏi trống không "giá bao nhiêu" mà trước đó ĐÃ HỎI 1 quả
	if (strings.Contains(norm, "gia bao nhieu") || strings.Contains(norm, "bao nhieu") || strings.Contains(norm, "xin gia")) && sess != nil && sess.LastProduct.MaSP != "" {
		p := sess.LastProduct
		tag, taste := resolveTasteAndTag(&p)
		msg := fmt.Sprintf("Dạ lô %s (%s) bên em chất ăn rất ngon: %s\n\n"+
			"Dạ Anh/Chị đang dự tính lấy ăn gia đình hay lấy số lượng cho shop để em báo giá chuẩn nhất cho mình ạ?",
			p.TenSP, tag, taste)
		return &BotReply{Message: msg, Product: &p, ShouldSendImg: false}
	}

	// TRƯỜNG HỢP 4: Khách hỏi chung theo nhóm danh mục
	if strings.Contains(norm, "hong tao") {
		return &BotReply{Message: buildCategorySummary("hong_tao", "Hồng Táo", "🍎", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "tao") {
		return &BotReply{Message: buildCategorySummary("tao", "Táo", "🍎", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "nho sua") {
		return &BotReply{Message: buildCategorySummary("nho_sua", "Nho Sữa", "🍇", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "nho") {
		return &BotReply{Message: buildCategorySummary("nho", "Nho", "🍇", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "cam") {
		return &BotReply{Message: buildCategorySummary("cam", "Cam", "🍊", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "quyt") {
		return &BotReply{Message: buildCategorySummary("quyt", "Quýt", "🍊", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "dua") {
		return &BotReply{Message: buildCategorySummary("dua", "Dưa", "🍈", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "kiwi") {
		return &BotReply{Message: buildCategorySummary("kiwi", "Kiwi", "🥝", available), Product: nil, ShouldSendImg: false}
	}

	// Chào hỏi mặc định
	return &BotReply{
		Message: "Dạ Tổng kho trái cây nhập khẩu & Nông sản HP FRUIT (Bồ Đề - Long Biên) xin chào Anh/Chị ạ! Bên em sẵn rất nhiều mã hoa quả chuẩn hàng bay, hàng cont và nông sản sạch tươi ngon mỗi ngày. Anh/Chị đang quan tâm dòng quả nào để em gửi hình ảnh và báo giá chi tiết ạ?",
		Product: nil,
		ShouldSendImg: false,
	}
}
