package casino

import (
	"hash/fnv"
	"strings"
)

// Category represents a persona response category.
type Category string

const (
	GeneralSarcasm Category = "general_sarcasm"
	WinGrudging    Category = "win_grudging"
	LossMocking    Category = "loss_mocking"
	ProtectedUser  Category = "protected_user"
	Trade          Category = "trade"
	RecklessBet    Category = "reckless_bet"
	TinyBet        Category = "tiny_bet"
	Lore           Category = "lore"
)

// WeightedLine holds a line and its probability weight.
type WeightedLine struct {
	Line   string
	Weight int
}

// PersonaTable maps categories to weighted lines.
var PersonaTable = map[Category][]WeightedLine{
	GeneralSarcasm: {
		{"Don Rat sees you’ve returned. Hope wasn’t on the menu, but you brought it anyway.", 10},
		{"You walk in like a winner, but you smell like a rat who’s been losing all week.", 8},
		{"Don Rat admires your optimism. It’s the only valuable thing you have left.", 7},
		{"You again? Don Rat was enjoying the peace.", 6},
		{"Your confidence is adorable. Misguided, but adorable.", 6},
		{"Don Rat expected nothing from you, and you still managed to disappoint.", 5},
		{"You scurry in with dreams and leave with dust. Classic rat behavior.", 5},
		{"Don Rat appreciates your dedication to losing. Truly inspiring.", 4},
	},

	WinGrudging: {
		{"You won? Don Rat demands a recount.", 10},
		{"Beginner’s luck? Or did you bribe the universe?", 8},
		{"Don Rat is shocked. Physically. Emotionally. Existentially.", 7},
		{"Enjoy this moment. It won’t happen again.", 6},
		{"Even a blind rat finds cheese once in a while.", 5},
		{"Don Rat allows this victory. Don’t get comfortable.", 5},
		{"You won? Don Rat must be slipping.", 4},
	},

	LossMocking: {
		{"There it is. The natural order restored.", 10},
		{"Don Rat predicted this outcome before you even placed the bet.", 9},
		{"You lose with such consistency it’s almost a talent.", 8},
		{"Your credits vanish faster than your excuses.", 7},
		{"Don Rat thanks you for your generous donation.", 6},
		{"You gamble like a rat who’s allergic to winning.", 5},
		{"Another loss? Don Rat is starting to feel bad. Almost.", 4},
	},

	ProtectedUser: {
		{"Ah, the untouchable one arrives. Don Rat bows… reluctantly.", 10},
		{"Even when you lose, fate refuses to punish you. Annoying.", 8},
		{"Don Rat sees the universe still bends around you. Typical.", 7},
		{"You’re the only rat who can fall upward.", 6},
		{"Losses bounce off you like bullets off Don Rat’s finest suit.", 5},
		{"If privilege had a mascot, it’d be you, little rat.", 5},
		{"Don Rat doesn’t bother taking your credits. The cosmos won’t allow it.", 4},
	},

	Trade: {
		{"Transferring credits? Don Rat sees you’re redistributing your failures.", 10},
		{"Giving away your credits? Bold strategy. Stupid, but bold.", 8},
		{"Don Rat approves of this transaction. Mostly because it’s not his credits.", 7},
		{"You trade like a rat who’s never seen numbers before.", 6},
		{"Don Rat watches your financial decisions with great amusement.", 5},
	},

	RecklessBet: {
		{"That’s a big bet for a rat with such a tiny brain.", 10},
		{"Don Rat admires your recklessness. It keeps the lights on.", 8},
		{"You bet like you’re trying to impress someone. You’re not.", 7},
		{"This is either bravery or stupidity. Don Rat knows which.", 6},
		{"Your wallet is screaming. Don Rat is laughing.", 5},
	},

	TinyBet: {
		{"That’s your bet? Don Rat has seen bigger crumbs.", 10},
		{"You call that a wager? My pet rat bets more in his sleep.", 8},
		{"Don Rat expected more from you. Not much more, but more.", 7},
		{"A tiny bet for a tiny rat.", 6},
		{"Don Rat almost missed that. It was so small.", 5},
	},

	Lore: {
		{"Don Rat built this casino on the tears of rats like you.", 10},
		{"Every credit you lose goes straight into Don Rat’s retirement fund.", 8},
		{"Don Rat didn’t become king of the sewer by playing fair.", 7},
		{"This casino runs on luck, fear, and your poor decisions.", 6},
		{"Don Rat sees all. Especially your mistakes.", 5},
	},
}

// GetLine returns a deterministic weighted line for a category and identity key.
func GetLine(cat Category, identityKey string) string {
	lines := PersonaTable[cat]
	if len(lines) == 0 {
		return "Don Rat is speechless. A rare event."
	}

	totalWeight := 0
	for _, wl := range lines {
		totalWeight += wl.Weight
	}

	r := weightedHash(identityKey+":"+string(cat), totalWeight)
	acc := 0

	for _, wl := range lines {
		acc += wl.Weight
		if r < acc {
			return wl.Line
		}
	}

	return lines[len(lines)-1].Line
}

// PersonaResponse returns a persona-aware line based on context.
func PersonaResponse(userID string, cat Category) string {
	if strings.EqualFold(strings.TrimSpace(userID), "hellomimiz") {
		return GetLine(ProtectedUser, userID)
	}
	return GetLine(cat, userID)
}

func weightedHash(key string, max int) int {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(key))
	return int(hasher.Sum32() % uint32(max))
}
