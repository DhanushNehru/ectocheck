// Package ollama is a minimal client for a local Ollama server's chat API.
// It exists so the rest of ectocheck never thinks about HTTP.
package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to one Ollama server with one model.
type Client struct {
	base  string
	model string
	http  *http.Client
}

// New builds a client for base (e.g. http://127.0.0.1:11434) and model.
func New(base, model string) *Client {
	return &Client{
		base:  strings.TrimRight(base, "/"),
		model: model,
		http:  &http.Client{Timeout: 5 * time.Minute},
	}
}

type chatRequest struct {
	Model    string       `json:"model"`
	Messages []Message    `json:"messages"`
	Stream   bool         `json:"stream"`
	Options  *chatOptions `json:"options,omitempty"`
}

type chatOptions struct {
	Temperature float64 `json:"temperature"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// Message is one chat turn.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Message Message `json:"message"`
}

// Chat sends a system+user pair and returns the assistant's reply.
// Temperature is pinned low: ectocheck wants judgment, not creativity.
func (c *Client) Chat(system, user string) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []Message{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Stream:  false,
		Options: &chatOptions{Temperature: 0.1, NumPredict: 120},
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Post(c.base+"/api/chat", "application/json", bytes.NewReader(buf))
	if err != nil {
		return "", fmt.Errorf("ollama chat: %w (is `ollama serve` running?)", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama chat: %s: %s", resp.Status, truncate(string(body), 200))
	}
	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("ollama chat: decoding reply: %w", err)
	}
	return strings.TrimSpace(parsed.Message.Content), nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
