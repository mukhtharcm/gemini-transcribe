package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	defaultModel   = "gemini-2.5-flash"
	defaultBaseURL = "https://generativelanguage.googleapis.com"
	apiURLTemplate = "%s/v1beta/models/%s:generateContent?key=%s"
)

type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Text       string    `json:"text,omitempty"`
	InlineData *BlobData `json:"inline_data,omitempty"`
}

type BlobData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

func main() {
	var (
		inputFile  string
		apiKey     string
		model      string
		baseURL    string
		prompt     string
		outputJSON bool
		verbose    bool
	)

	flag.StringVar(&inputFile, "i", "", "Input audio/video file (required)")
	flag.StringVar(&inputFile, "input", "", "Input audio/video file (required)")
	flag.StringVar(&apiKey, "k", "", "Gemini API key (or set GEMINI_API_KEY)")
	flag.StringVar(&apiKey, "key", "", "Gemini API key (or set GEMINI_API_KEY)")
	flag.StringVar(&model, "m", defaultModel, "Gemini model to use")
	flag.StringVar(&model, "model", defaultModel, "Gemini model to use")
	flag.StringVar(&baseURL, "base-url", "", "Custom API base URL (or set GEMINI_BASE_URL)")
	flag.StringVar(&baseURL, "b", "", "Custom API base URL (or set GEMINI_BASE_URL)")
	flag.StringVar(&prompt, "p", "Transcribe this audio accurately. Output only the transcription, no extra commentary.", "Custom prompt")
	flag.StringVar(&prompt, "prompt", "Transcribe this audio accurately. Output only the transcription, no extra commentary.", "Custom prompt")
	flag.BoolVar(&outputJSON, "json", false, "Output as JSON")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "gemini-transcribe - Transcribe audio/video using Gemini API\n\n")
		fmt.Fprintf(os.Stderr, "Usage: gemini-transcribe -i <file> [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gemini-transcribe -i audio.mp3\n")
		fmt.Fprintf(os.Stderr, "  gemini-transcribe -i video.mp4 -m gemini-2.5-flash\n")
		fmt.Fprintf(os.Stderr, "  gemini-transcribe -i recording.wav --json\n")
		fmt.Fprintf(os.Stderr, "  gemini-transcribe -i audio.ogg -b https://gemini-proxy.example.workers.dev\n")
		fmt.Fprintf(os.Stderr, "\nSupported formats: mp3, wav, ogg, flac, m4a, mp4, webm, mov, avi, mkv\n")
	}

	flag.Parse()

	// Get API key
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		// Try config file
		if home, err := os.UserHomeDir(); err == nil {
			keyFile := filepath.Join(home, ".config", "gemini", "api_key")
			if data, err := os.ReadFile(keyFile); err == nil {
				apiKey = strings.TrimSpace(string(data))
			}
		}
	}
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: API key required. Use -k flag, set GEMINI_API_KEY, or store in ~/.config/gemini/api_key")
		os.Exit(1)
	}

	// Get base URL
	if baseURL == "" {
		baseURL = os.Getenv("GEMINI_BASE_URL")
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	// Remove trailing slash if present
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Validate input
	if inputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: Input file required. Use -i flag")
		flag.Usage()
		os.Exit(1)
	}

	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: File not found: %s\n", inputFile)
		os.Exit(1)
	}

	// Convert to audio if needed
	audioData, mimeType, err := prepareAudio(inputFile, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error preparing audio: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Audio size: %d bytes, MIME: %s\n", len(audioData), mimeType)
		fmt.Fprintf(os.Stderr, "Sending to Gemini (%s)...\n", model)
	}

	// Call Gemini API
	transcription, err := transcribe(apiKey, model, baseURL, audioData, mimeType, prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error transcribing: %v\n", err)
		os.Exit(1)
	}

	// Output
	if outputJSON {
		result := map[string]string{
			"transcription": transcription,
			"model":         model,
			"file":          inputFile,
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Println(transcription)
	}
}

func prepareAudio(inputFile string, verbose bool) ([]byte, string, error) {
	ext := strings.ToLower(filepath.Ext(inputFile))

	// Check if ffmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		// No ffmpeg, try to read file directly
		if verbose {
			fmt.Fprintln(os.Stderr, "ffmpeg not found, reading file directly...")
		}
		data, err := os.ReadFile(inputFile)
		if err != nil {
			return nil, "", err
		}
		mimeType := getMimeType(ext)
		return data, mimeType, nil
	}

	// Audio formats that Gemini accepts well
	audioExts := map[string]bool{
		".mp3": true, ".wav": true, ".ogg": true,
		".flac": true, ".m4a": true, ".aac": true,
	}

	// If already a good audio format and small enough, use directly
	if audioExts[ext] {
		info, err := os.Stat(inputFile)
		if err == nil && info.Size() < 20*1024*1024 { // Under 20MB
			data, err := os.ReadFile(inputFile)
			if err != nil {
				return nil, "", err
			}
			return data, getMimeType(ext), nil
		}
	}

	// Convert to mp3 using ffmpeg
	if verbose {
		fmt.Fprintln(os.Stderr, "Converting to mp3 with ffmpeg...")
	}

	tmpFile, err := os.CreateTemp("", "gemini-transcribe-*.mp3")
	if err != nil {
		return nil, "", err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// ffmpeg command: extract audio, convert to mp3, mono, 16kHz for speech
	cmd := exec.Command("ffmpeg",
		"-i", inputFile,
		"-vn",              // No video
		"-acodec", "libmp3lame",
		"-ar", "16000",     // 16kHz sample rate (good for speech)
		"-ac", "1",         // Mono
		"-b:a", "64k",      // 64kbps (sufficient for speech)
		"-y",               // Overwrite
		tmpPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, "", fmt.Errorf("ffmpeg failed: %v\n%s", err, stderr.String())
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, "", err
	}

	return data, "audio/mpeg", nil
}

func getMimeType(ext string) string {
	mimeTypes := map[string]string{
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".flac": "audio/flac",
		".m4a":  "audio/mp4",
		".aac":  "audio/aac",
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mov":  "video/quicktime",
		".avi":  "video/x-msvideo",
		".mkv":  "video/x-matroska",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func transcribe(apiKey, model, baseURL string, audioData []byte, mimeType, prompt string) (string, error) {
	// Build request with inline data (base64 encoded)
	req := GeminiRequest{
		Contents: []Content{
			{
				Parts: []Part{
					{
						InlineData: &BlobData{
							MimeType: mimeType,
							Data:     base64.StdEncoding.EncodeToString(audioData),
						},
					},
					{
						Text: prompt,
					},
				},
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf(apiURLTemplate, baseURL, model, apiKey)
	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v\nBody: %s", err, string(body))
	}

	if geminiResp.Error != nil {
		return "", fmt.Errorf("API error (%d): %s", geminiResp.Error.Code, geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no transcription in response")
	}

	return strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text), nil
}
