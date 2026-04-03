package webhook

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

// InteractionHandler processes a Discord interaction and returns a response.
type InteractionHandler interface {
	HandleInteraction(i *discordgo.Interaction) *discordgo.InteractionResponse
}

// Handler serves Discord interaction webhooks with Ed25519 signature verification.
type Handler struct {
	bot       InteractionHandler
	publicKey ed25519.PublicKey
}

// NewHandler creates a Handler that verifies Discord signatures using publicKeyHex
// (the hex-encoded Ed25519 public key from your Discord application settings).
func NewHandler(bot InteractionHandler, publicKeyHex string) (*Handler, error) {
	key, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %w", err)
	}
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: got %d, want %d", len(key), ed25519.PublicKeySize)
	}
	return &Handler{bot: bot, publicKey: ed25519.PublicKey(key)}, nil
}

// ServeHTTP handles incoming Discord interaction POST requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	sig := r.Header.Get("X-Signature-Ed25519")
	ts := r.Header.Get("X-Signature-Timestamp")
	if !h.verify(sig, ts, body) {
		http.Error(w, "Invalid request signature", http.StatusUnauthorized)
		return
	}

	var interaction discordgo.Interaction
	if err := json.Unmarshal(body, &interaction); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if interaction.Type == discordgo.InteractionPing {
		h.writeJSON(w, &discordgo.InteractionResponse{Type: discordgo.InteractionResponsePong})
		return
	}

	resp := h.bot.HandleInteraction(&interaction)
	if resp == nil {
		http.Error(w, "Unknown interaction", http.StatusBadRequest)
		return
	}
	h.writeJSON(w, resp)
}

func (h *Handler) verify(signatureHex, timestamp string, body []byte) bool {
	sig, err := hex.DecodeString(signatureHex)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	msg := make([]byte, 0, len(timestamp)+len(body))
	msg = append(msg, []byte(timestamp)...)
	msg = append(msg, body...)
	return ed25519.Verify(h.publicKey, msg, sig)
}

func (h *Handler) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("webhook: encoding response: %v", err)
	}
}
