package youtubetranscript

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type options struct {
	lang               []string
	preserveFormatting bool
}

func WithLang(lang string) Option {
	return func(o *options) {
		o.lang = append(o.lang, lang)
	}
}

func WithPreserveFormatting(preserveFormatting bool) Option {
	return func(o *options) {
		o.preserveFormatting = preserveFormatting
	}
}

type Option func(*options)

type ReturnTranscript struct {
	Text     string
	Start    float64
	Duration float64
}

func GetTranscript(ctx context.Context, videoID string, opts ...Option) (string, error) {
	var yt youtubeTranscript
	for _, opt := range opts {
		opt(&yt.options)
	}
	return yt.GetTranscript(ctx, videoID)
}

type youtubeTranscript struct {
	options options
}

func (yt *youtubeTranscript) GetTranscript(ctx context.Context, videoID string) (string, error) {
	transcripts, err := yt.listTranscripts(videoID)

	if err != nil {
		return "", fmt.Errorf("failed to list transcripts: %w", err)
	}

	transcript, err := transcripts.FindTranscript(yt.options.lang)

	if err != nil {
		return "", fmt.Errorf("failed to find transcripts for language %s: %w", yt.options.lang, err)
	}

	captions, err := transcript.Fetch(yt.options.preserveFormatting)

	if err != nil {
		return "", fmt.Errorf("failed to fetch transcript: %w", err)
	}

	var cps []string

	for _, caption := range captions {
		cps = append(cps, caption.Text)

	}

	return strings.Join(cps, " "), nil
}

func (yt *youtubeTranscript) listTranscripts(videoID string) (*TranscriptList, error) {
	tlf := TranscriptListFetcher{
		httpClient: http.DefaultClient,
	}

	transcriptList, err := tlf.Fetch(videoID)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch transcript list: %w", err)
	}
	return transcriptList, nil
}
