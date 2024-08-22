package youtubetranscript

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"aqwari.net/xml/xmltree"
)

var (
	ErrVideoUnavailable           = errors.New("video is unavailable")
	ErrTooManyRequests            = errors.New("too many requests")
	ErrYoutubeRequestFailed       = errors.New("youtube request failed")
	ErrNoTranscriptFound          = errors.New("no transcript found")
	ErrNotTranslatable            = errors.New("transcript is not translatable")
	ErrFailedToCreateConsetCookie = errors.New("failed to create consent cookie")
	ErrInvalidVideoID             = errors.New("invalid video ID")
	ErrTranscriptDisabled         = errors.New("transcript is disabled")
)

const WATCH_URL = "https://www.youtube.com/watch?v=%s"

type TranscriptListFetcher struct {
	httpClient *http.Client
}

func NewTranscriptListFetcher(httpClient *http.Client) *TranscriptListFetcher {
	return &TranscriptListFetcher{httpClient: httpClient}
}

func (t *TranscriptListFetcher) Fetch(videoID string) (*TranscriptList, error) {
	html, err := t.fetchVideoHTML(videoID)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch video HTML: %w", err)
	}

	captionsJSON, err := t.extractCaptionsJSON(html, videoID)

	if err != nil {
		return nil, fmt.Errorf("failed to extract captions JSON: %w", err)

	}

	return NewTranscriptList(t.httpClient, videoID,
		captionsJSON,
	)
}

func (t *TranscriptListFetcher) createConsetCookie(html string, videoID string) error {
	re := regexp.MustCompile(`name="v" value="(.*?)"`)
	match := re.FindStringSubmatch(html)
	if match == nil {
		return ErrFailedToCreateConsetCookie
	}
	u, _ := url.Parse(fmt.Sprintf(WATCH_URL, videoID))
	t.httpClient.Jar.SetCookies(u, []*http.Cookie{
		{
			Name:   "CONSENT",
			Value:  "YES+" + match[1],
			Domain: ".youtube.com",
		},
	})
	return nil
}

func (t *TranscriptListFetcher) fetchVideoHTML(videoID string) (string, error) {
	html, err := t.fetchHTML(videoID)

	if err != nil {
		return "", fmt.Errorf("failed to fetch video HTML: %w", err)
	}

	if strings.Contains(html, `action="https://consent.youtube.com/s"`) {
		t.createConsetCookie(html, videoID)
		html, err = t.fetchHTML(videoID)
		if err != nil {
			return "", fmt.Errorf("failed to fetch video HTML: %w", err)
		}
		if strings.Contains(html, `action="https://consent.youtube.com/s"`) {
			return "", ErrFailedToCreateConsetCookie
		}
	}

	return html, nil
}

func (t *TranscriptListFetcher) fetchHTML(videoID string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(WATCH_URL, videoID), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch video HTML: %w", err)
	}
	if err != nil {
		return "", fmt.Errorf("failed to fetch video HTML: %w", err)
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(b), nil
}

func (t *TranscriptListFetcher) extractCaptionsJSON(videoHTML string, videoID string) (map[string]any, error) {
	splittedHTML := strings.Split(videoHTML, `"captions":`)

	if len(splittedHTML) <= 1 {
		if strings.HasPrefix(videoID, "http://") || strings.HasPrefix(videoID, "https://") {
			return nil, ErrInvalidVideoID
		}
		if strings.Contains(videoHTML, `class="g-recaptcha`) {
			return nil, ErrTooManyRequests
		}
		if !strings.Contains(videoHTML, `"playabilityStatus":`) {
			return nil, ErrVideoUnavailable
		}

		return nil, ErrTranscriptDisabled
	}

	htmlChunk := strings.Split(splittedHTML[1], `,"videoDetails`)

	htmlChunk[0] = strings.Replace(htmlChunk[0], "\n", "", -1)

	var captions map[string]any

	err := json.Unmarshal([]byte(htmlChunk[0]), &captions)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal captions JSON: %w", err)
	}
	captions = captions["playerCaptionsTracklistRenderer"].(map[string]any)
	if len(captions) == 0 {
		return nil, ErrTranscriptDisabled
	}

	if _, ok := captions["captionTracks"]; !ok {
		return nil, ErrNoTranscriptFound

	}

	return captions, nil
}

type TranscriptList struct {
	VideoID                    string
	ManuallyCreatedTranscripts map[string]any
	GeneratedTranscripts       map[string]any
	Translationlanguages       []translationLanguage
}

func (tl *TranscriptList) FindTranscript(languageCodes []string) (Transcript, error) {

	transcriptDics := []map[string]any{tl.ManuallyCreatedTranscripts, tl.GeneratedTranscripts}
	for _, languageCode := range languageCodes {
		for _, transcriptDict := range transcriptDics {
			if transcript, ok := transcriptDict[languageCode]; ok {
				return transcript.(Transcript), nil
			}
		}
	}

	return Transcript{}, ErrNoTranscriptFound
}

type translationLanguage struct {
	Language     string
	LanguageCode string
}

func NewTranscriptList(httpClient *http.Client, videoID string, captionsJSON map[string]any) (*TranscriptList, error) {
	var translationLanguages []translationLanguage

	for _, caption := range captionsJSON["translationLanguages"].([]any) {
		caption := caption.(map[string]any)
		languageName := caption["languageName"].(map[string]any)

		translationLanguages = append(translationLanguages, translationLanguage{
			Language:     languageName["simpleText"].(string),
			LanguageCode: caption["languageCode"].(string),
		})
	}

	manuallyCreatedTranscripts := make(map[string]any)
	generatedTranscripts := make(map[string]any)

	for _, caption := range captionsJSON["captionTracks"].([]any) {
		caption := caption.(map[string]any)
		kind, _ := caption["kind"].(string)
		ts := Transcript{
			httpClient:          httpClient,
			videoID:             videoID,
			requestURL:          caption["baseUrl"].(string),
			language:            caption["name"].(map[string]any)["simpleText"].(string),
			languageCode:        caption["languageCode"].(string),
			isGenerated:         kind == "asr",
			transltionLanguages: caption["isTranslatable"],
		}
		key := caption["languageCode"].(string)
		if kind == "asr" {
			generatedTranscripts[key] = ts
		} else {
			manuallyCreatedTranscripts[key] = ts
		}
	}

	return &TranscriptList{
		VideoID:                    videoID,
		ManuallyCreatedTranscripts: manuallyCreatedTranscripts,
		GeneratedTranscripts:       generatedTranscripts,
		Translationlanguages:       translationLanguages,
	}, nil
}

type Transcript struct {
	httpClient          *http.Client
	videoID             string
	requestURL          string
	language            string
	languageCode        string
	isGenerated         bool
	transltionLanguages any
}

func (t *Transcript) Fetch(preserveFormatting bool) ([]ReturnTranscript, error) {
	req, err := http.NewRequest("GET", t.requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Accept-Language", "en-US")
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transcript: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrYoutubeRequestFailed
	}

	tp := TranscriptParser{
		preserveFormatting: preserveFormatting,
	}

	b, err := io.ReadAll(resp.Body)

	if err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	resp.Body.Close()

	return tp.Parse(b)
}

type TranscriptParser struct {
	preserveFormatting bool
}

func (t *TranscriptParser) Parse(b []byte) ([]ReturnTranscript, error) {

	htmlRegex := t.getHTMLRegex()

	xmlTree, err := xmltree.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML data: %w", err)
	}

	transcript := []ReturnTranscript{}
	for _, xmlElement := range xmlTree.Children {
		if xmlElement.Name.Local == "text" {
			content := html.UnescapeString(string(xmlElement.Content))
			text := htmlRegex.ReplaceAllString(content, "")
			var start, duration float64

			for _, attr := range xmlElement.StartElement.Attr {
				if attr.Name.Local == "start" {
					start, err = strconv.ParseFloat(attr.Value, 64)
					if err != nil {
						return nil, fmt.Errorf("failed to parse start: %w", err)
					}
				}

				if attr.Name.Local == "dur" {
					duration, err = strconv.ParseFloat(attr.Value, 64)
					if err != nil {
						return nil, fmt.Errorf("failed to parse duration: %w", err)
					}
				}
			}

			transcript = append(transcript, ReturnTranscript{
				Text:     text,
				Start:    start,
				Duration: duration,
			})
		}
	}

	return transcript, nil
}

func (t *TranscriptParser) getHTMLRegex() *regexp.Regexp {
	var formattingTags = []string{
		"strong",
		"em",
		"b",
		"i",
		"mark",
		"small",
		"del",
		"ins",
		"sub",
		"sup",
	}
	formatsRegex := strings.Join(formattingTags, "|")
	formatsRegex = fmt.Sprintf(`<\/?(!\/?(%s)\b).*?\b>`, formatsRegex)
	htmlRegex := regexp.MustCompile(formatsRegex)
	return htmlRegex
}
