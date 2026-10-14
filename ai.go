package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float32       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

var aiClient = &http.Client{Timeout: 5 * time.Second}

func buildFruitKnowledge(available []Product) string {
	var sb strings.Builder
	sb.WriteString("Bạn là nhân viên tư vấn của Tổng kho hoa quả & nông sản HP FRUIT (Bồ Đề, Long Biên, Hà Nội).\n")
	sb.WriteString("Tư vấn ngắn gọn (1-2 câu), thân thiện, lễ phép.\n")
	for _, p := range available {
		gia := p.GiaLeThung
		if gia == "" {
			gia = p.GiaSiLo
		}
		sb.WriteString(fmt.Sprintf("- %s: %s (%s)\n", p.TenSP, gia, p.ChatAn))
	}
	return sb.String()
}

func AskSmartAI(userMsg string, available []Product) string {
	apiKey := os.Getenv("GROQ_API_KEY")
	apiURL := "https://api.groq.com/openai/v1/chat/completions"
	modelName := "llama-3.3-70b-versatile"

	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
		apiURL = "https://api.openai.com/v1/chat/completions"
		modelName = "gpt-4o-mini"
	}

	if apiKey == "" {
		return ""
	}

	reqPayload := ChatRequest{
		Model:       modelName,
		Temperature: 0.3,
		MaxTokens:   200,
		Messages: []ChatMessage{
			{Role: "system", Content: buildFruitKnowledge(available)},
			{Role: "user", Content: userMsg},
		},
	}

	jsonBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return ""
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := aiClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return ""
	}

	if len(chatResp.Choices) > 0 {
		return strings.TrimSpace(chatResp.Choices[0].Message.Content)
	}
	return ""
}
