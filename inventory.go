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

// FindProductInMemory tìm kiếm tức thời trong RAM không qua Gemini API
func FindProductInMemory(query string) string {
	q := removeDiacritics(strings.TrimSpace(query))
	if len(q) < 2 {
		return ""
	}

	inventoryMutex.RLock()
	items := make([]Product, len(Inventory))
	copy(items, Inventory)
	inventoryMutex.RUnlock()

	var matched []Product
	words := strings.Fields(q)

	for _, p := range items {
		name := removeDiacritics(p.TenSP + " " + p.MaSP + " " + p.QuyCach + " " + p.DanhMuc)
		matchAll := true
		for _, w := range words {
			if !strings.Contains(name, w) {
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

	if len(matched) == 1 {
		p := matched[0]
		status := "còn hàng sẵn kho"
		if p.SoLuong <= 0 {
			status = "tạm hết hàng"
		}
		return fmt.Sprintf("Dạ bên em đang sẵn %s (%s):\n- Giá sỉ lô: %s\n- Giá lẻ thùng: %s\n- Tình trạng: %s\n\nMình lấy số lượng bao nhiêu thùng để em hỗ trợ lên đơn và báo giá tốt nhất cho mình nhé?",
			p.TenSP, p.QuyCach, p.GiaSiLo, p.GiaLeThung, status)
	}

	if len(matched) <= 4 {
		res := "Dạ kho bên em đang có sẵn các mã sau:\n"
		for _, p := range matched {
			res += fmt.Sprintf("• %s (%s) - Sỉ: %s | Lẻ: %s\n", p.TenSP, p.QuyCach, p.GiaSiLo, p.GiaLeThung)
		}
		res += "\nMình quan tâm loại/size nào để em tư vấn chi tiết hơn ạ?"
		return res
	}

	return ""
}
