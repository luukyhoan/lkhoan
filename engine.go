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
			taste = "Dòng táo hoàng gia, cơm giòn đanh, thơm nức và ngọt đậm sâu."
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
		case dm == "kiwi" || strings.Contains(n, "kiwi"):
			taste = "Ruột vàng mọng nước, ngọt dịu thanh mát chuẩn vị New Zealand."
		case dm == "le" || strings.Contains(n, "le"):
			taste = "Quả sáng mã, mọng ngập nước, cắn giòn ngọt đậm đà."
		case strings.Contains(dm, "nho") || strings.Contains(n, "nho"):
			taste = "Trái đanh cứng, chùm khít cuống xanh, vị ngọt thơm giòn tan."
		default:
			taste = "Hàng tuyển chuẩn ngon loại 1, bao tươi ngon ngọt từng trái."
		}
	}
	return taste
}

func limitItems(items []string, max int) []string {
	if len(items) <= max {
		return items
	}
	return items[:max]
}

func buildFullMenuSummary(available []Product) string {
	catMap := make(map[string][]string)

	for _, p := range available {
		if p.SoLuong <= 0 {
			continue
		}
		cat := strings.ToLower(strings.TrimSpace(p.DanhMuc))
		name := p.TenSP

		name = strings.ReplaceAll(name, "Táo ", "")
		name = strings.ReplaceAll(name, "Cam ", "")
		name = strings.ReplaceAll(name, "Quýt ", "")
		name = strings.ReplaceAll(name, "Dưa ", "")
		name = strings.ReplaceAll(name, "Lê ", "")

		catMap[cat] = append(catMap[cat], name)
	}

	var sections []string
	if items, ok := catMap["tao"]; ok && len(items) > 0 {
		sections = append(sections, fmt.Sprintf("🍎 Táo: %s", strings.Join(limitItems(items, 5), ", ")))
	}
	if items, ok := catMap["hong_tao"]; ok && len(items) > 0 {
		sections = append(sections, fmt.Sprintf("🍎 Hồng Táo: %s", strings.Join(limitItems(items, 3), ", ")))
	}
	camQuyt := append(catMap["cam"], catMap["quyt"]...)
	if len(camQuyt) > 0 {
		sections = append(sections, fmt.Sprintf("🍊 Cam & Quýt: %s", strings.Join(limitItems(camQuyt, 4), ", ")))
	}
	
	// Gom toàn bộ họ nhà nho
	var nhoAll []string
	for k, list := range catMap {
		if strings.Contains(k, "nho") {
			nhoAll = append(nhoAll, list...)
		}
	}
	if len(nhoAll) > 0 {
		sections = append(sections, fmt.Sprintf("🍇 Nho: %s", strings.Join(limitItems(nhoAll, 5), ", ")))
	}

	if items, ok := catMap["dua"]; ok && len(items) > 0 {
		sections = append(sections, fmt.Sprintf("🍈 Dưa: %s", strings.Join(limitItems(items, 3), ", ")))
	}
	if items, ok := catMap["le"]; ok && len(items) > 0 {
		sections = append(sections, fmt.Sprintf("🍐 Lê: %s", strings.Join(limitItems(items, 3), ", ")))
	}
	if items, ok := catMap["kiwi"]; ok && len(items) > 0 {
		sections = append(sections, fmt.Sprintf("🥝 Kiwi: %s", strings.Join(limitItems(items, 3), ", ")))
	}
	if items, ok := catMap["viet_quat"]; ok && len(items) > 0 {
		sections = append(sections, fmt.Sprintf("🫐 Việt Quất: %s", strings.Join(limitItems(items, 2), ", ")))
	}
	if items, ok := catMap["man"]; ok && len(items) > 0 {
		sections = append(sections, fmt.Sprintf("🍑 Mận: %s", strings.Join(limitItems(items, 3), ", ")))
	}

	return fmt.Sprintf("Dạ hiện tại kho HP FRUIT đang sẵn rất nhiều dòng quả nhập khẩu & nông sản tươi mới mỗi ngày phục vụ sỉ & lẻ sau ạ:\n\n%s\n\nDạ Anh/Chị đang quan tâm dòng quả nào để em gửi ảnh chi tiết và báo giá ạ?",
		strings.Join(sections, "\n"))
}

// getProductsByCategory: Hỗ trợ tìm kiếm thông minh, gom cả họ nho nếu hỏi chung
func getProductsByCategory(categoryName string, available []Product) []Product {
	var list []Product
	for _, p := range available {
		if p.SoLuong <= 0 {
			continue
		}
		cat := strings.ToLower(p.DanhMuc)
		
		if categoryName == "nho" {
			// Nếu hỏi "nho" chung: gom tất cả nho_sua, nho_do, nho_den, nho_ngon...
			if strings.Contains(cat, "nho") {
				list = append(list, p)
			}
		} else {
			if strings.EqualFold(cat, categoryName) {
				list = append(list, p)
			}
		}
	}
	return list
}

func handleCategoryInquiry(categoryName, label, icon string, isWholesale, isRetail bool, sess *UserSession, available []Product) *BotReply {
	prods := getProductsByCategory(categoryName, available)
	if len(prods) == 0 {
		return &BotReply{
			Message:       fmt.Sprintf("Dạ hiện tại kho HP FRUIT đang tạm hết các mã %s mới, hàng đợt tới về em sẽ báo mình ngay nhé ạ!", label),
			Product:       nil,
			ShouldSendImg: false,
		}
	}

	if len(prods) == 1 {
		p := prods[0]
		sess.LastProduct = p
		taste := resolveTaste(&p)

		if isWholesale {
			msg := fmt.Sprintf("Dạ em gửi Anh/Chị chính sách giá sỉ ưu đãi cho đại lý/shop bên em:\n"+
				"✨ Sản phẩm: %s\n"+
				"📦 Quy cách: %s\n"+
				"💰 Giá sỉ lô: %s\n"+
				"🍇 Hương vị / Chất ăn: %s\n\n"+
				"Anh/Chị dự tính vào số lượng bao nhiêu thùng để em chuẩn bị gửi xe ạ?",
				p.TenSP, p.QuyCach, p.GiaSiLo, taste)
			return &BotReply{Message: msg, Product: &p, ShouldSendImg: true}
		}

		if isRetail {
			msg := fmt.Sprintf("Dạ em gửi Anh/Chị thông tin lô hàng chuẩn ngon bên em ạ:\n"+
				"✨ Sản phẩm: %s\n"+
				"📦 Quy cách: %s\n"+
				"💰 Giá lẻ thùng: %s\n"+
				"🍇 Hương vị / Chất ăn: %s\n\n"+
				"Anh/Chị lấy mấy thùng để em lên đơn giao sớm cho mình ạ?",
				p.TenSP, p.QuyCach, p.GiaLeThung, taste)
			return &BotReply{Message: msg, Product: &p, ShouldSendImg: true}
		}

		msg := fmt.Sprintf("Dạ bên em sẵn mã %s hàng về tươi mới mỗi ngày ạ.\n"+
			"🍇 Hương vị / Chất ăn: %s\n\n"+
			"Bên em có chính sách giá riêng cho khách ăn gia đình và khách lấy sỉ cho shop/đại lý. Không biết Anh/Chị dự tính lấy dùng gia đình hay lấy cho shop để em báo giá tốt nhất cho mình ạ?",
			p.TenSP, taste)
		return &BotReply{Message: msg, Product: &p, ShouldSendImg: true}
	}

	var items []string
	for _, p := range prods {
		items = append(items, fmt.Sprintf("%s %s (%s)", icon, p.TenSP, p.QuyCach))
	}
	msg := fmt.Sprintf("Dạ bên em sẵn các mã %s hàng về tươi mới mỗi ngày ạ:\n\n%s\n\nDạ không biết mình đang cần tìm mã size nào để em gửi ảnh chi tiết và báo giá ạ?",
		label, strings.Join(items, "\n"))
	return &BotReply{Message: msg, Product: nil, ShouldSendImg: false}
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

func detectExplicitFruitCategory(norm string) string {
	if strings.Contains(norm, "hong tao") {
		return "hong_tao"
	}
	if strings.Contains(norm, "nho sua") {
		return "nho_sua"
	}
	if strings.Contains(norm, "nho do") {
		return "nho_do"
	}
	if strings.Contains(norm, "nho den") {
		return "nho_den"
	}
	if strings.Contains(norm, "nho ngon tay") {
		return "nho_ngon"
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
	if strings.Contains(norm, "le") {
		return "le"
	}
	if strings.Contains(norm, "man") {
		return "man"
	}
	if strings.Contains(norm, "viet quat") {
		return "viet_quat"
	}
	return ""
}

func findSpecificProduct(userMsg string, available []Product) *Product {
	norm := expandAliases(normalizeText(userMsg))
	words := strings.Fields(norm)
	explicitCat := detectExplicitFruitCategory(norm)

	if explicitCat == "" {
		hasFruitWord := false
		for _, w := range words {
			if w == "tao" || w == "cam" || w == "quyt" || w == "nho" || w == "dua" || w == "kiwi" || w == "le" || w == "man" {
				hasFruitWord = true
				break
			}
		}
		if !hasFruitWord {
			return nil
		}
	}

	var bestProd *Product
	highestScore := 0

	for i := range available {
		p := &available[i]
		if p.SoLuong <= 0 {
			continue
		}

		pNorm := normalizeText(p.TenSP)
		pCode := strings.ToLower(p.MaSP)
		pCat := strings.ToLower(p.DanhMuc)

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
			if explicitCat == "kiwi" && pCat != "kiwi" {
				continue
			}
			if explicitCat == "le" && pCat != "le" {
				continue
			}
			if explicitCat == "nho_sua" && pCat != "nho_sua" {
				continue
			}
			if explicitCat == "nho_do" && pCat != "nho_do" {
				continue
			}
			if explicitCat == "nho_den" && pCat != "nho_den" {
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
					if w != "tao" && w != "nho" && w != "cam" && w != "quyt" && w != "dua" && w != "kiwi" && w != "le" && w != "co" && w != "khong" && w != "gia" && w != "bao" && w != "nhieu" {
						score += 40
					}
				} else if len([]rune(w)) >= 3 && strings.HasPrefix(pw, w) {
					if w != "cam" && w != "tao" && w != "nho" && w != "kiwi" {
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

func isGeneralMenuQuery(norm string) bool {
	patterns := []string{
		"hom nay co qua gi", "co qua gi", "co trai cay gi", "co nhung qua gi",
		"kho co gi", "san nhung qua gi", "co nhung loai nao", "cac loai qua",
		"danh sach hoa qua", "co nhung mat hang nao", "co hang gi", "menu", "bang gia hom nay",
	}
	for _, p := range patterns {
		if strings.Contains(norm, p) {
			return true
		}
	}
	return false
}

func containsWord(norm, target string) bool {
	for _, w := range strings.Fields(norm) {
		if w == target {
			return true
		}
	}
	return false
}

func ProcessCustomerMessage(userMsg string, sess *UserSession, available []Product) *BotReply {
	rawNorm := normalizeText(userMsg)
	norm := expandAliases(rawNorm)

	// 1. Menu tổng quan hôm nay có quả gì
	if isGeneralMenuQuery(rawNorm) || isGeneralMenuQuery(norm) {
		return &BotReply{Message: buildFullMenuSummary(available), Product: nil, ShouldSendImg: false}
	}

	// 2. Mặc cả / mua nhiều / số lượng
	discountPatterns := []string{"mua nhieu", "gia tot hon", "giam gia", "bot khong", "chiet khau", "bot gia", "gia uu dai", "lay nhieu", "so luong nhieu", "co bot", "co giam"}
	isAskingDiscount := false
	for _, dp := range discountPatterns {
		if strings.Contains(norm, dp) {
			isAskingDiscount = true
			break
		}
	}

	// 3. Khách sỉ
	wholesalePatterns := []string{"mua ban", "mua buon", "mua si", "gia si", "ban cho", "ban cua hang", "cua hang", "dai ly", "kinh doanh", "vao so luong", "xe hang"}
	isWholesale := false
	for _, wp := range wholesalePatterns {
		if strings.Contains(norm, wp) {
			isWholesale = true
			break
		}
	}
	if !isWholesale {
		if containsWord(norm, "si") || containsWord(norm, "buon") || containsWord(norm, "ban") || containsWord(norm, "shop") || containsWord(norm, "lo") {
			isWholesale = true
		}
	}

	// 4. Khách lẻ
	retailPatterns := []string{"mua an", "mua bieu", "mua dung", "nha dung", "nha su dung", "an thu", "dung thu", "gia dinh", "1 thung", "may can", "hop", "le thung"}
	isRetail := false
	if !isWholesale && !isAskingDiscount {
		for _, rp := range retailPatterns {
			if strings.Contains(norm, rp) {
				isRetail = true
				break
			}
		}
		if !isRetail {
			if containsWord(norm, "an") || containsWord(norm, "bieu") || containsWord(norm, "dung") || containsWord(norm, "le") {
				isRetail = true
			}
		}
	}

	isAskingPrice := strings.Contains(norm, "gia") || strings.Contains(norm, "bao nhieu") || strings.Contains(norm, "nhieu tien") || strings.Contains(norm, "xin gia")

	// MẶC CẢ TRONG PHIÊN
	if isAskingDiscount && sess != nil && sess.LastProduct.MaSP != "" {
		p := sess.LastProduct
		taste := resolveTaste(&p)

		msg := fmt.Sprintf("Dạ chắc chắn rồi ạ! Với mã %s bên em luôn có chính sách giá sỉ trợ giá cực tốt khi mình lấy theo số lượng thùng hoặc vào cả lô ạ:\n\n"+
			"📦 Quy cách: %s\n"+
			"💰 Giá sỉ lô tham khảo: %s\n"+
			"🍇 Chất ăn: %s\n\n"+
			"Anh/Chị dự tính lấy số lượng khoảng bao nhiêu thùng để em báo mức chiết khấu sát nhất và hỗ trợ gửi xe cho mình ạ?",
			p.TenSP, p.QuyCach, p.GiaSiLo, taste)

		return &BotReply{Message: msg, Product: &p, ShouldSendImg: false}
	}

	// CHỐT SỈ/LẺ TRONG PHIÊN
	if sess != nil && sess.LastProduct.MaSP != "" && (isRetail || isWholesale) {
		p := sess.LastProduct
		taste := resolveTaste(&p)

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
	}

	// TÌM KIẾM ĐÍCH DANH THEO TÊN RIÊNG (vd: "cam úc", "nho sữa mizuki", "envy sz30"...)
	matchedProd := findSpecificProduct(userMsg, available)
	if matchedProd != nil {
		sess.LastProduct = *matchedProd
		p := matchedProd
		taste := resolveTaste(p)

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

		msg := fmt.Sprintf("Dạ bên em sẵn mã %s hàng về tươi mới mỗi ngày ạ.\n"+
			"🍇 Hương vị / Chất ăn: %s\n\n"+
			"Bên em có chính sách giá riêng cho khách ăn gia đình và khách lấy sỉ cho shop/đại lý. Không biết Anh/Chị dự tính lấy dùng gia đình hay lấy cho shop để em báo giá tốt nhất cho mình ạ?",
			p.TenSP, taste)
		return &BotReply{Message: msg, Product: p, ShouldSendImg: true}
	}

	// KIỂM TRA DÒNG CON (ENVY, DAZZLE, QUEEN, MIZUKI...)
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
			if len(prods) == 1 {
				p := &prods[0]
				sess.LastProduct = *p
				taste := resolveTaste(p)
				msg := fmt.Sprintf("Dạ bên em sẵn mã %s hàng về tươi mới mỗi ngày ạ.\n"+
					"🍇 Hương vị / Chất ăn: %s\n\n"+
					"Bên em có chính sách giá riêng cho khách ăn gia đình và khách lấy sỉ cho shop/đại lý. Không biết Anh/Chị dự tính lấy dùng gia đình hay lấy cho shop để em báo giá tốt nhất cho mình ạ?",
					p.TenSP, taste)
				return &BotReply{Message: msg, Product: p, ShouldSendImg: true}
			} else if len(prods) > 1 {
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

	// KIỂM TRA DANH MỤC
	if strings.Contains(norm, "hong tao") {
		return handleCategoryInquiry("hong_tao", "Hồng Táo", "🍎", isWholesale, isRetail, sess, available)
	}
	if strings.Contains(norm, "tao") {
		return handleCategoryInquiry("tao", "Táo", "🍎", isWholesale, isRetail, sess, available)
	}
	if strings.Contains(norm, "cam") {
		return handleCategoryInquiry("cam", "Cam", "🍊", isWholesale, isRetail, sess, available)
	}
	if strings.Contains(norm, "quyt") {
		return handleCategoryInquiry("quyt", "Quýt", "🍊", isWholesale, isRetail, sess, available)
	}
	
	// PHÂN BIỆT RÕ TỪNG LOẠI NHO VÀ NHO CHUNG
	if strings.Contains(norm, "nho sua") {
		return handleCategoryInquiry("nho_sua", "Nho Sữa", "🍇", isWholesale, isRetail, sess, available)
	}
	if strings.Contains(norm, "nho do") {
		return handleCategoryInquiry("nho_do", "Nho Đỏ", "🍇", isWholesale, isRetail, sess, available)
	}
	if strings.Contains(norm, "nho den") {
		return handleCategoryInquiry("nho_den", "Nho Đen", "🍇", isWholesale, isRetail, sess, available)
	}
	if strings.Contains(norm, "nho ngon tay") {
		return handleCategoryInquiry("nho_ngon", "Nho Ngón Tay", "🍇", isWholesale, isRetail, sess, available)
	}
	if strings.Contains(norm, "nho") {
		// Gom toàn bộ họ nho nếu khách chỉ hỏi "có nho không"
		return handleCategoryInquiry("nho", "Nho", "🍇", isWholesale, isRetail, sess, available)
	}

	if strings.Contains(norm, "dua") {
		return handleCategoryInquiry("dua", "Dưa", "🍈", isWholesale, isRetail, sess, available)
	}
	if strings.Contains(norm, "kiwi") {
		return handleCategoryInquiry("kiwi", "Kiwi", "🥝", isWholesale, isRetail, sess, available)
	}
	if strings.Contains(norm, "le") {
		return handleCategoryInquiry("le", "Lê", "🍐", isWholesale, isRetail, sess, available)
	}
	if strings.Contains(norm, "man") {
		return handleCategoryInquiry("man", "Mận", "🍑", isWholesale, isRetail, sess, available)
	}
	if strings.Contains(norm, "viet quat") {
		return handleCategoryInquiry("viet_quat", "Việt Quất", "🫐", isWholesale, isRetail, sess, available)
	}

	// KHÁCH HỎI DỒN GIÁ KHI ĐÃ CÓ QUẢ TRONG PHIÊN
	if isAskingPrice && sess != nil && sess.LastProduct.MaSP != "" {
		p := sess.LastProduct
		taste := resolveTaste(&p)
		msg := fmt.Sprintf("Dạ bên em sẵn mã %s hàng về tươi mới mỗi ngày ạ.\n"+
			"🍇 Hương vị / Chất ăn: %s\n\n"+
			"Bên em có chính sách giá riêng cho khách ăn gia đình và khách lấy sỉ cho shop/đại lý. Không biết Anh/Chị dự tính lấy dùng gia đình hay lấy cho shop để em báo giá tốt nhất cho mình ạ?",
			p.TenSP, taste)
		return &BotReply{Message: msg, Product: &p, ShouldSendImg: false}
	}

	// CHÀO HỎI MẶC ĐỊNH
	return &BotReply{
		Message: "Dạ Tổng kho trái cây nhập khẩu & Nông sản HP FRUIT (Bồ Đề - Long Biên) xin chào Anh/Chị ạ! Bên em sẵn rất nhiều mã hoa quả chuẩn hàng bay, hàng cont và nông sản sạch tươi ngon mỗi ngày. Anh/Chị đang quan tâm dòng quả nào để em gửi hình ảnh và báo giá chi tiết ạ?",
		Product: nil,
		ShouldSendImg: false,
	}
}
