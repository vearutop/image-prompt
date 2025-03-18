// Package openai provides OpenAI API client.
package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// ImagePrompter can ask LLM about an image.
type ImagePrompter struct {
	AuthKey   string
	Model     string            // default "gpt-4o-mini".
	Transport http.RoundTripper // default http.DefaultTransport.
}

// ModelName returns the name of LLM.
func (ip *ImagePrompter) ModelName() string {
	if ip.Model == "" {
		return "gpt-4o-mini"
	}

	return ip.Model
}

// PromptImage asks LLM about JPEG image.
func (ip *ImagePrompter) PromptImage(ctx context.Context, prompt string, jpegImage io.Reader) (string, error) {
	img, err := io.ReadAll(jpegImage)
	if err != nil {
		return "", err
	}

	type ImageURL struct {
		URL string `json:"url"`
	}

	type Content struct {
		Type     string   `json:"type"`
		Text     string   `json:"text,omitempty"`
		ImageURL ImageURL `json:"image_url,omitempty"`
	}

	type Message struct {
		Role    string    `json:"role"`
		Content []Content `json:"content"`
	}

	type Req struct {
		Model     string    `json:"model"`
		Messages  []Message `json:"messages"`
		MaxTokens int       `json:"max_tokens"`
	}

	type Response struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int    `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role        string        `json:"role"`
				Content     string        `json:"content"`
				Refusal     interface{}   `json:"refusal"`
				Annotations []interface{} `json:"annotations"`
			} `json:"message"`
			Logprobs     interface{} `json:"logprobs"`
			FinishReason string      `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens        int `json:"prompt_tokens"`
			CompletionTokens    int `json:"completion_tokens"`
			TotalTokens         int `json:"total_tokens"`
			PromptTokensDetails struct {
				CachedTokens int `json:"cached_tokens"`
				AudioTokens  int `json:"audio_tokens"`
			} `json:"prompt_tokens_details"`
			CompletionTokensDetails struct {
				ReasoningTokens          int `json:"reasoning_tokens"`
				AudioTokens              int `json:"audio_tokens"`
				AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
				RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
			} `json:"completion_tokens_details"`
		} `json:"usage"`
		ServiceTier       string `json:"service_tier"`
		SystemFingerprint string `json:"system_fingerprint"`

		Error struct {
			Message string      `json:"message"`
			Type    string      `json:"type"`
			Param   interface{} `json:"param"`
			Code    string      `json:"code"`
		} `json:"error"`
	}

	req := Req{}
	req.Model = ip.Model
	req.Messages = append(req.Messages, Message{
		Role: "user",
		Content: []Content{
			{Type: "text", Text: prompt},
			{Type: "image_url", ImageURL: ImageURL{
				URL: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(img),
			}},
		},
	})
	req.MaxTokens = 300

	if req.Model == "" {
		req.Model = "gpt-4o-mini"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+ip.AuthKey)

	tr := ip.Transport
	if tr == nil {
		tr = http.DefaultTransport
	}

	resp, err := tr.RoundTrip(r)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close() //nolint:errcheck

	cont, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	re := Response{}

	if err := json.Unmarshal(cont, &re); err != nil {
		return "", err
	}

	if re.Error.Message != "" {
		return "", errors.New(re.Error.Message)
	}

	if len(re.Choices) == 0 {
		return "", errors.New("no choices found")
	}

	return strings.Trim(re.Choices[0].Message.Content, `" \t`), nil
}
