# CLI

Command-line admin tool for managing the Dotachi platform.

## Building

```bash
go build -o dotachi-cli .
```

## Usage

```bash
# Login (saves token to ~/.dotachi/token)
dotachi-cli login --phone 09000000000 --password admin123

# Node management
dotachi-cli nodes list
dotachi-cli nodes add --name node-1 --host 1.2.3.4 --port 7443 --secret mysecret
dotachi-cli nodes ping --id 1

# Room management
dotachi-cli rooms list [--node 1] [--search "name"]
dotachi-cli rooms create --name "Room 1" --node 1 --max-players 10 [--private --password abc] [--game dota2]
dotachi-cli rooms assign --room 1 --user 5

# User management
dotachi-cli users list [--search "phone or name"]

# Shard management
dotachi-cli shards add --user 5 --amount 50000 --description "Payment received"
dotachi-cli shards remove --user 5 --amount 10000 --description "Refund"

# Pricing info
dotachi-cli pricing

# System overview
dotachi-cli status
```

## Configuration

- `~/.dotachi/config` - API base URL
- `~/.dotachi/token` - JWT token (saved after login)
- `DOTACHI_API` env var - overrides config file
