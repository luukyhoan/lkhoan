package main

import (
	"context"
	"fmt"
	"io"
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

type DriveImageFile struct {
	Filename string
	Data     []byte
}

func (d *DriveHelper) GetImageFilesFromFolder(folderURL string) []DriveImageFile {
	if d == nil || d.srv == nil {
		return nil
	}
	folderID := ExtractFolderID(folderURL)
	if folderID == "" {
		return nil
	}

	q := fmt.Sprintf("'%s' in parents and mimeType contains 'image/' and trashed = false", folderID)
	r, err := d.srv.Files.List().Q(q).Fields("files(id, name)").PageSize(4).Do()
	if err != nil {
		log.Printf("[Drive] Lỗi đọc folder %s: %v", folderID, err)
		return nil
	}

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
