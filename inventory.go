package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type InventoryManager struct {
	spreadsheetID string
	credFile      string
	mu            sync.RWMutex
	products      []Product
}

func extractFolderID(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	re := regexp.MustCompile(`folders/([a-zA-Z0-9_-]+)`)
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return matches[1]
	}
	return input
}

func NewInventoryManager(sheetID, credFile string) *InventoryManager {
	im := &InventoryManager{
		spreadsheetID: sheetID,
		credFile:      credFile,
	}
	im.Reload()
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			im.Reload()
		}
	}()
	return im
}

func (im *InventoryManager) Reload() {
	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithCredentialsFile(im.credFile), option.WithScopes(sheets.SpreadsheetsReadonlyScope))
	if err != nil {
		log.Printf("[Sheets] Lỗi khởi tạo service: %v", err)
		return
	}

	readRange := "Bang_Gia!A2:H"
	resp, err := srv.Spreadsheets.Values.Get(im.spreadsheetID, readRange).Do()
	if err != nil {
		readRange = "Sheet1!A2:H"
		resp, err = srv.Spreadsheets.Values.Get(im.spreadsheetID, readRange).Do()
		if err != nil {
			log.Printf("[Sheets] Không thể đọc dữ liệu: %v", err)
			return
		}
	}

	var list []Product
	for _, row := range resp.Values {
		if len(row) == 0 {
			continue
		}

		getVal := func(idx int) string {
			if idx < len(row) && row[idx] != nil {
				return fmt.Sprintf("%v", row[idx])
			}
			return ""
		}

		maSP := strings.TrimSpace(getVal(0))
		tenSP := strings.TrimSpace(getVal(1))
		if maSP == "" || tenSP == "" {
			continue
		}

		qtyStr := strings.TrimSpace(getVal(6))
		qty, _ := strconv.Atoi(qtyStr)
		if qtyStr == "" {
			qty = 10
		}

		folderRaw := getVal(7)
		folderID := extractFolderID(folderRaw)

		p := Product{
			MaSP:        maSP,
			TenSP:       tenSP,
			DanhMuc:     strings.TrimSpace(getVal(2)),
			QuyCach:     strings.TrimSpace(getVal(3)),
			GiaLeThung:  strings.TrimSpace(getVal(4)),
			GiaSiLo:     strings.TrimSpace(getVal(5)),
			SoLuong:     qty,
			FolderAnhID: folderID,
		}
		list = append(list, p)
	}

	im.mu.Lock()
	im.products = list
	im.mu.Unlock()
	log.Printf("[Sheets] Đã load thành công %d mã hàng vào RAM", len(list))
}

func (im *InventoryManager) GetAvailableProducts() []Product {
	im.mu.RLock()
	defer im.mu.RUnlock()

	var available []Product
	for _, p := range im.products {
		if p.SoLuong >= 2 {
			available = append(available, p)
		}
	}
	return available
}
