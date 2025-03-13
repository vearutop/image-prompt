// Package main provides an app to prompt for a remote image from command line.
//
// # Installation
//
// Get Ollama from https://ollama.com/.
// Then install llava with
//
//	ollama run llava:7b
//
// Then you can run llava for remote images.
//
//	go install github.com/vearutop/image-prompt@latest
//	image-prompt https://vearutop.p1cs.art/thumb/1200w/3bu7hfd29vjyc.jpg
//
// "A vintage Chevrolet Nomad station wagon, painted in a striking turquoise hue with silver
// trim and a white roof, is on display at an outdoor car show. The vehicle, featuring the
// iconic "Chevy bowtie" emblem on its front grille, is parked in front of a venue that appears
// to be a commercial building with a sports bar theme, identifiable by signage and decor.
// Spectators are visible around the car, admiring its classic design and color scheme."
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/vearutop/image-prompt/cloudflare"
	"github.com/vearutop/image-prompt/gemini"
	"github.com/vearutop/image-prompt/ollama"
	"github.com/vearutop/image-prompt/openai"
)

func main() { //nolint:cyclop,funlen
	var (
		prompt    string
		model     string
		cfWorker  string
		openaiKey string
		geminiKey string
	)

	flag.StringVar(&prompt, "prompt", "Generate a detailed caption for this image, don't name the places or items unless you're sure.", "prompt")
	flag.StringVar(&model, "model", "", "model name")
	flag.StringVar(&cfWorker, "cf", "", "CloudFlare worker URL (example https://MY_AUTH_KEY@llava.xxxxxx.workers.dev/)")
	flag.StringVar(&openaiKey, "openai", "", "OpenAI API KEY")
	flag.StringVar(&geminiKey, "gemini", "", "Gemini API KEY")
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()

		return
	}

	img := flag.Arg(0)

	var (
		image io.ReadCloser
		err   error
		p     interface {
			PromptImage(ctx context.Context, prompt string, image io.ReadCloser) (string, error)
		}
	)

	if strings.HasPrefix(img, "http://") || strings.HasPrefix(img, "https://") {
		resp, err := http.Get(img)
		if err != nil {
			log.Fatal(err)
		}

		image = resp.Body
		defer image.Close()
	} else {
		image, err = os.Open(img)
		if err != nil {
			log.Fatal(err)
		}
	}

	switch {
	case cfWorker != "":
		p, err = cloudflare.NewImagePrompter(cfWorker)
		if err != nil {
			log.Fatal(err)
		}
	case openaiKey != "":
		p = &openai.ImagePrompter{AuthKey: openaiKey}
	case geminiKey != "":
		p = &gemini.ImagePrompter{AuthKey: geminiKey}
	default:
		p = &ollama.ImagePrompter{
			Model: model,
		}
	}

	result, err := p.PromptImage(context.Background(), prompt, image)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
}
