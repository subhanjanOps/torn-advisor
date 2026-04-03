# Torn Advisor Engine

A rule-based decision engine for [Torn City](https://www.torn.com/) that recommends optimal next actions based on player state.

## Architecture

```
Torn API → tornSDK → Provider → Engine → Bot/UI
```

- **tornSDK** — data access layer (separate module)
- **advisor-engine** — decision logic (this repo)
- **Discord bot** — interface (future)
- **AI layer** — optional intelligence (future)

## Project Structure

```
advisor-engine/
├── engine/
│   ├── models.go          # PlayerState, Action structs
│   ├── rule_interface.go   # Rule interface
│   ├── rules.go            # Built-in rules (Xanax, Gym, Crime, Travel, Rehab, War)
│   ├── planner.go          # Sorts actions by priority
│   ├── priority.go         # Priority level constants
│   └── engine.go           # Engine runner
├── providers/
│   └── torn/
│       └── provider.go     # Converts tornSDK data → PlayerState
├── cmd/
│   └── advisor/
│       └── main.go         # CLI entry point
├── go.mod
└── README.md
```

## Usage

```bash
export TORN_API_KEY="your-api-key"
go run ./cmd/advisor/
```

## Rules

| Rule | Condition | Priority |
|------|-----------|----------|
| War | War is active | 95 |
| Xanax | Drug cooldown == 0 | 90 |
| Rehab | Addiction > 50 | 85 |
| Gym | Energy > 0 AND Happy > 4000 | 80 |
| Crime | Nerve == Max | 70 |
| Travel | Travel cooldown == 0 | 60 |

## Adding Custom Rules

Implement the `Rule` interface:

```go
type Rule interface {
    Evaluate(state engine.PlayerState) *engine.Action
}
```

Then register it when creating the engine:

```go
rules := engine.DefaultRules()
rules = append(rules, MyCustomRule{})
eng := engine.NewEngine(rules)
```