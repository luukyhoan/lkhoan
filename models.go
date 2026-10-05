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

type MetaCallback struct {
	Entry []struct {
		Messaging []struct {
			Sender struct {
				ID string `json:"id"`
			} `json:"sender"`
			Message struct {
				Text string `json:"text"`
			} `json:"message"`
			Postback struct {
				Payload string `json:"payload"`
			} `json:"postback"`
		} `json:"messaging"`
	} `json:"entry"`
}
type UserSession struct {
	LastProduct Product
	LastAction  string // "WAITING_ROLE"
}
