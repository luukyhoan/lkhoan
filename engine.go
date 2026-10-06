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

// buildCategorySummary gom tất cả mã hàng còn hàng theo đúng form
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

// buildHinhThucSummary lọc sản phẩm theo cột Hinh_Thuc (Hàng Bay, Hàng Cont, Nông Sản Việt)
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

	return fmt.Sprintf("Dạ bên em sẵn các mã %s hàng về tươi mới mỗi ngày ạ:\n\n%s\n\nAnh/Chị quan tâm mã nào để em gửi ảnh thực tế và báo giá chi tiết ạ?",
		title, strings.Join(items, "\n"))
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

func findSpecificProduct(userMsg string, available []Product) *Product {
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

		pWords := strings.Fields(pNorm)
		for _, w := range words {
			if len([]rune(w)) < 2 {
				continue
			}
			for _, pw := range pWords {
				if w == pw {
					// Loại bỏ từ chung chung
					if w != "tao" && w != "nho" && w != "cam" && w != "quyt" && w != "dua" && w != "kiwi" && w != "co" && w != "khong" && w != "gia" && w != "bao" && w != "nhieu" {
						score += 40
					}
				} else if len([]rune(w)) >= 3 && strings.HasPrefix(pw, w) {
					if w != "cam" && w != "tao" && w != "nho" {
						score += 35
					}
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

	// 1. NẾU KHÁCH ĐANG CHỐT SỈ/LẺ CHO SẢN PHẨM ĐÃ CÓ TRONG PHIÊN
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

	// 2. ƯU TIÊN KIỂM TRA KHÁCH HỎI CẢ DÒNG CON (ENVY, DAZZLE, QUEEN, MIZUKI...) CHƯA CHỈ SIZE
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
			hasSpecificSize := false
			for _, w := range strings.Fields(norm) {
				if strings.HasPrefix(w, "s") || strings.HasPrefix(w, "sz") || strings.Contains(w, "30") || strings.Contains(w, "70") || strings.Contains(w, "80") || strings.Contains(w, "90") {
					hasSpecificSize = true
					break
				}
			}
			if !hasSpecificSize {
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
	}

	// 3. ƯU TIÊN KIỂM TRA KHÁCH HỎI THEO DANH MỤC LỚN (CAM, TÁO, NHO, QUÝT, DƯA, KIWI...)
	// Kiểm tra xem khách có kèm tên định danh riêng không (ví dụ "cam úc", "táo queen")
	hasSpecificDetail := false
	specificClues := []string{"uc", "nam phi", "kieng", "s40", "s55", "mfc", "dazzle", "envy", "queen", "rockit", "koru", "delecta", "mizuki", "ginseng"}
	for _, clue := range specificClues {
		if strings.Contains(norm, clue) {
			hasSpecificDetail = true
			break
		}
	}

	// Nếu khách chỉ hỏi chung chung về loại quả mà không kèm tên riêng cụ thể -> Gửi toàn bộ danh sách mã còn hàng
	if !hasSpecificDetail {
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
	}

	// 4. LỌC THEO HÌNH THỨC (HÀNG BAY, HÀNG CONT, NÔNG SẢN VIỆT)
	if strings.Contains(norm, "hang bay") || strings.Contains(norm, "di bay") || strings.Contains(norm, "air") {
		return &BotReply{Message: buildHinhThucSummary("bay", "Hàng Bay (Air Cargo)", "✈️", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "hang cont") || strings.Contains(norm, "di cont") {
		return &BotReply{Message: buildHinhThucSummary("cont", "Hàng Cont Lạnh", "🚢", available), Product: nil, ShouldSendImg: false}
	}
	if strings.Contains(norm, "nong san") || strings.Contains(norm, "hang viet") || strings.Contains(norm, "viet nam") {
		return &BotReply{Message: buildHinhThucSummary("viet", "Nông Sản Việt Nam", "🌾", available), Product: nil, ShouldSendImg: false}
	}

	// 5. TÌM KIẾM ĐÍCH DANH MÃ SẢN PHẨM KHÁCH NÊU CỤ THỂ
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

	// 6. KHÁCH HỎI DỒN GIÁ MÀ TRƯỚC ĐÓ ĐÃ CHỌN 1 MÃ QUẢ
	if isAskingPrice && sess != nil && sess.LastProduct.MaSP != "" {
		p := sess.LastProduct
		taste := resolveTaste(&p)
		msg := fmt.Sprintf("Dạ bên em sẵn mã %s hàng về tươi mới mỗi ngày ạ.\n"+
			"🍇 Hương vị / Chất ăn: %s\n\n"+
			"Bên em có chính sách giá riêng cho khách ăn gia đình và khách lấy sỉ cho shop/đại lý. Không biết Anh/Chị dự tính lấy dùng gia đình hay lấy cho shop để em báo giá tốt nhất cho mình ạ?",
			p.TenSP, taste)
		return &BotReply{Message: msg, Product: &p, ShouldSendImg: false}
	}

	// 7. CHÀO HỎI MẶC ĐỊNH
	return &BotReply{
		Message: "Dạ Tổng kho trái cây nhập khẩu & Nông sản HP FRUIT (Bồ Đề - Long Biên) xin chào Anh/Chị ạ! Bên em sẵn rất nhiều mã hoa quả chuẩn hàng bay, hàng cont và nông sản sạch tươi ngon mỗi ngày. Anh/Chị đang quan tâm dòng quả nào để em gửi hình ảnh và báo giá chi tiết ạ?",
		Product: nil,
		ShouldSendImg: false,
	}
}
