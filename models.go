package main

type Product struct {
	MaSP        string `json:"ma_sp"`
	TenSP       string `json:"ten_sp"`
	DanhMuc     string `json:"danh_muc"`
	QuyCach     string `json:"quy_cach"`
	GiaLeThung  string `json:"gia_le_thung"`
	GiaSiLo     string `json:"gia_si_lo"`
	SoLuong     int    `json:"so_luong"`
	FolderAnhID string `json:"folder_anh_id"`
}

type GeminiBotResponse struct {
	Message       string   `json:"message"`
	SelectedCodes []string `json:"selected_codes"`
}

type UserSession struct {
	LastProduct Product
}

type MatchResult struct {
	Message     string
	LastProduct *Product
	PhotoURL    string
}
