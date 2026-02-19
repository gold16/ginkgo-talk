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

// AIMode represents the text processing mode.
type AIMode string

const (
	ModeRaw       AIMode = "raw"       // No processing, direct input
	ModeTidy      AIMode = "tidy"      // Remove duplicates, filler words, add punctuation
	ModeFormal    AIMode = "formal"    // Tidy + convert to formal/written style
	ModeTranslate AIMode = "translate" // Translate to/from English
)

// AIProcessor handles text processing via LLM API.
type AIProcessor struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewAIProcessor creates a new AI processor.
// It first reads saved config from disk, then falls back to environment variables.
func NewAIProcessor() *AIProcessor {
	// Load persistent config first
	cfg := LoadConfig()

	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("DEEPSEEK_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}

	model := cfg.Model
	if model == "" {
		model = os.Getenv("DEEPSEEK_MODEL")
	}
	if model == "" {
		model = "deepseek-chat"
	}

	return &AIProcessor{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsAvailable returns true if the API key is configured.
func (ai *AIProcessor) IsAvailable() bool {
	return ai.apiKey != ""
}

// SetAPIKey sets the API key at runtime.
func (ai *AIProcessor) SetAPIKey(key string) {
	ai.apiKey = strings.TrimSpace(key)
}

// Process processes text according to the given mode.
func (ai *AIProcessor) Process(text string, mode AIMode) (string, error) {
	if !ai.IsAvailable() {
		return text, fmt.Errorf("AI not configured: set DEEPSEEK_API_KEY environment variable")
	}

	if mode == ModeRaw || strings.TrimSpace(text) == "" {
		return text, nil
	}

	prompt := buildPrompt(text, mode)
	return ai.callAPI(prompt)
}

func buildPrompt(text string, mode AIMode) string {
	switch mode {
	case ModeTidy:
		return fmt.Sprintf(`你是一个语音转文字的文本整理助手。请对以下语音识别的原始文本进行整理：

规则：
1. 去除重复的词语（如说了两遍的词）
2. 去除口头禅和语气词（如"那个"、"就是"、"嗯"、"啊"、"然后"等）
3. 补充正确的标点符号
4. 保持原始含义，不添加、不删减实质内容
5. 只输出整理后的文本，不要解释

原始文本：%s`, text)

	case ModeFormal:
		return fmt.Sprintf(`你是一个语音转文字的写作助手。请将以下口语化的语音识别文本转换为正式书面语：

规则：
1. 去除所有重复词语和口头禅
2. 将口语表达转换为书面语
3. 优化语序，使表达更清晰流畅
4. 补充正确的标点符号
5. 保持原始含义不变，不添加新内容
6. 只输出修饰后的文本，不要解释

原始文本：%s`, text)

	case ModeTranslate:
		return fmt.Sprintf(`你是一个翻译助手。请判断以下文本的语言：
- 如果是中文，翻译成英文
- 如果是英文，翻译成中文
- 如果是其他语言，翻译成中文

规则：
1. 先去除口头禅和重复词语，再翻译
2. 翻译要自然流畅
3. 只输出翻译结果，不要解释

原始文本：%s`, text)

	default:
		return text
	}
}

// ChatMessage represents a message in the chat completion API.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the API request body.
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

// ChatResponse represents the API response.
type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (ai *AIProcessor) callAPI(prompt string) (string, error) {
	reqBody := ChatRequest{
		Model: ai.model,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   2048,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", ai.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ai.apiKey)

	resp, err := ai.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}
