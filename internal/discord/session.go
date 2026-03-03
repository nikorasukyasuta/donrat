package discord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
)

// NewSession creates a configured Discord session for slash-command interactions.
func NewSession(token string) (*discordgo.Session, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("create discord session: %w", err)
	}
	session.Identify.Intents = discordgo.IntentsGuilds
	return session, nil
}

// RegisterCommands overwrites application commands either globally or for one guild.
func RegisterCommands(ctx context.Context, session *discordgo.Session, appID string, guildID string, logger zerolog.Logger) ([]*discordgo.ApplicationCommand, error) {
	_ = ctx
	commands, err := session.ApplicationCommandBulkOverwrite(appID, guildID, SlashCommands)
	if err != nil {
		return nil, fmt.Errorf("register slash commands: %w", err)
	}
	logger.Info().Int("count", len(commands)).Str("guild_id", guildID).Msg("slash commands registered")
	return commands, nil
}
