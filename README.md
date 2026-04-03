<div align="center">

# 🎯 Torn Advisor Engine

[![CI](https://github.com/subhanjanOps/torn-advisor/actions/workflows/ci.yml/badge.svg)](https://github.com/subhanjanOps/torn-advisor/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Coverage](https://img.shields.io/badge/coverage-90.5%25-brightgreen)](coverage.out)

**A rule-based decision engine for [Torn City](https://www.torn.com/) that analyzes your player state and recommends the optimal next actions — so you never waste energy, nerve, or cooldowns.**

[Getting Started](#-getting-started) · [Rules](#-rules) · [Configuration](#-configuration) · [Contributing](#-contributing)

</div>

---

## 📖 Overview

Torn Advisor connects to the [Torn API](https://api.torn.com/) via [tornSDK](https://github.com/subhanjanOps/tornSDK), evaluates **9 gameplay rules** against your live player state, and outputs a **prioritized action plan** telling you exactly what to do next.

```
$ TORN_API_KEY=xxx go run ./cmd/advisor/

=== Torn Advisor — Action Plan ===

1. [hospital] Heal Up (priority 98)
   Life is below 50% — heal before taking any action.

2. [drug] Take Xanax (priority 90)
   Xanax cooldown is ready — take Xanax for an energy boost.

3. [gym] Train at Gym (priority 80)
   Energy and happiness are sufficient — train your stats.
```

---

## 🏗️ Architecture

```mermaid
flowchart LR
    A[🌐 Torn API] -->|HTTP| B[📦 tornSDK]
    B -->|typed data| C[🔌 Provider]
    C -->|PlayerState| C2[💾 Cache]
    C2 -->|cached state| D[⚙️ Engine]
    D -->|Action Plan| E[🖥️ CLI]
    D -->|Action Plan| G[🤖 Discord Bot]
    D -->|Action Plan| W[🌐 Webhook Server]
    F[📄 Config JSON] -.->|priorities| D
    H[🔐 Key Store] -.->|per-user keys| G
    H -.->|per-user keys| W
    style A fill:#e74c3c,color:#fff
    style B fill:#3498db,color:#fff
    style C fill:#2ecc71,color:#fff
    style C2 fill:#1abc9c,color:#fff
    style D fill:#f39c12,color:#fff
    style E fill:#9b59b6,color:#fff
    style F fill:#95a5a6,color:#fff
    style G fill:#7289da,color:#fff
    style H fill:#e67e22,color:#fff
    style W fill:#e91e63,color:#fff
```

| Layer | Package | Responsibility |
|:------|:--------|:---------------|
| **Domain** | `domain/` | Core types (`PlayerState`, `Action`, `Rule` interface) — zero dependencies |
| **Engine** | `engine/` | Evaluates rules, builds sorted action plan |
| **Rules** | `rules/` | 9 individual rule implementations with configurable priorities |
| **Provider** | `providers/torn/` | Adapts tornSDK to the `StateProvider` interface |
| **Cache** | `providers/cache/` | TTL-based caching wrapper to reduce Torn API calls |
| **Config** | `config/` | Loads rule priorities from JSON (with sensible defaults) |
| **Key Store** | `store/` | AES-256-GCM encrypted per-user API key storage |
| **Bot** | `bot/` | Discord bot with slash commands, scheduler, multi-user support |
| **Webhook** | `webhook/` | HTTP handler for Discord Interactions endpoint (Ed25519 verified) |
| **CLI** | `cmd/advisor/` | CLI entry point — wires everything, reads env vars, prints output |
| **Bot Entry** | `cmd/bot/` | Discord bot entry point — connects to Discord gateway, starts scheduler |
| **Webhook Entry** | `cmd/webhook/` | Webhook server entry point — HTTP server for serverless/cloud deployments |

### Design Principles

- **Clean Architecture** — dependency arrows point inward; `domain` has zero imports
- **Interface-driven** — `Rule` and `StateProvider` are interfaces, enabling easy testing and extension
- **Configurable** — rule priorities can be overridden via a JSON config file without changing code

---

## 📂 Project Structure

```
torn-advisor/
├── .github/workflows/
│   └── ci.yml                 # GitHub Actions: test, lint, integration
├── bot/
│   ├── bot.go                 # 🤖 Discord bot (commands, scheduler, multi-user)
│   └── bot_test.go            # Bot tests (mocked session + provider)
├── cmd/
│   ├── advisor/
│   │   ├── main.go            # CLI entry point
│   │   └── main_test.go       # CLI tests
│   ├── bot/
│   │   └── main.go            # Discord bot entry point (gateway mode)
│   └── webhook/
│       └── main.go            # Webhook server entry point (HTTP mode)
├── config/
│   ├── config.go              # Priority loading from JSON
│   └── config_test.go         # Config tests
├── domain/
│   ├── models.go              # PlayerState, Action, Rule, StateProvider
│   └── priority.go            # Named priority constants
├── engine/
│   ├── engine.go              # Engine runner
│   ├── engine_test.go         # Engine tests
│   ├── planner.go             # Sort + filter actions
│   ├── planner_test.go        # Planner tests
│   └── rule_interface.go      # Type aliases for domain interfaces
├── providers/
│   ├── cache/
│   │   ├── provider.go        # 💾 TTL-based caching provider
│   │   └── provider_test.go   # Cache tests
│   └── torn/
│       ├── provider.go        # tornSDK → PlayerState adapter
│       └── provider_test.go   # Provider tests (mocked API)
├── rules/
│   ├── hospital.go … booster.go  # 9 individual rule files
│   ├── defaults.go            # Default rule set factory
│   └── rules_test.go          # Rule tests
├── store/
│   ├── keystore.go            # 🔐 AES-256-GCM encrypted key store
│   └── keystore_test.go       # Key store tests
├── webhook/
│   ├── handler.go             # 🌐 HTTP handler with Ed25519 signature verification
│   └── handler_test.go        # Webhook tests
├── tests/
│   └── integration_test.go    # End-to-end test (real API)
├── Dockerfile                 # Multi-stage Docker build
├── docker-compose.yml         # One-command deployment
├── .dockerignore
├── .env.example               # Template environment file
├── config.example.json        # Example priorities config
├── Makefile                   # Build, test, lint, cover targets
├── go.mod
└── LICENSE                    # MIT
```

---

## 🚀 Getting Started

### Prerequisites

| Tool | Version | Purpose |
|:-----|:--------|:--------|
| [Go](https://go.dev/dl/) | 1.25+ | Language runtime |
| [tornSDK](https://github.com/subhanjanOps/tornSDK) | v0.1.0 | Torn API data access |
| [golangci-lint](https://golangci-lint.run/) | latest | Linting (optional) |

### Installation

```bash
git clone https://github.com/subhanjanOps/torn-advisor.git
cd torn-advisor
go mod download
```

> **Note:** The project uses a local `replace` directive for `tornSDK`. Ensure the `tornSDK` repo is cloned at `../tornSDK` relative to this project.

### Run

```bash
# Required: set your Torn API key
export TORN_API_KEY="your-api-key-here"

# Optional: load custom priorities
export ADVISOR_CONFIG="./config.example.json"

# Run the advisor
go run ./cmd/advisor/
```

### Build

```bash
make build          # → bin/advisor
make run            # build + run
```

---

## 📋 Rules

The engine ships with **9 built-in rules**. Each rule evaluates a specific game condition and returns a prioritized action recommendation (or nothing if the condition isn't met).

| # | Rule | Emoji | Condition | Default Priority | Category |
|:-:|:-----|:-----:|:----------|:----------------:|:---------|
| 1 | **Hospital** | 🏥 | Life < 50% of max | `98` | `hospital` |
| 2 | **Chain** | ⛓️ | Faction chain is active | `97` | `chain` |
| 3 | **War** | ⚔️ | Faction war is active | `95` | `war` |
| 4 | **Xanax** | 💊 | Drug cooldown = 0 | `90` | `drug` |
| 5 | **Rehab** | 🩺 | Addiction > 50 | `85` | `rehab` |
| 6 | **Gym** | 🏋️ | Energy > 0 AND Happy > 4000 | `80` | `gym` |
| 7 | **Crime** | 🔫 | Nerve = Max AND Max > 0 | `70` | `crime` |
| 8 | **Travel** | ✈️ | Travel cooldown = 0 | `60` | `travel` |
| 9 | **Booster** | ⚡ | Booster cooldown = 0 | `55` | `booster` |

> **Priority order:** Higher numbers execute first. The engine evaluates all rules, then sorts the resulting actions by priority descending.

### Priority Levels Reference

| Constant | Value | Use Case |
|:---------|:-----:|:---------|
| `PriorityUrgent` | 100 | Life-threatening situations |
| `PriorityVeryImportant` | 90 | Time-sensitive opportunities |
| `PriorityImportant` | 70 | Valuable but not urgent |
| `PriorityNormal` | 50 | Routine actions |
| `PriorityLow` | 30 | Nice-to-have |

---

## ⚙️ Configuration

Rule priorities can be customized without modifying code by providing a JSON config file:

```bash
export ADVISOR_CONFIG="./my-priorities.json"
```

**Example `config.example.json`:**

```json
{
  "hospital": 98,
  "chain": 97,
  "war": 95,
  "xanax": 90,
  "rehab": 85,
  "gym": 80,
  "crime": 70,
  "travel": 60,
  "booster": 55
}
```

- Only include the rules you want to override — omitted rules keep their defaults
- Set a rule to `0` to effectively disable it
- Swap priorities to change the recommended order (e.g., make gym higher priority than xanax)

---

## 🧩 Adding Custom Rules

Implement the `Rule` interface from the `domain` package:

```go
package myrules

import "github.com/subhanjanOps/torn-advisor/domain"

type FarmingRule struct {
    Priority int
}

func (r FarmingRule) Evaluate(state domain.PlayerState) *domain.Action {
    if state.Energy > 50 {
        return &domain.Action{
            Name:        "Farm NPCs",
            Description: "Energy is sufficient — farm NPCs for cash.",
            Priority:    r.Priority,
            Category:    "farming",
        }
    }
    return nil
}
```

Register it alongside the default rules:

```go
cfg := config.DefaultPriorities()
ruleSet := rules.DefaultRulesWithConfig(cfg)
ruleSet = append(ruleSet, myrules.FarmingRule{Priority: 65})
eng := engine.NewEngine(ruleSet)
```

---

## 🧪 Testing

```bash
make test            # Run all unit tests
make test-verbose    # Run with verbose output
make cover           # Run with coverage report
make cover-html      # Generate HTML coverage report
make lint            # Run golangci-lint
```

### Test Coverage

| Package | Coverage |
|:--------|:--------:|
| `rules/` | 100% |
| `engine/` | 100% |
| `config/` | 100% |
| `providers/torn/` | 100% |
| `cmd/advisor/` | 100% (excl. `main()`) |
| **Overall** | **~90.5%** |

### Integration Tests

Integration tests hit the real Torn API and are gated behind a build tag:

```bash
export TORN_API_KEY="your-api-key"
go test ./... -tags=integration -run TestIntegration -count=1
```

---

## 🔄 CI/CD

GitHub Actions runs automatically on every push and PR to `main`:

```mermaid
flowchart TD
    A[Push / PR] --> B[🔨 Build]
    B --> C[🧪 Test + Coverage]
    A --> D[🔍 Lint]
    C --> E{Push to main?}
    D --> E
    E -->|Yes| F[🌐 Integration Test]
    E -->|No| G[✅ Done]
    F --> G
    style B fill:#3498db,color:#fff
    style C fill:#2ecc71,color:#fff
    style D fill:#f39c12,color:#fff
    style F fill:#e74c3c,color:#fff
    style G fill:#9b59b6,color:#fff
```

| Job | Trigger | What it does |
|:----|:--------|:-------------|
| **test** | Push & PR | Build, run tests with `-race`, generate coverage |
| **lint** | Push & PR | Run `golangci-lint` |
| **integration** | Push to `main` only | Full pipeline test against live Torn API (requires `TORN_API_KEY` secret) |

---

## 🤖 Discord Bot

The project includes a full-featured Discord bot that exposes the advisor engine via slash commands.

### Bot Commands

| Command | Description |
|:--------|:------------|
| `/register <api_key>` | Store your Torn API key (AES-256-GCM encrypted) |
| `/unregister` | Remove your stored API key |
| `/advise` | Get your personalized action plan |
| `/status` | Show current player stats (energy, nerve, happy, life) |
| `/config` | Display current rule priority configuration |
| `/schedule` | Enable periodic urgent-action alerts in the current channel |
| `/unschedule` | Disable periodic alerts |

### Features

- **Multi-user** — each Discord user registers their own Torn API key
- **Encrypted storage** — API keys are encrypted at rest with AES-256-GCM
- **Response caching** — Torn API responses are cached (30s TTL) to avoid rate limits
- **Scheduled alerts** — every 15 min, posts urgent actions (priority ≥ 90) to opted-in channels
- **Ephemeral responses** — sensitive commands (register, errors) are only visible to the user

### Setup

1. **Create a Discord Application** at [discord.com/developers](https://discord.com/developers/applications)
2. Create a Bot user and copy the **Bot Token**
3. Copy the **Application ID**
4. Generate an encryption key: `openssl rand -hex 32`
5. Invite the bot to your server with the `applications.commands` and `bot` scopes

### Run Locally

```bash
export DISCORD_BOT_TOKEN="your-bot-token"
export DISCORD_APP_ID="your-app-id"
export ENCRYPTION_KEY="your-64-hex-char-key"

go run ./cmd/bot/
```

### Run with Docker

```bash
cp .env.example .env
# Edit .env with your values
docker compose up -d
```

### Webhook Mode (HTTP Server)

For serverless or cloud deployments, run the bot as an HTTP server that receives Discord interactions via webhook instead of maintaining a persistent gateway connection.

1. In your Discord Application settings, set the **Interactions Endpoint URL** to your server (e.g., `https://your-server.com/`)
2. Copy the **Public Key** from the application's General Information page

```bash
export DISCORD_PUBLIC_KEY="your-ed25519-public-key-hex"
export ENCRYPTION_KEY="your-64-hex-char-key"
export WEBHOOK_PORT="8080"   # optional, default 8080

go run ./cmd/webhook/
```

> **Note:** Webhook mode does not support the scheduler (periodic advice). Use gateway mode (`cmd/bot`) if you need scheduled alerts.

### Environment Variables

| Variable | Required | Description |
|:---------|:--------:|:------------|
| `DISCORD_BOT_TOKEN` | ✅ (gateway) | Discord bot token |
| `DISCORD_APP_ID` | ✅ (gateway) | Discord application ID |
| `DISCORD_PUBLIC_KEY` | ✅ (webhook) | Ed25519 public key from Discord app settings |
| `WEBHOOK_PORT` | | Webhook server port (default: `8080`) |
| `ENCRYPTION_KEY` | ✅ | 32-byte hex key for API key encryption |
| `KEY_STORE_PATH` | | Path to encrypted key file (default: `keys.json`) |
| `ADVISOR_CONFIG` | | Path to custom priorities JSON |

---

## 🗺️ Roadmap

- [x] Discord bot integration
- [x] Multi-user support with encrypted key storage
- [x] Response caching / rate limiting
- [x] Scheduled periodic advice
- [x] Docker containerization
- [x] Graceful shutdown with context propagation
- [x] Webhook mode (HTTP server for Discord Interactions)
- [ ] AI layer for adaptive recommendations
- [ ] Web dashboard with real-time updates
- [ ] Battle target selection rule
- [ ] OC (Organized Crime) timing rule
- [ ] Company work/train rule

---

## 📄 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**Built for the streets of Torn City** 🏙️

[Torn City](https://www.torn.com/) · [tornSDK](https://github.com/subhanjanOps/tornSDK) · [Report Bug](https://github.com/subhanjanOps/torn-advisor/issues)

</div>