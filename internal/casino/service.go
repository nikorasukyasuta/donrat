package casino

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"donrat/internal/models"
	"donrat/internal/mongo"
	"donrat/internal/utils"
	"donrat/internal/wallet"
)

// Service defines casino game behavior.
type Service interface {
	Coinflip(ctx context.Context, user models.UserRef, amount int64) (*Result, error)
	Slots(ctx context.Context, user models.UserRef, amount int64) (*Result, error)
	Roulette(ctx context.Context, user models.UserRef, amount int64, color string) (*Result, error)
	Dice(ctx context.Context, user models.UserRef, amount int64, guess int64) (*Result, error)
	Blackjack(ctx context.Context, user models.UserRef, amount int64) (*Result, error)
	War(ctx context.Context, user models.UserRef, amount int64) (*Result, error)
	Poker(ctx context.Context, user models.UserRef, amount int64) (*Result, error)
}

// Result captures game output.
type Result struct {
	Game          string
	Won           bool
	Amount        int64
	NetDelta      int64
	BalanceBefore int64
	BalanceAfter  int64
	Details       string
}

type service struct {
	wallet wallet.Service
	rng    utils.RNG
	logger zerolog.Logger
}

// NewService constructs a casino game service.
func NewService(walletService wallet.Service, rng utils.RNG, logger zerolog.Logger) Service {
	return &service{wallet: walletService, rng: rng, logger: logger}
}

// Coinflip resolves /bet coinflip <amount>.
func (s *service) Coinflip(ctx context.Context, user models.UserRef, amount int64) (*Result, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	beforeWallet, err := s.wallet.GetBalance(ctx, user)
	if err != nil {
		return nil, err
	}

	won := s.rng.CoinFlip()
	if won {
		afterWallet, err := s.wallet.AddCredits(ctx, user, amount, models.TransactionTypeBetWin, map[string]string{"game": "coinflip"})
		if err != nil {
			return nil, err
		}
		result := &Result{
			Game:          "coinflip",
			Won:           true,
			Amount:        amount,
			NetDelta:      amount,
			BalanceBefore: beforeWallet.Balance,
			BalanceAfter:  afterWallet.Balance,
			Details:       "Heads. Don Rat lets you keep the crumbs.",
		}
		s.logger.Info().Str("game", "coinflip").Str("user_id", user.ID).Int64("amount", amount).Bool("won", true).Msg("game resolved")
		return result, nil
	}

	afterWallet, err := s.wallet.SubtractCredits(ctx, user, amount, models.TransactionTypeBetLoss, map[string]string{"game": "coinflip"})
	if err != nil {
		if err == mongo.ErrInsufficientFunds {
			return nil, fmt.Errorf("not enough credits for this bet")
		}
		return nil, err
	}

	result := &Result{
		Game:          "coinflip",
		Won:           false,
		Amount:        amount,
		NetDelta:      afterWallet.Balance - beforeWallet.Balance,
		BalanceBefore: beforeWallet.Balance,
		BalanceAfter:  afterWallet.Balance,
		Details:       "Tails. Don Rat chuckles at your misery.",
	}
	s.logger.Info().Str("game", "coinflip").Str("user_id", user.ID).Int64("amount", amount).Bool("won", false).Msg("game resolved")
	return result, nil
}

// Slots resolves /slots <amount>.
func (s *service) Slots(ctx context.Context, user models.UserRef, amount int64) (*Result, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	beforeWallet, err := s.wallet.GetBalance(ctx, user)
	if err != nil {
		return nil, err
	}

	symbols := []string{"🧀", "🍀", "💀", "💎", "🪙"}
	a := symbols[s.rng.Intn(len(symbols))]
	b := symbols[s.rng.Intn(len(symbols))]
	c := symbols[s.rng.Intn(len(symbols))]

	multiplier := int64(0)
	switch {
	case a == b && b == c && a == "💎":
		multiplier = 5
	case a == b && b == c:
		multiplier = 3
	case a == b || b == c || a == c:
		multiplier = 1
	default:
		multiplier = 0
	}

	if multiplier > 0 {
		winAmount := amount * multiplier
		afterWallet, err := s.wallet.AddCredits(ctx, user, winAmount, models.TransactionTypeBetWin, map[string]string{"game": "slots", "roll": a + b + c})
		if err != nil {
			return nil, err
		}
		return &Result{
			Game:          "slots",
			Won:           true,
			Amount:        amount,
			NetDelta:      winAmount,
			BalanceBefore: beforeWallet.Balance,
			BalanceAfter:  afterWallet.Balance,
			Details:       fmt.Sprintf("%s %s %s — Don Rat squints and pays you %d credits.", a, b, c, winAmount),
		}, nil
	}

	afterWallet, err := s.wallet.SubtractCredits(ctx, user, amount, models.TransactionTypeBetLoss, map[string]string{"game": "slots", "roll": a + b + c})
	if err != nil {
		if err == mongo.ErrInsufficientFunds {
			return nil, fmt.Errorf("not enough credits for this bet")
		}
		return nil, err
	}
	return &Result{
		Game:          "slots",
		Won:           false,
		Amount:        amount,
		NetDelta:      afterWallet.Balance - beforeWallet.Balance,
		BalanceBefore: beforeWallet.Balance,
		BalanceAfter:  afterWallet.Balance,
		Details:       fmt.Sprintf("%s %s %s — Don Rat keeps the house edge.", a, b, c),
	}, nil
}

// Roulette resolves /roulette <amount> <color>.
func (s *service) Roulette(ctx context.Context, user models.UserRef, amount int64, color string) (*Result, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	if color != "red" && color != "black" && color != "green" {
		return nil, fmt.Errorf("color must be red, black, or green")
	}

	beforeWallet, err := s.wallet.GetBalance(ctx, user)
	if err != nil {
		return nil, err
	}

	number := s.rng.Intn(37)
	outcome := rouletteColor(number)
	won := color == outcome

	if won {
		multiplier := int64(1)
		if color == "green" {
			multiplier = 14
		}
		winAmount := amount * multiplier
		afterWallet, err := s.wallet.AddCredits(ctx, user, winAmount, models.TransactionTypeBetWin, map[string]string{
			"game":    "roulette",
			"guess":   color,
			"outcome": outcome,
			"number":  fmt.Sprintf("%d", number),
		})
		if err != nil {
			return nil, err
		}
		return &Result{
			Game:          "roulette",
			Won:           true,
			Amount:        amount,
			NetDelta:      winAmount,
			BalanceBefore: beforeWallet.Balance,
			BalanceAfter:  afterWallet.Balance,
			Details:       fmt.Sprintf("Wheel lands on %d (%s). Don Rat pays %d credits.", number, outcome, winAmount),
		}, nil
	}

	afterWallet, err := s.wallet.SubtractCredits(ctx, user, amount, models.TransactionTypeBetLoss, map[string]string{
		"game":    "roulette",
		"guess":   color,
		"outcome": outcome,
		"number":  fmt.Sprintf("%d", number),
	})
	if err != nil {
		if err == mongo.ErrInsufficientFunds {
			return nil, fmt.Errorf("not enough credits for this bet")
		}
		return nil, err
	}
	return &Result{
		Game:          "roulette",
		Won:           false,
		Amount:        amount,
		NetDelta:      afterWallet.Balance - beforeWallet.Balance,
		BalanceBefore: beforeWallet.Balance,
		BalanceAfter:  afterWallet.Balance,
		Details:       fmt.Sprintf("Wheel lands on %d (%s). Don Rat pockets your bet.", number, outcome),
	}, nil
}

// Dice resolves /dice <amount> <guess>.
func (s *service) Dice(ctx context.Context, user models.UserRef, amount int64, guess int64) (*Result, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	if guess < 1 || guess > 6 {
		return nil, fmt.Errorf("guess must be between 1 and 6")
	}

	beforeWallet, err := s.wallet.GetBalance(ctx, user)
	if err != nil {
		return nil, err
	}

	roll := int64(s.rng.Intn(6) + 1)
	if roll == guess {
		winAmount := amount * 5
		afterWallet, err := s.wallet.AddCredits(ctx, user, winAmount, models.TransactionTypeBetWin, map[string]string{
			"game":  "dice",
			"guess": fmt.Sprintf("%d", guess),
			"roll":  fmt.Sprintf("%d", roll),
		})
		if err != nil {
			return nil, err
		}
		return &Result{
			Game:          "dice",
			Won:           true,
			Amount:        amount,
			NetDelta:      winAmount,
			BalanceBefore: beforeWallet.Balance,
			BalanceAfter:  afterWallet.Balance,
			Details:       fmt.Sprintf("Dice shows %d. You guessed right. Don Rat pays %d credits.", roll, winAmount),
		}, nil
	}

	afterWallet, err := s.wallet.SubtractCredits(ctx, user, amount, models.TransactionTypeBetLoss, map[string]string{
		"game":  "dice",
		"guess": fmt.Sprintf("%d", guess),
		"roll":  fmt.Sprintf("%d", roll),
	})
	if err != nil {
		if err == mongo.ErrInsufficientFunds {
			return nil, fmt.Errorf("not enough credits for this bet")
		}
		return nil, err
	}
	return &Result{
		Game:          "dice",
		Won:           false,
		Amount:        amount,
		NetDelta:      afterWallet.Balance - beforeWallet.Balance,
		BalanceBefore: beforeWallet.Balance,
		BalanceAfter:  afterWallet.Balance,
		Details:       fmt.Sprintf("Dice shows %d. Wrong guess. Don Rat keeps your credits.", roll),
	}, nil
}

func rouletteColor(number int) string {
	if number == 0 {
		return "green"
	}
	if number%2 == 0 {
		return "black"
	}
	return "red"
}

// Blackjack resolves /blackjack <amount>.
func (s *service) Blackjack(ctx context.Context, user models.UserRef, amount int64) (*Result, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	beforeWallet, err := s.wallet.GetBalance(ctx, user)
	if err != nil {
		return nil, err
	}

	playerA := drawCardValue(s.rng.Intn(13) + 1)
	playerB := drawCardValue(s.rng.Intn(13) + 1)
	dealerA := drawCardValue(s.rng.Intn(13) + 1)
	dealerB := drawCardValue(s.rng.Intn(13) + 1)

	player := blackjackScore([]int{playerA, playerB})
	dealer := blackjackScore([]int{dealerA, dealerB})

	if player > 21 {
		afterWallet, debitErr := s.wallet.SubtractCredits(ctx, user, amount, models.TransactionTypeBetLoss, map[string]string{"game": "blackjack", "player": fmt.Sprintf("%d", player), "dealer": fmt.Sprintf("%d", dealer)})
		if debitErr != nil {
			if debitErr == mongo.ErrInsufficientFunds {
				return nil, fmt.Errorf("not enough credits for this bet")
			}
			return nil, debitErr
		}
		return &Result{Game: "blackjack", Won: false, Amount: amount, NetDelta: afterWallet.Balance - beforeWallet.Balance, BalanceBefore: beforeWallet.Balance, BalanceAfter: afterWallet.Balance, Details: fmt.Sprintf("You bust at %d while Don Rat shows %d.", player, dealer)}, nil
	}

	if dealer > 21 || player > dealer {
		afterWallet, creditErr := s.wallet.AddCredits(ctx, user, amount, models.TransactionTypeBetWin, map[string]string{"game": "blackjack", "player": fmt.Sprintf("%d", player), "dealer": fmt.Sprintf("%d", dealer)})
		if creditErr != nil {
			return nil, creditErr
		}
		return &Result{Game: "blackjack", Won: true, Amount: amount, NetDelta: amount, BalanceBefore: beforeWallet.Balance, BalanceAfter: afterWallet.Balance, Details: fmt.Sprintf("Blackjack hand %d beats Don Rat's %d.", player, dealer)}, nil
	}

	if player == dealer {
		return &Result{Game: "blackjack", Won: false, Amount: amount, NetDelta: 0, BalanceBefore: beforeWallet.Balance, BalanceAfter: beforeWallet.Balance, Details: fmt.Sprintf("Push at %d. Don Rat calls it even.", player)}, nil
	}

	afterWallet, debitErr := s.wallet.SubtractCredits(ctx, user, amount, models.TransactionTypeBetLoss, map[string]string{"game": "blackjack", "player": fmt.Sprintf("%d", player), "dealer": fmt.Sprintf("%d", dealer)})
	if debitErr != nil {
		if debitErr == mongo.ErrInsufficientFunds {
			return nil, fmt.Errorf("not enough credits for this bet")
		}
		return nil, debitErr
	}
	return &Result{Game: "blackjack", Won: false, Amount: amount, NetDelta: afterWallet.Balance - beforeWallet.Balance, BalanceBefore: beforeWallet.Balance, BalanceAfter: afterWallet.Balance, Details: fmt.Sprintf("Don Rat holds %d over your %d.", dealer, player)}, nil
}

// War resolves /war <amount>.
func (s *service) War(ctx context.Context, user models.UserRef, amount int64) (*Result, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	beforeWallet, err := s.wallet.GetBalance(ctx, user)
	if err != nil {
		return nil, err
	}

	playerRank := s.rng.Intn(13) + 2
	dealerRank := s.rng.Intn(13) + 2

	if playerRank > dealerRank {
		afterWallet, creditErr := s.wallet.AddCredits(ctx, user, amount, models.TransactionTypeBetWin, map[string]string{"game": "war", "player": fmt.Sprintf("%d", playerRank), "dealer": fmt.Sprintf("%d", dealerRank)})
		if creditErr != nil {
			return nil, creditErr
		}
		return &Result{Game: "war", Won: true, Amount: amount, NetDelta: amount, BalanceBefore: beforeWallet.Balance, BalanceAfter: afterWallet.Balance, Details: fmt.Sprintf("Your card %s beats Don Rat's %s.", rankLabel(playerRank), rankLabel(dealerRank))}, nil
	}

	if playerRank == dealerRank {
		return &Result{Game: "war", Won: false, Amount: amount, NetDelta: 0, BalanceBefore: beforeWallet.Balance, BalanceAfter: beforeWallet.Balance, Details: fmt.Sprintf("War tie at %s. Don Rat calls it a stale hand.", rankLabel(playerRank))}, nil
	}

	afterWallet, debitErr := s.wallet.SubtractCredits(ctx, user, amount, models.TransactionTypeBetLoss, map[string]string{"game": "war", "player": fmt.Sprintf("%d", playerRank), "dealer": fmt.Sprintf("%d", dealerRank)})
	if debitErr != nil {
		if debitErr == mongo.ErrInsufficientFunds {
			return nil, fmt.Errorf("not enough credits for this bet")
		}
		return nil, debitErr
	}
	return &Result{Game: "war", Won: false, Amount: amount, NetDelta: afterWallet.Balance - beforeWallet.Balance, BalanceBefore: beforeWallet.Balance, BalanceAfter: afterWallet.Balance, Details: fmt.Sprintf("Don Rat's %s crushes your %s.", rankLabel(dealerRank), rankLabel(playerRank))}, nil
}

func drawCardValue(rank int) int {
	if rank == 1 {
		return 11
	}
	if rank >= 10 {
		return 10
	}
	return rank
}

func blackjackScore(cards []int) int {
	total := 0
	aces := 0
	for _, card := range cards {
		total += card
		if card == 11 {
			aces++
		}
	}
	for total > 21 && aces > 0 {
		total -= 10
		aces--
	}
	return total
}

func rankLabel(rank int) string {
	switch rank {
	case 11:
		return "J"
	case 12:
		return "Q"
	case 13:
		return "K"
	case 14:
		return "A"
	default:
		return fmt.Sprintf("%d", rank)
	}
}

// Poker resolves /poker <amount> with a 5-card showdown against Don Rat.
func (s *service) Poker(ctx context.Context, user models.UserRef, amount int64) (*Result, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	beforeWallet, err := s.wallet.GetBalance(ctx, user)
	if err != nil {
		return nil, err
	}

	player := drawPokerHand(s.rng)
	dealer := drawPokerHand(s.rng)
	playerScore, playerName := scorePokerHand(player)
	dealerScore, dealerName := scorePokerHand(dealer)

	if playerScore > dealerScore {
		winAmount := amount * 2
		afterWallet, creditErr := s.wallet.AddCredits(ctx, user, winAmount, models.TransactionTypeBetWin, map[string]string{
			"game":         "poker",
			"player_rank":  playerName,
			"dealer_rank":  dealerName,
			"player_score": fmt.Sprintf("%d", playerScore),
			"dealer_score": fmt.Sprintf("%d", dealerScore),
		})
		if creditErr != nil {
			return nil, creditErr
		}
		return &Result{Game: "poker", Won: true, Amount: amount, NetDelta: winAmount, BalanceBefore: beforeWallet.Balance, BalanceAfter: afterWallet.Balance, Details: fmt.Sprintf("Your %s beats Don Rat's %s.", playerName, dealerName)}, nil
	}

	if playerScore == dealerScore {
		return &Result{Game: "poker", Won: false, Amount: amount, NetDelta: 0, BalanceBefore: beforeWallet.Balance, BalanceAfter: beforeWallet.Balance, Details: fmt.Sprintf("Standoff: %s vs %s. Don Rat calls a draw.", playerName, dealerName)}, nil
	}

	afterWallet, debitErr := s.wallet.SubtractCredits(ctx, user, amount, models.TransactionTypeBetLoss, map[string]string{
		"game":         "poker",
		"player_rank":  playerName,
		"dealer_rank":  dealerName,
		"player_score": fmt.Sprintf("%d", playerScore),
		"dealer_score": fmt.Sprintf("%d", dealerScore),
	})
	if debitErr != nil {
		if debitErr == mongo.ErrInsufficientFunds {
			return nil, fmt.Errorf("not enough credits for this bet")
		}
		return nil, debitErr
	}
	return &Result{Game: "poker", Won: false, Amount: amount, NetDelta: afterWallet.Balance - beforeWallet.Balance, BalanceBefore: beforeWallet.Balance, BalanceAfter: afterWallet.Balance, Details: fmt.Sprintf("Don Rat's %s crushes your %s.", dealerName, playerName)}, nil
}

func drawPokerHand(rng utils.RNG) []int {
	hand := make([]int, 5)
	for i := range hand {
		hand[i] = rng.Intn(13) + 2
	}
	return hand
}

func scorePokerHand(hand []int) (int, string) {
	counts := map[int]int{}
	for _, card := range hand {
		counts[card]++
	}

	pairs := 0
	three := false
	four := false
	for _, count := range counts {
		switch count {
		case 4:
			four = true
		case 3:
			three = true
		case 2:
			pairs++
		}
	}

	if four {
		return 700, "Four of a Kind"
	}
	if three && pairs == 1 {
		return 600, "Full House"
	}
	if three {
		return 400, "Three of a Kind"
	}
	if pairs == 2 {
		return 300, "Two Pair"
	}
	if pairs == 1 {
		return 200, "Pair"
	}

	high := 0
	for _, card := range hand {
		if card > high {
			high = card
		}
	}
	return 100 + high, "High Card"
}
