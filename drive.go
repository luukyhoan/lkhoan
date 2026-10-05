package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
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

// ExtractFolderID bóc tách folder ID từ link Google Drive
func ExtractFolderID(folderURL string) string {
	folderURL = strings.TrimSpace(folderURL)
	if folderURL == "" {
		return ""
	}
	if strings.Contains(folderURL, "/folders/") {
		parts := strings.Split(folderURL, "/folders/")
		if len(parts) > 1 {
			idPart := parts[1]
			if idx := strings.Index(idPart, "?"); idx != -1 {
				return idPart[:idx]
			}
			return idPart
		}
	}
	if u, err := url.Parse(folderURL); err == nil {
		if id := u.Query().Get("id"); id != "" {
			return id
		}
	}
	return folderURL
}

// GetImageLinksInFolder lấy link xem trực tiếp của tối đa 5 ảnh trong folder
func (d *DriveHelper) GetImageLinksInFolder(folderURL string) []string {
	if d == nil || d.srv == nil {
		return nil
	}
	folderID := ExtractFolderID(folderURL)
	if folderID == "" {
		return nil
	}

	q := fmt.Sprintf("'%s' in parents and mimeType contains 'image/' and trashed = false", folderID)
	r, err := d.srv.Files.List().Q(q).Fields("files(id, name)").PageSize(5).Do()
	if err != nil {
		log.Printf("[Drive] Lỗi đọc ảnh trong folder %s: %v", folderID, err)
		return nil
	}

	var directLinks []string
	for _, f := range r.Files {
		// Link download/view trực tiếp để Facebook attachment tải được
		link := fmt.Sprintf("https://drive.google.com/uc?export=view&id=%s", f.Id)
		directLinks = append(directLinks, link)
	}
	return directLinks
}
