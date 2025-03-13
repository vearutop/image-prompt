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

	"github.com/swaggest/usecase/status"
)

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
		Auth:    auth,
		BaseURL: baseURL,
	}, nil
}

type ImagePrompter struct {
	BaseURL   string
	Auth      string
	Transport http.RoundTripper // default http.DefaultTransport.
}

func (ip *ImagePrompter) PromptImage(ctx context.Context, prompt string, image io.ReadCloser) (string, error) {
	baseURL := ip.BaseURL
	if baseURL == "" {
		return "", errors.New("baseURL is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, image)
	if err != nil {
		return "", err
	}

	defer image.Close()

	req.Header.Set("Authorization", ip.Auth)
	req.Header.Set("Prompt", prompt)

	tr := ip.Transport
	if tr == nil {
		tr = http.DefaultTransport
	}

	resp, err := tr.RoundTrip(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	cont, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == http.StatusServiceUnavailable && bytes.Contains(cont, []byte("Worker exceeded resource limits")) {
		return "", status.ResourceExhausted
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
