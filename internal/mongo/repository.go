package mongo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"donrat/internal/models"
)

var (
	// ErrInsufficientFunds is returned when a wallet lacks credits.
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// TransferResult contains balances after a transfer.
type TransferResult struct {
	FromBefore int64
	FromAfter  int64
	ToBefore   int64
	ToAfter    int64
}

// HouseAnalytics summarizes overall casino performance.
type HouseAnalytics struct {
	TotalTransactions int64
	UniqueRats        int64
	TotalWagered      int64
	TotalPayouts      int64
	HouseNet          int64
	GameBreakdown     map[string]int64
	TopGame           string
}

// EnsureWallet returns an existing wallet or creates a default one.
func (s *Store) EnsureWallet(ctx context.Context, user models.UserRef, defaultBalance int64, isProtected bool) (*models.Wallet, error) {
	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"username":     user.Username,
			"is_protected": isProtected,
			"updated_at":   now,
		},
		"$setOnInsert": bson.M{
			"user_id":        user.ID,
			"balance":        defaultBalance,
			"last_operation": "wallet_initialized",
			"created_at":     now,
		},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	res := s.Wallets.FindOneAndUpdate(ctx, bson.M{"user_id": user.ID}, update, opts)
	if res.Err() != nil && !errors.Is(res.Err(), mongo.ErrNoDocuments) {
		return nil, fmt.Errorf("ensure wallet: %w", res.Err())
	}

	var wallet models.Wallet
	if err := res.Decode(&wallet); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Create a new wallet explicitly if none exists
			wallet = models.Wallet{
				UserID:        user.ID,
				Username:      user.Username,
				Balance:       defaultBalance,
				IsProtected:   isProtected,
				LastOperation: "wallet_initialized",
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			if _, err := s.Wallets.InsertOne(ctx, wallet); err != nil {
				return nil, fmt.Errorf("insert wallet: %w", err)
			}
			return &wallet, nil
		}
		return nil, fmt.Errorf("decode wallet: %w", err)
	}

	return &wallet, nil
}

// GetWallet retrieves a wallet by Discord user ID.
func (s *Store) GetWallet(ctx context.Context, userID string) (*models.Wallet, error) {
	var wallet models.Wallet
	err := s.Wallets.FindOne(ctx, bson.M{"user_id": userID}).Decode(&wallet)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("wallet not found")
		}
		return nil, fmt.Errorf("get wallet: %w", err)
	}
	return &wallet, nil
}

// Credit adds credits to a wallet and returns before/after balances.
func (s *Store) Credit(ctx context.Context, userID string, amount int64, operation string) (int64, int64, error) {
	before, err := s.GetWallet(ctx, userID)
	if err != nil {
		return 0, 0, err
	}

	res, err := s.Wallets.UpdateOne(ctx, bson.M{"user_id": userID}, bson.M{
		"$inc": bson.M{"balance": amount},
		"$set": bson.M{"updated_at": time.Now().UTC(), "last_operation": operation},
	})
	if err != nil {
		return 0, 0, fmt.Errorf("credit wallet: %w", err)
	}
	if res.MatchedCount == 0 {
		return 0, 0, fmt.Errorf("wallet not found")
	}

	after, err := s.GetWallet(ctx, userID)
	if err != nil {
		return 0, 0, err
	}

	return before.Balance, after.Balance, nil
}

// Debit subtracts credits from a wallet atomically.
func (s *Store) Debit(ctx context.Context, userID string, amount int64, operation string) (int64, int64, error) {
	before, err := s.GetWallet(ctx, userID)
	if err != nil {
		return 0, 0, err
	}

	filter := bson.M{"user_id": userID, "balance": bson.M{"$gte": amount}}
	update := bson.M{
		"$inc": bson.M{"balance": -amount},
		"$set": bson.M{"updated_at": time.Now().UTC(), "last_operation": operation},
	}
	result, err := s.Wallets.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, 0, fmt.Errorf("debit wallet: %w", err)
	}
	if result.MatchedCount == 0 {
		return before.Balance, before.Balance, ErrInsufficientFunds
	}

	after, err := s.GetWallet(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	return before.Balance, after.Balance, nil
}

// InsertTransaction writes an immutable transaction event.
func (s *Store) InsertTransaction(ctx context.Context, tx models.Transaction) error {
	tx.CreatedAt = time.Now().UTC()
	if _, err := s.Transactions.InsertOne(ctx, tx); err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}
	return nil
}

// Transfer moves credits from one wallet to another.
func (s *Store) Transfer(ctx context.Context, from models.UserRef, to models.UserRef, amount int64, protectedSender bool) (TransferResult, error) {
	result := TransferResult{}
	session, err := s.Client.StartSession()
	if err != nil {
		return result, fmt.Errorf("start mongo session: %w", err)
	}
	defer session.EndSession(ctx)

	callback := func(sc mongo.SessionContext) (interface{}, error) {
		var fromWallet models.Wallet
		if err := s.Wallets.FindOne(sc, bson.M{"user_id": from.ID}).Decode(&fromWallet); err != nil {
			return nil, fmt.Errorf("find sender wallet: %w", err)
		}
		var toWallet models.Wallet
		if err := s.Wallets.FindOne(sc, bson.M{"user_id": to.ID}).Decode(&toWallet); err != nil {
			return nil, fmt.Errorf("find receiver wallet: %w", err)
		}

		now := time.Now().UTC()
		result.FromBefore = fromWallet.Balance
		result.ToBefore = toWallet.Balance

		if !protectedSender {
			if fromWallet.Balance < amount {
				return nil, ErrInsufficientFunds
			}
			if _, err := s.Wallets.UpdateOne(sc, bson.M{"user_id": from.ID}, bson.M{
				"$inc": bson.M{"balance": -amount},
				"$set": bson.M{"updated_at": now, "last_operation": "trade_sent"},
			}); err != nil {
				return nil, fmt.Errorf("debit sender wallet: %w", err)
			}
			result.FromAfter = fromWallet.Balance - amount
		} else {
			result.FromAfter = fromWallet.Balance
		}

		if _, err := s.Wallets.UpdateOne(sc, bson.M{"user_id": to.ID}, bson.M{
			"$inc": bson.M{"balance": amount},
			"$set": bson.M{"updated_at": now, "last_operation": "trade_received"},
		}); err != nil {
			return nil, fmt.Errorf("credit receiver wallet: %w", err)
		}
		result.ToAfter = toWallet.Balance + amount

		senderType := models.TransactionTypeTradeOut
		senderAmount := amount
		if protectedSender {
			senderType = models.TransactionTypeProtectedSkip
			senderAmount = 0
		}

		txDocs := []interface{}{
			models.Transaction{
				UserID:      from.ID,
				Username:    from.Username,
				Type:        senderType,
				Amount:      senderAmount,
				BalanceFrom: result.FromBefore,
				BalanceTo:   result.FromAfter,
				Meta: map[string]string{
					"reason":       "trade",
					"counterparty": to.ID,
				},
				CreatedAt: now,
			},
			models.Transaction{
				UserID:      to.ID,
				Username:    to.Username,
				Type:        models.TransactionTypeTradeIn,
				Amount:      amount,
				BalanceFrom: result.ToBefore,
				BalanceTo:   result.ToAfter,
				Meta: map[string]string{
					"reason":       "trade",
					"counterparty": from.ID,
				},
				CreatedAt: now,
			},
		}
		if _, err := s.Transactions.InsertMany(sc, txDocs); err != nil {
			return nil, fmt.Errorf("insert transfer transactions: %w", err)
		}

		return nil, nil
	}

	_, err = session.WithTransaction(ctx, callback)
	if err != nil {
		if errors.Is(err, ErrInsufficientFunds) {
			return result, ErrInsufficientFunds
		}
		if strings.Contains(strings.ToLower(err.Error()), "transaction") {
			s.logger.Warn().Err(err).Msg("transfer transaction failed; check replica set configuration")
		}
		return result, fmt.Errorf("transfer credits: %w", err)
	}

	return result, nil
}

// ListTopWallets returns wallets ordered by highest balance.
func (s *Store) ListTopWallets(ctx context.Context, limit int64) ([]models.Wallet, error) {
	if limit <= 0 {
		limit = 10
	}
	opts := options.Find().SetSort(bson.D{{Key: "balance", Value: -1}, {Key: "updated_at", Value: 1}}).SetLimit(limit)
	cursor, err := s.Wallets.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("list top wallets: %w", err)
	}
	defer cursor.Close(ctx)

	var wallets []models.Wallet
	if err := cursor.All(ctx, &wallets); err != nil {
		return nil, fmt.Errorf("decode top wallets: %w", err)
	}

	return wallets, nil
}

// RecentTransactions returns recent wallet transactions for a user.
func (s *Store) RecentTransactions(ctx context.Context, userID string, limit int64) ([]models.Transaction, error) {
	if limit <= 0 {
		limit = 10
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit)
	cursor, err := s.Transactions.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, fmt.Errorf("find recent transactions: %w", err)
	}
	defer cursor.Close(ctx)

	var txs []models.Transaction
	if err := cursor.All(ctx, &txs); err != nil {
		return nil, fmt.Errorf("decode transactions: %w", err)
	}

	return txs, nil
}

// LastTransactionByType fetches latest transaction of a given type for a user.
func (s *Store) LastTransactionByType(ctx context.Context, userID string, txType string) (*models.Transaction, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})
	var tx models.Transaction
	err := s.Transactions.FindOne(ctx, bson.M{"user_id": userID, "type": txType}, opts).Decode(&tx)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find last transaction by type: %w", err)
	}
	return &tx, nil
}

// GetHouseAnalytics computes global casino metrics from transaction data.
func (s *Store) GetHouseAnalytics(ctx context.Context) (*HouseAnalytics, error) {
	analytics := &HouseAnalytics{GameBreakdown: map[string]int64{}}

	totalTransactions, err := s.Transactions.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("count transactions: %w", err)
	}
	analytics.TotalTransactions = totalTransactions

	uniqueRats, err := s.Wallets.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("count wallets: %w", err)
	}
	analytics.UniqueRats = uniqueRats

	wagered, err := s.sumTransactionAmounts(ctx, models.TransactionTypeBetLoss)
	if err != nil {
		return nil, err
	}
	analytics.TotalWagered = wagered

	payouts, err := s.sumTransactionAmounts(ctx, models.TransactionTypeBetWin)
	if err != nil {
		return nil, err
	}
	analytics.TotalPayouts = payouts
	analytics.HouseNet = wagered - payouts

	breakdownPipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"type": bson.M{"$in": []string{models.TransactionTypeBetWin, models.TransactionTypeBetLoss}}}}},
		bson.D{{Key: "$group", Value: bson.M{"_id": "$meta.game", "count": bson.M{"$sum": 1}}}},
		bson.D{{Key: "$sort", Value: bson.M{"count": -1}}},
	}
	cursor, err := s.Transactions.Aggregate(ctx, breakdownPipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate game breakdown: %w", err)
	}
	defer cursor.Close(ctx)

	type gameRow struct {
		ID    string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	var rows []gameRow
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("decode game breakdown: %w", err)
	}

	for _, row := range rows {
		gameName := row.ID
		if strings.TrimSpace(gameName) == "" {
			gameName = "unknown"
		}
		analytics.GameBreakdown[gameName] = row.Count
		if analytics.TopGame == "" {
			analytics.TopGame = gameName
		}
	}

	if analytics.TopGame == "" {
		analytics.TopGame = "n/a"
	}

	return analytics, nil
}

func (s *Store) sumTransactionAmounts(ctx context.Context, txType string) (int64, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"type": txType}}},
		bson.D{{Key: "$group", Value: bson.M{"_id": nil, "sum": bson.M{"$sum": "$amount"}}}},
	}
	cursor, err := s.Transactions.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("aggregate sum for %s: %w", txType, err)
	}
	defer cursor.Close(ctx)

	type sumRow struct {
		Sum int64 `bson:"sum"`
	}
	var rows []sumRow
	if err := cursor.All(ctx, &rows); err != nil {
		return 0, fmt.Errorf("decode sum for %s: %w", txType, err)
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].Sum, nil
}
