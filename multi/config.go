package multi

// WeightedPrompt is a prompt with usage probability.
type WeightedPrompt struct {
	Prompt string `json:"prompt" default:"Generate a detailed caption for this image, don't name the places, items or people unless you're sure." title:"Prompt text"`
	Weight int    `json:"weight" default:"1" title:"Prompt weight, prompts with higher weight are picked more often"`
}

// WeightedProvider is a Provider with usage probability.
type WeightedProvider struct {
	Provider Provider `json:"provider" title:"LLM Provider"`
	Weight   int      `json:"weight" default:"1" title:"Provider weight, providers with higher weight are picked more often"`
}

// Exhaustive list of supported providers.
const (
	Gemini     = ProviderType("gemini")
	CloudFlare = ProviderType("cloudflare")
	Ollama     = ProviderType("ollama")
	OpenAI     = ProviderType("openai")
)

// ProviderType enumerates supported types.
type ProviderType string

// Enum is a JSON schema helper.
func (p ProviderType) Enum() []any {
	return []any{
		Gemini,
		CloudFlare,
		Ollama,
		OpenAI,
	}
}

// Provider describes LLM service.
type Provider struct {
	Type        ProviderType `json:"type" title:"Type of provider"`
	AuthKey     string       `json:"auth_key,omitempty" title:"Auth/API key when applicable"`
	BaseURL     string       `json:"base_url,omitempty" title:"Base URL (for cloudflare, ollama)"`
	Model       string       `json:"model,omitempty" title:"Model"`
	Concurrency int          `json:"concurrency,omitempty" title:"Max request concurrency" default:"1"`
}

// Config defines prompts and services.
type Config struct {
	Prompts   []WeightedPrompt   `json:"prompts" minLength:"1" title:"Prompts"`
	Providers []WeightedProvider `json:"providers" minLength:"1" title:"LLM Providers"`
}
