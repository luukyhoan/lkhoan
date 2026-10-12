package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Trạng thái hội thoại của khách
type ConversationState int

const (
	StateBrowsing ConversationState = iota // Đang tham khảo / hỏi giá
	StateWaitingQuantity                   // Đã chọn sản phẩm, chờ khách chốt số lượng
	StateWaitingInfo                       // Đã có số lượng, chờ địa chỉ & SĐT
	StateOrderConfirmed                    // Đã xác nhận đơn hàng
)

type Product struct {
	TenSP       string
	Gia         string
	FolderAnhID string
	MoTa        string
}

type UserSession struct {
	State                ConversationState
	CurrentProduct       *Product
	Quantity             int
	CustomerPhone        string
	CustomerAddress      string
	LastAdminMessageTime time.Time
	FollowupTimer        *time.Timer
	InvitedToGroup       bool
}

type BotReply struct {
	Message      string
	ShouldSendImg bool
	Product      *Product
}

// 1. Kiểm tra khách có đang hỏi giá hoặc yêu cầu báo giá không
func isAskingPrice(text string) bool {
	lower := strings.ToLower(text)
	priceKeywords := []string{
		"bao nhiêu", "nhiêu", "giá", "bn", "bao tiền", "mấy tiền",
		"báo giá", "xin giá", "bnhieu", "bnh", "giá c đã", "báo giá c đã",
		"giá thế nào", "giá sao", "nhiêu 1 thùng", "bao nhiêu 1 thùng",
	}
	for _, kw := range priceKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// 2. Tách số điện thoại Việt Nam (10 chữ số)
func extractPhoneNumber(text string) string {
	// Bắt các định dạng 09x, 08x, 07x, 03x, 05x hoặc +84
	re := regexp.MustCompile(`(?:(?:\+84|84|0)[3|5|7|8|9])([0-9]{8})\b`)
	match := re.FindString(text)
	return strings.ReplaceAll(match, " ", "")
}

// 3. Tách số lượng chỉ khi khách có ý định đặt mua (VD: "lấy 2 thùng", "cho 1 thùng")
func extractOrderQuantity(text string) int {
	lower := strings.ToLower(text)

	// Nếu câu chứa từ hỏi giá thì KHÔNG ĐƯỢC tính là đặt mua
	if isAskingPrice(lower) {
		return 0
	}

	// Trường hợp 1: Có từ khóa mua hàng đi kèm ("lấy 2 thùng", "cho chị 1 thùng", "gửi 3 thùng")
	reIntent := regexp.MustCompile(`(?:lấy|cho|gửi|ship|đặt|nhặt|bốc|mua|chốt)\s*(?:chị|anh|em|mình)?\s*(\d+)\s*(?:thùng|hộp|kg|dây|set)?`)
	if match := reIntent.FindStringSubmatch(lower); len(match) > 1 {
		qty, _ := strconv.Atoi(match[1])
		return qty
	}

	// Trường hợp 2: Khách nhắn gọn chỉ mỗi số lượng (VD: "1", "2", "1 thùng", "2 thùng")
	reOnlyQty := regexp.MustCompile(`^(\d+)\s*(thùng|hộp|kg)?$`)
	if match := reOnlyQty.FindStringSubmatch(strings.TrimSpace(lower)); len(match) > 1 {
		qty, _ := strconv.Atoi(match[1])
		return qty
	}

	return 0
}

// 4. Kiểm tra chuỗi có thực sự là địa chỉ nhận hàng hợp lệ không
func isValidAddress(text string) bool {
	lower := strings.ToLower(text)

	// Tuyệt đối không nhận nếu chứa từ hỏi giá, thắc mắc
	if isAskingPrice(lower) || strings.Contains(lower, "không") || strings.Contains(lower, "chưa") {
		return false
	}

	// Phải đủ dài
	if len(strings.TrimSpace(text)) < 8 {
		return false
	}

	// Chứa các từ chỉ địa danh hoặc số nhà
	addressIndicators := []string{
		"thôn", "xóm", "xã", "phường", "quận", "huyện", "tỉnh", "thành phố",
		"tp", "đường", "phố", "ngõ", "ngách", "số", "nhà", "chợ", "bến xe", "quốc lộ",
	}
	for _, ind := range addressIndicators {
		if strings.Contains(lower, ind) {
			return true
		}
	}

	// Hoặc chứa số nhà kết hợp chữ (VD: 52 Nghĩa Dũng)
	hasDigit := regexp.MustCompile(`\d+`).MatchString(text)
	hasWord := regexp.MustCompile(`[a-zA-ZÀ-ỹ]{3,}`).MatchString(text)
	return hasDigit && hasWord
}

// 5. Tìm sản phẩm phù hợp trong bảng tính
func matchProduct(text string, products []Product) *Product {
	lower := strings.ToLower(text)
	for i := range products {
		name := strings.ToLower(products[i].TenSP)
		if strings.Contains(lower, name) {
			return &products[i]
		}
	}
	// Khớp tương đối theo loại trái cây
	keywords := map[string][]string{
		"kiwi":   {"kiwi", "kiwy"},
		"táo":    {"táo", "tao", "envy", "dazzle", "queen"},
		"nho":    {"nho", "sweet globe", "autumn crisp", "muscat"},
		"quýt":   {"quýt", "quyt", "úc"},
		"sầu":    {"sầu", "sầu riêng", "monthong"},
		"cherry": {"cherry", "chery"},
	}

	for key, kwList := range keywords {
		for _, kw := range kwList {
			if strings.Contains(lower, kw) {
				for i := range products {
					if strings.Contains(strings.ToLower(products[i].TenSP), key) {
						return &products[i]
					}
				}
			}
		}
	}
	return nil
}

// 6. Xử lý tin nhắn khách hàng chính
func ProcessCustomerMessage(text string, sess *UserSession, products []Product) BotReply {
	trimmed := strings.TrimSpace(text)
	lower := strings.ToLower(trimmed)

	// ==========================================================
	// ƯU TIÊN 1: NẾU KHÁCH HỎI GIÁ HOẶC BẢO BÁO GIÁ
	// ==========================================================
	if isAskingPrice(lower) {
		// Tìm sản phẩm khách đang nhắc tới trong câu hỏi
		matched := matchProduct(trimmed, products)
		if matched != nil {
			sess.CurrentProduct = matched
		}

		// Nếu xác định được sản phẩm
		if sess.CurrentProduct != nil {
			// Reset trạng thái: Không bắt khách nhập địa chỉ/SĐT ở bước này
			sess.State = StateWaitingQuantity
			sess.Quantity = 0
			sess.CustomerAddress = ""

			msg := fmt.Sprintf("Dạ mã %s bên em đang sẵn hàng tại kho, giá %s/thùng ạ.\n\nAnh/Chị lấy mấy thùng để bên em đóng gói gửi hàng sớm cho mình nhé ạ?",
				sess.CurrentProduct.TenSP, sess.CurrentProduct.Gia)

			return BotReply{
				Message:       msg,
				ShouldSendImg: true,
				Product:       sess.CurrentProduct,
			}
		}

		// Nếu chưa rõ mã trái cây nào
		return BotReply{
			Message: "Dạ bên em sẵn rất nhiều loại hoa quả nhập khẩu theo thùng giá sỉ tận kho. Anh/Chị đang quan tâm loại nào (Táo, Nho, Kiwi, Quýt, Cherry...) để em báo giá chuẩn nhất cho mình ạ?",
		}
	}

	// ==========================================================
	// ƯU TIÊN 2: BẮT SỐ ĐIỆN THOẠI & ĐỊA CHỈ (KHI ĐANG CHỐT ĐƠN)
	// ==========================================================
	phone := extractPhoneNumber(trimmed)
	if phone != "" {
		sess.CustomerPhone = phone
	}

	if isValidAddress(trimmed) {
		sess.CustomerAddress = trimmed
	}

	// Kiểm tra nếu khách nhắn số lượng muốn mua
	qty := extractOrderQuantity(trimmed)
	if qty > 0 {
		sess.Quantity = qty
		sess.State = StateWaitingInfo
	}

	// ==========================================================
	// ƯU TIÊN 3: XỬ LÝ THEO TRẠNG THÁI HIỆN TẠI
	// ==========================================================

	// Trường hợp: Đã có sản phẩm và khách vừa chốt số lượng -> Xin địa chỉ & SĐT
	if sess.State == StateWaitingInfo && sess.CurrentProduct != nil {
		// Nếu đã đủ cả SĐT và Địa chỉ -> Xác nhận hoàn tất đơn hàng
		if sess.CustomerPhone != "" && sess.CustomerAddress != "" {
			sess.State = StateOrderConfirmed
			msg := fmt.Sprintf("Dạ Tổng kho HP FRUIT xin xác nhận lại đơn đặt hàng của Anh/Chị ạ:\n\n🍎 Sản phẩm: %s\n📦 Số lượng: %d thùng\n📍 Địa chỉ nhận hàng: %s\n📞 Số điện thoại: %s\n\nBên em đang chuẩn bị đóng hàng chuẩn đẹp gửi cho mình ngay ạ. Em cảm ơn Anh/Chị nhiều nhé!",
				sess.CurrentProduct.TenSP, sess.Quantity, sess.CustomerAddress, sess.CustomerPhone)

			return BotReply{Message: msg}
		}

		// Nếu còn thiếu SĐT hoặc Địa chỉ
		if sess.CustomerPhone == "" && sess.CustomerAddress == "" {
			return BotReply{
				Message: fmt.Sprintf("Dạ vâng, em đã lưu số lượng %d thùng mã %s của Anh/Chị rồi ạ!\n\nAnh/Chị cho em xin thông tin: ĐỊA CHỈ NHẬN HÀNG và SỐ ĐIỆN THOẠI để bên em lên đơn gửi hàng sớm cho mình nhé ạ!",
					sess.Quantity, sess.CurrentProduct.TenSP),
			}
		} else if sess.CustomerPhone == "" {
			return BotReply{
				Message: fmt.Sprintf("Dạ em đã ghi nhận địa chỉ nhận hàng của mình tại: %s ạ.\n\nAnh/Chị cho em xin thêm SỐ ĐIỆN THOẠI nhận hàng để nhà xe/shipper liên hệ giao hàng nhé ạ!", sess.CustomerAddress),
			}
		} else if sess.CustomerAddress == "" {
			return BotReply{
				Message: fmt.Sprintf("Dạ em đã ghi nhận số điện thoại của mình là %s ạ.\n\nAnh/Chị cho em xin thêm ĐỊA CHỈ NHẬN HÀNG cụ thể để bên em gửi hàng chuẩn xác nhé ạ!", sess.CustomerPhone),
			}
		}
	}

	// ==========================================================
	// ƯU TIÊN 4: KHÁCH HỎI TÊN SẢN PHẨM MỚI
	// ==========================================================
	matched := matchProduct(trimmed, products)
	if matched != nil {
		sess.CurrentProduct = matched
		sess.State = StateWaitingQuantity
		sess.Quantity = 0

		return BotReply{
			Message: fmt.Sprintf("Dạ mã %s bên em đang sẵn hàng tại kho, giá %s/thùng ạ.\n\nAnh/Chị lấy số lượng bao nhiêu thùng để em hỗ trợ lên đơn cho mình nhé ạ?",
				matched.TenSP, matched.Gia),
			ShouldSendImg: true,
			Product:       matched,
		}
	}

	// Câu chào hỏi mặc định
	return BotReply{
		Message: "Dạ em chào Anh/Chị ạ! Anh/Chị đang quan tâm mã trái cây nào bên em (Kiwi, Táo, Nho, Quýt, Cherry...) để em kiểm tra kho và báo giá chi tiết cho mình nhé ạ!",
	}
}
