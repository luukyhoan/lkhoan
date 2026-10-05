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

// resolveTasteAndTag ưu tiên lấy từ Google Sheet (Cột I & J), nếu trống mới dùng câu mặc định
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
		n := normalizeText(p.TenSP)
		switch {
		case strings.Contains(n, "quyt"):
			taste = "Tép căng mọng nước, vị ngọt đậm thơm lừng, bóc vỏ dóc không dập."
		case strings.Contains(n, "tao"):
			taste = "Cơm dày giòn đanh, ngọt đậm sâu, cắn ngập miệng bao giòn."
		case strings.Contains(n, "nho"):
			taste = "Trái đanh cứng, chùm khít cuống xanh, vị ngọt thơm giòn tan."
		case strings.Contains(n, "le"):
			taste = "Thịt trắng phau, giòn rụm nhiều nước, vị ngọt thanh mát."
		case strings.Contains(n, "kiwi"):
			taste = "Ruột vàng mọng nước, ngọt dịu thanh mát chuẩn vị New Zealand."
		case strings.Contains(n, "cherry"):
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

	// 1. Phản hồi nhu cầu sỉ/lẻ cho quả đã chọn
	if sess != nil && sess.LastProduct.MaSP != "" && (isRetail || isWholesale) {
		p := sess.LastProduct
		tag, taste := resolveTasteAndTag(&p)

		if isRetail {
			openers := []string{
				"Dạ em gửi Anh/Chị thông tin lô hàng chuẩn ngon bên em ạ:",
				"Dạ Anh/Chị dùng gia đình thì lấy dòng này ăn cực mê ạ, em gửi thông tin:",
			}
			msg := fmt.Sprintf("%s\n"+
				"✨ Sản phẩm: %s\n"+
				"🏷️ Hình thức: %s\n"+
				"📦 Quy cách: %s\n"+
				"💰 Giá lẻ thùng: %s\n"+
				"🍇 Hương vị / Chất ăn: %s\n\n"+
				"Anh/Chị lấy mấy thùng để em lên đơn giao sớm cho mình ạ?",
				pickRandom(openers), p.TenSP, tag, p.QuyCach, p.GiaLeThung, taste)

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

	// 2. Tìm kiếm theo tên sản phẩm
	var matchedProd *Product
	for i := range available {
		p := &available[i]
		if strings.Contains(norm, normalizeText(p.TenSP)) || (p.MaSP != "" && strings.Contains(norm, strings.ToLower(p.MaSP))) {
			matchedProd = p
			break
		}
	}

	if matchedProd == nil {
		keywords := []string{"quyt", "tao", "nho", "le", "kiwi", "cherry", "cam", "dua", "ngo", "bap", "man", "viet quat", "luu"}
		for _, kw := range keywords {
			if strings.Contains(norm, kw) {
				for i := range available {
					if strings.Contains(normalizeText(available[i].TenSP), kw) {
						matchedProd = &available[i]
						break
					}
				}
				if matchedProd != nil {
					break
				}
			}
		}
	}

	if matchedProd != nil {
		sess.LastProduct = *matchedProd
		tag, taste := resolveTasteAndTag(matchedProd)

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

		msg := fmt.Sprintf("Dạ bên em vừa về lô %s (%s) tươi mới loại 1 chuẩn ngon ạ.\n\n"+
			"Bên em có chính sách giá ưu đãi riêng cho khách ăn gia đình và khách lấy sỉ cho cửa hàng/đại lý. Không biết Anh/Chị dự tính lấy số lượng dùng thử hay lấy cho shop để em báo giá tốt nhất cho mình ạ?",
			matchedProd.TenSP, tag)

		return &BotReply{Message: msg, Product: matchedProd, ShouldSendImg: true}
	}

	// 3. Chào hỏi mặc định
	return &BotReply{
		Message: "Dạ Tổng kho trái cây nhập khẩu & Nông sản HP FRUIT (Bồ Đề - Long Biên) xin chào Anh/Chị ạ! Bên em sẵn rất nhiều mã hoa quả chuẩn hàng bay, hàng cont và nông sản sạch tươi ngon mỗi ngày. Anh/Chị đang quan tâm dòng quả nào để em gửi hình ảnh và báo giá chi tiết ạ?",
		Product: nil,
		ShouldSendImg: false,
	}
}
