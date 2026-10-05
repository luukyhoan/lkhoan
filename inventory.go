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

type InventoryManager struct {
	SheetID         string
	CredentialsFile string
	mu              sync.RWMutex
	products        []Product
}

var (
	GlobalInventoryMgr *InventoryManager
	Inventory          []Product
	inventoryMutex     sync.RWMutex
)

func NewInventoryManager(sheetID, credentialsFile string) *InventoryManager {
	mgr := &InventoryManager{
		SheetID:         sheetID,
		CredentialsFile: credentialsFile,
	}
	GlobalInventoryMgr = mgr

	if err := mgr.Refresh(); err != nil {
		log.Printf("[Sheets] Khởi tạo dữ liệu kho thất bại: %v", err)
	}

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

func (im *InventoryManager) GetAvailableProducts() []Product {
	im.mu.RLock()
	defer im.mu.RUnlock()
	res := make([]Product, len(im.products))
	copy(res, im.products)
	return res
}

func removeDiacritics(str string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, str)
	return strings.ToLower(result)
}

// FindProductInMemory tìm kiếm nhanh trong RAM, giữ kín giá tiền và kèm ảnh
func FindProductInMemory(query string) *MatchResult {
	q := removeDiacritics(strings.TrimSpace(query))
	if len(q) < 2 {
		return nil
	}

	skipKeywords := []string{"chao", "xin chao", "alo", "dia chi", "so dien thoai", "thung", "lay", "mua an", "kinh doanh", "gia dinh", "bieu"}
	for _, kw := range skipKeywords {
		if q == kw {
			return nil
		}
	}

	inventoryMutex.RLock()
	items := make([]Product, len(Inventory))
	copy(items, Inventory)
	inventoryMutex.RUnlock()

	var matched []Product
	words := strings.Fields(q)

	for _, p := range items {
		if p.SoLuong <= 0 {
			continue
		}

		searchTarget := removeDiacritics(p.TenSP + " " + p.MaSP + " " + p.DanhMuc + " " + p.QuyCach)
		matchAll := true
		for _, w := range words {
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
		return nil
	}

	// Khớp 1 sản phẩm cụ thể
	if len(matched) == 1 {
		p := matched[0]
		msg := fmt.Sprintf("Dạ bên em đang sẵn %s (%s) hàng bay mới về, quả chắc cuống tươi chuẩn đẹp ạ.\n\nAnh/Chị dự tính lấy số lượng dùng gia đình, làm quà biếu hay lấy cho shop/cửa hàng để em hỗ trợ chính sách giá tốt nhất cho mình ạ?",
			p.TenSP, p.QuyCach)

		return &MatchResult{
			Message:     msg,
			LastProduct: &p,
			PhotoURL:    p.FolderAnhID,
		}
	}

	// Khớp theo nhóm quả (cam, nho, táo...)
	if len(matched) <= 5 {
		msg := "Dạ kho HP FRUIT đang sẵn các dòng sau hàng mới về tuyển chọn rất đẹp ạ:\n"
		for _, p := range matched {
			msg += fmt.Sprintf("• %s (%s)\n", p.TenSP, p.QuyCach)
		}
		msg += "\nAnh/Chị đang quan tâm dòng nào, dự tính dùng gia đình hay lấy cho cửa hàng để em gửi hình ảnh thực tế và báo chính sách giá tốt nhất ạ?"

		return &MatchResult{
			Message:     msg,
			LastProduct: &matched[0],
			PhotoURL:    matched[0].FolderAnhID,
		}
	}

	return nil
}
