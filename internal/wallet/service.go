package wallet

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"donrat/internal/models"
	"donrat/internal/mongo"
)

// Service defines wallet use-cases.
type Service interface {
	EnsureWallet(ctx context.Context, user models.UserRef) (*models.Wallet, error)
	GetBalance(ctx context.Context, user models.UserRef) (*models.Wallet, error)
	AddCredits(ctx context.Context, user models.UserRef, amount int64, txType string, meta map[string]string) (*models.Wallet, error)
	SubtractCredits(ctx context.Context, user models.UserRef, amount int64, txType string, meta map[string]string) (*models.Wallet, error)
	TransferCredits(ctx context.Context, from models.UserRef, to models.UserRef, amount int64) (*mongo.TransferResult, error)
	ClaimDaily(ctx context.Context, user models.UserRef, amount int64) (*models.Wallet, bool, time.Time, error)
	TopWallets(ctx context.Context, limit int64) ([]models.Wallet, error)
	RecentTransactions(ctx context.Context, user models.UserRef, limit int64) ([]models.Transaction, error)
	HouseAnalytics(ctx context.Context) (*mongo.HouseAnalytics, error)
	IsProtected(user models.UserRef) bool
}

type service struct {
	store          *mongo.Store
	defaultBalance int64
	protectedUser  string
	logger         zerolog.Logger
}

// NewService constructs a wallet service.
func NewService(store *mongo.Store, defaultBalance int64, protectedUser string, logger zerolog.Logger) Service {
	return &service{
		store:          store,
		defaultBalance: defaultBalance,
		protectedUser:  protectedUser,
		logger:         logger,
	}
}

// IsProtected returns whether the user matches protected username or ID.
func (s *service) IsProtected(user models.UserRef) bool {
	target := strings.TrimSpace(strings.ToLower(s.protectedUser))
	if target == "" {
		return false
	}
	return strings.ToLower(user.ID) == target || strings.ToLower(user.Username) == target
}

// EnsureWallet ensures a rat wallet exists.
func (s *service) EnsureWallet(ctx context.Context, user models.UserRef) (*models.Wallet, error) {
	wallet, err := s.store.EnsureWallet(ctx, user, s.defaultBalance, s.IsProtected(user))
	if err != nil {
		return nil, fmt.Errorf("ensure wallet for %s: %w", user.ID, err)
	}
	return wallet, nil
}

// GetBalance returns the current wallet state.
func (s *service) GetBalance(ctx context.Context, user models.UserRef) (*models.Wallet, error) {
	if _, err := s.EnsureWallet(ctx, user); err != nil {
		return nil, err
	}
	wallet, err := s.store.GetWallet(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get wallet balance for %s: %w", user.ID, err)
	}
	return wallet, nil
}

// AddCredits adds credits and logs a transaction.
func (s *service) AddCredits(ctx context.Context, user models.UserRef, amount int64, txType string, meta map[string]string) (*models.Wallet, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	if _, err := s.EnsureWallet(ctx, user); err != nil {
		return nil, err
	}

	before, after, err := s.store.Credit(ctx, user.ID, amount, txType)
	if err != nil {
		return nil, err
	}

	err = s.store.InsertTransaction(ctx, models.Transaction{
		UserID:      user.ID,
		Username:    user.Username,
		Type:        txType,
		Amount:      amount,
		BalanceFrom: before,
		BalanceTo:   after,
		Meta:        withDefaultMeta(meta),
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	wallet, err := s.store.GetWallet(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return wallet, nil
}

// SubtractCredits subtracts credits and logs a transaction; protected rats never lose credits.
func (s *service) SubtractCredits(ctx context.Context, user models.UserRef, amount int64, txType string, meta map[string]string) (*models.Wallet, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	if _, err := s.EnsureWallet(ctx, user); err != nil {
		return nil, err
	}

	if s.IsProtected(user) {
		wallet, err := s.store.GetWallet(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		err = s.store.InsertTransaction(ctx, models.Transaction{
			UserID:      user.ID,
			Username:    user.Username,
			Type:        models.TransactionTypeProtectedSkip,
			Amount:      0,
			BalanceFrom: wallet.Balance,
			BalanceTo:   wallet.Balance,
			Meta:        withDefaultMeta(meta),
			CreatedAt:   time.Now().UTC(),
		})
		if err != nil {
			return nil, err
		}
		return wallet, nil
	}

	before, after, err := s.store.Debit(ctx, user.ID, amount, txType)
	if err != nil {
		if errors.Is(err, mongo.ErrInsufficientFunds) {
			return nil, err
		}
		return nil, err
	}

	err = s.store.InsertTransaction(ctx, models.Transaction{
		UserID:      user.ID,
		Username:    user.Username,
		Type:        txType,
		Amount:      amount,
		BalanceFrom: before,
		BalanceTo:   after,
		Meta:        withDefaultMeta(meta),
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	wallet, err := s.store.GetWallet(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return wallet, nil
}

// TransferCredits moves credits and transaction-logs both sides.
func (s *service) TransferCredits(ctx context.Context, from models.UserRef, to models.UserRef, amount int64) (*mongo.TransferResult, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	if from.ID == to.ID {
		return nil, fmt.Errorf("cannot trade with yourself")
	}
	if _, err := s.EnsureWallet(ctx, from); err != nil {
		return nil, err
	}
	if _, err := s.EnsureWallet(ctx, to); err != nil {
		return nil, err
	}

	protectedSender := s.IsProtected(from)
	transferResult, err := s.store.Transfer(ctx, from, to, amount, protectedSender)
	if err != nil {
		if errors.Is(err, mongo.ErrInsufficientFunds) {
			return nil, err
		}
		return nil, err
	}

	s.logger.Info().
		Str("from_user_id", from.ID).
		Str("to_user_id", to.ID).
		Int64("amount", amount).
		Bool("protected_sender", protectedSender).
		Msg("wallet transfer completed")

	return &transferResult, nil
}

// ClaimDaily grants once-per-day credits if available.
func (s *service) ClaimDaily(ctx context.Context, user models.UserRef, amount int64) (*models.Wallet, bool, time.Time, error) {
	if amount <= 0 {
		return nil, false, time.Time{}, fmt.Errorf("daily amount must be greater than zero")
	}
	if _, err := s.EnsureWallet(ctx, user); err != nil {
		return nil, false, time.Time{}, err
	}

	lastDaily, err := s.store.LastTransactionByType(ctx, user.ID, models.TransactionTypeDailyBonus)
	if err != nil {
		return nil, false, time.Time{}, err
	}

	now := time.Now().UTC()
	if lastDaily != nil && sameDayUTC(lastDaily.CreatedAt, now) {
		walletState, getErr := s.store.GetWallet(ctx, user.ID)
		if getErr != nil {
			return nil, false, time.Time{}, getErr
		}
		next := nextUTCMidnight(now)
		return walletState, false, next, nil
	}

	walletState, err := s.AddCredits(ctx, user, amount, models.TransactionTypeDailyBonus, map[string]string{"reason": "daily_claim"})
	if err != nil {
		return nil, false, time.Time{}, err
	}

	return walletState, true, nextUTCMidnight(now), nil
}

// TopWallets returns richest rats ordered by credits.
func (s *service) TopWallets(ctx context.Context, limit int64) ([]models.Wallet, error) {
	return s.store.ListTopWallets(ctx, limit)
}

// RecentTransactions returns recent wallet events for a user.
func (s *service) RecentTransactions(ctx context.Context, user models.UserRef, limit int64) ([]models.Transaction, error) {
	if _, err := s.EnsureWallet(ctx, user); err != nil {
		return nil, err
	}
	return s.store.RecentTransactions(ctx, user.ID, limit)
}

// HouseAnalytics returns house-wide casino metrics.
func (s *service) HouseAnalytics(ctx context.Context) (*mongo.HouseAnalytics, error) {
	return s.store.GetHouseAnalytics(ctx)
}

func sameDayUTC(a time.Time, b time.Time) bool {
	a = a.UTC()
	b = b.UTC()
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func nextUTCMidnight(now time.Time) time.Time {
	now = now.UTC()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
}

func withDefaultMeta(meta map[string]string) map[string]string {
	if meta == nil {
		return map[string]string{}
	}
	return meta
}
