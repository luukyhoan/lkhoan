package main

type Product struct {
	MaSP        string `json:"ma_sp"`        // Cột A (0)
	TenSP       string `json:"ten_sp"`       // Cột B (1)
	DanhMuc     string `json:"danh_muc"`     // Cột C (2)
	QuyCach     string `json:"quy_cach"`     // Cột D (3)
	GiaLeThung  string `json:"gia_le_thung"`  // Cột E (4)
	GiaSiLo     string `json:"gia_si_lo"`     // Cột F (5)
	SoLuong     int    `json:"so_luong"`     // Cột G (6)
	FolderAnhID string `json:"folder_anh_id"` // Cột H (7)
	HinhThuc    string `json:"hinh_thuc"`    // Cột I (8)
	ChatAn      string `json:"chat_an"`      // Cột J (9)
}

type UserSession struct {
	LastProduct Product
}
