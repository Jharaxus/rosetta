package tts

import (
	"context"
	"fmt"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
)

// GoogleSynthesizer wraps the GCP TTS client and implements SpeechSynthesizer.
// Close must be called when the server shuts down.
type GoogleSynthesizer struct {
	client *texttospeech.Client
}

// NewGoogleSynthesizer initialises the GCP TTS client.
// Returns an error if GOOGLE_APPLICATION_CREDENTIALS is missing or invalid.
func NewGoogleSynthesizer(ctx context.Context) (*GoogleSynthesizer, error) {
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("tts: create GCP client: %w", err)
	}
	return &GoogleSynthesizer{client: client}, nil
}

// Close tears down the underlying gRPC connection.
func (g *GoogleSynthesizer) Close() {
	_ = g.client.Close()
}

// Synthesize calls the Google Cloud TTS API and returns OGG/Opus bytes.
// ctx is the request context; cancellation aborts the gRPC call.
func (g *GoogleSynthesizer) Synthesize(ctx context.Context, text, voice string) ([]byte, error) {
	resp, err := g.client.SynthesizeSpeech(ctx, &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: "de-DE",
			Name:         voice,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_OGG_OPUS,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("tts: SynthesizeSpeech: %w", err)
	}
	return resp.AudioContent, nil
}
