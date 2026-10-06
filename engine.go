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

func resolveTasteAndTag(p *Product) (string, string) {
	tag := p.HinhThuc
	if tag == "" {
		n := normalizeText(p.TenSP)
		if strings.Contains(n, "bay") || strings.Contains(n, "air") {
			tag = "Hàng Bay (Air Cargo) tươi rói vừa đáp"
		} else if strings.Contains(n, "viet") || strings.Contains(n, "da lat") || strings.Contains(n, "son la") {
			tag = "Nông sản Việt tuyển chọn tận vườn"
		} else {
			tag = "Hàng Cont lạnh chuẩn loại 1"
		}
	}

	taste := p.ChatAn
	if taste == "" {
		dm := strings.ToLower(p.DanhMuc)
		n := normalizeText(p.TenSP)
		switch {
		case dm == "hong_tao" || strings.Contains(n, "hong tao"):
			taste = "Quả đanh giòn, vị ngọt thanh mát, cắn xốp nhẹ giòn tan."
		case dm == "tao" || strings.Contains(n, "tao"):
			taste = "Cơm dày giòn đanh, vị ngọt đậm sâu, cắn ngập miệng bao giòn không xốp."
		case dm == "quyt" || strings.Contains(n, "quyt"):
			taste = "Tép căng mọng nước, vị ngọt đậm thơm lừng, bóc vỏ dóc không dập."
		case strings.Contains(dm, "nho") || strings.Contains(n, "nho"):
			taste = "Trái đanh cứng, chùm khít cuống xanh, vị ngọt thơm giòn tan."
		case dm == "le" || strings.Contains(n, "le"):
			taste = "Thịt trắng phau, giòn rụm nhiều nước, vị ngọt thanh mát."
		case dm == "kiwi" || strings.Contains(n, "kiwi"):
			taste = "Ruột vàng mọng nước, ngọt dịu thanh mát chuẩn vị New Zealand."
		case dm == "cherry" || strings.Contains(n, "cherry"):
			taste = "Size VIP trái to thẫm màu, cuống xanh giòn ngọt đẫm vị."
		default:
			taste = "Hàng tuyển tươi ngon, bao chuẩn chất lượng ngọt mát từng quả."
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

// buildCategorySummary tạo thực đơn ngắn khi khách chỉ hỏi chung chung tên nhóm quả
func buildCategorySummary(categoryName, label string, available []Product) string {
	var items []string
	for _, p := range available {
		// Chỉ lấy mã thuộc danh mục đó và CÒN HÀNG (SoLuong > 0)
		if strings.EqualFold(p.DanhMuc, categoryName) && p.SoLuong > 0 {
			items = append(items, fmt.Sprintf("🔹 %s (%s)", p.TenSP, p.QuyCach))
			if len(items) >= 6 { // Lấy tối đa 6 dòng nổi bật nhất để không làm rối mắt
				break
			}
		}
	}

	if len(items) == 0 {
		return fmt.Sprintf("Dạ hiện tại các mã %s bên em đang tạm hết hàng mới, đợt tới về em báo mình ngay nhé ạ!", label)
	}

	return fmt.Sprintf("Dạ hiện tại kho HP FRUIT đang sẵn các dòng %s tươi mới chuẩn ngon sau ạ:\n\n%s\n\nAnh/Chị đang quan tâm loại nào trong danh sách trên để em gửi hình ảnh và báo giá chi tiết cho mình ạ?",
		label, strings.Join(items, "\n"))
}

func findSpecificProduct(userMsg string, available []Product) *Product {
	norm := normalizeText(userMsg)

	targetCategory := ""
	if strings.Contains(norm, "hong tao") {
		targetCategory = "hong_tao"
	} else if strings.Contains(norm, "tao") {
		targetCategory = "tao"
	} else if strings.Contains(norm, "nho sua") {
		targetCategory = "nho_sua"
	} else if strings.Contains(norm, "nho") {
		targetCategory = "nho"
	} else if strings.Contains(norm, "quyt") {
		targetCategory = "quyt"
	} else if strings.Contains(norm, "cam") {
		targetCategory = "cam"
	}

	var bestProd *Product
	highestScore := 0

	for i := range available {
		p := &available[i]
		pNorm := normalizeText(p.TenSP)
		pCode := strings.ToLower(p.MaSP)
		pCat := strings.ToLower(p.DanhMuc)

		if targetCategory == "tao" && pCat == "hong_tao" {
			continue
		}
		if targetCategory == "hong_tao" && pCat != "hong_tao" {
			continue
		}

		score := 0

		if pCode != "" && strings.Contains(norm, pCode) {
			score += 100
		}
		if strings.Contains(norm, pNorm) {
			score += 60
		}

		words := strings.Fields(norm)
		for _, w := range words {
			if len([]rune(w)) <= 1 {
				continue
			}
			if strings.Contains(pNorm, w) {
				// Điểm cộng lớn cho các tên riêng đặc thù
				if w != "tao" && w != "nho" && w != "cam" && w != "quyt" && w != "co" && w != "khong" {
					score += 30
				} else {
					score += 5
				}
			}
		}

		if score > highestScore {
			highestScore = score
			bestProd = p
		}
	}

	// Đạt điểm tối thiểu 30 mới xem là đã xác định đích danh một loại quả
	if highestScore >= 30 {
		return bestProd
	}
	return nil
}

func ProcessCustomerMessage(userMsg string, sess *UserSession, available []Product) *BotReply {
	norm := normalizeText(userMsg)

	retailKeywords := []string{"le", "mua an", "an thu", "dung thu", "gia dinh", "1 thung", "may can", "an", "le thung", "hop", "an thu xem"}
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

	// 1. Phản hồi nhu cầu sỉ/lẻ cho sản phẩm khách đang trao đổi dở
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

	// 2. Kiểm tra xem khách đang hỏi đích danh 1 mã hay hỏi chung chung một danh mục
	matchedProd := findSpecificProduct(userMsg, available)

	if matchedProd != nil {
		sess.LastProduct = *matchedProd
		tag, taste := resolveTasteAndTag(matchedProd)

		if isRetail {
			msg := fmt.Sprintf("Dạ em gửi Anh/Chị thông tin lô hàng chuẩn ngon bên em ạ:\n"+
				"✨ Sản phẩm: %s\n"+
				"🏷️️ Hình thức: %s\n"+
				"📦 Quy cách: %s\n"+
				"💰 Giá lẻ thùng: %s\n"+
				"🍇 Hương vị / Chất ăn: %s\n\n"+
				"Anh/Chị lấy mấy thùng để em lên đơn giao sớm cho mình ạ?",
				matchedProd.TenSP, tag, matchedProd.QuyCach, matchedProd.GiaLeThung, taste)
			return &BotReply{Message: msg, Product: matchedProd, ShouldSendImg: true}
		}

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

		msg := fmt.Sprintf("Dạ bên em sẵn %s (%s) hàng về tươi mới chuẩn loại 1 ạ.\n\n"+
			"Bên em có chính sách giá ưu đãi riêng cho khách ăn gia đình và khách lấy sỉ cho cửa hàng/đại lý. Không biết Anh/Chị dự tính lấy số lượng dùng thử hay lấy cho shop để em báo giá tốt nhất cho mình ạ?",
			matchedProd.TenSP, tag)

		return &BotReply{Message: msg, Product: matchedProd, ShouldSendImg: true}
	}

	// 3. Khách hỏi chung chung theo danh mục -> Gửi menu danh sách rút gọn (không bắn ảnh ồ ạt)
	if strings.Contains(norm, "hong tao") {
		return &BotReply{Message: buildCategorySummary("hong_tao", "Hồng Táo", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "tao") {
		return &BotReply{Message: buildCategorySummary("tao", "Táo", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "nho sua") {
		return &BotReply{Message: buildCategorySummary("nho_sua", "Nho Sữa", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "nho") {
		return &BotReply{Message: buildCategorySummary("nho", "Nho", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "cam") {
		return &BotReply{Message: buildCategorySummary("cam", "Cam", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "quyt") {
		return &BotReply{Message: buildCategorySummary("quyt", "Quýt", available), Product: nil, ShouldSendImg: false}
	}

	// 4. Chào hỏi mặc định
	return &BotReply{
		Message: "Dạ Tổng kho trái cây nhập khẩu & Nông sản HP FRUIT (Bồ Đề - Long Biên) xin chào Anh/Chị ạ! Bên em sẵn rất nhiều mã hoa quả chuẩn hàng bay, hàng cont và nông sản sạch tươi ngon mỗi ngày. Anh/Chị đang quan tâm dòng quả nào để em gửi hình ảnh và báo giá chi tiết ạ?",
		Product: nil,
		ShouldSendImg: false,
	}
}
