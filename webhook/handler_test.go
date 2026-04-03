package webhook

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bwmarrin/discordgo"
)

type mockBot struct {
	resp *discordgo.InteractionResponse
}

func (m *mockBot) HandleInteraction(_ *discordgo.Interaction) *discordgo.InteractionResponse {
	return m.resp
}

func signRequest(privKey ed25519.PrivateKey, timestamp string, body []byte) string {
	msg := make([]byte, 0, len(timestamp)+len(body))
	msg = append(msg, []byte(timestamp)...)
	msg = append(msg, body...)
	return hex.EncodeToString(ed25519.Sign(privKey, msg))
}

func newTestHandler(t *testing.T, bot *mockBot) (*Handler, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}
	h, err := NewHandler(bot, hex.EncodeToString(pub))
	if err != nil {
		t.Fatalf("creating handler: %v", err)
	}
	return h, priv
}

func TestNewHandler_InvalidHex(t *testing.T) {
	_, err := NewHandler(&mockBot{}, "not-hex")
	if err == nil {
		t.Fatal("expected error for invalid hex")
	}
}

func TestNewHandler_WrongKeyLength(t *testing.T) {
	_, err := NewHandler(&mockBot{}, "abcd")
	if err == nil {
		t.Fatal("expected error for wrong key length")
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler(t, &mockBot{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestInvalidSignature(t *testing.T) {
	h, _ := newTestHandler(t, &mockBot{})
	body := []byte(`{"type":1}`)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("X-Signature-Ed25519", hex.EncodeToString(make([]byte, 64)))
	req.Header.Set("X-Signature-Timestamp", "12345")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMissingSignature(t *testing.T) {
	h, _ := newTestHandler(t, &mockBot{})
	body := []byte(`{"type":1}`)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	// No signature headers.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestPing(t *testing.T) {
	h, priv := newTestHandler(t, &mockBot{})
	body := []byte(`{"type":1}`)
	ts := "1609459200"
	sig := signRequest(priv, ts, body)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("X-Signature-Ed25519", sig)
	req.Header.Set("X-Signature-Timestamp", ts)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
	}

	var resp discordgo.InteractionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Type != discordgo.InteractionResponsePong {
		t.Fatalf("got type %d, want %d (Pong)", resp.Type, discordgo.InteractionResponsePong)
	}
}

func TestApplicationCommand(t *testing.T) {
	expected := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "hello"},
	}
	h, priv := newTestHandler(t, &mockBot{resp: expected})

	body := []byte(`{"type":2,"data":{"id":"1","name":"advise"}}`)
	ts := "1609459200"
	sig := signRequest(priv, ts, body)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("X-Signature-Ed25519", sig)
	req.Header.Set("X-Signature-Timestamp", ts)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
	}

	var resp discordgo.InteractionResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Type != discordgo.InteractionResponseChannelMessageWithSource {
		t.Fatalf("got type %d, want %d", resp.Type, discordgo.InteractionResponseChannelMessageWithSource)
	}
}

func TestUnknownCommand(t *testing.T) {
	h, priv := newTestHandler(t, &mockBot{resp: nil})

	body := []byte(`{"type":2,"data":{"id":"1","name":"unknown"}}`)
	ts := "1609459200"
	sig := signRequest(priv, ts, body)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("X-Signature-Ed25519", sig)
	req.Header.Set("X-Signature-Timestamp", ts)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestBadJSON(t *testing.T) {
	h, priv := newTestHandler(t, &mockBot{})

	body := []byte(`{not-json}`)
	ts := "1609459200"
	sig := signRequest(priv, ts, body)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("X-Signature-Ed25519", sig)
	req.Header.Set("X-Signature-Timestamp", ts)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
