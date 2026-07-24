# Technical Debt & Refactoring Opportunities

Generated from a comprehensive codebase analysis. Items are organized by
category and priority.

## 🟠 Code Duplication

### Need to figure out a better way to deal with model helper functions

We have a lot of functions that currently live in model but dont touch the db.
Model should really just have the db logic so we should figure out a way to extract these.

### 3. Next-pick logic duplicated
**Files:**
- `server/model/draft.go:1135` (`nextPick()`)
- `server/draft/draftActor.go:920` (`getNextPick()`)

Both implement the "determine the next player to pick" algorithm. The `model`
version queries the database; the `draft` version operates on in-memory state.
This is a source of potential divergence bugs.

**Fix:** Extract the core algorithm into a shared helper and have both
implementations delegate to it.

## 🟡 Long Functions to Break Up

### 7. `getDraft` — 170 lines
**File:** `server/model/draft.go:394-565`

Does: query draft header, get current pick, iterate players, load sub-picks
for each player.

**Suggested extractions:** `queryDraftHeader`, `loadPlayers`,
`loadPicksForPlayer`.

### 8. `getDraftsForUser` — 152 lines
**File:** `server/model/draft.go:178-330`

Does: prepare first query, prepare second query, iterate rows, load current
pick, iterate inner rows.

**Suggested extractions:** `queryDraftsForUser`, `buildPlayerQuery`,
`loadCurrentPick`.

### 9. `HandlerRegisterPost` — 124 lines
**File:** `server/handler/authPageHandler.go:163-288`

Seven nearly identical validation blocks (CSRF, username taken, password
match, length, complexity, register, session). Each block follows the same
pattern with slightly different logic.

**Suggested extractions:** Create individual `validate*` functions or use a
validation chain pattern.

### 10. `handlePick` — 130 lines
**File:** `server/draft/draftActor.go:570-699`

Does: validate pick, store the pick, reload draft, get next pick, check if
complete, make next pick available, build discord event, send notification.

**Suggested extractions:** `completePick`, `advanceToNextPick`,
`notifyPickComplete`.

### 11. `handleSkipCurrentPick` — 90 lines
**File:** `server/draft/draftActor.go:733-823`

Shares much of its structure with `handlePick`. Both could share a
`handlePickAdvancement` helper.

### 12. `PostPickNotification` — 96 lines
**File:** `server/discord/discordWebhookBus.go:166-262`

Complex conditional logic for draft-complete / skip / normal pick messages
with markdown formatting embedded inline.

**Suggested extractions:** `buildPickNotificationContent`, `sendWebhook`.

---

## 🔵 Repeated Error Handling Boilerplate

### 13. Statement/rows close pattern (~90 occurrences)
Every query function in the `model` package repeats:
```go
defer func() {
    if err := stmt.Close(); err != nil {
        log.Error(ctx, "FuncName: Failed to close statement", "error", err)
    }
}()
```

And for rows:
```go
defer func() {
    if err := rows.Close(); err != nil {
        log.Error(ctx, "FuncName: Failed to close rows", "error", err)
    }
}()
```

**Fix:** Create `database.CloseStatement(ctx, stmt)` and
`database.CloseRows(ctx, rows)` helpers. Spans `model/draft.go`,
`model/user.go`, `model/team.go`, `model/match.go`, `model/matchTeam.go`,
`model/discord.go`, and `tbaHandler/tbaHandler.go`.

### 14. "Failed to get username" handler boilerplate (~9 occurrences)
Every protected handler starts with:
```go
username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
if err != nil {
    log.Error(c.Request().Context(), "Failed to get username", "error", err)
    return c.String(http.StatusInternalServerError, "An error occurred")
}
```

**Files:**
- `server/handler/homePageHandler.go:15`
- `server/handler/draftScoreHandler.go:22`
- `server/handler/draftProfilePageHandler.go:22`
- `server/handler/draftAdminHandler.go:32`
- `server/handler/leaderboardHandler.go:18`
- `server/handler/teamHandler.go:14`
- `server/handler/userProfilePageHandler.go:20`
- `server/handler/createDraftHandler.go:20`
- `server/handler/landingPageHandler.go:33`

**Fix:** Extract a `h.getAuthenticatedUsername(c) (string, error)` helper on
the `Handler` struct.

### 15. CSRF cookie generation error handling repeated
**File:** `server/handler/authPageHandler.go`

The `generateCSRFCookie` error path (5–7 lines) is repeated at lines 18, 48,
109, 149, 191, 166, 193, 207, 223, 251.

**Fix:** Extract error handling into a helper or restructure control flow.

---

## 🟣 Inconsistent Patterns

### 16. `Render` vs manual byte-buffer rendering
**File:** `server/handler/utils.go:14-25`

- `Render(c, component)` renders directly to `c.Response()`.
- `RenderError(c, status, component)` buffers then calls `c.HTML()`.
-   The pick notifier (`draftPickPageHandler.go:257`) renders to
  `strings.Builder` manually.

**Fix:** Add a `RenderToString(c, component)` helper and use it consistently,
or make `RenderError` the standard for all rendering.

### 17. SQL `Prepare` inconsistency
-   Most model functions use `db.Prepare(ctx, database, query)` from
  `server/database/databaseDriver.go:58`.
-   Some functions use `database.PrepareContext(ctx, query)` directly (e.g.,
  `uninvitePlayer` at `draft.go:839`, `getOutstandingInvitesForDraft` at
  `draft.go:893`, `getOverallLeaderboard` at `draft.go:1582`,
  `getDraftPickRows` at `discord.go:107`).

**Fix:** Standardize on a single `Prepare` wrapper across the codebase.

### 18. Error message casing inconsistency
Three styles appear with no consistent pattern:
- `errors.New("lowercase")`
- `fmt.Errorf("Capital letter")`
- `errors.New("Sentence case.")`

**Fix:** Choose one convention (e.g., lowercase, no trailing period per Go
convention) and apply consistently.

### 19. `Match.String()` uses value receiver
**File:** `server/model/match.go:22`

All other `String()` methods in the codebase use pointer receivers
(`*DraftModel`, `*DraftPlayer`, `*Pick`, `*User`, `*MatchTeam`). `Match` uses
a value receiver, which is inconsistent and can cause unexpected copies.

### 20. `database` vs `db` field naming
- Model functions use `database` as the parameter name for `*sql.DB`.
- `sql_*_store.go` files use `db` as the struct field name.

**Fix:** Pick one convention and apply consistently.

### 21. Import alias inconsistency
- `db "server/database"` used in `model/` files.
- `draftView "server/view/draft"` used in `handler/` files.
- `"server/database"` imported without alias in `tbaHandler.go`.
- `"server/view/draft"` imported without alias in `createDraftHandler.go`.

**Fix:** Standardize import aliasing conventions.

---

## 🟢 Interface & Design Improvements

### 22. Handler struct has 11 ungrouped fields
**File:** `server/handler/handler.go:15-31`

The `Handler` struct has grown to 11 fields. Every new dependency gets added
here linearly.

**Fix:** Group into logical sub-structs:
```go
type Handler struct {
    Stores     StorageGroup    // DraftStore, UserStore, TeamStore
    Services   ServiceGroup    // TBAHandler, Scorer, DiscordWebhookBus
    Config     ConfigGroup     // SecureHttpCookie, MinPasswordLength, etc.
}
```

### 23. No interface for `AvatarStore`
**File:** `server/cache/avatarStore.go`

Handlers depend on `*cache.AvatarStore` directly. An interface would make
testing easier.

### 24. No interface for `DiscordWebhookBus`
**File:** `server/discord/discordWebhookBus.go`

Used directly throughout the codebase. An interface like `DiscordNotifier {
PostPickNotification(...) }` would allow mocking in tests.

### 25. No interface for `TBAHandler`
**File:** `server/tbaHandler/tbaHandler.go`

Always passed as a concrete pointer. A `TBAInterface` would decouple the
scorer and draft packages from the HTTP implementation.

### 26. Unused `PickListener` interface
**File:** `server/picking/pickNotifier.go:21-23`

The `PickListener` interface is defined but never implemented. Its method
signature doesn't match `PickNotifier.ReceivePickEvent`.

**Fix:** Either remove it or align it with the actual usage.

### 27. Assert package used in user-facing handlers (contradicts AGENTS.md)
**Files:**
- `server/handler/draftScoreHandler.go:73`
- Various locations in `server/model/draft.go`
- Various locations in `server/draft/draftActor.go`

AGENTS.md states: *"Never use assert.Fatal in authentication hot paths or
user- facing handlers — always return errors gracefully."* The assert package
calls `log.Fatal` → `os.Exit(1)`, meaning any assertion failure in these paths
crashes the entire server.

---

## 🟤 TODOs & Known Technical Debt

| # | File | Line | TODO |
|---|------|------|------|
| 28 | `server/draft/draftActor.go` | 734 | `// TODO: Wrap SkipPick and MakePickAvailable in a database transaction` |
| 29 | `server/draft/draftActor.go` | 826 | `// TODO: Wrap DeletePick and ResetPick in a database transaction` |
| 30 | `server/draft/draftActorMap.go` | 12 | `// TODO should we LRU this?` |
| 31 | `server/utils/utils.go` | 109 | `// todo we should make it so this in configurable per draft` |
| 32 | `server/draft/draftActor.go` | 82 | `// TODO Does tba handler need to be a pointer?` |
| 33 | `server/draft/draftActor.go` | 902 | `// TODO: Add store method for transferring ownership when available` |
| 34 | `server/handler/draftPickPageHandler.go` | 62 | `// TODO we could move this to the actor so we dont have to call the db` |
| 35 | `server/handler/draftProfilePageHandler.go` | 35 | `// TODO I think this should go through the draft manager` |
| 36 | `server/handler/adminPageHandler.go` | 134 | `// TODO Need to start draft watch dog` |
| 37 | `server/handler/authPageHandler.go` | 16 | `// We can probably do this in the middleware` (unresolved design question) |
| 38 | `server/model/user.go` | 178-179 | `// Should we move more logic here? No...` (design uncertainty) |
| 39 | `server/model/user.go` | 326-329 | `//If the count is greater than one there is a problem...Do we want to invalidate the session` |

---

## ⚪ Minor Cleanups & Miscellaneous

### 40. Repeated env-var parsing in `main.go`
**File:** `server/main.go:79-117`

Four nearly identical blocks for parsing `minPasswordLength`,
`redisRateLimitDB`, `redisAvatarDB`, `postsPerMinute`, and `rateLimitEnabled`
from environment variables:
```go
val := default
if var != "" {
    parsed, err := strconv.Atoi(var) // or ParseInt, ParseBool
    if err == nil {
        val = parsed
    }
}
```

The `metrics/db.go` file (lines 45–69) already has `getEnvAsInt` and
`getEnvAsDuration` helpers — evidence the pattern was recognized but not
applied uniformly.

**Fix:** Create shared helpers: `getEnvInt(key string, default int) int`,
`getEnvBool(key string, default bool) bool`, `getEnvInt64(key string, default
int64) int64`.

### 41. Route registration is one big function
**File:** `server/server.go:161-216`

All 30+ routes are registered in a single function.

**Fix:** Group into `registerAuthRoutes(app, auth)`,
`registerDraftRoutes(protected, auth)`, `registerAdminRoutes(admin)`.
