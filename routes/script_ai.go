package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type aiChatRequest struct {
	Messages      []aiMessage `json:"messages"`
	CurrentSource string      `json:"current_source"`
}

type aiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func handleScriptAI(c *gin.Context) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "gemini_key_missing",
			"message": "GEMINI_API_KEY is not configured.",
		})
		return
	}

	var req aiChatRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messages array is required"})
		return
	}

	modelName := "gemini-2.5-flash"

	systemPrompt := dslSystemPrompt
	if req.CurrentSource != "" {
		systemPrompt += "\n\n## Current Editor Content\n\nThe user is currently editing this SentanylScript source:\n\n```dsl\n" +
			req.CurrentSource + "\n```\n"
	}

	var contents []map[string]interface{}
	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" || role == "model" {
			role = "model"
		} else {
			role = "user"
		}
		contents = append(contents, map[string]interface{}{
			"role":  role,
			"parts": []map[string]string{{"text": msg.Content}},
		})
	}

	geminiReq := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature":     0.4,
			"maxOutputTokens": 16384,
			"thinkingConfig": map[string]interface{}{
				"thinkingBudget": 4096,
			},
		},
	}

	body, _ := json.Marshal(geminiReq)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		modelName, apiKey)

	const timeoutDuration = 180 * time.Second

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutDuration)
	defer cancel()

	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded || strings.Contains(err.Error(), "deadline") || strings.Contains(err.Error(), "timeout") {
			log.Printf("[AI] Gemini API timeout after %v: %v", timeoutDuration, err)
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error":   "gemini_timeout",
				"message": fmt.Sprintf("Gemini API took too long (max %v). Try a smaller prompt.", timeoutDuration),
			})
			return
		}

		log.Printf("[AI] Gemini API unreachable: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "gemini_unavailable",
			"message": "Failed to reach Gemini API",
		})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		log.Printf("[AI] Gemini API returned %d: %s", resp.StatusCode, string(respBody))
		switch {
		case resp.StatusCode == 401 || resp.StatusCode == 403:
			c.JSON(http.StatusBadGateway, gin.H{"error": "gemini_key_invalid", "message": "Invalid Gemini API key"})
		case resp.StatusCode == 429:
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "gemini_quota_exceeded", "message": "Gemini quota exceeded"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal",
				"message": fmt.Sprintf("Gemini error %d", resp.StatusCode),
			})
		}
		return
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text    string `json:"text"`
					Thought bool   `json:"thought,omitempty"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		log.Printf("[AI] Failed to parse Gemini response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal", "message": "Failed to parse Gemini response"})
		return
	}

	reply := ""
	if len(geminiResp.Candidates) > 0 {
		for _, part := range geminiResp.Candidates[0].Content.Parts {
			if !part.Thought {
				reply += part.Text
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"reply": reply})
}
