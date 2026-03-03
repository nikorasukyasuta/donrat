package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	walletsCollection      = "wallets"
	transactionsCollection = "transactions"
)

// Store wraps MongoDB collections used by the bot.
type Store struct {
	Client       *mongo.Client
	DB           *mongo.Database
	Wallets      *mongo.Collection
	Transactions *mongo.Collection
	logger       zerolog.Logger
}

// Connect initializes a MongoDB client and returns a Store.
func Connect(ctx context.Context, uri string, database string, timeout time.Duration, logger zerolog.Logger) (*Store, error) {
	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	if err := client.Ping(connectCtx, nil); err != nil {
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	db := client.Database(database)
	store := &Store{
		Client:       client,
		DB:           db,
		Wallets:      db.Collection(walletsCollection),
		Transactions: db.Collection(transactionsCollection),
		logger:       logger,
	}

	return store, nil
}

// EnsureIndexes creates required indexes.
func (s *Store) EnsureIndexes(ctx context.Context) error {
	walletIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true).SetName("wallet_user_id_unique")},
	}
	if _, err := s.Wallets.Indexes().CreateMany(ctx, walletIndexes); err != nil {
		return fmt.Errorf("create wallet indexes: %w", err)
	}

	transactionIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}, Options: options.Index().SetName("transaction_user_time")},
		{Keys: bson.D{{Key: "type", Value: 1}, {Key: "created_at", Value: -1}}, Options: options.Index().SetName("transaction_type_time")},
	}
	if _, err := s.Transactions.Indexes().CreateMany(ctx, transactionIndexes); err != nil {
		return fmt.Errorf("create transaction indexes: %w", err)
	}

	return nil
}

// Close gracefully disconnects the MongoDB client.
func (s *Store) Close(ctx context.Context) error {
	if err := s.Client.Disconnect(ctx); err != nil {
		return fmt.Errorf("disconnect mongo: %w", err)
	}
	return nil
}
