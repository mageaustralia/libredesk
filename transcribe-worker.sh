#!/bin/bash
# Transcription worker - watches /home/ubuntu/libredesk/transcribe-queue/ for jobs
# Each job is a file named: {conversation_uuid}_{media_uuid}_{filename}

QUEUE_DIR="/home/ubuntu/libredesk/transcribe-queue"
MODEL="/home/ubuntu/whisper-models/ggml-base.en.bin"
UPLOAD_DIR="/home/ubuntu/libredesk/uploads"

mkdir -p "$QUEUE_DIR"

echo "Transcription worker started, watching $QUEUE_DIR"

inotifywait -m -e create --format '%f' "$QUEUE_DIR" 2>/dev/null | while read jobfile; do
    sleep 0.5  # Let file finish writing
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
    
    # Convert to WAV
    TMPWAV=$(mktemp /tmp/whisper-XXXXXX.wav)
    ffmpeg -i "$INPUT" -ar 16000 -ac 1 -c:a pcm_s16le "$TMPWAV" -y -loglevel error 2>&1
    
    if [ ! -s "$TMPWAV" ]; then
        echo "Error: ffmpeg conversion failed for $FILENAME"
        rm -f "$TMPWAV" "$JOBPATH"
        continue
    fi
    
    # Transcribe
    TRANSCRIPT=$(whisper -m "$MODEL" -f "$TMPWAV" --no-timestamps -t 2 -p 1 2>/dev/null | sed '/^$/d' | tr -s ' ')
    rm -f "$TMPWAV"
    
    if [ -z "$TRANSCRIPT" ] || [ "$TRANSCRIPT" = "[BLANK_AUDIO]" ]; then
        echo "No transcript content for $FILENAME"
        rm -f "$JOBPATH"
        continue
    fi
    
    echo "Transcript: $TRANSCRIPT"
    
    # Insert as activity message via psql
    ESCAPED=$(echo "$TRANSCRIPT" | sed "s/'/''/g")
    CONTENT="<p><strong>Voicemail Transcript</strong></p><p>$ESCAPED</p>"
    
    docker exec libredesk_db psql -U libredesk -d libredesk -c "
        INSERT INTO conversation_messages (uuid, type, status, conversation_id, content, text_content, content_type, private, sender_id, sender_type, created_at, updated_at)
        SELECT gen_random_uuid(), 'outgoing', 'sent', c.id, '$CONTENT', '$CONTENT', 'html', true, 1, 'agent', NOW(), NOW()
        FROM conversations c WHERE c.uuid = '$CONV_UUID';
    " 2>&1
    
    echo "Transcript inserted for conversation $CONV_UUID"
    rm -f "$JOBPATH"
done
