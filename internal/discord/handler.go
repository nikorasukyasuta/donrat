package discord

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"

	"donrat/internal/casino"
	"donrat/internal/models"
	"donrat/internal/mongo"
	"donrat/internal/wallet"
)

const (
	embedColorInfo    = 0x3498DB
	embedColorSuccess = 0x2ECC71
	embedColorDanger  = 0xE74C3C
)

// Handler routes Discord slash interactions.
type Handler struct {
	wallet   wallet.Service
	casino   casino.Service
	presence *PresenceManager
	logger   zerolog.Logger
	owner    string
	ownerMu  sync.Mutex
	ownerID  string
}

// NewHandler creates a command interaction handler.
func NewHandler(walletService wallet.Service, casinoService casino.Service, presenceManager *PresenceManager, logger zerolog.Logger, ownerName string) *Handler {
	return &Handler{
		wallet:   walletService,
		casino:   casinoService,
		presence: presenceManager,
		logger:   logger,
		owner:    ownerName,
	}
}

// HandleInteraction dispatches slash commands.
func (h *Handler) HandleInteraction(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	if interaction.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := interaction.ApplicationCommandData()
	ctx := context.Background()

	switch data.Name {
	case "balance":
		h.handleBalance(ctx, session, interaction)
	case "wallet":
		h.handleWallet(ctx, session, interaction)
	case "bet":
		h.handleBet(ctx, session, interaction)
	case "slots":
		h.handleSlots(ctx, session, interaction)
	case "roulette":
		h.handleRoulette(ctx, session, interaction)
	case "dice":
		h.handleDice(ctx, session, interaction)
	case "blackjack":
		h.handleBlackjack(ctx, session, interaction)
	case "war":
		h.handleWar(ctx, session, interaction)
	case "poker":
		h.handlePoker(ctx, session, interaction)
	case "leaderboard":
		h.handleLeaderboard(ctx, session, interaction)
	case "daily":
		h.handleDaily(ctx, session, interaction)
	case "history":
		h.handleHistory(ctx, session, interaction)
	case "house":
		h.handleHouse(ctx, session, interaction)
	case "trade":
		h.handleTrade(ctx, session, interaction)
	case "donrat":
		h.handleDonRat(session, interaction)
	default:
		h.respondError(session, interaction.Interaction, "Unknown command.")
	}
}

func (h *Handler) handleBalance(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	user := interactionUser(interaction)
	walletState, err := h.wallet.GetBalance(ctx, user)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "balance").Msg("command failed")
		h.respondError(session, interaction.Interaction, "Don Rat couldn't fetch your wallet.")
		return
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "Rat", Value: fmt.Sprintf("<@%s>", user.ID), Inline: true},
		{Name: "Wallet", Value: fmt.Sprintf("%d credits", walletState.Balance), Inline: true},
	}
	if walletState.IsProtected {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "House Note", Value: "Don Rat's untouchable rat. You never lose credits.", Inline: false})
	}

	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "Rat Wallet Ledger",
		Description: "Don Rat inspects your pockets.",
		Color:       embedColorInfo,
		Fields:      fields,
	})
}

func (h *Handler) handleWallet(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	user := interactionUser(interaction)
	walletState, err := h.wallet.EnsureWallet(ctx, user)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "wallet").Msg("wallet creation failed")
		h.respondError(session, interaction.Interaction, "Don Rat couldn't set up your wallet.")
		return
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "Rat", Value: fmt.Sprintf("<@%s>", user.ID), Inline: true},
		{Name: "Starting Balance", Value: fmt.Sprintf("%d credits", walletState.Balance), Inline: true},
	}
	if walletState.IsProtected {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "House Note", Value: "Don Rat's favorite. You never lose credits.", Inline: false})
	}
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Don Rat Note",
		Value:  casino.PersonaResponse(user.Username, casino.Lore),
		Inline: false,
	})

	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "Welcome to Don Rat's Casino",
		Description: "Your wallet is ready. Time to test your luck, rat.",
		Color:       embedColorSuccess,
		Fields:      fields,
	})
	h.updatePresence(session, "wallet", user.Username)
}

func (h *Handler) handleBet(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	data := interaction.ApplicationCommandData()
	if len(data.Options) == 0 || data.Options[0].Name != "coinflip" {
		h.respondError(session, interaction.Interaction, "Only /bet coinflip <amount> is supported.")
		return
	}
	options := data.Options[0].Options
	amountOpt := findOption(options, "amount")
	if amountOpt == nil {
		h.respondError(session, interaction.Interaction, "Amount is required.")
		return
	}
	amount := amountOpt.IntValue()
	if amount <= 0 {
		h.respondError(session, interaction.Interaction, "Amount must be greater than zero.")
		return
	}

	user := interactionUser(interaction)
	result, err := h.casino.Coinflip(ctx, user, amount)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "bet").Str("user_id", user.ID).Msg("coinflip failed")
		h.respondError(session, interaction.Interaction, "Don Rat rejects this bet: "+err.Error())
		return
	}

	status := "Loss"
	color := embedColorDanger
	if result.Won {
		status = "Win"
		color = embedColorSuccess
	}
	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "Coinflip Ledger",
		Description: result.Details,
		Color:       color,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Result", Value: status, Inline: true},
			{Name: "Bet", Value: fmt.Sprintf("%d", result.Amount), Inline: true},
			{Name: "Net", Value: fmt.Sprintf("%+d", result.NetDelta), Inline: true},
			{Name: "Balance", Value: fmt.Sprintf("%d → %d", result.BalanceBefore, result.BalanceAfter), Inline: false},
			{Name: "Don Rat Note", Value: pickResultPersona(user, result.Won), Inline: false},
			{Name: "Table Presence", Value: h.describePresence("coinflip", user.Username), Inline: false},
		},
	})
	h.updatePresence(session, "coinflip", user.Username)
}

func (h *Handler) handleSlots(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	data := interaction.ApplicationCommandData()
	amountOpt := findOption(data.Options, "amount")
	if amountOpt == nil {
		h.respondError(session, interaction.Interaction, "Amount is required.")
		return
	}
	amount := amountOpt.IntValue()
	if amount <= 0 {
		h.respondError(session, interaction.Interaction, "Amount must be greater than zero.")
		return
	}

	user := interactionUser(interaction)
	result, err := h.casino.Slots(ctx, user, amount)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "slots").Str("user_id", user.ID).Msg("slots failed")
		h.respondError(session, interaction.Interaction, "Don Rat shuts down the slots: "+err.Error())
		return
	}

	status := "Loss"
	color := embedColorDanger
	if result.Won {
		status = "Win"
		color = embedColorSuccess
	}
	fields := []*discordgo.MessageEmbedField{
		{Name: "Result", Value: status, Inline: true},
		{Name: "Bet", Value: fmt.Sprintf("%d", result.Amount), Inline: true},
		{Name: "Net", Value: fmt.Sprintf("%+d", result.NetDelta), Inline: true},
		{Name: "Balance", Value: fmt.Sprintf("%d → %d", result.BalanceBefore, result.BalanceAfter), Inline: false},
		{Name: "Don Rat Note", Value: pickResultPersona(user, result.Won), Inline: false},
		{Name: "Table Presence", Value: h.describePresence("slots", user.Username), Inline: false},
	}

	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "Slots Ledger",
		Description: result.Details,
		Color:       color,
		Fields:      fields,
	})
	h.updatePresence(session, "slots", user.Username)
}

func (h *Handler) handleRoulette(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	data := interaction.ApplicationCommandData()
	amountOpt := findOption(data.Options, "amount")
	colorOpt := findOption(data.Options, "color")
	if amountOpt == nil || colorOpt == nil {
		h.respondError(session, interaction.Interaction, "Amount and color are required.")
		return
	}

	amount := amountOpt.IntValue()
	if amount <= 0 {
		h.respondError(session, interaction.Interaction, "Amount must be greater than zero.")
		return
	}
	color := strings.ToLower(colorOpt.StringValue())
	if color != "red" && color != "black" && color != "green" {
		h.respondError(session, interaction.Interaction, "Color must be red, black, or green.")
		return
	}

	user := interactionUser(interaction)
	result, err := h.casino.Roulette(ctx, user, amount, color)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "roulette").Str("user_id", user.ID).Msg("roulette failed")
		h.respondError(session, interaction.Interaction, "Don Rat spins you away: "+err.Error())
		return
	}

	status := "Loss"
	colorHex := embedColorDanger
	if result.Won {
		status = "Win"
		colorHex = embedColorSuccess
	}

	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "Roulette Ledger",
		Description: result.Details,
		Color:       colorHex,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Result", Value: status, Inline: true},
			{Name: "Bet", Value: fmt.Sprintf("%d on %s", result.Amount, strings.ToUpper(color)), Inline: true},
			{Name: "Net", Value: fmt.Sprintf("%+d", result.NetDelta), Inline: true},
			{Name: "Balance", Value: fmt.Sprintf("%d → %d", result.BalanceBefore, result.BalanceAfter), Inline: false},
			{Name: "Don Rat Note", Value: pickResultPersona(user, result.Won), Inline: false},
			{Name: "Table Presence", Value: h.describePresence("roulette", user.Username), Inline: false},
		},
	})
	h.updatePresence(session, "roulette", user.Username)
}

func (h *Handler) handleDice(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	data := interaction.ApplicationCommandData()
	amountOpt := findOption(data.Options, "amount")
	guessOpt := findOption(data.Options, "guess")
	if amountOpt == nil || guessOpt == nil {
		h.respondError(session, interaction.Interaction, "Amount and guess are required.")
		return
	}

	amount := amountOpt.IntValue()
	if amount <= 0 {
		h.respondError(session, interaction.Interaction, "Amount must be greater than zero.")
		return
	}
	guess := guessOpt.IntValue()
	if guess < 1 || guess > 6 {
		h.respondError(session, interaction.Interaction, "Guess must be between 1 and 6.")
		return
	}

	user := interactionUser(interaction)
	result, err := h.casino.Dice(ctx, user, amount, guess)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "dice").Str("user_id", user.ID).Msg("dice failed")
		h.respondError(session, interaction.Interaction, "Don Rat shakes the dice cup shut: "+err.Error())
		return
	}

	status := "Loss"
	colorHex := embedColorDanger
	if result.Won {
		status = "Win"
		colorHex = embedColorSuccess
	}

	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "Dice Ledger",
		Description: result.Details,
		Color:       colorHex,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Result", Value: status, Inline: true},
			{Name: "Bet", Value: fmt.Sprintf("%d", result.Amount), Inline: true},
			{Name: "Net", Value: fmt.Sprintf("%+d", result.NetDelta), Inline: true},
			{Name: "Balance", Value: fmt.Sprintf("%d → %d", result.BalanceBefore, result.BalanceAfter), Inline: false},
			{Name: "Don Rat Note", Value: pickResultPersona(user, result.Won), Inline: false},
			{Name: "Table Presence", Value: h.describePresence("dice", user.Username), Inline: false},
		},
	})
	h.updatePresence(session, "dice", user.Username)
}

func (h *Handler) handleBlackjack(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	data := interaction.ApplicationCommandData()
	amountOpt := findOption(data.Options, "amount")
	if amountOpt == nil {
		h.respondError(session, interaction.Interaction, "Amount is required.")
		return
	}
	amount := amountOpt.IntValue()
	if amount <= 0 {
		h.respondError(session, interaction.Interaction, "Amount must be greater than zero.")
		return
	}

	user := interactionUser(interaction)
	result, err := h.casino.Blackjack(ctx, user, amount)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "blackjack").Str("user_id", user.ID).Msg("blackjack failed")
		h.respondError(session, interaction.Interaction, "Don Rat folds your hand: "+err.Error())
		return
	}

	status := "Loss"
	color := embedColorDanger
	if result.NetDelta == 0 {
		status = "Push"
		color = embedColorInfo
	}
	if result.Won {
		status = "Win"
		color = embedColorSuccess
	}

	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "Blackjack Ledger",
		Description: result.Details,
		Color:       color,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Result", Value: status, Inline: true},
			{Name: "Bet", Value: fmt.Sprintf("%d", result.Amount), Inline: true},
			{Name: "Net", Value: fmt.Sprintf("%+d", result.NetDelta), Inline: true},
			{Name: "Balance", Value: fmt.Sprintf("%d → %d", result.BalanceBefore, result.BalanceAfter), Inline: false},
			{Name: "Don Rat Note", Value: pickResultPersona(user, result.Won), Inline: false},
			{Name: "Table Presence", Value: h.describePresence("blackjack", user.Username), Inline: false},
		},
	})
	h.updatePresence(session, "blackjack", user.Username)
}

func (h *Handler) handleWar(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	data := interaction.ApplicationCommandData()
	amountOpt := findOption(data.Options, "amount")
	if amountOpt == nil {
		h.respondError(session, interaction.Interaction, "Amount is required.")
		return
	}
	amount := amountOpt.IntValue()
	if amount <= 0 {
		h.respondError(session, interaction.Interaction, "Amount must be greater than zero.")
		return
	}

	user := interactionUser(interaction)
	result, err := h.casino.War(ctx, user, amount)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "war").Str("user_id", user.ID).Msg("war failed")
		h.respondError(session, interaction.Interaction, "Don Rat burns the deck: "+err.Error())
		return
	}

	status := "Loss"
	color := embedColorDanger
	if result.NetDelta == 0 {
		status = "Tie"
		color = embedColorInfo
	}
	if result.Won {
		status = "Win"
		color = embedColorSuccess
	}

	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "Card War Ledger",
		Description: result.Details,
		Color:       color,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Result", Value: status, Inline: true},
			{Name: "Bet", Value: fmt.Sprintf("%d", result.Amount), Inline: true},
			{Name: "Net", Value: fmt.Sprintf("%+d", result.NetDelta), Inline: true},
			{Name: "Balance", Value: fmt.Sprintf("%d → %d", result.BalanceBefore, result.BalanceAfter), Inline: false},
			{Name: "Don Rat Note", Value: pickResultPersona(user, result.Won), Inline: false},
			{Name: "Table Presence", Value: h.describePresence("war", user.Username), Inline: false},
		},
	})
	h.updatePresence(session, "war", user.Username)
}

func (h *Handler) handlePoker(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	data := interaction.ApplicationCommandData()
	amountOpt := findOption(data.Options, "amount")
	if amountOpt == nil {
		h.respondError(session, interaction.Interaction, "Amount is required.")
		return
	}
	amount := amountOpt.IntValue()
	if amount <= 0 {
		h.respondError(session, interaction.Interaction, "Amount must be greater than zero.")
		return
	}

	user := interactionUser(interaction)
	result, err := h.casino.Poker(ctx, user, amount)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "poker").Str("user_id", user.ID).Msg("poker failed")
		h.respondError(session, interaction.Interaction, "Don Rat sweeps the poker table: "+err.Error())
		return
	}

	status := "Loss"
	color := embedColorDanger
	if result.NetDelta == 0 {
		status = "Tie"
		color = embedColorInfo
	}
	if result.Won {
		status = "Win"
		color = embedColorSuccess
	}

	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "Poker Ledger",
		Description: result.Details,
		Color:       color,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Result", Value: status, Inline: true},
			{Name: "Bet", Value: fmt.Sprintf("%d", result.Amount), Inline: true},
			{Name: "Net", Value: fmt.Sprintf("%+d", result.NetDelta), Inline: true},
			{Name: "Balance", Value: fmt.Sprintf("%d → %d", result.BalanceBefore, result.BalanceAfter), Inline: false},
			{Name: "Don Rat Note", Value: pickResultPersona(user, result.Won), Inline: false},
			{Name: "Table Presence", Value: h.describePresence("poker", user.Username), Inline: false},
		},
	})
	h.updatePresence(session, "poker", user.Username)
}

func (h *Handler) handleLeaderboard(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	data := interaction.ApplicationCommandData()
	limit := int64(10)
	if limitOpt := findOption(data.Options, "limit"); limitOpt != nil {
		if parsed := limitOpt.IntValue(); parsed > 0 {
			limit = parsed
		}
	}

	wallets, err := h.wallet.TopWallets(ctx, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "leaderboard").Msg("leaderboard failed")
		h.respondError(session, interaction.Interaction, "Don Rat cannot count your wallets right now.")
		return
	}

	lines := []string{}
	for idx, walletRow := range wallets {
		lines = append(lines, fmt.Sprintf("%d. <@%s> — %d credits", idx+1, walletRow.UserID, walletRow.Balance))
	}
	if len(lines) == 0 {
		lines = append(lines, "No rats on the board yet.")
	}

	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "Casino Leaderboard",
		Description: strings.Join(lines, "\n"),
		Color:       embedColorInfo,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Don Rat Note", Value: casino.PersonaResponse("leaderboard", casino.Lore), Inline: false},
			{Name: "Table Presence", Value: h.describePresence("leaderboard", ""), Inline: false},
		},
	})
	h.updatePresence(session, "leaderboard", "")
}

func (h *Handler) handleDaily(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	const dailyAmount int64 = 250
	user := interactionUser(interaction)
	walletState, claimed, nextClaim, err := h.wallet.ClaimDaily(ctx, user, dailyAmount)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "daily").Str("user_id", user.ID).Msg("daily claim failed")
		h.respondError(session, interaction.Interaction, "Don Rat's vault clerk is missing: "+err.Error())
		return
	}

	color := embedColorInfo
	status := "Already claimed"
	details := fmt.Sprintf("Next daily claim at %s UTC", nextClaim.Format("2006-01-02 15:04"))
	if claimed {
		color = embedColorSuccess
		status = "Claimed"
		details = fmt.Sprintf("You received %d credits.", dailyAmount)
	}

	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "Daily Stipend Ledger",
		Description: details,
		Color:       color,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Status", Value: status, Inline: true},
			{Name: "Wallet", Value: fmt.Sprintf("%d credits", walletState.Balance), Inline: true},
			{Name: "Don Rat Note", Value: casino.PersonaResponse(user.Username, casino.GeneralSarcasm), Inline: false},
			{Name: "Table Presence", Value: h.describePresence("daily", user.Username), Inline: false},
		},
	})
	h.updatePresence(session, "daily", user.Username)
}

func (h *Handler) handleHistory(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	data := interaction.ApplicationCommandData()
	limit := int64(5)
	if limitOpt := findOption(data.Options, "limit"); limitOpt != nil {
		if parsed := limitOpt.IntValue(); parsed > 0 {
			limit = parsed
		}
	}

	user := interactionUser(interaction)
	history, err := h.wallet.RecentTransactions(ctx, user, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "history").Str("user_id", user.ID).Msg("history failed")
		h.respondError(session, interaction.Interaction, "Don Rat misplaced your ledger.")
		return
	}

	entries := []string{}
	for _, tx := range history {
		entries = append(entries, fmt.Sprintf("%s | %s | %+d | %d→%d", tx.CreatedAt.Format(time.RFC3339), tx.Type, tx.Amount, tx.BalanceFrom, tx.BalanceTo))
	}
	if len(entries) == 0 {
		entries = append(entries, "No transaction history yet.")
	}

	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "Wallet History Ledger",
		Description: strings.Join(entries, "\n"),
		Color:       embedColorInfo,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Don Rat Note", Value: casino.PersonaResponse(user.Username, casino.Lore), Inline: false},
			{Name: "Table Presence", Value: h.describePresence("history", user.Username), Inline: false},
		},
	})
	h.updatePresence(session, "history", user.Username)
}

func (h *Handler) handleHouse(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	allowed, err := h.canAccessHouse(session, interaction)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "house").Msg("house authorization failed")
		h.respondError(session, interaction.Interaction, "Don Rat cannot verify your rank right now.")
		return
	}
	if !allowed {
		h.respondError(session, interaction.Interaction, "Only administrators and the bot owner may access house analytics.")
		return
	}

	analytics, err := h.wallet.HouseAnalytics(ctx)
	if err != nil {
		h.logger.Error().Err(err).Str("command", "house").Msg("house analytics failed")
		h.respondError(session, interaction.Interaction, "Don Rat's accountants are hiding in the walls.")
		return
	}

	breakdown := []string{}
	for game, count := range analytics.GameBreakdown {
		breakdown = append(breakdown, fmt.Sprintf("%s: %d", game, count))
	}
	if len(breakdown) == 0 {
		breakdown = append(breakdown, "No game rounds logged yet.")
	}

	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "House Analytics Ledger",
		Description: "Don Rat's back-office numbers.",
		Color:       embedColorInfo,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Total Transactions", Value: fmt.Sprintf("%d", analytics.TotalTransactions), Inline: true},
			{Name: "Unique Rats", Value: fmt.Sprintf("%d", analytics.UniqueRats), Inline: true},
			{Name: "Top Game", Value: analytics.TopGame, Inline: true},
			{Name: "Total Wagered", Value: fmt.Sprintf("%d", analytics.TotalWagered), Inline: true},
			{Name: "Total Payouts", Value: fmt.Sprintf("%d", analytics.TotalPayouts), Inline: true},
			{Name: "House Net", Value: fmt.Sprintf("%+d", analytics.HouseNet), Inline: true},
			{Name: "Game Volume", Value: strings.Join(breakdown, "\n"), Inline: false},
			{Name: "Don Rat Note", Value: casino.PersonaResponse("house", casino.Lore), Inline: false},
			{Name: "Table Presence", Value: h.describePresence("house", ""), Inline: false},
		},
	})
	h.updatePresence(session, "house", "")
}

func (h *Handler) canAccessHouse(session *discordgo.Session, interaction *discordgo.InteractionCreate) (bool, error) {
	user := interactionUser(interaction)
	ownerID, err := h.getBotOwnerID(session)
	if err != nil {
		return false, err
	}
	if ownerID != "" && user.ID == ownerID {
		return true, nil
	}

	if interaction.Member != nil {
		permissions := interaction.Member.Permissions
		if permissions&discordgo.PermissionAdministrator != 0 {
			return true, nil
		}
	}

	return false, nil
}

func (h *Handler) getBotOwnerID(session *discordgo.Session) (string, error) {
	h.ownerMu.Lock()
	if h.ownerID != "" {
		defer h.ownerMu.Unlock()
		return h.ownerID, nil
	}
	h.ownerMu.Unlock()

	app, err := session.Application("@me")
	if err != nil {
		return "", fmt.Errorf("fetch bot application: %w", err)
	}

	resolvedOwnerID := ""
	if app.Owner != nil {
		resolvedOwnerID = app.Owner.ID
	}

	h.ownerMu.Lock()
	h.ownerID = resolvedOwnerID
	h.ownerMu.Unlock()

	return resolvedOwnerID, nil
}

func (h *Handler) handleTrade(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	data := interaction.ApplicationCommandData()
	userOpt := findOption(data.Options, "user")
	amountOpt := findOption(data.Options, "amount")
	if userOpt == nil || amountOpt == nil {
		h.respondError(session, interaction.Interaction, "User and amount are required.")
		return
	}
	amount := amountOpt.IntValue()
	if amount <= 0 {
		h.respondError(session, interaction.Interaction, "Amount must be greater than zero.")
		return
	}

	receiver := userOpt.UserValue(session)
	if receiver == nil {
		h.respondError(session, interaction.Interaction, "Could not resolve rat recipient.")
		return
	}

	senderRef := interactionUser(interaction)
	receiverRef := models.UserRef{ID: receiver.ID, Username: receiver.Username}

	result, err := h.wallet.TransferCredits(ctx, senderRef, receiverRef, amount)
	if err != nil {
		if errors.Is(err, mongo.ErrInsufficientFunds) {
			h.respondError(session, interaction.Interaction, "Don Rat says your wallet is too thin for this trade.")
			return
		}
		h.logger.Error().Err(err).Str("command", "trade").Str("from_user_id", senderRef.ID).Str("to_user_id", receiverRef.ID).Msg("trade failed")
		h.respondError(session, interaction.Interaction, "Don Rat rejects this trade: "+err.Error())
		return
	}

	note := "Credits moved cleanly."
	if result.FromBefore == result.FromAfter {
		note = casino.PersonaResponse(senderRef.Username, casino.ProtectedUser)
	} else {
		note = casino.PersonaResponse(senderRef.Username, casino.Trade)
	}

	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       "Trade Ledger",
		Description: fmt.Sprintf("<@%s> sent %d credits to <@%s>.", senderRef.ID, amount, receiverRef.ID),
		Color:       embedColorInfo,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Sender Balance", Value: fmt.Sprintf("%d → %d", result.FromBefore, result.FromAfter), Inline: true},
			{Name: "Receiver Balance", Value: fmt.Sprintf("%d → %d", result.ToBefore, result.ToAfter), Inline: true},
			{Name: "Don Rat Note", Value: note, Inline: false},
			{Name: "Table Presence", Value: h.describePresence("trade", senderRef.Username), Inline: false},
		},
	})
	h.updatePresence(session, "trade", senderRef.Username)
}

func (h *Handler) handleDonRat(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	user := interactionUser(interaction)
	message := casino.PersonaResponse(user.Username, casino.GeneralSarcasm)
	h.respondEmbed(session, interaction.Interaction, &discordgo.MessageEmbed{
		Title:       h.owner,
		Description: message + " Now move, before I tax your whiskers.",
		Color:       embedColorInfo,
	})
	h.updatePresence(session, "donrat", user.Username)
}

func (h *Handler) respondEmbed(session *discordgo.Session, interaction *discordgo.Interaction, embed *discordgo.MessageEmbed) {
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to send interaction response")
	}
}

func (h *Handler) respondError(session *discordgo.Session, interaction *discordgo.Interaction, message string) {
	err := session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: strings.TrimSpace(message),
		},
	})
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to send error interaction response")
	}
}

func interactionUser(interaction *discordgo.InteractionCreate) models.UserRef {
	if interaction.Member != nil && interaction.Member.User != nil {
		return models.UserRef{ID: interaction.Member.User.ID, Username: interaction.Member.User.Username}
	}
	if interaction.User != nil {
		return models.UserRef{ID: interaction.User.ID, Username: interaction.User.Username}
	}
	return models.UserRef{ID: "unknown", Username: "unknown-rat"}
}

func findOption(options []*discordgo.ApplicationCommandInteractionDataOption, name string) *discordgo.ApplicationCommandInteractionDataOption {
	for _, option := range options {
		if option.Name == name {
			return option
		}
	}
	return nil
}

func (h *Handler) updatePresence(session *discordgo.Session, game string, username string) {
	if h.presence == nil {
		return
	}
	if err := h.presence.MarkActive(session, game, username); err != nil {
		h.logger.Warn().Err(err).Str("game", game).Msg("failed to update active presence")
	}
}

func pickResultPersona(user models.UserRef, won bool) string {
	if won {
		return casino.PersonaResponse(user.Username, casino.WinGrudging)
	}
	return casino.PersonaResponse(user.Username, casino.LossMocking)
}

func (h *Handler) describePresence(game string, username string) string {
	if h.presence == nil {
		return "🐀 Running Don Rat Casino"
	}
	return h.presence.DescribeActivity(game, username)
}
