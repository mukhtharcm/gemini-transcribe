# gemini-transcribe

A lightweight Go CLI for transcribing audio/video files using Google Gemini API.

## Features

- 🎤 Uses raw Gemini API with AI Studio API key
- 🎬 Automatic audio extraction from video via ffmpeg
- 📁 Supports audio: mp3, wav, ogg, flac, m4a, aac
- 🎥 Supports video: mp4, webm, mov, avi, mkv
- ⚡ Single binary, minimal dependencies

## Installation

### From source

```bash
git clone https://github.com/mukhtharcm/gemini-transcribe
cd gemini-transcribe
go build -o gemini-transcribe .
```

### Using go install

```bash
go install github.com/mukhtharcm/gemini-transcribe@latest
```

## Requirements

- **Gemini API key** from [AI Studio](https://aistudio.google.com/apikey)
- **ffmpeg** (optional) - for video files or audio format conversion

## Usage

```bash
# Set API key
export GEMINI_API_KEY="your-api-key"

# Transcribe audio file
gemini-transcribe -i audio.mp3

# Transcribe video (extracts audio automatically)
gemini-transcribe -i video.mp4

# Use specific model
gemini-transcribe -i audio.wav -m gemini-2.5-pro

# Custom transcription prompt
gemini-transcribe -i audio.mp3 -p "Transcribe with speaker labels"

# JSON output
gemini-transcribe -i audio.mp3 --json

# Verbose mode
gemini-transcribe -i audio.mp3 -v
```

## Options

| Flag | Description | Default |
|------|-------------|---------|
| `-i, --input` | Input audio/video file | (required) |
| `-k, --key` | Gemini API key | `$GEMINI_API_KEY` |
| `-m, --model` | Gemini model | `gemini-2.5-flash` |
| `-p, --prompt` | Custom prompt | Transcribe accurately |
| `--json` | Output as JSON | false |
| `-v, --verbose` | Verbose output | false |

## How it works

1. Reads audio file (or extracts audio from video via ffmpeg)
2. Converts to mp3 if needed (16kHz mono for optimal speech recognition)
3. Base64 encodes and sends to Gemini API inline
4. Returns transcription text

## Why Gemini?

Gemini models have excellent multilingual speech recognition and can handle:
- Multiple languages in one audio
- Accented speech
- Background noise
- Long-form audio

## License

MIT
