# gemini-transcribe

A fast CLI tool to transcribe audio and video files using Google's Gemini API.

## Features

- 🎤 Transcribe audio files (mp3, wav, ogg, flac, m4a, aac)
- 🎬 Transcribe video files (mp4, webm, mov, avi, mkv) - requires ffmpeg
- 🔌 Custom API endpoint support (for proxies)
- 📝 JSON output option
- ⚡ Uses inline base64 encoding (no file upload API needed)

## Installation

### From source (requires Go 1.21+)

```bash
git clone https://github.com/mukhtharcm/gemini-transcribe
cd gemini-transcribe
go build -o gemini-transcribe .
sudo mv gemini-transcribe /usr/local/bin/
```

### Or with go install

```bash
go install github.com/mukhtharcm/gemini-transcribe@latest
```

## Usage

```bash
# Basic usage
gemini-transcribe -i audio.mp3

# With API key flag
gemini-transcribe -i audio.ogg -k YOUR_API_KEY

# With custom base URL (for proxies)
gemini-transcribe -i voice.ogg -k YOUR_API_KEY -b https://your-proxy.workers.dev

# JSON output
gemini-transcribe -i audio.mp3 --json

# Verbose mode
gemini-transcribe -i audio.mp3 -v

# Custom model
gemini-transcribe -i audio.mp3 -m gemini-2.0-flash

# Custom prompt
gemini-transcribe -i audio.mp3 -p "Transcribe this audio in Spanish"
```

## Options

| Flag | Long | Description | Default |
|------|------|-------------|---------|
| `-i` | `--input` | Input audio/video file (required) | - |
| `-k` | `--key` | Gemini API key | env/config |
| `-m` | `--model` | Gemini model to use | `gemini-2.5-flash` |
| `-b` | `--base-url` | Custom API base URL | Google's API |
| `-p` | `--prompt` | Custom transcription prompt | Default prompt |
| `-v` | `--verbose` | Verbose output | `false` |
| | `--json` | Output as JSON | `false` |

## API Key Configuration

The API key is resolved in this order:

1. `-k` / `--key` flag
2. `GEMINI_API_KEY` environment variable
3. `~/.config/gemini/api_key` file

### Setup config file

```bash
mkdir -p ~/.config/gemini
echo "YOUR_API_KEY" > ~/.config/gemini/api_key
chmod 600 ~/.config/gemini/api_key
```

## Supported Formats

### Audio
- MP3 (`.mp3`)
- WAV (`.wav`)
- OGG (`.ogg`)
- FLAC (`.flac`)
- M4A (`.m4a`)
- AAC (`.aac`)

### Video (requires ffmpeg)
- MP4 (`.mp4`)
- WebM (`.webm`)
- MOV (`.mov`)
- AVI (`.avi`)
- MKV (`.mkv`)

Video files are automatically converted to audio using ffmpeg before transcription.

## Using with a Proxy

If you need to use a proxy (e.g., Cloudflare Worker), use the `-b` flag:

```bash
gemini-transcribe -i audio.ogg -b https://gemini-proxy.example.workers.dev
```

The proxy should forward requests to `https://generativelanguage.googleapis.com`.

## Integration with Clawdbot

Add to your `clawdbot.json`:

```json
{
  "audio": {
    "transcription": {
      "command": [
        "gemini-transcribe",
        "-k", "YOUR_API_KEY",
        "-b", "https://your-proxy.workers.dev",
        "-i", "{{MediaPath}}"
      ],
      "timeoutSeconds": 60
    }
  }
}
```

## License

MIT
