package conversation

import (
	"fmt"
	"os"
	"strings"

	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
)

const transcribeQueueDir = "/libredesk/transcribe-queue/"

// audioContentTypes lists MIME types that should be transcribed.
var audioContentTypes = map[string]bool{
	"audio/wav":    true,
	"audio/wave":   true,
	"audio/x-wav":  true,
	"audio/mpeg":   true,
	"audio/mp3":    true,
	"audio/ogg":    true,
	"audio/x-m4a":  true,
	"audio/mp4":    true,
	"audio/aac":    true,
	"audio/flac":   true,
	"audio/x-flac": true,
	"audio/webm":   true,
	"audio/amr":    true,
	"audio/3gpp":   true,
}

// transcribeAudioAttachments transcribes audio attachments using the configured provider.
func (m *Manager) transcribeAudioAttachments(conversationUUID string, media []mmodels.Media) {
	// Check if transcription is enabled.
	aiSettings, err := m.settingsStore.GetAISettings()
	if err != nil {
		m.lo.Error("error fetching AI settings for transcription", "error", err)
		return
	}

	if !aiSettings.TranscriptionEnabled {
		return
	}

	for _, med := range media {
		ct := strings.ToLower(med.ContentType)
		if !audioContentTypes[ct] {
			continue
		}

		m.lo.Info("transcribing audio", "uuid", med.UUID, "filename", med.Filename, "provider", aiSettings.TranscriptionProvider)

		switch aiSettings.TranscriptionProvider {
		case "openai":
			go m.transcribeViaAPI(conversationUUID, med)
		case "local":
			m.transcribeViaLocal(conversationUUID, med)
		default:
			m.lo.Warn("unknown transcription provider", "provider", aiSettings.TranscriptionProvider)
		}
	}
}

// transcribeViaAPI sends the audio to OpenAI's Whisper API.
func (m *Manager) transcribeViaAPI(conversationUUID string, med mmodels.Media) {
	// Read the audio file.
	audioData, err := m.mediaStore.GetBlob(med.UUID)
	if err != nil {
		m.lo.Error("error reading audio file for transcription", "error", err, "uuid", med.UUID)
		return
	}

	if m.TranscribeFunc == nil {
		m.lo.Error("transcription function not configured")
		return
	}

	transcript, err := m.TranscribeFunc(audioData, med.Filename)
	if err != nil {
		m.lo.Error("error transcribing audio via API", "error", err, "uuid", med.UUID)
		return
	}

	if transcript == "" {
		m.lo.Info("empty transcript from API", "uuid", med.UUID)
		return
	}

	m.insertTranscript(conversationUUID, med.Filename, transcript)
}

// transcribeViaLocal writes a job file for the local whisper worker.
func (m *Manager) transcribeViaLocal(conversationUUID string, med mmodels.Media) {
	os.MkdirAll(transcribeQueueDir, 0755)

	jobContent := fmt.Sprintf("%s|%s|%s", conversationUUID, med.UUID, med.Filename)
	jobPath := transcribeQueueDir + med.UUID + ".job"
	if err := os.WriteFile(jobPath, []byte(jobContent), 0644); err != nil {
		m.lo.Error("error writing transcription job", "error", err)
	}
}

// insertTranscript inserts a transcript as a private note on the conversation.
func (m *Manager) insertTranscript(conversationUUID, filename, transcript string) {
	content := fmt.Sprintf("<p><strong>Voicemail Transcript</strong></p><p>%s</p>", transcript)

	_, err := m.db.Exec(`
		INSERT INTO conversation_messages (uuid, type, status, conversation_id, content, text_content, content_type, private, sender_id, sender_type, created_at, updated_at)
		SELECT gen_random_uuid(), 'outgoing', 'sent', c.id, $1, $2, 'html', true, 1, 'agent', NOW(), NOW()
		FROM conversations c WHERE c.uuid = $3`,
		content, transcript, conversationUUID)

	if err != nil {
		m.lo.Error("error inserting transcript", "error", err, "conversation", conversationUUID)
		return
	}

	m.lo.Info("transcript inserted", "conversation", conversationUUID, "filename", filename)
}
