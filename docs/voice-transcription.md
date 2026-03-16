# Voice Transcription Setup (Local whisper.cpp)

LibreDesk can automatically transcribe audio attachments (voicemails, voice notes) into text using a local [whisper.cpp](https://github.com/ggerganov/whisper.cpp) installation. Transcripts appear as private notes on the conversation.

## Requirements

- Linux host (tested on Ubuntu 22.04+ / ARM64 and x86_64)
- ~500MB disk space (whisper.cpp build + model)
- ~250MB RAM during transcription
- `ffmpeg` installed
- `inotifywait` (from `inotify-tools`)
- PostgreSQL client (`psql`) accessible via `docker exec`

## 1. Install Dependencies

```bash
sudo apt update
sudo apt install -y ffmpeg inotify-tools build-essential
```

## 2. Build whisper.cpp

```bash
cd /home/ubuntu
git clone https://github.com/ggerganov/whisper.cpp.git
cd whisper.cpp
cmake -B build
cmake --build build -j$(nproc)
sudo cp build/bin/whisper-cli /usr/local/bin/whisper
```

## 3. Download a Model

The `base.en` model is recommended for voicemail transcription — it's fast and accurate for English:

```bash
mkdir -p /home/ubuntu/whisper-models
cd /home/ubuntu/whisper.cpp
bash models/download-ggml-model.sh base.en
cp models/ggml-base.en.bin /home/ubuntu/whisper-models/
```

Other model options:
| Model | Size | RAM | Speed | Notes |
|-------|------|-----|-------|-------|
| `tiny.en` | 75MB | ~125MB | Fastest | Lower accuracy |
| `base.en` | 142MB | ~210MB | Fast | Good balance (recommended) |
| `small.en` | 466MB | ~600MB | Moderate | Higher accuracy |

## 4. Create the Worker Script

Create `/home/ubuntu/libredesk/transcribe-worker.sh`:

```bash
#!/bin/bash
# LibreDesk Voice Transcription Worker
# Watches for transcription jobs and processes them with whisper.cpp

QUEUE_DIR="/home/ubuntu/libredesk/transcribe-queue"
MODEL="/home/ubuntu/whisper-models/ggml-base.en.bin"
UPLOAD_DIR="/home/ubuntu/libredesk/uploads"
DB_CONTAINER="libredesk_db"
DB_USER="libredesk"
DB_NAME="libredesk"

mkdir -p "$QUEUE_DIR"

echo "Transcription worker started, watching $QUEUE_DIR"

inotifywait -m -e create --format '%f' "$QUEUE_DIR" 2>/dev/null | while read jobfile; do
    sleep 0.5
    JOBPATH="$QUEUE_DIR/$jobfile"
    [ ! -f "$JOBPATH" ] && continue

    # Parse job: conversation_uuid|media_uuid|filename
    IFS='|' read -r CONV_UUID MEDIA_UUID FILENAME < "$JOBPATH"

    echo "Transcribing: $FILENAME (media: $MEDIA_UUID, conv: $CONV_UUID)"

    INPUT="$UPLOAD_DIR/$MEDIA_UUID"
    if [ ! -f "$INPUT" ]; then
        echo "Error: file not found $INPUT"
        rm -f "$JOBPATH"
        continue
    fi

    # Convert to 16kHz mono WAV (required by whisper)
    TMPWAV=$(mktemp /tmp/whisper-XXXXXX.wav)
    ffmpeg -i "$INPUT" -ar 16000 -ac 1 -c:a pcm_s16le "$TMPWAV" -y -loglevel error 2>&1

    if [ ! -s "$TMPWAV" ]; then
        echo "Error: ffmpeg conversion failed for $FILENAME"
        rm -f "$TMPWAV" "$JOBPATH"
        continue
    fi

    # Transcribe with whisper
    TRANSCRIPT=$(whisper -m "$MODEL" -f "$TMPWAV" --no-timestamps -t 2 -p 1 2>/dev/null | sed '/^$/d' | tr -s ' ')
    rm -f "$TMPWAV"

    if [ -z "$TRANSCRIPT" ] || [ "$TRANSCRIPT" = "[BLANK_AUDIO]" ]; then
        echo "No transcript content for $FILENAME"
        rm -f "$JOBPATH"
        continue
    fi

    echo "Transcript: $TRANSCRIPT"

    # Insert as private note via psql
    ESCAPED=$(echo "$TRANSCRIPT" | sed "s/'/''/g")
    CONTENT="<p><strong>Voicemail Transcript</strong></p><p>$ESCAPED</p>"

    docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "
        INSERT INTO conversation_messages (uuid, type, status, conversation_id, content, text_content, content_type, private, sender_id, sender_type, created_at, updated_at)
        SELECT gen_random_uuid(), 'outgoing', 'sent', c.id, '$CONTENT', '$CONTENT', 'html', true, 1, 'agent', NOW(), NOW()
        FROM conversations c WHERE c.uuid = '$CONV_UUID';
    " 2>&1

    echo "Transcript inserted for conversation $CONV_UUID"
    rm -f "$JOBPATH"
done
```

Make it executable:

```bash
chmod +x /home/ubuntu/libredesk/transcribe-worker.sh
```

## 5. Create the Systemd Service

Create `/etc/systemd/system/libredesk-transcribe.service`:

```ini
[Unit]
Description=LibreDesk Voice Transcription Worker
After=docker.service
Requires=docker.service

[Service]
Type=simple
User=ubuntu
ExecStart=/home/ubuntu/libredesk/transcribe-worker.sh
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable libredesk-transcribe
sudo systemctl start libredesk-transcribe
```

## 6. Enable in LibreDesk

1. Go to **Admin > AI Settings**
2. Scroll to **Voice Transcription**
3. Toggle **Enable Transcription** on
4. Select **Local whisper.cpp (self-hosted)** as the provider
5. Click **Save**

## Verification

Check that the worker is running:

```bash
sudo systemctl status libredesk-transcribe
```

View worker logs:

```bash
sudo journalctl -u libredesk-transcribe -f
```

Send a test voicemail or audio file to your support inbox. The transcription should appear as a private note within a few seconds.

## Troubleshooting

**Worker not starting**: Check that `inotifywait` is installed (`apt install inotify-tools`).

**Transcription fails**: Ensure `ffmpeg` and `whisper` are in the PATH. Test manually:
```bash
whisper -m /home/ubuntu/whisper-models/ggml-base.en.bin -f /path/to/test.wav --no-timestamps
```

**Empty transcripts**: The audio may be too short or silent. Check the original file.

**DB insert fails**: Verify the Docker container name matches (`libredesk_db`) and the system user ID is `1`:
```bash
docker exec libredesk_db psql -U libredesk -d libredesk -c "SELECT id, first_name FROM users WHERE id = 1;"
```
