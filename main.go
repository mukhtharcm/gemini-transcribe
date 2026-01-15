package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func getAPIKey(flagKey string) string {
	// Priority: flag > env > config file
	if flagKey != "" {
		return flagKey
	}
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		return key
	}
	// Try config file
	home, _ := os.UserHomeDir()
	keyFile := filepath.Join(home, ".config", "gemini", "api_key")
	if data, err := os.ReadFile(keyFile); err == nil {
		return strings.TrimSpace(string(data))
	}
	return ""
}

func getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		".mp3":  "audio/mp3",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".flac": "audio/flac",
		".m4a":  "audio/m4a",
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mov":  "video/quicktime",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "audio/mp3"
}

func main() {
	input := flag.String("i", "", "Input audio/video file")
	apiKey := flag.String("k", "", "Gemini API key")
	model := flag.String("m", "gemini-2.0-flash", "Model to use")
	prompt := flag.String("p", "Transcribe this audio accurately. Output only the transcription.", "Custom prompt")
	jsonOut := flag.Bool("json", false, "JSON output")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	if *input == "" {
		fmt.Fprintln(os.Stderr, "Usage: gemini-transcribe -i <audio-file>")
		os.Exit(1)
	}

	key := getAPIKey(*apiKey)
	if key == "" {
		fmt.Fprintln(os.Stderr, "Error: No API key. Set GEMINI_API_KEY or use -k flag or store in ~/.config/gemini/api_key")
		os.Exit(1)
	}

	// Read file
	data, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "File: %s (%d bytes)\n", *input, len(data))
		fmt.Fprintf(os.Stderr, "Model: %s\n", *model)
	}

	// Upload file first
	uploadURL := fmt.Sprintf("https://generativelanguage.googleapis.com/upload/v1beta/files?key=%s", key)
	
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	
	part, _ := writer.CreateFormFile("file", filepath.Base(*input))
	part.Write(data)
	writer.Close()

	req, _ := http.NewRequest("POST", uploadURL, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Upload error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var uploadResp struct {
		File struct {
			URI   string `json:"uri"`
			State string `json:"state"`
		} `json:"file"`
	}
	json.NewDecoder(resp.Body).Decode(&uploadResp)

	if uploadResp.File.URI == "" {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Upload failed: %s\n", string(body))
		os.Exit(1)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Uploaded: %s\n", uploadResp.File.URI)
	}

	// Generate content
	genURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", *model, key)
	
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"file_data": map[string]string{
						"mime_type": getMimeType(*input),
						"file_uri":  uploadResp.File.URI,
					}},
					{"text": *prompt},
				},
			},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ = http.NewRequest("POST", genURL, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Generate error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var genResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&genResp)

	if genResp.Error.Message != "" {
		fmt.Fprintf(os.Stderr, "API Error: %s\n", genResp.Error.Message)
		os.Exit(1)
	}

	if len(genResp.Candidates) > 0 && len(genResp.Candidates[0].Content.Parts) > 0 {
		text := genResp.Candidates[0].Content.Parts[0].Text
		if *jsonOut {
			out, _ := json.Marshal(map[string]string{"transcript": text})
			fmt.Println(string(out))
		} else {
			fmt.Println(text)
		}
	} else {
		fmt.Fprintln(os.Stderr, "No transcription returned")
		os.Exit(1)
	}
}
