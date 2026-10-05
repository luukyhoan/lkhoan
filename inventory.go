package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// InventoryManager quản lý danh mục và đồng bộ tồn kho
type InventoryManager struct {
	SheetID         string
	CredentialsFile string
	mu              sync.RWMutex
	products        []Product
}

// Biến toàn cục để tra cứu nhanh trong RAM
var (
	GlobalInventoryMgr *InventoryManager
	Inventory          []Product
	inventoryMutex     sync.RWMutex
)

// NewInventoryManager khởi tạo bộ quản lý kho và chạy đồng bộ nền
func NewInventoryManager(sheetID, credentialsFile string) *InventoryManager {
	mgr := &InventoryManager{
		SheetID:         sheetID,
		CredentialsFile: credentialsFile,
	}
	GlobalInventoryMgr = mgr

	// Tải lần đầu tiên khi khởi động
	if err := mgr.Refresh(); err != nil {
		log.Printf("[Sheets] Khởi tạo dữ liệu kho thất bại: %v", err)
	}

	// Chu kỳ tự động tải lại dữ liệu từ Google Sheets mỗi 5 phút
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := mgr.Refresh(); err != nil {
				log.Printf("[Sheets] Không thể đọc dữ liệu: %v", err)
			}
		}
	}()

	return mgr
}

// Refresh đọc dữ liệu từ tab Bang_Gia trên Google Sheets vào RAM
func (im *InventoryManager) Refresh() error {
	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithCredentialsFile(im.CredentialsFile), option.WithScopes(sheets.SpreadsheetsReadonlyScope))
	if err != nil {
		return fmt.Errorf("không thể kết nối Sheets Service: %w", err)
	}

	readRange := "Bang_Gia!A2:H"
	resp, err := srv.Spreadsheets.Values.Get(im.SheetID, readRange).Do()
	if err != nil {
		return err
	}

	var prods []Product
	for _, row := range resp.Values {
		if len(row) < 2 {
			continue
		}

		p := Product{
			MaSP:  fmt.Sprintf("%v", row[0]),
			TenSP: fmt.Sprintf("%v", row[1]),
		}

		if len(row) > 2 {
			p.DanhMuc = fmt.Sprintf("%v", row[2])
		}
		if len(row) > 3 {
			p.QuyCach = fmt.Sprintf("%v", row[3])
		}
		if len(row) > 4 {
			p.GiaLeThung = fmt.Sprintf("%v", row[4])
		}
		if len(row) > 5 {
			p.GiaSiLo = fmt.Sprintf("%v", row[5])
		}
		if len(row) > 6 {
			slStr := strings.TrimSpace(fmt.Sprintf("%v", row[6]))
			if sl, e := strconv.Atoi(slStr); e == nil {
				p.SoLuong = sl
			}
		}
		if len(row) > 7 {
			p.FolderAnhID = fmt.Sprintf("%v", row[7])
		}

		if p.MaSP != "" || p.TenSP != "" {
			prods = append(prods, p)
		}
	}

	im.mu.Lock()
	im.products = prods
	im.mu.Unlock()

	inventoryMutex.Lock()
	Inventory = prods
	inventoryMutex.Unlock()

	log.Printf("[Sheets] Đã load thành công %d mã hàng vào RAM", len(prods))
	return nil
}

// GetAvailableProducts lấy danh sách sản phẩm
func (im *InventoryManager) GetAvailableProducts() []Product {
	im.mu.RLock()
	defer im.mu.RUnlock()
	res := make([]Product, len(im.products))
	copy(res, im.products)
	return res
}

// removeDiacritics loại bỏ dấu tiếng Việt để đối soát từ khóa
func removeDiacritics(str string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, str)
	return strings.ToLower(result)
}

// FindProductInMemory tìm kiếm thông minh từ RAM: nhận diện mã, tên hoặc nhóm quả
func FindProductInMemory(query string) string {
	q := removeDiacritics(strings.TrimSpace(query))
	if len(q) < 2 {
		return ""
	}

	// Bỏ qua các câu chào hỏi hoặc câu khẳng định số lượng (để Gemini tiếp tục hội thoại)
	skipKeywords := []string{"chao", "xin chao", "alo", "dia chi", "so dien thoai", "thung", "lay", "mua an", "kinh doanh"}
	for _, kw := range skipKeywords {
		if q == kw {
			return ""
		}
	}

	inventoryMutex.RLock()
	items := make([]Product, len(Inventory))
	copy(items, Inventory)
	inventoryMutex.RUnlock()

	var matched []Product
	words := strings.Fields(q)

	// 1. Tìm khớp trực tiếp theo tên, mã hoặc danh mục
	for _, p := range items {
		// Bỏ qua sản phẩm đã hết hàng trong kho nếu tìm nhóm chung
		if p.SoLuong <= 0 {
			continue
		}

		searchTarget := removeDiacritics(p.TenSP + " " + p.MaSP + " " + p.DanhMuc + " " + p.QuyCach)
		matchAll := true
		for _, w := range words {
			// Bỏ qua các từ nối phổ biến khi tìm
			if w == "co" || w == "khong" || w == "em" || w == "shop" || w == "ban" || w == "cho" || w == "gia" {
				continue
			}
			if !strings.Contains(searchTarget, w) {
				matchAll = false
				break
			}
		}
		if matchAll {
			matched = append(matched, p)
		}
	}

	if len(matched) == 0 {
		return ""
	}

	// Trường hợp 1: Khách hỏi đúng 1 mã quả cụ thể (ví dụ: "kiwi nzl s22", "cam nam phi s55")
	if len(matched) == 1 {
		p := matched[0]
		return fmt.Sprintf("Dạ bên em đang sẵn %s (%s) hàng chuẩn nhập khẩu cao cấp ạ:\n• Giá lẻ: %s/thùng\n• Giá sỉ lô: %s\n\nAnh/Chị dự tính lấy số lượng dùng gia đình, làm quà biếu hay lấy cho cửa hàng/đại lý để em áp dụng chính sách giá tốt nhất ạ?",
			p.TenSP, p.QuyCach, p.GiaLeThung, p.GiaSiLo)
	}

	// Trường hợp 2: Khách hỏi theo nhóm/chủng loại (ví dụ: "có cam không", "bên em có nho không", "táo")
	if len(matched) <= 6 {
		res := "Dạ kho HP FRUIT đang sẵn các dòng sau mới về, chất lượng tuyển chọn rất đẹp ạ:\n"
		for _, p := range matched {
			res += fmt.Sprintf("• %s (%s) — Lẻ: %s/thùng | Sỉ lô: %s\n", p.TenSP, p.QuyCach, p.GiaLeThung, p.GiaSiLo)
		}
		res += "\nAnh/Chị đang quan tâm dòng nào, dự tính dùng gia đình hay lấy cho cửa hàng để em gửi hình ảnh thực tế và tư vấn kỹ hơn ạ?"
		return res
	}

	return ""
}
