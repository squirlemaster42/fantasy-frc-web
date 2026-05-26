# Draft Agent

AI-powered agent that automates fantasy FRC draft picks by integrating with the Fantasy FRC web application.

## Overview

Draft Agent is a Go application that uses OpenCode's AI API to automatically make picks in a Fantasy FRC draft. It creates users, starts a draft, invites players, and uses AI to determine which teams to pick based on persona prompts.

## How It Works

1. **User Configuration** - Loads user personas from `userConfig.json`
2. **Authentication** - Creates user accounts and logs them in
3. **Draft Creation** - Creates a new draft and invites all configured users
4. **AI Picks** - For each turn, calls OpenCode with the player's persona prompt and current draft state to determine the pick

## Configuration

### userConfig.json

Defines users and their drafting personas:

```json
[
    {
        "Username": "UserOne",
        "Password": "UserOne",
        "DraftPersona": {
            "Model": "Test",
            "PersonaPrompt": "You are an expert at picking teams for a fantasy First Robotics competition draft..."
        }
    }
]
```

- **Username/Password**: Credentials for the Fantasy FRC web app
- **PersonaPrompt**: Instructions given to the AI for making picks

### System Prompt

Located in `main.go`, defines the base prompt for pick requests:

```
Respond with a single team number. Do not respond with anything else. Do not give an explanation for your pick, just give the team number.
```

This is combined with the user's persona prompt and current draft state to create the full prompt sent to OpenCode.

## Usage

### Prerequisites

- Go 1.26.2+
- Running Fantasy FRC server on `http://localhost:7331`
- OpenCode CLI installed and available in PATH
- `frc-worlds-2025.csv` file with valid team numbers (optional, for validation)

### Running

```bash
cd draftAgent
go run .
```

### Configuration

Edit `userConfig.json` to configure:
- Number of players in the draft
- Usernames and passwords (must match accounts in Fantasy FRC)
- Persona prompts that define how the AI picks teams

## Architecture

### Key Files

| File | Purpose |
|------|---------|
| `main.go` | Entry point, coordinates draft flow and AI calls |
| `fantasyCaller.go` | HTTP client functions for interacting with Fantasy FRC web app |
| `userConfig.json` | User and persona configuration |

### Flow

1. `initUsers()` - Loads config, creates users, authenticates
2. `initDraft()` - Creates draft, invites users, starts draft
3. Loop until draft reaches "Teams Playing" status:
   - Identify current picking player
   - Build prompt with persona + current picks
   - Call OpenCode API to get pick
   - Submit pick via HTTP request

## Environment

- **Target Server**: `http://localhost:7331` (configurable in `fantasyCaller.go`)
