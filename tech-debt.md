# Technical Debt & Refactoring Opportunities

Generated from a comprehensive codebase analysis of the current `main` branch. Items are organized by priority and include current file paths/line numbers.

Last updated: 2026-08-10

## 🚨 Critical — fix first

| # | File | Lines | Issue | Recommendation |
|---|------|-------|-------|----------------|
| 1 | `server/background/draftDaemon.go` | 51–74 | `defer d.mu.RUnlock()` is inside the `for` loop. Defer is function-scoped, so the RLock is **never released between iterations** and the daemon will deadlock on the second tick. | ~~Move the lock/unlock to explicit calls inside the loop body.~~ **Fixed** — removed the lock from `Run` and rely on `IsRunning()`; added `tickInterval` for testability. |
| 2 | `server/database/databaseDriver.go` | 62–70 | `Prepare()` calls `log.Fatal` if statement preparation fails. | ~~Return the error; do not crash the server on a transient DB failure.~~ **Fixed** — `Prepare` now crashes via `assert` for schema/syntax/statement errors (`42xxx`, `22xxx`, `26xxx` SQLSTATE classes) and returns wrapped errors for transient failures. `background/cleanup.go` updated to use `database.Prepare` so all prepares go through the wrapper. |
| 3 | `server/assert/assert.go` / call sites | 29–73 | `RunAssert`, `NoError`, `AssertCF`, and `NoErrorCF` all call `log.Fatal` → `os.Exit(1)`. | ~~Audit call sites.~~ **Fixed** — audited all call sites; removed dead `assert` setup in `queryDraftRow`, converted `makePick` id-mismatch and `getDraftScore` zero-id checks to returned errors, and kept true invariants in `main.go`, `draftDaemon.go`, `databaseDriver.go`, and `getDraftScore` map-length check. Package is documented in `AGENTS.md` as intentional fail-fast behavior. |
| 4 | `server/draft/draftActor.go` | 555–603 | `handlePick` is not transactional: `MakePick`, `reloadDraftState`, `advanceToNextPick`, and notification are separate DB calls. | ~~Wrap the whole pick-and-advance flow in a transaction.~~ **Fixed** — `DraftStore` now has `RunInTransaction` and `WithTx`; `handlePick` runs `MakePick` + `MakePickAvailable`/`TEAMS_PLAYING` state transition inside one transaction; `advanceToNextPick` removed as dead code. |
| 5 | `server/draft/draftActor.go` | 637–693 | `handleSkipCurrentPick` has the same transaction gap as `handlePick`. | ~~Wrap skip-and-advance in a transaction.~~ **Fixed** — `handleSkipCurrentPick` uses `RunInTransaction` for `SkipPick` + `MakePickAvailable`/`TEAMS_PLAYING` transition; `handleUndoLastPick` uses `RunInTransaction` for `DeletePick` + `ResetPick`. |
| 6 | `server/model/draft.go` | 526–557, 559–597 | `getDraft` loads all picks once, then `loadDraftPlayers` calls `getDraftPlayerPicks` **per player**, causing an N+1 query. | ~~Load all picks once and map them to players in memory.~~ **Fixed** — `loadDraftPlayers` now groups the already-loaded `draftModel.Picks` by player in memory; `getPicks` orders by `AvailableTime` to preserve semantics, and `getDraftPlayerPicks` was removed. |
| 7 | `server/scorer/scorer.go` | 304–369 | `scoringRunner` infinite loop never checks `ctx.Done()`, so it leaks on shutdown. | ~~Add `ctx.Done()` checks and a graceful exit path.~~ **Fixed** — `RunScorer` now returns a `<-chan struct{}` wait handle, `main.go` waits on shutdown, and `MatchQueue.PopMatch` is context-aware so the loop can exit on cancellation. |

---

## 🔴 High impact

| # | File | Lines | Issue | Recommendation |
|---|------|-------|-------|----------------|
| 8 | `server/model/draft.go` | 168–269 | `getDraftsForUser` calls `loadDraftPickingInfo` and `loadPlayersForDraftInList` for **every row** returned. | ~~Fold these into the main query or batch the per-draft calls.~~ **Fixed** — players/invites and current-pick info are now loaded in two batched queries using `ANY($1::int[])`, reducing the request to a constant number of DB round-trips. |
| 9 | `server/model/draft.go` / `server/model/team.go` | ~1438–1550 | `getOverallLeaderboard` and `getDraftScore` call `getScore` for every team, generating at least one DB query per row. | ~~Join team score data in the leaderboard query.~~ **Fixed** — added `getScoresBatch` to score many teams in a single query; `getScore` now delegates to it, and both `getDraftScore` and `getOverallLeaderboard` batch all team scores in one call. |
| 10 | `server/tbaHandler/tbaHandler.go` | 100–179 | `makeRequest` returns `nil` for most errors and has no `error` return value; callers silently ignore failures. | ~~Return `([]byte, error)` and propagate errors.~~ **Fixed** — `makeRequest` now returns `([]byte, error)`; all public request methods and the `TBAInterface` return errors; callers in `scorer`, `draft/pickValidator`, `handler/adminPageHandler`, and `cache/avatarStore` handle or propagate them. |
| 11 | `server/handler/tbaWebhookEventHandler.go` | 56–60 | Invalid TBA webhook HMAC returns `200 OK` and logs the full request body. | ~~Return `401/403` and avoid logging untrusted bodies.~~ **Fixed** — invalid HMAC now returns `401 Unauthorized` and the request body is no longer logged on auth failure or JSON decode errors. |
| 12 | `server/draft/draftActor.go` | 306–380 | `handleAcceptInvite` performs multiple independent DB writes without a transaction. | ~~Wrap in a transaction.~~ **Fixed** — `handleAcceptInvite` now runs under `RunInTransaction`, locks the draft row with `SELECT ... FOR UPDATE`, checks player count, then atomically accepts the invite, adds the player, and cancels outstanding invites. Schema migration adds unique constraints to prevent duplicate players/pending invites. |
| 13 | `server/model/draft.go` | 324–354 | `createDraft` inserts the draft and the owner player in two separate statements. | ~~Wrap in a transaction.~~ **Fixed** — `SQLDraftStore.CreateDraft` now wraps `createDraft` in `RunInTransaction`. |
| 14 | `server/model/user.go` / `server/authentication/password.go` | 121–145 | Dummy bcrypt hash used for username-enumeration resistance may not be a valid bcrypt hash. | ~~Generate a valid dummy hash at startup.~~ **Fixed** — `BcryptPasswordHasher` in `server/authentication` generates the dummy hash on demand using the same cost as real passwords, eliminating a timing leak between unknown and existing usernames. |
| 15 | `server/handler/authPageHandler.go` / `server/authentication/validation.go` | 166–198 | Username format, length, and whitespace are not validated on registration. | ~~Add username normalization and validation rules.~~ **Fixed** — registration validation is centralized in `server/authentication`, trimmed whitespace, rejects spaces, enforces 3–32 character length, and restricts characters to letters, digits, underscores, and hyphens. |
| 16 | `server/utils/utils.go` / `server/utils/pick_window_config.go` / `config/pick-windows.json` | 90, 110–139 | `PICK_TIME = 1h` and daily pick windows (8–22, 17–22) are hardcoded globally. | ~~Move to env/config and add constants.~~ **Fixed** — pick time and per-day windows are now loaded from `config/pick-windows.json` via `PICK_WINDOWS_CONFIG_FILE`; defaults are preserved when the file is missing. Config is threaded through `DraftActorMap` / `DraftActor`. |

---

## 🟡 Medium / design debt

| # | File | Lines | Issue | Recommendation |
|---|------|-------|-------|----------------|
| 17 | `server/draft/draftActorMap.go` | 12 | `// TODO should we LRU this?` Actor cache is unbounded. | **Fixed** — replaced `sync.Map` with a custom stdlib LRU cache; size configurable via `DRAFT_ACTOR_CACHE_SIZE` (default 128); evicted actors are shut down asynchronously. |
| 18 | `server/handler/adminPageHandler.go` | 134 | `// TODO Need to start draft watch dog` — drafts started at runtime were not added to the pick daemon, so expired picks were not auto-skipped. | **Fixed** — wired `DraftDaemon` into `handler.ServiceGroup`; both the UI `HandleStartDraft` and the admin `StartDraftCommand` now add the draft to the daemon after transitioning to `PICKING`. |
| 19 | `server/utils/utils.go` | 109 | `// todo we should make it so this in configurable per draft` — pick windows are global. | Make pick windows configurable per draft. |
| 20 | `server/model/draft.go` ~1135 / `server/draft/draftActor.go` ~920 | — | Next-pick snake-draft algorithm exists in two places (`nextPick` vs `getNextPick`). | **Fixed** — the snake-draft algorithm is already shared via `model.DetermineNextPick`. |
| 21 | `server/handler/adminPageHandler.go` | 154–206, 282–358 | `SkipPickCommand` and `AdminPickCommand` reimplement actor send/receive logic instead of reusing helpers. | **Fixed** — `SkipPickCommand`, `AdminPickCommand`, and `ModifyPickTimeCommand` now route through `draft.SkipCurrentPick`, `draft.MakePick`, and `draft.ModifyCurrentPickExpirationTime`. |
| 22 | `server/draft/draftActor.go` | 555–603, 637–693 | `handlePick` and `handleSkipCurrentPick` are 60+ lines and mix validation, DB, state, and notifications. | **Fixed** — extracted `prepareDraftAdvance`, `commitDraftAdvance`, and `publishPickOutcome`; both handlers now delegate to these helpers. |
| 23 | `server/discord/discordWebhookBus.go` / `server/handler/tbaWebhookEventHandler.go` | 158–169, 190–193 | Discord ID validation/formatting is duplicated in two places. | **Fixed** — extracted `discord.Identifier` and `discord.IsValidId` helpers and replaced both inline implementations. |
| 24 | `server/handler/*.go` | Many | `userUuid := c.Get("userUuid").(uuid.UUID)` + `getAuthenticatedUsername` block is repeated in almost every protected handler. | **Fixed** — added `requireUser` and `requireUserUuid` helpers in `handler/handler.go`; they centralize the context type assertion and username lookup, and redirect to `/login` when the UUID is missing or invalid. |
| 25 | `server/model/draft.go` | 1576 | `CanStartDraft` requires exactly `8` players as a magic number. | ~~`const DraftPlayerCount = 8`~~ **Fixed** — added `DraftPlayerCount`, `PicksPerPlayer`, and `PicksPerDraft` constants; replaced all backend and template magic numbers. |
| 26 | `server/model/draft.go` / `server/draft/draftActor.go` | 649, 864 | Draft completion uses `len(picks) < 64` magic number. | ~~`const PicksPerDraft = 64`~~ **Fixed** — `PicksPerDraft = DraftPlayerCount * PicksPerPlayer` and used in `handlePick`, `handleSkipCurrentPick`, and tests. |
| 27 | `server/tbaHandler/tbaHandler.go` / `server/utils/utils.go` | 200, 264 | Year `2026` is hardcoded in one endpoint; year `2024` is hardcoded in `MakeMatchKeysYearRequest`; `Events()` also hardcodes 2026 event keys. | **Fixed** — added `utils.TbaSeasonYear` and `utils.TbaHistoricMatchYear` constants and replaced all hardcoded years. |
| 28 | `server/handler/draftPickPageHandler.go` | 149–269 | WebSocket upgrader, ping/pong, watcher registration, and HTML rendering are all in one handler. | Extract a dedicated WebSocket/notifier service. |
| 29 | `server/model/match.go` | 22 | `Match.String()` uses a value receiver; all other `String()` methods use pointer receivers. | **Fixed** — `Match.String()` already uses a pointer receiver. |
| 30 | `server/model/*.go` | Many | Function parameter named `database` while the import alias is `db`; error strings mix lowercase, Title Case, and sentence case. | **Fixed** — all model parameters/fields renamed to `db`, imports standardized to unaliased `server/database`, and the one Title Case error (`RunInTransaction...`) was lowercased. |
| 31 | `server/main.go` | 55–117 | Required env vars are not validated early; malformed bool parsing silently defaults. | **Fixed** — added `utils.RequireEnv` and strict parsing helpers; `main.go` now validates all required vars, enforces `SERVER_PORT` range, and fails fast on malformed bool/int values. |
| 32 | `server/draft/draftActor.go` | 704 | Undo pick hardcodes a `3 * time.Hour` expiration reset. | ~~Reuse the configured pick window/expiration logic.~~ **Fixed** — undo now uses `d.pickConfig.GetPickExpirationTime(..., d.pickConfig.PickTime)` instead of the hardcoded `3 * time.Hour`. |

---

## 🟢 Lower priority / polish

| # | File | Lines | Issue | Recommendation |
|---|------|-------|-------|----------------|
| 33 | `server/database/databaseDriver.go` | 47–49 | Pool settings (`90`, `25`, `30m`) are hardcoded. | **Fixed** — configurable via `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, and `DB_CONN_MAX_LIFETIME`; defaults in `database/defaults.go`. |
| 34 | `server/middleware/ratelimit.go` | 59–64 | Login (5), register (3), window (15m), and default general rate (100) are hardcoded. | **Fixed** — login/register limits and window are configurable via `RATE_LIMIT_LOGIN_ATTEMPTS`, `RATE_LIMIT_REGISTER_ATTEMPTS`, and `RATE_LIMIT_AUTH_WINDOW`; defaults in `middleware/defaults.go`. General rate is already env-driven. |
| 35 | `server/server.go` | 234 | `Cache-Control: public, max-age=2592000` is a raw number. | **Fixed** — configurable via `STATIC_ASSET_MAX_AGE_SECONDS`; default and constant in `server/defaults.go`. |
| 36 | `server/cache/avatarStore.go` | 58 | Redis TTL is `4*7*24*time.Hour`. | **Fixed** — configurable via `AVATAR_CACHE_TTL`; default and constant in `cache/defaults.go`. |
| 37 | `server/model/user.go` / `server/authentication` | 35, 171, 239 | bcrypt cost `14`, session expiration `10 days`, and duplicate-session threshold are hardcoded. | **Fixed** — `BCRYPT_COST` is configurable via env; session expiration is configurable via `SESSION_EXPIRATION_DAYS`; defaults and constants live in `authentication/defaults.go` and `model/defaults.go`. Duplicate-session threshold check no longer exists (was removed during refactor). |
| 38 | `server/background/cleanup.go` | 59 | Cleanup deletes sessions expiring before `now + 2 hours`. | **Fixed** — configurable via `SESSION_CLEANUP_LEEWAY_HOURS`; default and constant in `background/defaults.go`. |
| 39 | `server/utils/utils.go` | 14–25 | `Events()` returns a hardcoded 2026 event list. | **Fixed** — event codes are configurable via `TBA_EVENT_CODES` (comma-separated); defaults and constants in `utils/defaults.go`. The year is already centralized in `utils.TbaSeasonYear`. |
| 40 | `server/scorer/scorer.go` | 230–239, 160–166, 72–109 | Alliance-selection scores, playoff match points, and bonus points are hardcoded. | **Fixed** — all scoring values are configurable via `SCORER_*` env vars; defaults and constants in `scorer/defaults.go`. |
| 41 | `server/handler/draftPickPageHandler.go` | 207, 227, 241/259 | WebSocket read deadline (120s), ping ticker (30s), and write deadlines (10s) are hardcoded. | **Fixed** — configurable via `WS_READ_TIMEOUT`, `WS_PING_INTERVAL`, and `WS_WRITE_TIMEOUT`; defaults and constants in `handler/defaults.go`. |
| 42 | `server/draft/draftActor.go` | 110, 250, 267 | Inbox buffer (100) and message/reply timeouts (5s) are repeated throughout the actor. | **Fixed** — configurable via `DRAFT_ACTOR_INBOX_BUFFER` and `DRAFT_ACTOR_REQUEST_TIMEOUT`; defaults and constants in `draft/defaults.go`. |
| 43 | `server/draft/pickValidator.go` | 44–61 | O(n×m) event-validity loop (lists are tiny). | **Fixed** — `PickValidator` now builds a set of valid events once and checks membership in O(1). Added unit tests for all validation paths. |
| 44 | `server/picking/pickNotifier.go` | 16 | `PickEvent.Err` field is never consumed. | Remove it or wire it into error UI. |
| 45 | `server/main.go` | 79–117 | Repeated env-var parsing blocks (copy of pattern in `metrics/db.go`). | Reuse `getEnvAsInt`/`getEnvAsDuration` from `metrics/db.go`. |
| 46 | `server/model/discord.go` | 81–97 | `getDraftPickRows` builds an `IN (...)` query with `fmt.Sprintf`/`strings.Join`. Safe today because of `$N` placeholders, but raw SQL construction in model code. | Document the safety invariant or add an IN-list helper. |
| 47 | `server/handler/utils.go` | 37–76 | `generateCSRFCookie` and `validateCSRFCookie` live in `handler` but are conceptually CSRF middleware. | Move them to `server/middleware` or `server/authentication`. |
| 48 | `server/handler/invitePageHandler.go` | 133 | `HandleDeclineInvite` ends with a direct `Render` instead of the shared `renderInviteTable` helper. | Use the helper for consistency. |
| 49 | `server/model/draft.go` | Multiple | Some functions return wrapped errors, others return generic `errors.New` or raw errors. | Standardize on wrapped, lowercase, no-trailing-period errors. |
| 50 | `server/draftAgent/fantasyCaller.go` | 177–227 | `parseCurrentDraftPicks` and HTML-traversal helpers are brittle and long. | Add tests or use the JSON API instead of scraping HTML. |

---

## 🟣 TODOs / unresolved design comments

| # | File | Line | TODO |
|---|------|------|------|
| 55 | `server/draft/draftActor.go` | 82 | `// TODO Does tba handler need to be a pointer?` |
| 56 | `server/draft/draftActor.go` | 902 | `// TODO: Add store method for transferring ownership when available` |
| 57 | `server/handler/draftPickPageHandler.go` | 62 | `// TODO we could move this to the actor so we dont have to call the db` |
| 58 | `server/handler/draftProfilePageHandler.go` | 35 | `// TODO I think this should go through the draft manager` |
| 59 | `server/handler/adminPageHandler.go` | 134 | `// TODO Need to start draft watch dog` |
| 60 | `server/handler/authPageHandler.go` | 16 | `// We can probably do this in the middleware` (unresolved design question) |
| 61 | `server/model/user.go` | 178–179 | `// Should we move more logic here? No...` (design uncertainty) |
| 62 | `server/model/user.go` | 326–329 | `//If the count is greater than one there is a problem...Do we want to invalidate the session` |
| 63 | `draftTester/main.go` | 188 | `// TODO We should make a list of valid teams to pick and then just flip a coin for if we will pick them` |

---

## 🧪 Test gaps

- `server/background/draftDaemon.go` — a two-tick test would catch the deadlock.
- `server/utils/utils.go` — `GetPickExpirationTime` boundary tests (window start/end, weekends, DST).
- `server/model/draft.go` — `DetermineNextPick` snake-draft wrap-around and edge cases.
- `server/tbaHandler/tbaHandler.go` — cache hit, 304, 404, 5xx, retry, and network-error paths.
- `server/handler/draftPickPageHandler.go` — WebSocket watcher registration, ping/pong, disconnect.
- `server/handler/adminPageHandler.go` — admin commands with mock stores.
- `server/scorer/scorer.go` — unit tests with synthetic `swagger.Match` data.
- `server/cache/avatarStore.go` — Redis-down fallback path.

---

## ✅ Already cleaned up in recent commits

- Handler struct grouped into `StorageGroup` / `ServiceGroup` / `ConfigGroup`.
- Route registration split into `registerPublicRoutes`, `registerProtectedRoutes`, `registerAdminRoutes`, etc.
- `getAuthenticatedUsername` helper extracted.
- `CloseStatement` / `CloseRows` helpers added and adopted in most model code.
- `HandlerRegisterPost` shortened and validation extracted.
- `AvatarStore` and `DiscordWebhookBus` interfaces now exist.
- `renderLoginWithError` / `renderRegisterWithError` helpers added.
