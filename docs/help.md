---
title: MMoney
description: Own your personal finance data — synced from Monarch Money
---

# MMoney

Own your personal finance data. MMoney syncs your complete financial history from [Monarch Money](https://www.monarchmoney.com) into your local raft-sqlite database — accounts, transactions, budgets, investments, net worth snapshots, credit scores — updated every 10 minutes.

**Why?** Financial platforms can change their terms, shut down, or lock you out. Your transaction history, net worth trends, and spending patterns are yours. MMoney gives you a permanent local archive that you control — query it, search it, build on it, back it up. No vendor lock-in, no API rate limits, no subscription required to access your own data.

## Prerequisites

1. **Sign up for [Monarch Money](https://www.monarchmoney.com)**. Monarch Money is a personal finance platform that aggregates all your bank accounts, credit cards, loans, investments, and property values in one place via Plaid and other data providers. Create an account and link your financial institutions through their app. MMoney uses Monarch Money as the data source — it handles the hard part of connecting to hundreds of banks securely.

2. **You need an email + password login**. MMoney authenticates via Monarch Money's REST API which requires email and password — Google/Apple social login tokens cannot be used directly.

   **If you signed up with Google or Apple**: Go to [Monarch Money Settings](https://app.monarchmoney.com/settings) > Security and **set a password** on your account. Your Google login will still work alongside the password. Once you have a password set, use your Google email address and the new password in the vault credential below.

3. **Enable MFA (recommended)**. If you use TOTP-based MFA (authenticator app), you'll need the TOTP secret key for automated login. When setting up MFA in Monarch Money, copy the secret key (the base32 string, not the QR code). If you don't use MFA, leave `mfa_secret` empty.

## Setup

### Step 1: Store your Monarch Money credentials in the vault

The vault app encrypts and stores your credentials securely. The `data` field contains your Monarch Money login details:

```bash
curl -X POST http://localhost:8080/apps/vault/api/credentials \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "monarch-money",
    "data": {
      "email": "you@example.com",
      "password": "your-monarch-password",
      "mfa_secret": "YOUR_TOTP_BASE32_SECRET"
    }
  }'
```

The response returns a `public_id` — save this for the next step:

```json
{
  "public_id": "abc123-def456-...",
  "name": "monarch-money",
  "created_at": "2026-06-13T..."
}
```

If you don't use MFA, set `"mfa_secret": ""`.

You can also create the credential through the Vault UI at `/apps/vault`.

### Step 2: Configure MMoney with your vault credential ID

Tell MMoney which vault credential to use for authentication:

```bash
curl -X POST http://localhost:8080/apps/ext/mmoney/api/config \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"key": "vault_credential_id", "value": "abc123-def456-..."}'
```

Replace `abc123-def456-...` with the `public_id` from step 1.

### Step 3: Run the first sync

```bash
curl -X POST http://localhost:8080/apps/ext/mmoney/api/sync \
  -H "Authorization: Bearer $TOKEN"
```

The first sync fetches your entire transaction history from Monarch Money. This may take a minute depending on how many transactions you have.

### Step 4: Verify

```bash
curl http://localhost:8080/apps/ext/mmoney/api/sync-status \
  -H "Authorization: Bearer $TOKEN"
```

You should see `"status": "ok"` with account and transaction counts.

After the first sync, the automation app syncs every 10 minutes automatically. You can also press the Sync button in the UI or call `POST /api/sync` manually.

## How Sync Works

- **Auth token refresh**: On each sync, MMoney authenticates with Monarch Money using your vault credentials. The token is cached for 9 minutes to avoid re-authenticating on every cycle.
- **Sliding window**: Each sync fetches transactions from 7 days before the last sync through today. This overlap ensures nothing is missed.
- **First sync**: Fetches all historical data (no start date filter).
- **Upsert**: All records use `INSERT OR REPLACE` keyed on Monarch's remote ID. Duplicates are impossible.
- **What syncs**: Accounts, transactions, categories, budgets (current ±1 month), recurring transactions (next 3 months), investment holdings, net worth snapshots, credit score history.

## Accounts

**GET /api/accounts** — List all synced financial accounts with balances, types, and status.

## Transactions

**GET /api/transactions** — List transactions with pagination and date filtering (`start`, `end`, `limit`, `offset`).

**GET /api/search** — Full-text search across transaction merchants, categories, and notes.

## Categories

**GET /api/categories** — List all transaction categories and their groups.

## Budgets

**GET /api/budgets** — List budget entries. Filter by `month` (YYYY-MM).

## Recurring

**GET /api/recurring** — List recurring transactions with frequency and next date.

## Investments

**GET /api/investments** — List investment holdings with ticker, quantity, basis, and current value.

## Net Worth

**GET /api/net-worth** — Historical net worth from account snapshots. Filter by `start` and `end`.

**GET /api/snapshots** — Raw account balance snapshots over time.

## Cashflow

**GET /api/cashflow** — Income, expense, savings, and savings rate. Filter by `start` and `end`.

## Reports

**GET /api/reports/cashflow** — Monthly income/expense bars, totals, and category breakdowns. Supports `recurring=1` filter to show only recurring transactions.

**GET /api/reports/fire** — FIRE (Financial Independence) calculator.

## FIRE Calculator

FIRE = when your investment portfolio's annual dollar growth exceeds your annual living expenses.

The traditional 25x rule is a proxy. The real question is simpler: **does your money grow faster than you spend it?**

- **Investment Growth ($/yr)**: How much your portfolio grew in the last 12 months
- **Annual Expenses**: How much you spent in the last 12 months
- **Surplus/Gap**: Growth minus expenses. Positive = you're FIRE. Negative = the gap to close.
- **Years to FIRE**: Projects forward using your current growth rate and expense inflation rate

The projection chart shows annual dollar growth at multiple return rates (current, 8%, 12%, 15%, 18%) with your expenses as a rising dashed line (accounting for inflation). Where any growth line crosses the expense line = FIRE for that scenario.

This is your personal Monte Carlo — one chart, multiple futures.

## Sync

**POST /api/sync** — Trigger a manual sync from Monarch Money.

**GET /api/sync-status** — Check last sync time and status.

## Configuration

**POST /api/config** — Set a config value (e.g. `vault_credential_id`).

**GET /api/config/{key}** — Get a config value.

## Build & Deploy

### Build from source

```bash
cd apps/mmoney && go build -o bin/mmoney-server ./cmd/mmoney-server
```

### Cross-compile

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o bin/mmoney-server-linux-amd64 ./cmd/mmoney-server

# Linux arm64
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o bin/mmoney-server-linux-arm64 ./cmd/mmoney-server
```

### Obfuscated release (all platforms)

```bash
make build-release
```

### Docker

```bash
./mmoney-server docker-build
```

### Download

Pre-built binaries are available on the [GitHub releases page](https://github.com/localitas/localitas/releases).

```bash
gh release download --repo localitas/localitas --pattern 'mmoney-server-*'
```
