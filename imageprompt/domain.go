// Package imageprompt defines image prompt domain.
package imageprompt

import (
	"context"
	"io"
)

type sentinelError string

func (e sentinelError) Error() string { return string(e) }

// Sentinel errors that can be matched.
const (
	ErrResourceExhausted = sentinelError("resource exhausted")
	ErrEmptyConfig       = sentinelError("empty config")
)

// ErrUnexpectedResponse contains unexpected response body.
type ErrUnexpectedResponse struct {
	Message      string
	ResponseBody []byte
}

func (ErrUnexpectedResponse) Error() string {
	return "unexpected response"
}

// Prompter defines LLM driver.
type Prompter interface {
	PromptImage(ctx context.Context, prompt string, jpegImage io.Reader) (string, error)
	ModelName() string
}
