package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"donrat/internal/casino"
	"donrat/internal/config"
	"donrat/internal/discord"
	"donrat/internal/mongo"
	"donrat/internal/utils"
	"donrat/internal/wallet"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	logger := utils.NewLogger(cfg.LogLevel, cfg.Environment)
	logger.Info().Msg("starting Don Rat casino bot")

	store, err := mongo.Connect(ctx, cfg.Mongo.URI, cfg.Mongo.Database, cfg.Mongo.ConnectTimeout, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to MongoDB")
	}
	defer func() {
		if closeErr := store.Close(context.Background()); closeErr != nil {
			logger.Error().Err(closeErr).Msg("failed to close Mongo connection")
		}
	}()

	if err := store.EnsureIndexes(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to ensure mongo indexes")
	}

	walletService := wallet.NewService(store, cfg.Casino.DefaultBalance, cfg.Discord.ProtectedUser, logger)
	casinoService := casino.NewService(walletService, utils.NewDeterministicRNG(cfg.Casino.RandomSeed), logger)
	presenceManager := discord.NewPresenceManager(logger)

	discordHandler := discord.NewHandler(walletService, casinoService, presenceManager, logger, cfg.Discord.OwnerName)
	discordSession, err := discord.NewSession(cfg.Discord.Token)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create discord session")
	}
	discordSession.AddHandler(discordHandler.HandleInteraction)

	if err := discordSession.Open(); err != nil {
		logger.Fatal().Err(err).Msg("failed to open discord connection")
	}
	defer discordSession.Close()
	if err := discordSession.UpdateGameStatus(0, "🐀 Running Don Rat Casino"); err != nil {
		logger.Warn().Err(err).Msg("failed to set startup presence")
	}
	go presenceManager.Start(ctx, discordSession)

	appID := discordSession.State.User.ID
	if _, err := discord.RegisterCommands(ctx, discordSession, appID, cfg.Discord.GuildID, logger); err != nil {
		logger.Fatal().Err(err).Msg("failed to register slash commands")
	}

	logger.Info().Str("bot_user", discordSession.State.User.Username).Msg("Don Rat is online and listening for slash commands")

	<-ctx.Done()
	logger.Info().Msg("shutdown signal received")
}
