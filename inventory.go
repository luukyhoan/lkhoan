package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type InventoryManager struct {
	sheetID         string
	credentialsFile string
	products        []Product
	mutex           sync.RWMutex
}

var GlobalInventory *InventoryManager

func NewInventoryManager(sheetID, credentialsFile string) *InventoryManager {
	mgr := &InventoryManager{
		sheetID:         sheetID,
		credentialsFile: credentialsFile,
	}
	GlobalInventory = mgr
	mgr.refreshData()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			mgr.refreshData()
		}
	}()

	return mgr
}

func (im *InventoryManager) refreshData() {
	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithCredentialsFile(im.credentialsFile), option.WithScopes(sheets.SpreadsheetsReadonlyScope))
	if err != nil {
		log.Printf("[Sheets] Lỗi tạo client: %v", err)
		return
	}

	readRange := "Bang_Gia!A2:J"
	resp, err := srv.Spreadsheets.Values.Get(im.sheetID, readRange).Do()
	if err != nil {
		log.Printf("[Sheets] Lỗi đọc dữ liệu sheet: %v", err)
		return
	}

	var prods []Product
	for _, row := range resp.Values {
		if len(row) < 2 {
			continue
		}

		getCol := func(idx int) string {
			if idx < len(row) {
				return strings.TrimSpace(fmt.Sprintf("%v", row[idx]))
			}
			return ""
		}

		qty, _ := strconv.Atoi(getCol(6))

		p := Product{
			MaSP:        getCol(0), // Cột A: Mã SP
			TenSP:       getCol(1), // Cột B: Tên SP
			XuatXu:      getCol(2), // Cột C: Xuất xứ
			QuyCach:     getCol(3), // Cột D: Quy cách
			GiaLeThung:  getCol(4), // Cột E: Giá lẻ
			GiaSiLo:     getCol(5), // Cột F: Giá sỉ
			SoLuong:     qty,       // Cột G: Số lượng
			FolderAnhID: getCol(7), // Cột H: Link Folder ảnh
			HinhThuc:    getCol(8), // Cột I: Hinh_Thuc (Hàng Bay / Hàng Cont / Nông Sản Việt)
			ChatAn:      getCol(9), // Cột J: Chat_An (Mô tả chất ăn lô thực tế)
		}

		if p.TenSP != "" {
			prods = append(prods, p)
		}
	}

	im.mutex.Lock()
	im.products = prods
	im.mutex.Unlock()

	log.Printf("[Sheets] Đã load thành công %d mã hàng vào RAM", len(prods))
}

func (im *InventoryManager) GetAvailableProducts() []Product {
	im.mutex.RLock()
	defer im.mutex.RUnlock()
	return im.products
}
