# MMoney

Personal finance management — synced from Monarch Money.

Part of the [Localitas](https://github.com/localitas) platform — a self-hosted, privacy-first personal computing system.

## Features

- Account balances and net worth tracking
- Transaction history with search and categorization
- Budget tracking with flexible and rollover budgets
- Investment portfolio with performance tracking
- Cashflow analysis by category and merchant
- FIRE (Financial Independence) calculator with projections
- Asset/liability snapshots over time

## Installation

### Development (via Localitas core)

```bash
# Clone the repo
git clone https://github.com/localitas/localitas-app-mmoney.git ~/localitas-app-mmoney

# Start with the Localitas dev cluster (builds and runs in Docker automatically)
cd ~/localitas && make dev-core
```

### Standalone

```bash
cd ~/localitas-app-mmoney

# Build and run locally
make build
./bin/mmoney-server serve --listen :8000

# Or via launchd (macOS)
make start

# Or via Docker
make start-docker
```

## Exposing to the Internet

Localitas apps are accessible remotely through Localitas's built-in tunnel service, powered by FRP. No port forwarding or dynamic DNS required.

1. Sign up at [localitas.com](https://localitas.com) and connect your local Localitas core
2. The tunnel automatically exposes your core (and all apps) at `https://{your-subdomain}.localitas.com`
3. This app is available at `https://{your-subdomain}.localitas.com/apps/ext/mmoney/`

All traffic is encrypted end-to-end. Authentication is handled by the Localitas core — only authorized users can access your apps.

## License

MIT
