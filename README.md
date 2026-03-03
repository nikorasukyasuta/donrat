# Don Rat Casino Bot (Go + Discord + MongoDB)

Production-ready Discord slash-command casino bot where every user is a rat, and Don Rat runs the house.

## Features

- Slash-command only interaction model.
- MongoDB-backed wallet ledger with transaction history.
- Deterministic game behavior using seeded RNG.
- Don Rat sarcastic persona responses.

## Project Structure

```text
.
├── cmd/
│   └── bot/
│       └── main.go
├── internal/
│   ├── casino/
│   │   ├── persona.go
│   │   └── service.go
│   ├── config/
│   │   └── config.go
│   ├── discord/
│   │   ├── commands.go
│   │   ├── handler.go
│   │   └── session.go
│   ├── models/
│   │   ├── user.go
│   │   └── wallet.go
│   ├── mongo/
│   │   ├── client.go
│   │   └── repository.go
│   ├── utils/
│   │   ├── logger.go
│   │   └── random.go
│   └── wallet/
│       └── service.go
├── .env.example
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── go.mod
```

## Slash Commands

- `/balance` – show your wallet balance.
- `/wallet` – create your rat wallet and join Don Rat's casino.
- `/bet coinflip <amount>` – bet credits on a coinflip.
- `/slots <amount>` – spin slots with weighted payout rules.
- `/roulette <amount> <color>` – bet on red, black, or green.
- `/dice <amount> <guess>` – bet on a dice guess from 1 to 6.
- `/blackjack <amount>` – play a quick blackjack hand.
- `/war <amount>` – draw a high card against Don Rat.
- `/poker <amount>` – five-card showdown against Don Rat.
- `/daily` – claim a daily credit stipend once per UTC day.
- `/leaderboard [limit]` – richest rats by wallet balance.
- `/history [limit]` – recent wallet transaction log.
- `/house` – casino-wide analytics (wagers, payouts, net, game volume).
- `/trade <user> <amount>` – transfer social credits.
- `/donrat` – get a Don Rat persona line.

All commands validate input, use embeds for responses, and log balance mutations in MongoDB.
Presence is updated dynamically to mimic active casino tables (coinflip, slots, roulette, dice, and trading).
When idle, the bot rotates game-themed rich presence statuses on a timer.
Game embeds now include a "Table Presence" field mirroring current rich presence context.

Additional casino operations:

- Daily rewards are logged as transactions and limited to one claim per UTC day.
- Leaderboard is computed from MongoDB wallet balances.
- History uses recent transaction entries from the wallet ledger.

## Don Rat Persona

Don Rat speaks as a sarcastic mafia boss to rats.

Example lines:

- “Don Rat sees you crawling back to the tables, little rat.”
- “Your wallet looks light. Don Rat approves.”
- “Even when you lose, you amuse Don Rat.”

Reusable persona generation lives in `internal/casino/persona.go`.
The Discord handlers now consume category-based persona responses (win/loss/trade/protected/general sarcasm).
Persona line selection is deterministic (hash-based weighted choice) for reproducible behavior.

## Environment Variables

Copy `.env.example` to `.env` and set values:

- `DISCORD_TOKEN` (required)
- `DISCORD_GUILD_ID` (optional; if empty, commands are global)
- `PROTECTED_USER` (default `hellomimiz`, supports ID or username)
- `DON_RAT_OWNER_NAME` (default `Don Rat`)
- `MONGO_URI` (default `mongodb://mongo:27017/?replicaSet=rs0`)
- `MONGO_DATABASE` (default `donrat`)
- `MONGO_CONNECT_TIMEOUT_SECONDS` (default `10`)
- `CASINO_DEFAULT_BALANCE` (default `1000`)
- `CASINO_RANDOM_SEED` (default `42`)
- `LOG_LEVEL` (default `info`)
- `ENVIRONMENT` (default `development`)

## Run Locally

```bash
cp .env.example .env
make tidy
make run
```

## Run With Docker

```bash
cp .env.example .env
docker compose up --build -d
```

This starts:

- `mongo` with replica set enabled (`rs0`)
- `mongo-init` for replica set initialization
- `bot` service

## Deploy to EC2 via ECR

Prerequisites:

- Local machine: `docker`, `aws` CLI, and SSH access to the EC2 instance.
- EC2 instance: `docker`, Docker Compose plugin (`docker compose`), and `aws` CLI.
- IAM permissions for ECR push/pull on both local and EC2 contexts.

1. Build and deploy in one command:

```bash
scripts/deploy_ec2_ecr.sh \
 <aws_account_id> <aws_region> <ecr_repo> <image_tag> \
 <ec2_user> <ec2_host> <ssh_key_path> <ec2_app_dir>
```

Example:

```bash
scripts/deploy_ec2_ecr.sh \
 123456789012 us-east-1 donrat-bot v1 \
 ubuntu ec2-12-34-56-78.compute-1.amazonaws.com ~/.ssh/my-key.pem /opt/donrat
```

What it does:

- Creates ECR repo if missing.
- Builds and pushes your bot image to ECR.
- Copies `docker-compose.ec2.yml` (as `docker-compose.yml`) and `.env` to EC2.
- Pulls the image and starts containers on EC2.

## Make Targets

- `make tidy` – resolve dependencies
- `make build` – build bot binary
- `make run` – run bot locally
- `make test` – run tests
- `make docker-up` – build/start containers
- `make docker-down` – stop containers and remove volumes
- `make docker-build ECR_REPO=donrat-bot IMAGE_TAG=v1` – build tagged image locally
- `make docker-tag ECR_REPO=donrat-bot IMAGE_TAG=v1 IMAGE_URI=<full-ecr-image-uri>` – tag image for push

## Startup Behavior

On startup, the bot:

1. Loads config via Viper.
2. Connects to MongoDB.
3. Ensures required indexes exist.
4. Registers slash commands.
5. Starts listening for interactions.

## License

MIT
