package bot

import (
	"context"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/subhanjanOps/torn-advisor/config"
	"github.com/subhanjanOps/torn-advisor/domain"
	"github.com/subhanjanOps/torn-advisor/store"
)

// --- Mocks ---

type mockProvider struct {
	state domain.PlayerState
	err   error
}

func (m *mockProvider) FetchPlayerState(_ context.Context) (domain.PlayerState, error) {
	return m.state, m.err
}

type mockSession struct {
	handlers       []interface{}
	commands       []*discordgo.ApplicationCommand
	interactions   []*interactionCall
	embeds         []*embedCall
	openCalled     bool
	closeCalled    bool
	openErr        error
	closeErr       error
	cmdCreateErr   error
	interactionErr error
}

type interactionCall struct {
	interaction *discordgo.Interaction
	resp        *discordgo.InteractionResponse
}

type embedCall struct {
	channelID string
	embed     *discordgo.MessageEmbed
}

func (m *mockSession) AddHandler(handler interface{}) func() {
	m.handlers = append(m.handlers, handler)
	return func() {}
}

func (m *mockSession) ApplicationCommandCreate(_ string, _ string, cmd *discordgo.ApplicationCommand, _ ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
	if m.cmdCreateErr != nil {
		return nil, m.cmdCreateErr
	}
	m.commands = append(m.commands, cmd)
	return cmd, nil
}

func (m *mockSession) InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
	m.interactions = append(m.interactions, &interactionCall{interaction: interaction, resp: resp})
	return m.interactionErr
}

func (m *mockSession) ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.embeds = append(m.embeds, &embedCall{channelID: channelID, embed: embed})
	return &discordgo.Message{}, nil
}

func (m *mockSession) Open() error {
	m.openCalled = true
	return m.openErr
}

func (m *mockSession) Close() error {
	m.closeCalled = true
	return m.closeErr
}

// --- Helpers ---

func testEncKey() string {
	return hex.EncodeToString([]byte("01234567890123456789012345678901"))
}

func newTestBot(t *testing.T, s *mockSession, mp *mockProvider) *Bot {
	t.Helper()
	dir := t.TempDir()
	ks, err := store.NewKeyStore(filepath.Join(dir, "keys.json"), testEncKey())
	if err != nil {
		t.Fatalf("creating keystore: %v", err)
	}
	factory := func(_ string) domain.StateProvider { return mp }
	return New(s, ks, factory, config.DefaultPriorities(), 30*time.Second)
}

func registerUser(t *testing.T, b *Bot, userID, apiKey string) {
	t.Helper()
	opts := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: "api_key", Type: discordgo.ApplicationCommandOptionString, Value: apiKey},
	}
	resp := b.BuildRegisterResponse(userID, opts)
	if resp.Data.Flags != discordgo.MessageFlagsEphemeral || !strings.Contains(resp.Data.Content, "registered") {
		t.Fatalf("register failed: %s", resp.Data.Content)
	}
}

// --- Tests ---

func TestNew(t *testing.T) {
	s := &mockSession{}
	b := newTestBot(t, s, &mockProvider{})

	if b.session != s {
		t.Error("session not set")
	}
}

func TestRegisterAndStart(t *testing.T) {
	s := &mockSession{}
	b := newTestBot(t, s, &mockProvider{})

	if err := b.RegisterAndStart("test-app-id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s.openCalled {
		t.Error("session.Open was not called")
	}
	if len(s.handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(s.handlers))
	}
	if len(s.commands) != 7 {
		t.Errorf("expected 7 commands registered, got %d", len(s.commands))
	}

	names := make(map[string]bool)
	for _, cmd := range s.commands {
		names[cmd.Name] = true
	}
	for _, expected := range []string{"advise", "status", "config", "register", "unregister", "schedule", "unschedule"} {
		if !names[expected] {
			t.Errorf("command %q not registered", expected)
		}
	}
}

func TestRegisterAndStart_OpenError(t *testing.T) {
	s := &mockSession{openErr: fmt.Errorf("connection failed")}
	b := newTestBot(t, s, &mockProvider{})

	err := b.RegisterAndStart("app-id")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "opening discord session") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRegisterAndStart_CommandCreateError(t *testing.T) {
	s := &mockSession{cmdCreateErr: fmt.Errorf("forbidden")}
	b := newTestBot(t, s, &mockProvider{})

	err := b.RegisterAndStart("app-id")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "registering command") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStop(t *testing.T) {
	s := &mockSession{}
	b := newTestBot(t, s, &mockProvider{})

	if err := b.Stop(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.closeCalled {
		t.Error("session.Close was not called")
	}
}

func TestStop_Error(t *testing.T) {
	s := &mockSession{closeErr: fmt.Errorf("close failed")}
	b := newTestBot(t, s, &mockProvider{})

	if err := b.Stop(); err == nil {
		t.Error("expected error")
	}
}

func TestRegisterAndAdvise(t *testing.T) {
	mp := &mockProvider{
		state: domain.PlayerState{
			Energy: 100, EnergyMax: 150,
			Nerve: 25, NerveMax: 60,
			Happy: 5000, Life: 100, LifeMax: 100,
		},
	}
	b := newTestBot(t, &mockSession{}, mp)

	// Before register: advise should fail.
	resp := b.BuildAdviseResponse("user1")
	if !strings.Contains(resp.Data.Content, "register") {
		t.Errorf("expected register prompt, got: %s", resp.Data.Content)
	}

	// Register and advise.
	registerUser(t, b, "user1", "test-key")
	resp = b.BuildAdviseResponse("user1")

	if len(resp.Data.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(resp.Data.Embeds))
	}
	if resp.Data.Embeds[0].Title != "Torn Advisor — Action Plan" {
		t.Errorf("unexpected title: %s", resp.Data.Embeds[0].Title)
	}
}

func TestBuildAdviseResponse_NoActions(t *testing.T) {
	mp := &mockProvider{
		state: domain.PlayerState{
			Energy: 0, EnergyMax: 150,
			Nerve: 0, NerveMax: 0,
			Happy: 0, Life: 100, LifeMax: 100,
			XanaxCooldown: 1, BoosterCooldown: 1, TravelCooldown: 1,
			Addiction: 0, ChainActive: false, WarActive: false,
		},
	}
	b := newTestBot(t, &mockSession{}, mp)
	registerUser(t, b, "user1", "key")

	resp := b.BuildAdviseResponse("user1")

	embed := resp.Data.Embeds[0]
	if !strings.Contains(embed.Description, "No actions recommended") {
		t.Errorf("expected 'no actions' message, got: %s", embed.Description)
	}
}

func TestBuildAdviseResponse_ProviderError(t *testing.T) {
	mp := &mockProvider{err: fmt.Errorf("api timeout")}
	b := newTestBot(t, &mockSession{}, mp)
	registerUser(t, b, "user1", "key")

	resp := b.BuildAdviseResponse("user1")

	if !strings.Contains(resp.Data.Content, "Failed to fetch player state") {
		t.Errorf("unexpected content: %s", resp.Data.Content)
	}
}

func TestBuildStatusResponse(t *testing.T) {
	mp := &mockProvider{
		state: domain.PlayerState{
			Energy: 80, EnergyMax: 150,
			Nerve: 25, NerveMax: 60,
			Happy: 5000, Life: 90, LifeMax: 100,
			ChainActive: true, WarActive: false,
		},
	}
	b := newTestBot(t, &mockSession{}, mp)
	registerUser(t, b, "user1", "key")

	resp := b.BuildStatusResponse("user1")

	embed := resp.Data.Embeds[0]
	if embed.Title != "Player Status" {
		t.Errorf("unexpected title: %s", embed.Title)
	}
	if len(embed.Fields) != 6 {
		t.Errorf("expected 6 fields, got %d", len(embed.Fields))
	}

	fieldMap := make(map[string]string)
	for _, f := range embed.Fields {
		fieldMap[f.Name] = f.Value
	}
	if fieldMap["Energy"] != "80 / 150" {
		t.Errorf("energy field: %s", fieldMap["Energy"])
	}
	if fieldMap["Chain Active"] != "true" {
		t.Errorf("chain field: %s", fieldMap["Chain Active"])
	}
}

func TestBuildStatusResponse_NotRegistered(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})

	resp := b.BuildStatusResponse("unknown-user")
	if !strings.Contains(resp.Data.Content, "register") {
		t.Error("expected register prompt")
	}
}

func TestBuildConfigResponse(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})

	resp := b.BuildConfigResponse()

	embed := resp.Data.Embeds[0]
	if embed.Title != "Rule Priorities" {
		t.Errorf("unexpected title: %s", embed.Title)
	}
	if !strings.Contains(embed.Description, "Hospital: **98**") {
		t.Errorf("missing Hospital: %s", embed.Description)
	}
}

func TestRegister_EmptyOpts(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})

	resp := b.BuildRegisterResponse("user1", nil)
	if !strings.Contains(resp.Data.Content, "required") {
		t.Errorf("expected error, got: %s", resp.Data.Content)
	}
}

func TestUnregister(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})
	registerUser(t, b, "user1", "key")

	resp := b.BuildUnregisterResponse("user1")
	if !strings.Contains(resp.Data.Content, "removed") {
		t.Errorf("unexpected: %s", resp.Data.Content)
	}

	// Should no longer be able to advise.
	resp = b.BuildAdviseResponse("user1")
	if !strings.Contains(resp.Data.Content, "register") {
		t.Error("expected register prompt after unregister")
	}
}

func TestScheduleAndUnschedule(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})

	// Schedule without registration should fail.
	resp := b.BuildScheduleResponse("user1", "ch1")
	if !strings.Contains(resp.Data.Content, "register") {
		t.Errorf("expected register prompt, got: %s", resp.Data.Content)
	}

	// Register then schedule.
	registerUser(t, b, "user1", "key")
	resp = b.BuildScheduleResponse("user1", "ch1")
	if !strings.Contains(resp.Data.Content, "enabled") {
		t.Errorf("expected enabled, got: %s", resp.Data.Content)
	}

	// Unschedule.
	resp = b.BuildUnscheduleResponse("user1")
	if !strings.Contains(resp.Data.Content, "disabled") {
		t.Errorf("expected disabled, got: %s", resp.Data.Content)
	}
}

func TestScheduledAdvice_SendsEmbed(t *testing.T) {
	mp := &mockProvider{
		state: domain.PlayerState{
			Life: 10, LifeMax: 100, // triggers hospital (priority 98)
			ChainActive:     true, // triggers chain (priority 97)
			XanaxCooldown:   1,
			BoosterCooldown: 1,
			TravelCooldown:  1,
		},
	}
	s := &mockSession{}
	b := newTestBot(t, s, mp)
	registerUser(t, b, "user1", "key")

	b.scheduleMu.Lock()
	b.scheduleChannels["user1"] = "channel-123"
	b.scheduleMu.Unlock()

	b.RunScheduledAdvice()

	if len(s.embeds) != 1 {
		t.Fatalf("expected 1 embed sent, got %d", len(s.embeds))
	}
	if s.embeds[0].channelID != "channel-123" {
		t.Errorf("wrong channel: %s", s.embeds[0].channelID)
	}
	if !strings.Contains(s.embeds[0].embed.Title, "Scheduled Advice") {
		t.Errorf("wrong title: %s", s.embeds[0].embed.Title)
	}
}

func TestScheduledAdvice_NoUrgent_NoMessage(t *testing.T) {
	mp := &mockProvider{
		state: domain.PlayerState{
			Energy: 0, Nerve: 0, NerveMax: 0, Happy: 0,
			Life: 100, LifeMax: 100,
			XanaxCooldown: 1, BoosterCooldown: 1, TravelCooldown: 1,
		},
	}
	s := &mockSession{}
	b := newTestBot(t, s, mp)
	registerUser(t, b, "user1", "key")

	b.scheduleMu.Lock()
	b.scheduleChannels["user1"] = "channel-123"
	b.scheduleMu.Unlock()

	b.RunScheduledAdvice()

	if len(s.embeds) != 0 {
		t.Errorf("expected no embeds (no urgent actions), got %d", len(s.embeds))
	}
}

func TestEmbedResponse(t *testing.T) {
	resp := embedResponse("Title", "Desc", 0xFF0000)
	if resp.Data.Embeds[0].Title != "Title" {
		t.Error("title mismatch")
	}
}

func TestErrorResponse(t *testing.T) {
	resp := errorResponse("something broke")
	if !strings.Contains(resp.Data.Content, "something broke") {
		t.Error("content mismatch")
	}
	if resp.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Error("expected ephemeral flag")
	}
}

func TestEphemeralResponse(t *testing.T) {
	resp := ephemeralResponse("done!")
	if resp.Data.Content != "done!" {
		t.Error("content mismatch")
	}
	if resp.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Error("expected ephemeral flag")
	}
}
