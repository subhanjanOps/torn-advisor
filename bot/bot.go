package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/subhanjanOps/torn-advisor/config"
	"github.com/subhanjanOps/torn-advisor/domain"
	"github.com/subhanjanOps/torn-advisor/engine"
	"github.com/subhanjanOps/torn-advisor/providers/cache"
	"github.com/subhanjanOps/torn-advisor/rules"
	"github.com/subhanjanOps/torn-advisor/store"
)

// ProviderFactory creates a StateProvider for the given Torn API key.
type ProviderFactory func(apiKey string) domain.StateProvider

// Bot wraps a Discord session and the Torn Advisor engine.
type Bot struct {
	session         Session
	keyStore        *store.KeyStore
	providerFactory ProviderFactory
	cfg             config.RulePriorities
	cacheTTL        time.Duration

	mu        sync.RWMutex
	providers map[string]*cache.Provider // discordUserID -> cached provider

	// Scheduler fields.
	scheduleInterval time.Duration
	scheduleChannels map[string]string // discordUserID -> channelID
	scheduleMu       sync.RWMutex

	// Lifecycle.
	cancel context.CancelFunc
	ctx    context.Context
}

// Session abstracts the discord session for testability.
type Session interface {
	AddHandler(handler interface{}) func()
	ApplicationCommandCreate(appID string, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
	InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error)
	Open() error
	Close() error
}

// New creates a Bot with the given dependencies.
func New(session Session, keyStore *store.KeyStore, factory ProviderFactory, cfg config.RulePriorities, cacheTTL time.Duration) *Bot {
	ctx, cancel := context.WithCancel(context.Background())
	return &Bot{
		session:          session,
		keyStore:         keyStore,
		providerFactory:  factory,
		cfg:              cfg,
		cacheTTL:         cacheTTL,
		providers:        make(map[string]*cache.Provider),
		scheduleChannels: make(map[string]string),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// slashCommands defines all the slash commands the bot registers.
var slashCommands = []*discordgo.ApplicationCommand{
	{
		Name:        "advise",
		Description: "Get your personalized Torn action plan",
	},
	{
		Name:        "status",
		Description: "Show your current Torn player status",
	},
	{
		Name:        "config",
		Description: "Show current rule priority configuration",
	},
	{
		Name:        "register",
		Description: "Register your Torn API key (sent via DM for security)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "api_key",
				Description: "Your Torn API key",
				Required:    true,
			},
		},
	},
	{
		Name:        "unregister",
		Description: "Remove your stored Torn API key",
	},
	{
		Name:        "schedule",
		Description: "Enable periodic advice in this channel (every 15 min)",
	},
	{
		Name:        "unschedule",
		Description: "Disable periodic advice",
	},
}

// RegisterAndStart registers slash commands, adds the interaction handler, and opens the session.
func (b *Bot) RegisterAndStart(appID string) error {
	b.session.AddHandler(b.handleInteraction)

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("opening discord session: %w", err)
	}

	for _, cmd := range slashCommands {
		if _, err := b.session.ApplicationCommandCreate(appID, "", cmd); err != nil {
			return fmt.Errorf("registering command %s: %w", cmd.Name, err)
		}
	}

	return nil
}

// StartScheduler begins the periodic advice loop. Call after RegisterAndStart.
func (b *Bot) StartScheduler(interval time.Duration) {
	b.scheduleInterval = interval
	go b.schedulerLoop()
}

// Stop cancels all in-flight operations and closes the Discord session.
func (b *Bot) Stop() error {
	b.cancel()
	if b.session != nil {
		return b.session.Close()
	}
	return nil
}

// HandleInteraction processes a Discord interaction and returns the response.
// Used by the webhook HTTP handler — does not require an active Discord session.
func (b *Bot) HandleInteraction(i *discordgo.Interaction) *discordgo.InteractionResponse {
	if i.Type != discordgo.InteractionApplicationCommand {
		return nil
	}

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	data := i.ApplicationCommandData()
	switch data.Name {
	case "advise":
		return b.handleAdvise(userID)
	case "status":
		return b.handleStatus(userID)
	case "config":
		return b.handleConfig()
	case "register":
		return b.handleRegister(userID, data.Options)
	case "unregister":
		return b.handleUnregister(userID)
	case "schedule":
		return b.handleSchedule(userID, i.ChannelID)
	case "unschedule":
		return b.handleUnschedule(userID)
	default:
		return nil
	}
}

func (b *Bot) getProvider(userID string) (domain.StateProvider, error) {
	apiKey, ok := b.keyStore.Get(userID)
	if !ok {
		return nil, fmt.Errorf("no API key registered — use `/register` first")
	}

	b.mu.RLock()
	cp, exists := b.providers[userID]
	b.mu.RUnlock()

	if exists {
		return cp, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Double-check.
	if cp, exists = b.providers[userID]; exists {
		return cp, nil
	}

	inner := b.providerFactory(apiKey)
	cp = cache.NewProvider(inner, b.cacheTTL)
	b.providers[userID] = cp
	return cp, nil
}

func (b *Bot) removeProvider(userID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.providers, userID)
}

func (b *Bot) handleInteraction(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	var resp *discordgo.InteractionResponse
	userID := interactionUserID(i)

	switch i.ApplicationCommandData().Name {
	case "advise":
		resp = b.handleAdvise(userID)
	case "status":
		resp = b.handleStatus(userID)
	case "config":
		resp = b.handleConfig()
	case "register":
		resp = b.handleRegister(userID, i.ApplicationCommandData().Options)
	case "unregister":
		resp = b.handleUnregister(userID)
	case "schedule":
		resp = b.handleSchedule(userID, i.ChannelID)
	case "unschedule":
		resp = b.handleUnschedule(userID)
	default:
		return
	}

	if err := b.session.InteractionRespond(i.Interaction, resp); err != nil {
		log.Printf("responding to interaction: %v", err)
	}
}

func interactionUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func (b *Bot) handleAdvise(userID string) *discordgo.InteractionResponse {
	provider, err := b.getProvider(userID)
	if err != nil {
		return errorResponse(err.Error())
	}

	state, err := provider.FetchPlayerState(b.ctx)
	if err != nil {
		return errorResponse(fmt.Sprintf("Failed to fetch player state: %v", err))
	}

	eng := engine.NewEngine(rules.DefaultRulesWithConfig(b.cfg))
	plan := eng.Run(state)

	if len(plan) == 0 {
		return embedResponse("Torn Advisor", "No actions recommended right now. You're all set!", 0x00FF00)
	}

	var sb strings.Builder
	for i, action := range plan {
		sb.WriteString(fmt.Sprintf("**%d.** `[%s]` **%s** (priority %d)\n", i+1, action.Category, action.Name, action.Priority))
		sb.WriteString(fmt.Sprintf("   %s\n\n", action.Description))
	}

	return embedResponse("Torn Advisor — Action Plan", sb.String(), 0x3498DB)
}

func (b *Bot) handleStatus(userID string) *discordgo.InteractionResponse {
	provider, err := b.getProvider(userID)
	if err != nil {
		return errorResponse(err.Error())
	}

	state, err := provider.FetchPlayerState(b.ctx)
	if err != nil {
		return errorResponse(fmt.Sprintf("Failed to fetch player state: %v", err))
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "Energy", Value: fmt.Sprintf("%d / %d", state.Energy, state.EnergyMax), Inline: true},
		{Name: "Nerve", Value: fmt.Sprintf("%d / %d", state.Nerve, state.NerveMax), Inline: true},
		{Name: "Happy", Value: fmt.Sprintf("%d", state.Happy), Inline: true},
		{Name: "Life", Value: fmt.Sprintf("%d / %d", state.Life, state.LifeMax), Inline: true},
		{Name: "Chain Active", Value: fmt.Sprintf("%v", state.ChainActive), Inline: true},
		{Name: "War Active", Value: fmt.Sprintf("%v", state.WarActive), Inline: true},
	}

	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:  "Player Status",
					Color:  0x2ECC71,
					Fields: fields,
				},
			},
		},
	}
}

func (b *Bot) handleConfig() *discordgo.InteractionResponse {
	cfg := b.cfg
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Hospital: **%d**\n", cfg.Hospital))
	sb.WriteString(fmt.Sprintf("Chain: **%d**\n", cfg.Chain))
	sb.WriteString(fmt.Sprintf("War: **%d**\n", cfg.War))
	sb.WriteString(fmt.Sprintf("Xanax: **%d**\n", cfg.Xanax))
	sb.WriteString(fmt.Sprintf("Rehab: **%d**\n", cfg.Rehab))
	sb.WriteString(fmt.Sprintf("Gym: **%d**\n", cfg.Gym))
	sb.WriteString(fmt.Sprintf("Crime: **%d**\n", cfg.Crime))
	sb.WriteString(fmt.Sprintf("Travel: **%d**\n", cfg.Travel))
	sb.WriteString(fmt.Sprintf("Booster: **%d**\n", cfg.Booster))

	return embedResponse("Rule Priorities", sb.String(), 0x9B59B6)
}

func (b *Bot) handleRegister(userID string, opts []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionResponse {
	if len(opts) == 0 {
		return errorResponse("API key is required")
	}
	apiKey := opts[0].StringValue()
	if apiKey == "" {
		return errorResponse("API key cannot be empty")
	}

	if err := b.keyStore.Set(userID, apiKey); err != nil {
		return errorResponse(fmt.Sprintf("Failed to store API key: %v", err))
	}

	// Invalidate any existing cached provider.
	b.removeProvider(userID)

	return ephemeralResponse("✅ Your Torn API key has been registered and encrypted. Use `/advise` to get started!")
}

func (b *Bot) handleUnregister(userID string) *discordgo.InteractionResponse {
	if err := b.keyStore.Delete(userID); err != nil {
		return errorResponse(fmt.Sprintf("Failed to remove API key: %v", err))
	}
	b.removeProvider(userID)

	b.scheduleMu.Lock()
	delete(b.scheduleChannels, userID)
	b.scheduleMu.Unlock()

	return ephemeralResponse("✅ Your API key and schedule have been removed.")
}

func (b *Bot) handleSchedule(userID, channelID string) *discordgo.InteractionResponse {
	if _, ok := b.keyStore.Get(userID); !ok {
		return errorResponse("No API key registered — use `/register` first")
	}

	b.scheduleMu.Lock()
	b.scheduleChannels[userID] = channelID
	b.scheduleMu.Unlock()

	return ephemeralResponse("✅ Periodic advice enabled in this channel. Use `/unschedule` to disable.")
}

func (b *Bot) handleUnschedule(userID string) *discordgo.InteractionResponse {
	b.scheduleMu.Lock()
	delete(b.scheduleChannels, userID)
	b.scheduleMu.Unlock()

	return ephemeralResponse("✅ Periodic advice disabled.")
}

func (b *Bot) schedulerLoop() {
	if b.scheduleInterval <= 0 {
		b.scheduleInterval = 15 * time.Minute
	}
	ticker := time.NewTicker(b.scheduleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.runScheduledAdvice()
		}
	}
}

func (b *Bot) runScheduledAdvice() {
	b.scheduleMu.RLock()
	snapshot := make(map[string]string, len(b.scheduleChannels))
	for uid, chID := range b.scheduleChannels {
		snapshot[uid] = chID
	}
	b.scheduleMu.RUnlock()

	for userID, channelID := range snapshot {
		b.sendScheduledAdvice(userID, channelID)
	}
}

func (b *Bot) sendScheduledAdvice(userID, channelID string) {
	provider, err := b.getProvider(userID)
	if err != nil {
		log.Printf("scheduler: user %s: %v", userID, err)
		return
	}

	state, err := provider.FetchPlayerState(b.ctx)
	if err != nil {
		log.Printf("scheduler: user %s: fetch error: %v", userID, err)
		return
	}

	eng := engine.NewEngine(rules.DefaultRulesWithConfig(b.cfg))
	plan := eng.Run(state)

	// Only post if there are high-priority actions (priority >= 90).
	var urgent []domain.Action
	for _, a := range plan {
		if a.Priority >= 90 {
			urgent = append(urgent, a)
		}
	}
	if len(urgent) == 0 {
		return
	}

	var sb strings.Builder
	for i, action := range urgent {
		sb.WriteString(fmt.Sprintf("**%d.** `[%s]` **%s** (priority %d)\n", i+1, action.Category, action.Name, action.Priority))
		sb.WriteString(fmt.Sprintf("   %s\n\n", action.Description))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "⏰ Scheduled Advice — Urgent Actions",
		Description: sb.String(),
		Color:       0xE74C3C,
	}

	if _, err := b.session.ChannelMessageSendEmbed(channelID, embed); err != nil {
		log.Printf("scheduler: user %s: send error: %v", userID, err)
	}
}

// --- Exported for testing / webhook ---

// BuildAdviseResponse generates the advise response.
func (b *Bot) BuildAdviseResponse(userID string) *discordgo.InteractionResponse {
	return b.handleAdvise(userID)
}

// BuildStatusResponse generates the status response.
func (b *Bot) BuildStatusResponse(userID string) *discordgo.InteractionResponse {
	return b.handleStatus(userID)
}

// BuildConfigResponse generates the config response.
func (b *Bot) BuildConfigResponse() *discordgo.InteractionResponse {
	return b.handleConfig()
}

// BuildRegisterResponse generates the register response.
func (b *Bot) BuildRegisterResponse(userID string, opts []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.InteractionResponse {
	return b.handleRegister(userID, opts)
}

// BuildUnregisterResponse generates the unregister response.
func (b *Bot) BuildUnregisterResponse(userID string) *discordgo.InteractionResponse {
	return b.handleUnregister(userID)
}

// BuildScheduleResponse generates the schedule response.
func (b *Bot) BuildScheduleResponse(userID, channelID string) *discordgo.InteractionResponse {
	return b.handleSchedule(userID, channelID)
}

// BuildUnscheduleResponse generates the unschedule response.
func (b *Bot) BuildUnscheduleResponse(userID string) *discordgo.InteractionResponse {
	return b.handleUnschedule(userID)
}

// RunScheduledAdvice exposes the scheduler tick for testing.
func (b *Bot) RunScheduledAdvice() {
	b.runScheduledAdvice()
}

func embedResponse(title, description string, color int) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       title,
					Description: description,
					Color:       color,
				},
			},
		},
	}
}

func errorResponse(msg string) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "⚠️ " + msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}
}

func ephemeralResponse(msg string) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}
}
