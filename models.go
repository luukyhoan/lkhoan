package main

import "time"

type Product struct {
	MaSP        string `json:"ma_sp"`
	TenSP       string `json:"ten_sp"`
	DanhMuc     string `json:"danh_muc"`
	QuyCach     string `json:"quy_cach"`
	GiaLeThung  string `json:"gia_le_thung"`
	GiaSiLo     string `json:"gia_si_lo"`
	SoLuong     int    `json:"so_luong"`
	FolderAnhID string `json:"folder_anh_id"`
	HinhThuc    string `json:"hinh_thuc"`
	ChatAn      string `json:"chat_an"`
}

type UserSession struct {
	LastProduct          Product
	FollowupTimer        *time.Timer
	InvitedToGroup       bool
	LastAdminMessageTime time.Time

	// Quản lý quy trình chốt đơn
	OrderStep            string // "" -> "AWAITING_INFO" -> "CONFIRMED"
	OrderQuantity        string
	CustomerPhone        string
	CustomerAddress      string
}
