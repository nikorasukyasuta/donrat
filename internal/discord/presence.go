package discord

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
)

// PresenceManager keeps bot status game-themed and rotates idle activities.
type PresenceManager struct {
	mu             sync.Mutex
	lastActiveAt   time.Time
	idleIndex      int
	idleActivities []string
	interval       time.Duration
	logger         zerolog.Logger
}

// NewPresenceManager creates a presence manager with rotating casino-themed activities.
func NewPresenceManager(logger zerolog.Logger) *PresenceManager {
	return &PresenceManager{
		idleActivities: []string{
			"🎰 Hosting Don Rat slots",
			"🎲 Taking coinflip bets",
			"🎡 Spinning roulette tables",
			"🎲 Rolling high-stakes dice",
			"🃏 Dealing blackjack hands",
			"🂡 Running rat card wars",
			"♠️ Hosting poker showdowns",
			"📊 Auditing house analytics",
			"💸 Watching rat trades",
		},
		interval: 45 * time.Second,
		logger:   logger,
	}
}

// DescribeActivity returns the game-themed status line for a game/user.
func (p *PresenceManager) DescribeActivity(game string, username string) string {
	return p.activityForGame(game, username)
}

// Start begins idle activity rotation until context cancellation.
func (p *PresenceManager) Start(ctx context.Context, session *discordgo.Session) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.rotateIdle(session); err != nil {
				p.logger.Warn().Err(err).Msg("failed to rotate idle presence")
			}
		}
	}
}

// MarkActive sets a game-specific activity and pauses idle rotation briefly.
func (p *PresenceManager) MarkActive(session *discordgo.Session, game string, username string) error {
	activity := p.activityForGame(game, username)
	if err := session.UpdateGameStatus(0, activity); err != nil {
		return fmt.Errorf("set active presence: %w", err)
	}

	p.mu.Lock()
	p.lastActiveAt = time.Now().UTC()
	p.mu.Unlock()
	return nil
}

func (p *PresenceManager) rotateIdle(session *discordgo.Session) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.lastActiveAt.IsZero() && time.Since(p.lastActiveAt) < p.interval {
		return nil
	}

	if len(p.idleActivities) == 0 {
		return nil
	}

	activity := p.idleActivities[p.idleIndex%len(p.idleActivities)]
	p.idleIndex++
	if err := session.UpdateGameStatus(0, activity); err != nil {
		return fmt.Errorf("set idle presence: %w", err)
	}

	return nil
}

func (p *PresenceManager) activityForGame(game string, username string) string {
	switch game {
	case "coinflip":
		return "🎲 Coinflip with " + username
	case "slots":
		return "🎰 Slots with " + username
	case "roulette":
		return "🎡 Roulette with " + username
	case "dice":
		return "🎲 Dice table with " + username
	case "trade":
		return "💸 Rat trades in progress"
	case "blackjack":
		return "🃏 Blackjack with " + username
	case "war":
		return "🂡 Card war with " + username
	case "poker":
		return "♠️ Poker with " + username
	case "leaderboard":
		return "🏆 Counting richest rats"
	case "daily":
		return "🪙 Handing daily stipends"
	case "history":
		return "📜 Reading rat ledgers"
	case "house":
		return "📊 Auditing house ledgers"
	case "wallet":
		return "💳 New rat joining casino"
	case "donrat":
		return "👑 Don Rat gives orders"
	default:
		return "🐀 Running Don Rat Casino"
	}
}
