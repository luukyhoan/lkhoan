package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type DriveHelper struct {
	srv *drive.Service
}

var GlobalDriveHelper *DriveHelper

func InitDriveHelper(credentialsFile string) {
	ctx := context.Background()
	srv, err := drive.NewService(ctx, option.WithCredentialsFile(credentialsFile), option.WithScopes(drive.DriveReadonlyScope))
	if err != nil {
		log.Printf("[Drive] Khởi tạo Drive Service thất bại: %v", err)
		return
	}
	GlobalDriveHelper = &DriveHelper{srv: srv}
	log.Println("[Drive] Khởi tạo Google Drive Service thành công")
}

// ExtractFolderID dùng regex bóc chuẩn xác ID thư mục kể cả khi có ?hl=vi hay tham số URL
func ExtractFolderID(folderURL string) string {
	folderURL = strings.TrimSpace(folderURL)
	if folderURL == "" {
		return ""
	}

	// Bắt ID dạng chuỗi ký tự sau /folders/
	re := regexp.MustCompile(`/folders/([a-zA-Z0-9_-]+)`)
	matches := re.FindStringSubmatch(folderURL)
	if len(matches) > 1 {
		return matches[1]
	}

	// Nếu link có dạng id=xxx
	reID := regexp.MustCompile(`[?&]id=([a-zA-Z0-9_-]+)`)
	matchesID := reID.FindStringSubmatch(folderURL)
	if len(matchesID) > 1 {
		return matchesID[1]
	}

	return folderURL
}

type DriveImageFile struct {
	Filename string
	Data     []byte
}

func (d *DriveHelper) GetImageFilesFromFolder(folderURL string) []DriveImageFile {
	if d == nil || d.srv == nil {
		log.Printf("[Drive] Drive Service chưa được khởi tạo")
		return nil
	}
	folderID := ExtractFolderID(folderURL)
	if folderID == "" {
		log.Printf("[Drive] Không tìm thấy Folder ID từ link: %s", folderURL)
		return nil
	}

	log.Printf("[Drive] Đang quét ảnh từ Folder ID: %s", folderID)

	q := fmt.Sprintf("'%s' in parents and mimeType contains 'image/' and trashed = false", folderID)
	r, err := d.srv.Files.List().Q(q).Fields("files(id, name)").PageSize(4).Do()
	if err != nil {
		log.Printf("[Drive] Lỗi đọc folder %s: %v (Kiểm tra lại quyền chia sẻ thư mục cho Service Account)", folderID, err)
		return nil
	}

	log.Printf("[Drive] Tìm thấy %d ảnh trong thư mục %s", len(r.Files), folderID)

	var results []DriveImageFile
	for _, f := range r.Files {
		resp, err := d.srv.Files.Get(f.Id).Download()
		if err != nil {
			log.Printf("[Drive] Không thể tải file %s: %v", f.Name, err)
			continue
		}
		b, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err == nil && len(b) > 0 {
			results = append(results, DriveImageFile{
				Filename: f.Name,
				Data:     b,
			})
		}
	}
	return results
}
