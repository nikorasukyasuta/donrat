package discord

import "github.com/bwmarrin/discordgo"

// SlashCommands defines the bot's global slash command contract.
var SlashCommands = []*discordgo.ApplicationCommand{
	{
		Name:        "balance",
		Description: "Show your rat wallet balance",
	},
	{
		Name:        "wallet",
		Description: "Create your rat wallet and join Don Rat's casino",
	},
	{
		Name:        "bet",
		Description: "Place a casino bet",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "coinflip",
				Description: "Bet on a coin flip",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "amount",
						Description: "Amount of social credits to bet",
						Required:    true,
						MinValue:    ptrFloat(1),
					},
				},
			},
		},
	},
	{
		Name:        "slots",
		Description: "Spin the Don Rat slots",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Amount of social credits to bet",
				Required:    true,
				MinValue:    ptrFloat(1),
			},
		},
	},
	{
		Name:        "roulette",
		Description: "Bet on Don Rat roulette",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Amount of social credits to bet",
				Required:    true,
				MinValue:    ptrFloat(1),
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "color",
				Description: "Roulette color to bet on",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "red", Value: "red"},
					{Name: "black", Value: "black"},
					{Name: "green", Value: "green"},
				},
			},
		},
	},
	{
		Name:        "dice",
		Description: "Bet on a six-sided dice roll",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Amount of social credits to bet",
				Required:    true,
				MinValue:    ptrFloat(1),
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "guess",
				Description: "Your dice guess from 1 to 6",
				Required:    true,
				MinValue:    ptrFloat(1),
				MaxValue:    6,
			},
		},
	},
	{
		Name:        "blackjack",
		Description: "Play a quick hand of Don Rat blackjack",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Amount of social credits to bet",
				Required:    true,
				MinValue:    ptrFloat(1),
			},
		},
	},
	{
		Name:        "war",
		Description: "Draw a card against Don Rat",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Amount of social credits to bet",
				Required:    true,
				MinValue:    ptrFloat(1),
			},
		},
	},
	{
		Name:        "poker",
		Description: "Play five-card showdown poker",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Amount of social credits to bet",
				Required:    true,
				MinValue:    ptrFloat(1),
			},
		},
	},
	{
		Name:        "leaderboard",
		Description: "Show richest rats in the casino",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "limit",
				Description: "How many rats to list (default 10)",
				Required:    false,
				MinValue:    ptrFloat(1),
				MaxValue:    25,
			},
		},
	},
	{
		Name:        "daily",
		Description: "Claim your daily social credit stipend",
	},
	{
		Name:        "history",
		Description: "Show your recent wallet transactions",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "limit",
				Description: "How many recent entries (default 5)",
				Required:    false,
				MinValue:    ptrFloat(1),
				MaxValue:    15,
			},
		},
	},
	{
		Name:        "house",
		Description: "Show Don Rat casino house analytics",
	},
	{
		Name:        "trade",
		Description: "Trade social credits with another rat",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "Rat to receive credits",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Amount of social credits to trade",
				Required:    true,
				MinValue:    ptrFloat(1),
			},
		},
	},
	{
		Name:        "donrat",
		Description: "Receive wisdom from Don Rat",
	},
}

func ptrFloat(value float64) *float64 {
	return &value
}
