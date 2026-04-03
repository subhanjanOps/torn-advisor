package bot

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

func TestStopNilSession(t *testing.T) {
	dir := t.TempDir()
	ks, err := store.NewKeyStore(filepath.Join(dir, "keys.json"), testEncKey())
	if err != nil {
		t.Fatalf("creating keystore: %v", err)
	}
	factory := func(_ string) domain.StateProvider { return &mockProvider{} }
	b := New(nil, ks, factory, config.DefaultPriorities(), 30*time.Second)

	if err := b.Stop(); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestHandleInteraction_Advise(t *testing.T) {
	mp := &mockProvider{state: domain.PlayerState{
		Energy: 100, EnergyMax: 150, Happy: 5000,
		Life: 100, LifeMax: 100,
		XanaxCooldown: 1, BoosterCooldown: 1, TravelCooldown: 1,
	}}
	b := newTestBot(t, &mockSession{}, mp)
	registerUser(t, b, "u1", "key")

	interaction := &discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand,
		Member: &discordgo.Member{
			User: &discordgo.User{ID: "u1"},
		},
		Data: discordgo.ApplicationCommandInteractionData{
			Name: "advise",
		},
	}
	// Marshal/unmarshal to set rawData correctly
	resp := b.HandleInteraction(interaction)
	if resp == nil {
		t.Fatal("expected response")
	}
}

func TestHandleInteraction_Status(t *testing.T) {
	mp := &mockProvider{state: domain.PlayerState{
		Energy: 80, EnergyMax: 150, Life: 90, LifeMax: 100,
	}}
	b := newTestBot(t, &mockSession{}, mp)
	registerUser(t, b, "u1", "key")

	resp := b.HandleInteraction(&discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand,
		User: &discordgo.User{ID: "u1"},
		Data: discordgo.ApplicationCommandInteractionData{Name: "status"},
	})
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Data.Embeds[0].Title != "Player Status" {
		t.Errorf("unexpected title: %s", resp.Data.Embeds[0].Title)
	}
}

func TestHandleInteraction_Config(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})
	resp := b.HandleInteraction(&discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand,
		User: &discordgo.User{ID: "u1"},
		Data: discordgo.ApplicationCommandInteractionData{Name: "config"},
	})
	if resp == nil || resp.Data.Embeds[0].Title != "Rule Priorities" {
		t.Error("expected config response")
	}
}

func TestHandleInteraction_Register(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})
	resp := b.HandleInteraction(&discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand,
		User: &discordgo.User{ID: "u1"},
		Data: discordgo.ApplicationCommandInteractionData{
			Name: "register",
			Options: []*discordgo.ApplicationCommandInteractionDataOption{
				{Name: "api_key", Type: discordgo.ApplicationCommandOptionString, Value: "mykey"},
			},
		},
	})
	if resp == nil || !strings.Contains(resp.Data.Content, "registered") {
		t.Error("expected register success")
	}
}

func TestHandleInteraction_Unregister(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})
	registerUser(t, b, "u1", "key")

	resp := b.HandleInteraction(&discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand,
		User: &discordgo.User{ID: "u1"},
		Data: discordgo.ApplicationCommandInteractionData{Name: "unregister"},
	})
	if resp == nil || !strings.Contains(resp.Data.Content, "removed") {
		t.Error("expected unregister response")
	}
}

func TestHandleInteraction_Schedule(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})
	registerUser(t, b, "u1", "key")

	resp := b.HandleInteraction(&discordgo.Interaction{
		Type:      discordgo.InteractionApplicationCommand,
		ChannelID: "ch1",
		User:      &discordgo.User{ID: "u1"},
		Data:      discordgo.ApplicationCommandInteractionData{Name: "schedule"},
	})
	if resp == nil || !strings.Contains(resp.Data.Content, "enabled") {
		t.Error("expected schedule response")
	}
}

func TestHandleInteraction_Unschedule(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})
	resp := b.HandleInteraction(&discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand,
		User: &discordgo.User{ID: "u1"},
		Data: discordgo.ApplicationCommandInteractionData{Name: "unschedule"},
	})
	if resp == nil || !strings.Contains(resp.Data.Content, "disabled") {
		t.Error("expected unschedule response")
	}
}

func TestHandleInteraction_Unknown(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})
	resp := b.HandleInteraction(&discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand,
		User: &discordgo.User{ID: "u1"},
		Data: discordgo.ApplicationCommandInteractionData{Name: "nonexistent"},
	})
	if resp != nil {
		t.Error("expected nil for unknown command")
	}
}

func TestHandleInteraction_NonCommand(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})
	resp := b.HandleInteraction(&discordgo.Interaction{
		Type: discordgo.InteractionPing,
	})
	if resp != nil {
		t.Error("expected nil for non-command interaction")
	}
}

func TestInteractionUserID_Member(t *testing.T) {
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "member-123"},
			},
		},
	}
	if got := interactionUserID(i); got != "member-123" {
		t.Errorf("got %q, want %q", got, "member-123")
	}
}

func TestInteractionUserID_User(t *testing.T) {
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			User: &discordgo.User{ID: "user-456"},
		},
	}
	if got := interactionUserID(i); got != "user-456" {
		t.Errorf("got %q, want %q", got, "user-456")
	}
}

func TestInteractionUserID_Empty(t *testing.T) {
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{},
	}
	if got := interactionUserID(i); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestHandleInteractionCallback_Success(t *testing.T) {
	mp := &mockProvider{state: domain.PlayerState{
		Energy: 100, EnergyMax: 150, Happy: 5000,
		Life: 100, LifeMax: 100,
	}}
	s := &mockSession{}
	b := newTestBot(t, s, mp)
	registerUser(t, b, "u1", "key")

	// Simulate discordgo calling handleInteraction.
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:   discordgo.InteractionApplicationCommand,
			Member: &discordgo.Member{User: &discordgo.User{ID: "u1"}},
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "config",
			},
		},
	}
	b.handleInteraction(nil, i)

	if len(s.interactions) != 1 {
		t.Fatalf("expected 1 interaction respond call, got %d", len(s.interactions))
	}
}

func TestHandleInteractionCallback_AllCommands(t *testing.T) {
	mp := &mockProvider{state: domain.PlayerState{
		Energy: 100, EnergyMax: 150, Happy: 5000,
		Life: 100, LifeMax: 100,
		XanaxCooldown: 1, BoosterCooldown: 1, TravelCooldown: 1,
	}}
	s := &mockSession{}
	b := newTestBot(t, s, mp)
	registerUser(t, b, "u1", "key")

	commands := []struct {
		name string
		data discordgo.ApplicationCommandInteractionData
	}{
		{name: "advise", data: discordgo.ApplicationCommandInteractionData{Name: "advise"}},
		{name: "status", data: discordgo.ApplicationCommandInteractionData{Name: "status"}},
		{name: "register", data: discordgo.ApplicationCommandInteractionData{
			Name: "register",
			Options: []*discordgo.ApplicationCommandInteractionDataOption{
				{Name: "api_key", Type: discordgo.ApplicationCommandOptionString, Value: "key2"},
			},
		}},
		{name: "unregister", data: discordgo.ApplicationCommandInteractionData{Name: "unregister"}},
		{name: "schedule", data: discordgo.ApplicationCommandInteractionData{Name: "schedule"}},
		{name: "unschedule", data: discordgo.ApplicationCommandInteractionData{Name: "unschedule"}},
	}

	for _, cmd := range commands {
		t.Run(cmd.name, func(t *testing.T) {
			before := len(s.interactions)
			i := &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					Type:      discordgo.InteractionApplicationCommand,
					ChannelID: "ch1",
					User:      &discordgo.User{ID: "u1"},
					Data:      cmd.data,
				},
			}
			b.handleInteraction(nil, i)
			if len(s.interactions) != before+1 {
				t.Errorf("expected InteractionRespond to be called for %s", cmd.name)
			}
		})
	}
}

func TestHandleInteractionCallback_NonCommand(t *testing.T) {
	s := &mockSession{}
	b := newTestBot(t, s, &mockProvider{})

	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionPing,
		},
	}
	b.handleInteraction(nil, i)

	if len(s.interactions) != 0 {
		t.Error("expected no respond calls for non-command interaction")
	}
}

func TestHandleInteractionCallback_UnknownCommand(t *testing.T) {
	s := &mockSession{}
	b := newTestBot(t, s, &mockProvider{})

	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			User: &discordgo.User{ID: "u1"},
			Data: discordgo.ApplicationCommandInteractionData{Name: "bogus"},
		},
	}
	b.handleInteraction(nil, i)

	if len(s.interactions) != 0 {
		t.Error("expected no respond calls for unknown command")
	}
}

func TestHandleInteractionCallback_RespondError(t *testing.T) {
	mp := &mockProvider{state: domain.PlayerState{Life: 100, LifeMax: 100}}
	s := &mockSession{interactionErr: fmt.Errorf("respond failed")}
	b := newTestBot(t, s, mp)

	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			User: &discordgo.User{ID: "u1"},
			Data: discordgo.ApplicationCommandInteractionData{Name: "config"},
		},
	}
	// Should not panic; just logs the error.
	b.handleInteraction(nil, i)
}

func TestStartScheduler(t *testing.T) {
	mp := &mockProvider{state: domain.PlayerState{
		Life: 10, LifeMax: 100, ChainActive: true,
		XanaxCooldown: 1, BoosterCooldown: 1, TravelCooldown: 1,
	}}
	s := &mockSession{}
	b := newTestBot(t, s, mp)
	registerUser(t, b, "u1", "key")

	b.scheduleMu.Lock()
	b.scheduleChannels["u1"] = "ch1"
	b.scheduleMu.Unlock()

	b.StartScheduler(50 * time.Millisecond)
	time.Sleep(120 * time.Millisecond)
	_ = b.Stop()

	if len(s.embeds) == 0 {
		t.Error("expected at least one scheduled embed")
	}
}

func TestSchedulerLoop_StopsOnCancel(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})
	b.scheduleInterval = 10 * time.Millisecond

	done := make(chan struct{})
	go func() {
		b.schedulerLoop()
		close(done)
	}()

	b.cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("schedulerLoop did not exit after cancel")
	}
}

func TestSchedulerLoop_DefaultInterval(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})
	// Don't set scheduleInterval — it should default to 15 min.
	// Cancel immediately so we just test the default-setting path.
	b.cancel()
	b.schedulerLoop()

	if b.scheduleInterval != 15*time.Minute {
		t.Errorf("expected 15m default, got %v", b.scheduleInterval)
	}
}

func TestSendScheduledAdvice_ProviderError(t *testing.T) {
	mp := &mockProvider{err: fmt.Errorf("oops")}
	s := &mockSession{}
	b := newTestBot(t, s, mp)
	registerUser(t, b, "u1", "key")

	// Should not panic; just logs the error.
	b.sendScheduledAdvice("u1", "ch1")
	if len(s.embeds) != 0 {
		t.Error("expected no embeds on provider error")
	}
}

func TestSendScheduledAdvice_NoRegistration(t *testing.T) {
	s := &mockSession{}
	b := newTestBot(t, s, &mockProvider{})

	// User not registered — should log and return silently.
	b.sendScheduledAdvice("unknown-user", "ch1")
	if len(s.embeds) != 0 {
		t.Error("expected no embeds for unregistered user")
	}
}

func TestSendScheduledAdvice_EmbedSendError(t *testing.T) {
	mp := &mockProvider{state: domain.PlayerState{
		Life: 10, LifeMax: 100, ChainActive: true,
		XanaxCooldown: 1, BoosterCooldown: 1, TravelCooldown: 1,
	}}
	s := &mockSessionWithEmbedErr{mockSession: mockSession{}}
	b := newTestBotWithSession(t, s, mp)
	registerUser(t, b, "u1", "key")

	// Should not panic; the embed send error is logged.
	b.sendScheduledAdvice("u1", "ch1")
}

func TestBuildStatusResponse_ProviderError(t *testing.T) {
	mp := &mockProvider{err: fmt.Errorf("timeout")}
	b := newTestBot(t, &mockSession{}, mp)
	registerUser(t, b, "u1", "key")

	resp := b.BuildStatusResponse("u1")
	if !strings.Contains(resp.Data.Content, "Failed to fetch") {
		t.Errorf("unexpected: %s", resp.Data.Content)
	}
}

func TestGetProvider_ConcurrentDoubleCheck(t *testing.T) {
	mp := &mockProvider{}
	b := newTestBot(t, &mockSession{}, mp)
	registerUser(t, b, "u1", "key")
	// Remove the cached provider so goroutines will race to create it.
	b.mu.Lock()
	delete(b.providers, "u1")
	b.mu.Unlock()

	// Slow factory ensures goroutines pile up on the write lock,
	// so the second goroutine hits the double-check path.
	origFactory := b.providerFactory
	b.providerFactory = func(apiKey string) domain.StateProvider {
		time.Sleep(100 * time.Millisecond)
		return origFactory(apiKey)
	}

	// Use a barrier so all goroutines call getProvider at the same instant.
	var ready sync.WaitGroup
	var start sync.WaitGroup
	start.Add(1)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		ready.Add(1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			ready.Done()
			start.Wait()
			p, err := b.getProvider("u1")
			if err != nil {
				t.Errorf("getProvider error: %v", err)
			}
			if p == nil {
				t.Error("expected non-nil provider")
			}
		}()
	}

	ready.Wait() // all goroutines spawned
	start.Done() // release them all at once
	wg.Wait()
}

func TestRegister_EmptyStringKey(t *testing.T) {
	b := newTestBot(t, &mockSession{}, &mockProvider{})
	opts := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: "api_key", Type: discordgo.ApplicationCommandOptionString, Value: ""},
	}
	resp := b.BuildRegisterResponse("u1", opts)
	if !strings.Contains(resp.Data.Content, "empty") {
		t.Errorf("expected empty error, got: %s", resp.Data.Content)
	}
}

func newTestBotWithBadStore(t *testing.T) *Bot {
	t.Helper()
	// Create a keystore in a temp dir, then remove the dir so save() fails.
	dir := t.TempDir()
	ksPath := filepath.Join(dir, "sub", "keys.json")
	// Create the subdir so NewKeyStore succeeds.
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatalf("creating subdir: %v", err)
	}
	ks, err := store.NewKeyStore(ksPath, testEncKey())
	if err != nil {
		t.Fatalf("creating keystore: %v", err)
	}
	// Remove the directory so writes will fail.
	if err := os.RemoveAll(filepath.Join(dir, "sub")); err != nil {
		t.Fatalf("removing subdir: %v", err)
	}
	factory := func(_ string) domain.StateProvider { return &mockProvider{} }
	return New(&mockSession{}, ks, factory, config.DefaultPriorities(), 30*time.Second)
}

func TestRegister_StoreError(t *testing.T) {
	b := newTestBotWithBadStore(t)
	opts := []*discordgo.ApplicationCommandInteractionDataOption{
		{Name: "api_key", Type: discordgo.ApplicationCommandOptionString, Value: "some-api-key"},
	}
	resp := b.BuildRegisterResponse("u1", opts)
	if !strings.Contains(resp.Data.Content, "Failed") {
		t.Errorf("expected failure message, got: %s", resp.Data.Content)
	}
}

func TestUnregister_StoreError(t *testing.T) {
	b := newTestBotWithBadStore(t)
	resp := b.BuildUnregisterResponse("u1")
	if !strings.Contains(resp.Data.Content, "Failed") {
		t.Errorf("expected failure message, got: %s", resp.Data.Content)
	}
}

// mockSessionWithEmbedErr is a mockSession that returns an error from ChannelMessageSendEmbed.
type mockSessionWithEmbedErr struct {
	mockSession
}

func (m *mockSessionWithEmbedErr) ChannelMessageSendEmbed(_ string, _ *discordgo.MessageEmbed, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	return nil, fmt.Errorf("embed send failed")
}

func newTestBotWithSession(t *testing.T, s Session, mp *mockProvider) *Bot {
	t.Helper()
	dir := t.TempDir()
	ks, err := store.NewKeyStore(filepath.Join(dir, "keys.json"), testEncKey())
	if err != nil {
		t.Fatalf("creating keystore: %v", err)
	}
	factory := func(_ string) domain.StateProvider { return mp }
	return New(s, ks, factory, config.DefaultPriorities(), 30*time.Second)
}
