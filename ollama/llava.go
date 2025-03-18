// Package ollama provides a client to a local or remote self-hosted LLM service, see https://ollama.com/.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// ImagePrompter can ask LLM about an image.
type ImagePrompter struct {
	BaseURL   string            // default "http://localhost:11434/api/generate".
	Model     string            // default "llava:7b".
	Transport http.RoundTripper // default http.DefaultTransport.
}

// ModelName returns the name of LLM.
func (ip *ImagePrompter) ModelName() string {
	if ip.Model == "" {
		return "llava:7b"
	}

	return ip.Model
}

// PromptImage asks LLM about JPEG image.
func (ip *ImagePrompter) PromptImage(ctx context.Context, prompt string, jpegImage io.Reader) (string, error) {
	type Req struct {
		Model  string   `json:"model"`
		Prompt string   `json:"prompt"`
		Stream bool     `json:"stream"`
		Images [][]byte `json:"images"`
	}

	cont, err := io.ReadAll(jpegImage)
	if err != nil {
		return "", err
	}

	r := Req{}

	r.Model = ip.Model
	r.Prompt = prompt
	r.Stream = false
	r.Images = append(r.Images, cont)

	if r.Model == "" {
		r.Model = "llava:7b"
	}

	body, err := json.Marshal(r)
	if err != nil {
		return "", err
	}

	baseURL := ip.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434/api/generate"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	tr := ip.Transport
	if tr == nil {
		tr = http.DefaultTransport
	}

	resp, err := tr.RoundTrip(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close() //nolint:errcheck

	cont, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type Resp struct {
		Response string `json:"response"`
	}

	re := Resp{}

	if err := json.Unmarshal(cont, &re); err != nil {
		return "", err
	}

	return strings.Trim(re.Response, `" \t`), nil
}
