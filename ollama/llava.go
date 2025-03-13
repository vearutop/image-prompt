package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type ImagePrompter struct {
	BaseURL   string            // default "http://localhost:11434/api/generate".
	Model     string            // default "llava:7b".
	Transport http.RoundTripper // default http.DefaultTransport.
}

func (ip *ImagePrompter) PromptImage(ctx context.Context, prompt string, image io.ReadCloser) (string, error) {
	type Req struct {
		Model  string   `json:"model"`
		Prompt string   `json:"prompt"`
		Stream bool     `json:"stream"`
		Images [][]byte `json:"images"`
	}

	cont, err := io.ReadAll(image)
	if err != nil {
		return "", err
	}

	if err := image.Close(); err != nil {
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

	defer resp.Body.Close()

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
