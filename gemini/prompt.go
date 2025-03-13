package gemini

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

// https://ai.google.dev/gemini-api/docs/vision?lang=rest&authuser=1

/*
curl "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=GEMINI_API_KEY" \
-H 'Content-Type: application/json' \
-X POST \
-d '{
  "contents": [{
    "parts":[{"text": "Explain how AI works"}]
    }]
   }'
*/

/*
IMG_PATH=/path/to/your/image1.jpeg

if [[ "$(base64 --version 2>&1)" = *"FreeBSD"* ]]; then
  B64FLAGS="--input"
else
  B64FLAGS="-w0"
fi

curl "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=$GOOGLE_API_KEY" \
    -H 'Content-Type: application/json' \
    -X POST \
    -d '{
      "contents": [{
        "parts":[
            {"text": "Caption this image."},
            {
              "inline_data": {
                "mime_type":"image/jpeg",
                "data": "'\$(base64 \$B64FLAGS \$IMG_PATH)'"
              }
            }
        ]
      }]
    }' 2> /dev/null
*/

type ImagePrompter struct {
	AuthKey   string
	Transport http.RoundTripper // default http.DefaultTransport.
}

type Response struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason     string `json:"finishReason"`
		CitationMetadata struct {
			CitationSources []struct {
				StartIndex int `json:"startIndex"`
				EndIndex   int `json:"endIndex"`
			} `json:"citationSources"`
		} `json:"citationMetadata"`
		AvgLogprobs float64 `json:"avgLogprobs"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
		PromptTokensDetails  []struct {
			Modality   string `json:"modality"`
			TokenCount int    `json:"tokenCount"`
		} `json:"promptTokensDetails"`
		CandidatesTokensDetails []struct {
			Modality   string `json:"modality"`
			TokenCount int    `json:"tokenCount"`
		} `json:"candidatesTokensDetails"`
	} `json:"usageMetadata"`
	ModelVersion string `json:"modelVersion"`
}

func (ip *ImagePrompter) PromptImage(ctx context.Context, prompt string, image io.ReadCloser) (string, error) {
	img, err := io.ReadAll(image)
	if err != nil {
		return "", err
	}

	if err := image.Close(); err != nil {
		return "", err
	}

	type InlineData struct {
		MimeType string `json:"mime_type"`
		Data     string `json:"data"`
	}

	type Part struct {
		Text       string      `json:"text,omitempty"`
		InlineData *InlineData `json:"inline_data,omitempty"`
	}

	type Content struct {
		Parts []Part `json:"parts"`
	}

	type Req struct {
		Contents []Content `json:"contents"`
	}

	req := Req{}
	req.Contents = []Content{
		{
			Parts: []Part{
				{Text: prompt},
				{InlineData: &InlineData{MimeType: "image/jpeg", Data: base64.StdEncoding.EncodeToString(img)}},
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	// println(string(body))

	r, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key="+ip.AuthKey,
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	r.Header.Set("Content-Type", "application/json")

	tr := ip.Transport
	if tr == nil {
		tr = http.DefaultTransport
	}

	resp, err := tr.RoundTrip(r)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	cont, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	re := Response{}

	if err := json.Unmarshal(cont, &re); err != nil {
		return "", err
	}

	if len(re.Candidates) == 0 {
		return "", errors.New("no candidates found")
	}

	c := re.Candidates[0]
	if len(c.Content.Parts) == 0 {
		return "", errors.New("no parts found")
	}

	return strings.Trim(c.Content.Parts[0].Text, "\" \t\n"), nil
}
