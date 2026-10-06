package main

type Product struct {
	MaSP        string `json:"ma_sp"`
	TenSP       string `json:"ten_sp"`
	XuatXu      string `json:"xuat_xu"`
	QuyCach     string `json:"quy_cach"`
	GiaLeThung  string `json:"gia_le_thung"`
	GiaSiLo     string `json:"gia_si_lo"`
	SoLuong     int    `json:"so_luong"`
	FolderAnhID string `json:"folder_anh_id"`
	HinhThuc    string `json:"hinh_thuc"` // Cột I trên Sheet
	ChatAn      string `json:"chat_an"`   // Cột J trên Sheet
}

type UserSession struct {
	LastProduct Product
}
