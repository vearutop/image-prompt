// Package cloudflare provides a client to self-hosted CloudFlare AI worker, see README.md for details.
package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/vearutop/image-prompt/imageprompt"
)

// NewImagePrompter creates a client to self-hosted CloudFlare AI worker.
func NewImagePrompter(baseURL string) (*ImagePrompter, error) {
	if baseURL == "" {
		return nil, errors.New("baseURL is empty")
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	auth := u.User.Username()
	u.User = nil

	baseURL = u.String()

	return &ImagePrompter{
		AuthKey: auth,
		BaseURL: baseURL,
	}, nil
}

// ImagePrompter can ask LLM about an image.
type ImagePrompter struct {
	BaseURL   string
	AuthKey   string
	Transport http.RoundTripper // default http.DefaultTransport.
}

// ModelName returns the name of LLM.
func (ip *ImagePrompter) ModelName() string {
	return "@cf/llava-hf/llava-1.5-7b-hf"
}

// PromptImage asks LLM about JPEG image.
func (ip *ImagePrompter) PromptImage(ctx context.Context, prompt string, jpegImage io.Reader) (string, error) {
	baseURL := ip.BaseURL
	if baseURL == "" {
		return "", errors.New("baseURL is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, jpegImage)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", ip.AuthKey)
	req.Header.Set("Prompt", prompt)

	tr := ip.Transport
	if tr == nil {
		tr = http.DefaultTransport
	}

	resp, err := tr.RoundTrip(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close() //nolint:errcheck

	cont, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == http.StatusServiceUnavailable && bytes.Contains(cont, []byte("Worker exceeded resource limits")) {
		return "", imageprompt.ErrResourceExhausted
	}

	type Resp struct {
		Description string `json:"description"`
	}

	re := Resp{}

	if err := json.Unmarshal(cont, &re); err != nil {
		return "", err
	}

	return strings.Trim(re.Description, `" \t`), nil
}
