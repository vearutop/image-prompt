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
	prompterExhaustedUntil smap[Provider, time.Time]
	prompterSemaphore      smap[Provider, chan struct{}]

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

type prompter struct {
	prompt string
	p      Provider
	sem    chan struct{}
}

func (ip *ImagePrompter) pp(cfg Config) (prompter, error) {
	if len(cfg.Providers) == 0 || len(cfg.Prompts) == 0 {
		return prompter{}, imageprompt.ErrEmptyConfig
	}

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
		exhausted, _ := ip.prompterExhaustedUntil.Load(pr.Provider)
		if !exhausted.IsZero() {
			if exhausted.Before(time.Now()) {
				ip.prompterExhaustedUntil.Store(pr.Provider, time.Time{})
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
		return prompter{
			p: provider,
		}, imageprompt.ErrResourceExhausted
	}

	if provider.Concurrency == 0 {
		provider.Concurrency = 1
	}

	sem, _ := ip.prompterSemaphore.Load(provider)
	if sem == nil {
		sem = make(chan struct{}, provider.Concurrency)
		ip.prompterSemaphore.Store(provider, sem)
	} else if cap(sem) != provider.Concurrency {
		// Wait for all in progress requests to finish.
		for i := 0; i < cap(sem); i++ {
			sem <- struct{}{}
		}

		// Replace semaphore.
		sem = make(chan struct{}, provider.Concurrency)
		ip.prompterSemaphore.Store(provider, sem)
	}

	return prompter{prompt: prompt, p: provider, sem: sem}, nil
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
	p, err := ip.pp(ip.cfgAccessor())
	if err != nil {
		return Result{}, err
	}

	p.sem <- struct{}{}
	defer func() { <-p.sem }()

	pr, err := p.p.prompter()
	if err != nil {
		return Result{}, err
	}

	res, err := pr.PromptImage(ctx, p.prompt, jpegImage)
	if err != nil {
		return Result{}, err
	}

	if errors.Is(err, imageprompt.ErrResourceExhausted) {
		ip.prompterExhaustedUntil.Store(p.p, time.Now().Add(time.Minute))

		return ip.PromptImage(ctx, jpegImage)
	}

	return Result{
		Text:   res,
		Model:  pr.ModelName(),
		Prompt: p.prompt,
	}, nil
}

type smap[K comparable, V any] struct {
	m sync.Map
}

func (m *smap[K, V]) Delete(key K) { m.m.Delete(key) }
func (m *smap[K, V]) Load(key K) (value V, ok bool) {
	v, ok := m.m.Load(key)
	if !ok {
		return value, ok
	}

	return v.(V), ok //nolint:errcheck
}

func (m *smap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, loaded := m.m.LoadAndDelete(key)
	if !loaded {
		return value, loaded
	}

	return v.(V), loaded //nolint:errcheck
}

func (m *smap[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	a, loaded := m.m.LoadOrStore(key, value)

	return a.(V), loaded //nolint:errcheck
}

func (m *smap[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(key, value any) bool {
		return f(key.(K), value.(V)) //nolint:errcheck
	})
}

func (m *smap[K, V]) Store(key K, value V) {
	m.m.Store(key, value)
}
