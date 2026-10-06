package main

import (
	"fmt"
	"strings"
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
		",", " ", ".", " ", "?", " ", "!", " ", "-", " ", "/", " ",
	)
	return replacer.Replace(s)
}

func expandAliases(norm string) string {
	words := strings.Fields(norm)
	var expanded []string

	aliasMap := map[string]string{
		"daz":    "dazzle",
		"dazz":   "dazzle",
		"dazle":  "dazzle",
		"roc":    "rockit",
		"rokit":  "rockit",
		"rock":   "rockit",
		"quen":   "queen",
		"qen":    "queen",
		"kor":    "koru",
		"mizu":   "mizuki",
		"gins":   "ginseng",
		"env":    "envy",
		"cher":   "cherry",
		"ever":   "evercrip",
		"posy":   "posy",
		"delec":  "delecta",
		"namphi": "nam phi",
	}

	for _, w := range words {
		expanded = append(expanded, w)
		if full, ok := aliasMap[w]; ok {
			expanded = append(expanded, full)
		}
	}
	return strings.Join(expanded, " ")
}

// resolveTaste ưu tiên 100% cột Chat_An (J) từ Google Sheet
func resolveTaste(p *Product) string {
	taste := strings.TrimSpace(p.ChatAn)
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
	return taste
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

func buildHinhThucSummary(keyword, title, icon string, available []Product) string {
	var items []string
	for _, p := range available {
		normHT := normalizeText(p.HinhThuc)
		if strings.Contains(normHT, keyword) && p.SoLuong > 0 {
			items = append(items, fmt.Sprintf("%s %s (%s)", icon, p.TenSP, p.QuyCach))
		}
	}

	if len(items) == 0 {
		return fmt.Sprintf("Dạ hiện tại kho HP FRUIT đang tạm hết các mã %s sẵn kho, hàng đợt tới về em báo mình ngay nhé ạ!", title)
	}

	return fmt.Sprintf("Dạ kho HP FRUIT đang sẵn các mã %s tươi mới cập kho sau ạ:\n\n%s\n\nAnh/Chị quan tâm mã nào để em gửi ảnh thực tế và báo giá chi tiết ạ?",
		title, strings.Join(items, "\n"))
}

func findFuzzyProduct(userMsg string, available []Product) *Product {
	norm := expandAliases(normalizeText(userMsg))
	words := strings.Fields(norm)

	var bestProd *Product
	highestScore := 0

	for i := range available {
		p := &available[i]
		pNorm := normalizeText(p.TenSP)
		pCode := strings.ToLower(p.MaSP)
		pCat := strings.ToLower(p.DanhMuc)

		score := 0

		if pCode != "" && strings.Contains(norm, pCode) {
			score += 150
		}
		if strings.Contains(norm, pNorm) {
			score += 80
		}
		if pCat != "" && strings.Contains(norm, pCat) {
			score += 20
		}

		pWords := strings.Fields(pNorm)
		for _, w := range words {
			if len([]rune(w)) < 2 {
				continue
			}
			for _, pw := range pWords {
				if w == pw {
					if w != "tao" && w != "nho" && w != "cam" && w != "quyt" && w != "co" && w != "khong" && w != "gia" && w != "bao" && w != "nhieu" {
						score += 40
					} else {
						score += 10
					}
				} else if len([]rune(w)) >= 3 && strings.HasPrefix(pw, w) {
					score += 35
				}
			}
		}

		if strings.Contains(norm, "hong tao") && pCat != "hong_tao" {
			score -= 60
		}
		if !strings.Contains(norm, "hong tao") && strings.Contains(norm, "tao") && pCat == "hong_tao" {
			score -= 60
		}

		if score > highestScore {
			highestScore = score
			bestProd = p
		}
	}

	if highestScore >= 30 {
		return bestProd
	}
	return nil
}

func ProcessCustomerMessage(userMsg string, sess *UserSession, available []Product) *BotReply {
	rawNorm := normalizeText(userMsg)
	norm := expandAliases(rawNorm)

	retailKeywords := []string{
		"le", "mua an", "an thu", "dung thu", "gia dinh", "1 thung", "may can", "an",
		"le thung", "hop", "mua an thu", "nha dung", "bieu", "tang", "bieu tang", "an nha",
	}
	wholesaleKeywords := []string{
		"si", "gia si", "lay si", "buon", "dai ly", "shop", "kinh doanh", "so luong",
		"vao so luong", "lo", "tuyen si", "xe hang", "ban lai", "cua hang",
	}

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

	isAskingPrice := strings.Contains(norm, "gia") || strings.Contains(norm, "bao nhieu") || strings.Contains(norm, "nhieu tien") || strings.Contains(norm, "xin gia")

	// 1. TÌM KIẾM SẢN PHẨM KHÁCH NHẮC TỚI
	matchedProd := findFuzzyProduct(userMsg, available)
	if matchedProd != nil {
		sess.LastProduct = *matchedProd
	}

	// 2. NẾU ĐÃ CÓ SẢN PHẨM TRONG NGỮ CẢNH
	if sess != nil && sess.LastProduct.MaSP != "" {
		p := sess.LastProduct
		taste := resolveTaste(&p)

		// 2.1. Khách mua lẻ / gia đình
		if isRetail {
			msg := fmt.Sprintf("Dạ em gửi Anh/Chị thông tin lô hàng chuẩn ngon bên em ạ:\n"+
				"✨ Sản phẩm: %s\n"+
				"📦 Quy cách: %s\n"+
				"💰 Giá lẻ thùng: %s\n"+
				"🍇 Hương vị / Chất ăn: %s\n\n"+
				"Anh/Chị lấy mấy thùng để em lên đơn giao sớm cho mình ạ?",
				p.TenSP, p.QuyCach, p.GiaLeThung, taste)

			return &BotReply{Message: msg, Product: &p, ShouldSendImg: false}
		}

		// 2.2. Khách mua sỉ / shop
		if isWholesale {
			msg := fmt.Sprintf("Dạ em gửi Anh/Chị chính sách giá sỉ ưu đãi cho đại lý/shop bên em:\n"+
				"✨ Sản phẩm: %s\n"+
				"📦 Quy cách: %s\n"+
				"💰 Giá sỉ lô: %s\n"+
				"🍇 Hương vị / Chất ăn: %s\n\n"+
				"Anh/Chị dự tính vào số lượng bao nhiêu thùng để em chuẩn bị gửi xe ạ?",
				p.TenSP, p.QuyCach, p.GiaSiLo, taste)

			return &BotReply{Message: msg, Product: &p, ShouldSendImg: false}
		}

		// 2.3. Khách hỏi quả cụ thể hoặc hỏi dồn giá: Đưa chất ăn và phân loại tệp khách
		if matchedProd != nil || isAskingPrice {
			msg := fmt.Sprintf("Dạ bên em sẵn %s hàng về tươi mới chuẩn loại 1 ạ.\n"+
				"🍇 Hương vị / Chất ăn: %s\n\n"+
				"Bên em có chính sách giá riêng cho khách ăn gia đình và khách lấy sỉ cho shop/đại lý. Không biết Anh/Chị dự tính lấy dùng gia đình hay lấy cho shop để em báo giá tốt nhất cho mình ạ?",
				p.TenSP, taste)

			shouldImg := (matchedProd != nil)
			return &BotReply{Message: msg, Product: &p, ShouldSendImg: shouldImg}
		}
	}

	// 3. KHÁCH HỎI LỌC THEO HÌNH THỨC (CỘT I)
	if strings.Contains(norm, "hang bay") || strings.Contains(norm, "di bay") || strings.Contains(norm, "air") {
		return &BotReply{Message: buildHinhThucSummary("bay", "Hàng Bay (Air Cargo)", "✈️", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "hang cont") || strings.Contains(norm, "di cont") {
		return &BotReply{Message: buildHinhThucSummary("cont", "Hàng Cont Lạnh", "🚢", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "nong san") || strings.Contains(norm, "hang viet") || strings.Contains(norm, "viet nam") {
		return &BotReply{Message: buildHinhThucSummary("viet", "Nông Sản Việt Nam", "🌾", available), Product: nil, ShouldSendImg: false}
	}

	// 4. KHÁCH HỎI THEO DANH MỤC (CỘT C)
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

	// 5. CHÀO HỎI MẶC ĐỊNH
	return &BotReply{
		Message: "Dạ Tổng kho trái cây nhập khẩu & Nông sản HP FRUIT (Bồ Đề - Long Biên) xin chào Anh/Chị ạ! Bên em sẵn rất nhiều mã hoa quả chuẩn hàng bay, hàng cont và nông sản sạch tươi ngon mỗi ngày. Anh/Chị đang quan tâm dòng quả nào để em gửi hình ảnh và báo giá chi tiết ạ?",
		Product: nil,
		ShouldSendImg: false,
	}
}
