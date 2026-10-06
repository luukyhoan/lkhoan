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

func resolveTaste(p *Product) string {
	taste := strings.TrimSpace(p.ChatAn)
	if taste == "" {
		dm := strings.ToLower(p.DanhMuc)
		n := normalizeText(p.TenSP)
		switch {
		case dm == "hong_tao" || strings.Contains(n, "hong tao"):
			taste = "Quả đanh giòn, vị ngọt thanh mát, cắn xốp nhẹ giòn tan."
		case strings.Contains(n, "envy"):
			taste = "Dòng táo hoàng gia, độ giòn đanh cơm, thơm nức và ngọt đậm sâu."
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
		case dm == "cam" || strings.Contains(n, "cam"):
			taste = "Tép mọng ngập nước, vị ngọt thanh dịu mát, vỏ mỏng thơm ngát."
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

	return fmt.Sprintf("Dạ bên em sẵn các mã %s hàng về tươi mới mỗi ngày ạ:\n\n%s\n\nDạ không biết mình đang cần tìm mã size nào để em gửi ảnh chi tiết và báo giá ạ?",
		label, strings.Join(items, "\n"))
}

func findAllMatchingSubline(keyword string, available []Product) []Product {
	var list []Product
	for _, p := range available {
		pNorm := normalizeText(p.TenSP)
		if strings.Contains(pNorm, keyword) && p.SoLuong > 0 {
			list = append(list, p)
		}
	}
	return list
}

// detectExplicitFruitCategory nhận diện rõ danh mục khách đề cập trong câu
func detectExplicitFruitCategory(norm string) string {
	if strings.Contains(norm, "hong tao") {
		return "hong_tao"
	}
	if strings.Contains(norm, "nho sua") {
		return "nho_sua"
	}
	if strings.Contains(norm, "cam") {
		return "cam"
	}
	if strings.Contains(norm, "quyt") {
		return "quyt"
	}
	if strings.Contains(norm, "tao") || strings.Contains(norm, "dazzle") || strings.Contains(norm, "envy") || strings.Contains(norm, "queen") || strings.Contains(norm, "rockit") {
		return "tao"
	}
	if strings.Contains(norm, "nho") {
		return "nho"
	}
	if strings.Contains(norm, "dua") {
		return "dua"
	}
	if strings.Contains(norm, "kiwi") {
		return "kiwi"
	}
	if strings.Contains(norm, "man") {
		return "man"
	}
	if strings.Contains(norm, "viet quat") {
		return "viet_quat"
	}
	return ""
}

// findSpecificProduct: Lọc tuyệt đối theo Danh Mục và loại bỏ mã hết hàng (SoLuong <= 0)
func findSpecificProduct(userMsg string, available []Product) *Product {
	norm := expandAliases(normalizeText(userMsg))
	words := strings.Fields(norm)
	explicitCat := detectExplicitFruitCategory(norm)

	var bestProd *Product
	highestScore := 0

	for i := range available {
		p := &available[i]

		// BỎ QUA HÀNG HẾT
		if p.SoLuong <= 0 {
			continue
		}

		pNorm := normalizeText(p.TenSP)
		pCode := strings.ToLower(p.MaSP)
		pCat := strings.ToLower(p.DanhMuc)

		// NẾU KHÁCH HỎI "CAM", CHỈ XÉT DANH MỤC "CAM", BỎ QUA HOÀN TOÀN CÁC DANH MỤC KHÁC
		if explicitCat != "" {
			if explicitCat == "cam" && pCat != "cam" {
				continue
			}
			if explicitCat == "tao" && (pCat != "tao" || pCat == "hong_tao") {
				continue
			}
			if explicitCat == "hong_tao" && pCat != "hong_tao" {
				continue
			}
			if explicitCat == "quyt" && pCat != "quyt" {
				continue
			}
			if explicitCat == "nho" && !strings.Contains(pCat, "nho") {
				continue
			}
		}

		score := 0

		if pCode != "" && strings.Contains(norm, pCode) {
			score += 150
		}
		if strings.Contains(norm, pNorm) {
			score += 80
		}

		pWords := strings.Fields(pNorm)
		for _, w := range words {
			if len([]rune(w)) < 2 {
				continue
			}
			for _, pw := range pWords {
				if w == pw {
					if w != "tao" && w != "nho" && w != "cam" && w != "quyt" && w != "dua" && w != "kiwi" && w != "co" && w != "khong" && w != "gia" && w != "bao" && w != "nhieu" {
						score += 40 // Điểm cho từ như: "uc", "nam phi", "kieng", "s55", "mfc"...
					}
				} else if len([]rune(w)) >= 3 && strings.HasPrefix(pw, w) {
					if w != "cam" && w != "tao" && w != "nho" {
						score += 35
					}
				}
			}
		}

		if score > highestScore {
			highestScore = score
			bestProd = p
		}
	}

	if highestScore >= 35 {
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

	// 1. NẾU KHÁCH ĐANG CHỐT SỈ/LẺ CHO SẢN PHẨM TRONG PHIÊN
	if sess != nil && sess.LastProduct.MaSP != "" && (isRetail || isWholesale) {
		p := sess.LastProduct
		taste := resolveTaste(&p)

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
	}

	// 2. TÌM KIẾM ĐÍCH DANH MÃ SẢN PHẨM (Ví dụ: "cam úc", "cam nam phi", "envy sz30"...)
	matchedProd := findSpecificProduct(userMsg, available)
	if matchedProd != nil {
		sess.LastProduct = *matchedProd
		p := matchedProd
		taste := resolveTaste(p)

		if isRetail {
			msg := fmt.Sprintf("Dạ em gửi Anh/Chị thông tin lô hàng chuẩn ngon bên em ạ:\n"+
				"✨ Sản phẩm: %s\n"+
				"📦 Quy cách: %s\n"+
				"💰 Giá lẻ thùng: %s\n"+
				"🍇 Hương vị / Chất ăn: %s\n\n"+
				"Anh/Chị lấy mấy thùng để em lên đơn giao sớm cho mình ạ?",
				p.TenSP, p.QuyCach, p.GiaLeThung, taste)
			return &BotReply{Message: msg, Product: p, ShouldSendImg: true}
		}

		if isWholesale {
			msg := fmt.Sprintf("Dạ em gửi Anh/Chị chính sách giá sỉ ưu đãi cho đại lý/shop bên em:\n"+
				"✨ Sản phẩm: %s\n"+
				"📦 Quy cách: %s\n"+
				"💰 Giá sỉ lô: %s\n"+
				"🍇 Hương vị / Chất ăn: %s\n\n"+
				"Anh/Chị dự tính vào số lượng bao nhiêu thùng để em chuẩn bị gửi xe ạ?",
				p.TenSP, p.QuyCach, p.GiaSiLo, taste)
			return &BotReply{Message: msg, Product: p, ShouldSendImg: true}
		}

		msg := fmt.Sprintf("Dạ bên em sẵn mã %s hàng về tươi mới mỗi ngày ạ.\n"+
			"🍇 Hương vị / Chất ăn: %s\n\n"+
			"Bên em có chính sách giá riêng cho khách ăn gia đình và khách lấy sỉ cho shop/đại lý. Không biết Anh/Chị dự tính lấy dùng gia đình hay lấy cho shop để em báo giá tốt nhất cho mình ạ?",
			p.TenSP, taste)
		return &BotReply{Message: msg, Product: p, ShouldSendImg: true}
	}

	// 3. NẾU KHÁCH HỎI CẢ DÒNG CON CHƯA CHỈ SIZE (ENVY, DAZZLE, QUEEN, MIZUKI...)
	sublines := []struct {
		kw    string
		label string
		icon  string
	}{
		{"envy", "Táo Envy", "🍎"},
		{"dazzle", "Táo Dazzle", "🍎"},
		{"rockit", "Táo Rockit", "🍎"},
		{"queen", "Táo Queen", "🍎"},
		{"mizuki", "Nho Sữa Mizuki", "🍇"},
	}

	for _, sub := range sublines {
		if strings.Contains(norm, sub.kw) {
			prods := findAllMatchingSubline(sub.kw, available)
			if len(prods) > 0 {
				var lines []string
				for _, p := range prods {
					lines = append(lines, fmt.Sprintf("%s %s (%s)", sub.icon, p.TenSP, p.QuyCach))
				}
				msg := fmt.Sprintf("Dạ bên em sẵn các mã %s hàng về tươi mới mỗi ngày ạ:\n\n%s\n\nDạ không biết mình đang cần tìm mã size nào để em gửi ảnh chi tiết và báo giá ạ?",
					sub.label, strings.Join(lines, "\n"))
				return &BotReply{Message: msg, Product: nil, ShouldSendImg: false}
			}
		}
	}

	// 4. LỌC TOÀN BỘ THEO DANH MỤC LỚN KHI KHÁCH CHỈ HỎI CHUNG (TÁO, CAM, QUÝT, NHO...)
	if strings.Contains(norm, "hong tao") {
		return &BotReply{Message: buildCategorySummary("hong_tao", "Hồng Táo", "🍎", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "tao") {
		return &BotReply{Message: buildCategorySummary("tao", "Táo", "🍎", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "cam") {
		return &BotReply{Message: buildCategorySummary("cam", "Cam", "🍊", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "quyt") {
		return &BotReply{Message: buildCategorySummary("quyt", "Quýt", "🍊", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "nho sua") {
		return &BotReply{Message: buildCategorySummary("nho_sua", "Nho Sữa", "🍇", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "nho") {
		return &BotReply{Message: buildCategorySummary("nho", "Nho", "🍇", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "dua") {
		return &BotReply{Message: buildCategorySummary("dua", "Dưa", "🍈", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "kiwi") {
		return &BotReply{Message: buildCategorySummary("kiwi", "Kiwi", "🥝", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "man") {
		return &BotReply{Message: buildCategorySummary("man", "Mận", "🍑", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "viet quat") {
		return &BotReply{Message: buildCategorySummary("viet_quat", "Việt Quất", "🫐", available), Product: nil, ShouldSendImg: false}
	}

	// 5. KHÁCH HỎI DỒN GIÁ KHI ĐÃ CÓ SẢN PHẨM LƯU TRONG PHIÊN
	if isAskingPrice && sess != nil && sess.LastProduct.MaSP != "" {
		p := sess.LastProduct
		taste := resolveTaste(&p)
		msg := fmt.Sprintf("Dạ bên em sẵn mã %s hàng về tươi mới mỗi ngày ạ.\n"+
			"🍇 Hương vị / Chất ăn: %s\n\n"+
			"Bên em có chính sách giá riêng cho khách ăn gia đình và khách lấy sỉ cho shop/đại lý. Không biết Anh/Chị dự tính lấy dùng gia đình hay lấy cho shop để em báo giá tốt nhất cho mình ạ?",
			p.TenSP, taste)
		return &BotReply{Message: msg, Product: &p, ShouldSendImg: false}
	}

	// 6. CHÀO HỎI MẶC ĐỊNH
	return &BotReply{
		Message: "Dạ Tổng kho trái cây nhập khẩu & Nông sản HP FRUIT (Bồ Đề - Long Biên) xin chào Anh/Chị ạ! Bên em sẵn rất nhiều mã hoa quả chuẩn hàng bay, hàng cont và nông sản sạch tươi ngon mỗi ngày. Anh/Chị đang quan tâm dòng quả nào để em gửi hình ảnh và báo giá chi tiết ạ?",
		Product: nil,
		ShouldSendImg: false,
	}
}
