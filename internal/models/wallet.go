package models

import "time"

// Wallet stores a rat's social credit balance.
type Wallet struct {
	ID            string    `bson:"_id,omitempty" json:"id"`
	UserID        string    `bson:"user_id" json:"user_id"`
	Username      string    `bson:"username" json:"username"`
	Balance       int64     `bson:"balance" json:"balance"`
	IsProtected   bool      `bson:"is_protected" json:"is_protected"`
	LastOperation string    `bson:"last_operation" json:"last_operation"`
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at" json:"updated_at"`
}

const (
	// TransactionTypeAward represents credit gains.
	TransactionTypeAward = "award"
	// TransactionTypeBetWin represents a successful bet result.
	TransactionTypeBetWin = "bet_win"
	// TransactionTypeBetLoss represents a losing bet result.
	TransactionTypeBetLoss = "bet_loss"
	// TransactionTypeTradeIn represents incoming transfer.
	TransactionTypeTradeIn = "trade_in"
	// TransactionTypeTradeOut represents outgoing transfer.
	TransactionTypeTradeOut = "trade_out"
	// TransactionTypeProtectedSkip represents attempted deduction skipped by protection rule.
	TransactionTypeProtectedSkip = "protected_skip"
	// TransactionTypeDailyBonus represents daily credit grants.
	TransactionTypeDailyBonus = "daily_bonus"
)

// Transaction stores a wallet balance mutation audit record.
type Transaction struct {
	ID          string            `bson:"_id,omitempty" json:"id"`
	UserID      string            `bson:"user_id" json:"user_id"`
	Username    string            `bson:"username" json:"username"`
	Type        string            `bson:"type" json:"type"`
	Amount      int64             `bson:"amount" json:"amount"`
	BalanceFrom int64             `bson:"balance_from" json:"balance_from"`
	BalanceTo   int64             `bson:"balance_to" json:"balance_to"`
	Meta        map[string]string `bson:"meta,omitempty" json:"meta,omitempty"`
	CreatedAt   time.Time         `bson:"created_at" json:"created_at"`
}
