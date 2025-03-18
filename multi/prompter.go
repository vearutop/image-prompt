// Package multi provides image prompter for multiple prompts and providers.
package multi

import (
	"context"
	"errors"
	"io"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/vearutop/image-prompt/cloudflare"
	"github.com/vearutop/image-prompt/gemini"
	"github.com/vearutop/image-prompt/imageprompt"
	"github.com/vearutop/image-prompt/ollama"
	"github.com/vearutop/image-prompt/openai"
)

// ImagePrompter can ask LLMs about an image.
type ImagePrompter struct {
	mu                     sync.Mutex
	prompterExhaustedUntil map[string]time.Time

	rng *rand.Rand

	cfgAccessor func() Config
}

// NewImagePrompter creates image prompter with config accessor.
//
// Configuration is reevaluated for each request and can be changed in runtime.
func NewImagePrompter(cfg func() Config) *ImagePrompter {
	mp := &ImagePrompter{}

	mp.rng = rand.New(rand.NewPCG(1, 1))
	mp.cfgAccessor = cfg

	return mp
}

func (ip *ImagePrompter) pp(cfg Config) (string, Provider, error) {
	if len(cfg.Providers) == 0 || len(cfg.Prompts) == 0 {
		return "", Provider{}, imageprompt.ErrEmptyConfig
	}

	ip.mu.Lock()
	defer ip.mu.Unlock()

	prompt := cfg.Prompts[0].Prompt
	provider := Provider{}

	sumWeight := 0
	for _, pr := range cfg.Prompts {
		sumWeight += pr.Weight
	}

	r := ip.rng.IntN(sumWeight)

	sumWeight = 0
	for _, pr := range cfg.Prompts {
		sumWeight += pr.Weight

		if sumWeight >= r {
			prompt = pr.Prompt

			break
		}
	}

	sumWeight = 0
	exhaustedFound := false

	for i, pr := range cfg.Providers {
		key := pr.Provider.string()

		exhausted := ip.prompterExhaustedUntil[pr.Provider.string()]
		if !exhausted.IsZero() {
			if exhausted.Before(time.Now()) {
				ip.prompterExhaustedUntil[key] = time.Time{}
			} else {
				pr.Weight = 0
				cfg.Providers[i] = pr
				exhaustedFound = true
			}
		}

		sumWeight += pr.Weight
	}

	r = ip.rng.IntN(sumWeight)

	for _, pr := range cfg.Providers {
		sumWeight += pr.Weight

		if sumWeight >= r {
			provider = pr.Provider

			break
		}
	}

	if provider.Type == "" && exhaustedFound {
		return "", provider, imageprompt.ErrResourceExhausted
	}

	return prompt, provider, nil
}

func (p Provider) string() string {
	return string(p.Type) + p.AuthKey + p.Model + p.BaseURL
}

func (p Provider) prompter() (imageprompt.Prompter, error) {
	switch p.Type {
	case OpenAI:
		return &openai.ImagePrompter{
			AuthKey: p.AuthKey,
			Model:   p.Model,
		}, nil
	case Gemini:
		return &gemini.ImagePrompter{
			AuthKey: p.AuthKey,
		}, nil
	case Ollama:
		return &ollama.ImagePrompter{
			Model: p.Model,
		}, nil
	case CloudFlare:
		return cloudflare.NewImagePrompter(p.BaseURL)
	default:
		return nil, errors.New("no provider")
	}
}

// Result is the prompt response.
type Result struct {
	Text   string `json:"text,omitempty"`
	Model  string `json:"model,omitempty"`
	Prompt string `json:"prompt,omitempty"`
}

// PromptImage asks LLM about JPEG image with one of predefined prompts.
func (ip *ImagePrompter) PromptImage(ctx context.Context, jpegImage io.Reader) (Result, error) {
	prompt, provider, err := ip.pp(ip.cfgAccessor())
	if err != nil {
		return Result{}, err
	}

	prompter, err := provider.prompter()
	if err != nil {
		return Result{}, err
	}

	res, err := prompter.PromptImage(ctx, prompt, jpegImage)
	if err != nil {
		return Result{}, err
	}

	if errors.Is(err, imageprompt.ErrResourceExhausted) {
		ip.mu.Lock()
		defer ip.mu.Unlock()

		ip.prompterExhaustedUntil[provider.string()] = time.Now().Add(time.Minute)

		return ip.PromptImage(ctx, jpegImage)
	}

	return Result{
		Text:   res,
		Model:  prompter.ModelName(),
		Prompt: prompt,
	}, nil
}
