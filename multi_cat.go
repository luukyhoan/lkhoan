package main

import (
	"fmt"
	"strings"
)

type MultiFruitCat struct {
	Name      string
	Keywords  []string
	Specifics []string
	Emoji     string
}

func getMultiCategories() []MultiFruitCat {
	return []MultiFruitCat{
		{
			Name:      "Nho",
			Keywords:  []string{"nho", "shine muscat", "mẫu đơn", "mau don"},
			Specifics: []string{"ngón tay", "ngon tay", "mizuki", "ginseng", "panda", "autumn", "kẹo", "keo", "đen", "den", "đỏ", "do"},
			Emoji:     "🍇",
		},
		{
			Name:      "Táo",
			Keywords:  []string{"táo", "tao", "apple"},
			Specifics: []string{"envy", "dazzle", "gala", "fuji", "rockit", "queen"},
			Emoji:     "🍎",
		},
		{
			Name:      "Lựu",
			Keywords:  []string{"lựu", "luu"},
			Specifics: []string{},
			Emoji:     "🏮",
		},
		{
			Name:      "Cam",
			Keywords:  []string{"cam", "orange"},
			Specifics: []string{"úc", "uc", "vàng", "vang", "ruột đỏ", "cara", "nam phi", "kiếng"},
			Emoji:     "🍊",
		},
		{
			Name:      "Quýt",
			Keywords:  []string{"quýt", "quyt", "mandarin"},
			Specifics: []string{"úc", "uc", "2ph", "đường", "duong"},
			Emoji:     "🍊",
		},
		{
			Name:      "Kiwi",
			Keywords:  []string{"kiwi", "kiqu", "kiu"},
			Specifics: []string{"vàng", "vang", "xanh", "zespri"},
			Emoji:     "🥝",
		},
		{
			Name:      "Lê",
			Keywords:  []string{"lê ", " lê", "le ", " le", "pear"},
			Specifics: []string{"hàn", "han", "nâu", "nau", "nam phi", "má đào", "ma dao", "sữa", "sua", "sơn đông"},
			Emoji:     "🍐",
		},
		{
			Name:      "Đào",
			Keywords:  []string{"đào", "dao"},
			Specifics: []string{},
			Emoji:     "🍑",
		},
		{
			Name:      "Mận",
			Keywords:  []string{"mận", "man"},
			Specifics: []string{},
			Emoji:     "🍑",
		},
		{
			Name:      "Hồng Giòn",
			Keywords:  []string{"hồng giòn", "hong gion", "hồng s6", "hong s6"},
			Specifics: []string{},
			Emoji:     "🍅",
		},
		{
			Name:      "Dưa",
			Keywords:  []string{"dưa", "dua", "melon"},
			Specifics: []string{"wow", "fuji", "lưới", "luoi", "hami"},
			Emoji:     "🍈",
		},
	}
}

// checkAndHandleMultiCategory kiểm tra xem khách có đang hỏi danh mục hoa quả (hoặc hỏi từ 2 loại quả trở lên)
func checkAndHandleMultiCategory(userMsg, rawNorm string, available []Product) *BotReply {
	lowerMsg := strings.ToLower(userMsg)
	cats := getMultiCategories()

	type MatchedCat struct {
		Cat   MultiFruitCat
		Prods []Product
	}
	var matched []MatchedCat

	for _, c := range cats {
		isMatch := false
		for _, kw := range c.Keywords {
			if strings.Contains(lowerMsg, kw) || strings.Contains(rawNorm, kw) {
				if c.Name == "Hồng Giòn" && (strings.Contains(lowerMsg, "hồng táo") || strings.Contains(rawNorm, "hong tao")) {
					continue
				}
				if c.Name == "Lê" && (strings.Contains(lowerMsg, "dazzle") || strings.Contains(rawNorm, "dazzle")) {
					continue
				}
				isMatch = true
				break
			}
		}

		if isMatch {
			var prods []Product
			for _, p := range available {
				pLow := strings.ToLower(p.TenSP)
				for _, kw := range c.Keywords {
					kwTrim := strings.TrimSpace(kw)
					if strings.Contains(pLow, kwTrim) {
						if c.Name == "Lê" && strings.Contains(pLow, "dazzle") {
							continue
						}
						prods = append(prods, p)
						break
					}
				}
			}
			if len(prods) > 0 {
				matched = append(matched, MatchedCat{Cat: c, Prods: prods})
			}
		}
	}

	// Chỉ kích hoạt khi khách hỏi từ 2 loại quả trở lên (ví dụ: táo và lựu, cam với quýt)
	if len(matched) < 2 {
		return nil
	}

	var res string
	for _, m := range matched {
		res += fmt.Sprintf("Dạ Tổng kho HP FRUIT đang sẵn các mã %s tươi ngon sau ạ:\n\n", strings.ToUpper(m.Cat.Name))
		for _, p := range m.Prods {
			price := p.GiaLeThung
			if price == "" {
				price = p.GiaSiLo
			}
			if price == "" {
				price = "Ib báo giá sỉ"
			}
			res += fmt.Sprintf("%s %s - Giá: %s/thùng\n", m.Cat.Emoji, p.TenSP, price)
		}
		res += "\n"
	}

	res += "Anh/Chị muốn lấy loại nào để em gửi hình ảnh thực tế và lên đơn gửi xe cho mình nhé ạ!"

	return &BotReply{
		Message:       strings.TrimSpace(res),
		Product:       &matched[0].Prods[0],
		ShouldSendImg: false,
	}
}
