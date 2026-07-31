# Technical Debt & Refactoring Opportunities

Generated from a comprehensive codebase analysis of the current `main` branch. Items are organized by priority and include current file paths/line numbers.

Last updated: 2026-07-31

## 🚨 Critical — fix first

| # | File | Lines | Issue | Recommendation |
|---|------|-------|-------|----------------|
| 1 | `server/background/draftDaemon.go` | 51–74 | `defer d.mu.RUnlock()` is inside the `for` loop. Defer is function-scoped, so the RLock is **never released between iterations** and the daemon will deadlock on the second tick. | ~~Move the lock/unlock to explicit calls inside the loop body.~~ **Fixed** — removed the lock from `Run` and rely on `IsRunning()`; added `tickInterval` for testability. |
| 2 | `server/database/databaseDriver.go` | 62–70 | `Prepare()` calls `log.Fatal` if statement preparation fails. | Return the error; do not crash the server on a transient DB failure. |
| 3 | `server/assert/assert.go` | 29–73 | `RunAssert`, `NoError`, `AssertCF`, and `NoErrorCF` all call `log.Fatal` → `os.Exit(1)`. | In production/request paths, return errors instead of terminating the process. |
| 4 | `server/draft/draftActor.go` | 555–603 | `handlePick` is not transactional: `MakePick`, `reloadDraftState`, `advanceToNextPick`, and notification are separate DB calls. | Wrap the whole pick-and-advance flow in a transaction. |
| 5 | `server/draft/draftActor.go` | 637–693 | `handleSkipCurrentPick` has the same transaction gap as `handlePick`. | Wrap skip-and-advance in a transaction. |
| 6 | `server/model/draft.go` | 526–557, 559–597 | `getDraft` loads all picks once, then `loadDraftPlayers` calls `getDraftPlayerPicks` **per player**, causing an N+1 query. | Load all picks once and map them to players in memory. |
| 7 | `server/scorer/scorer.go` | 304–369 | `scoringRunner` infinite loop never checks `ctx.Done()`, so it leaks on shutdown. | ~~Add `ctx.Done()` checks and a graceful exit path.~~ **Fixed** — `RunScorer` now returns a `<-chan struct{}` wait handle, `main.go` waits on shutdown, and `MatchQueue.PopMatch` is context-aware so the loop can exit on cancellation. |

---

## 🔴 High impact

| # | File | Lines | Issue | Recommendation |
|---|------|-------|-------|----------------|
| 8 | `server/model/draft.go` | 168–269 | `getDraftsForUser` calls `loadDraftPickingInfo` and `loadPlayersForDraftInList` for **every row** returned. | Fold these into the main query or batch the per-draft calls. |
| 9 | `server/model/draft.go` | ~1438–1550 | `getOverallLeaderboard` calls `getScore` for every team, generating at least one DB query per row. | Join team score data in the leaderboard query. |
| 10 | `server/tbaHandler/tbaHandler.go` | 100–179 | `makeRequest` returns `nil` for most errors and has no `error` return value; callers silently ignore failures. | Return `([]byte, error)` and propagate errors. |
| 11 | `server/handler/tbaWebhookEventHandler.go` | 56–60 | Invalid TBA webhook HMAC returns `200 OK` and logs the full request body. | Return `401/403` and avoid logging untrusted bodies. |
| 12 | `server/draft/draftActor.go` | 306–380 | `handleAcceptInvite` performs multiple independent DB writes without a transaction. | Wrap in a transaction. |
| 13 | `server/model/draft.go` | 324–354 | `createDraft` inserts the draft and the owner player in two separate statements. | Wrap in a transaction. |
| 14 | `server/model/user.go` | 121–145 | Dummy bcrypt hash used for username-enumeration resistance may not be a valid bcrypt hash. | Generate a valid dummy hash at startup. |
| 15 | `server/handler/authPageHandler.go` | 166–198 | Username format, length, and whitespace are not validated on registration. | Add username normalization and validation rules. |
| 16 | `server/utils/utils.go` | 90, 110–139 | `PICK_TIME = 1h` and daily pick windows (8–22, 17–22) are hardcoded globally. | Move to env/config and add constants. |

---

## 🟡 Medium / design debt

| # | File | Lines | Issue | Recommendation |
|---|------|-------|-------|----------------|
| 17 | `server/draft/draftActorMap.go` | 12 | `// TODO should we LRU this?` Actor cache is unbounded. | Decide on a max actor count / LRU policy and implement it. |
| 18 | `server/handler/adminPageHandler.go` | 134 | `// TODO Need to start draft watch dog` — no watchdog exists for stuck drafts. | Implement a watchdog that auto-skips expired picks, or remove the TODO. |
| 19 | `server/utils/utils.go` | 109 | `// todo we should make it so this in configurable per draft` — pick windows are global. | Make pick windows configurable per draft. |
| 20 | `server/model/draft.go` ~1135 / `server/draft/draftActor.go` ~920 | — | Next-pick snake-draft algorithm exists in two places (`nextPick` vs `getNextPick`). | Extract the core algorithm into a shared helper. |
| 21 | `server/handler/adminPageHandler.go` | 154–206, 282–358 | `SkipPickCommand` and `AdminPickCommand` reimplement actor send/receive logic instead of reusing helpers. | Route admin actions through `draft.SkipCurrentPick`, `draft.MakePick`, etc. |
| 22 | `server/draft/draftActor.go` | 555–603, 637–693 | `handlePick` and `handleSkipCurrentPick` are 60+ lines and mix validation, DB, state, and notifications. | Extract `completePick`, `advanceToNextPick`, `notifyPickComplete`. |
| 23 | `server/discord/discordWebhookBus.go` | 158–169 | Discord ID validation/formatting is duplicated in `handler/tbaWebhookEventHandler.go` ~190–193. | Move to a shared `discord.Identifier` helper. |
| 24 | `server/handler/*.go` | Many | `userUuid := c.Get("userUuid").(uuid.UUID)` + `getAuthenticatedUsername` block is repeated in almost every protected handler. | Consider a middleware that stashes the username in the context, or a single helper. |
| 25 | `server/model/draft.go` | 1576 | `CanStartDraft` requires exactly `8` players as a magic number. | `const DraftPlayerCount = 8`. |
| 26 | `server/model/draft.go` | 649, 864 | Draft completion uses `len(picks) < 64` magic number. | `const PicksPerDraft = 64` (or `DraftPlayerCount * DraftPlayerCount`). |
| 27 | `server/tbaHandler/tbaHandler.go` | 200, 264 | Year `2026` is hardcoded in one endpoint; year `2024` is hardcoded in `MakeMatchKeysYearRequest`. | `const TbaSeasonYear` and verify the 2024 endpoint is still needed. |
| 28 | `server/handler/draftPickPageHandler.go` | 149–269 | WebSocket upgrader, ping/pong, watcher registration, and HTML rendering are all in one handler. | Extract a dedicated WebSocket/notifier service. |
| 29 | `server/model/match.go` | 22 | `Match.String()` uses a value receiver; all other `String()` methods use pointer receivers. | Change to pointer receiver. |
| 30 | `server/model/*.go` | Many | Function parameter named `database` while the import alias is `db`; error strings mix lowercase, Title Case, and sentence case. | Standardize on `db` parameter names and lowercase, no-period errors. |
| 31 | `server/main.go` | 55–117 | Required env vars are not validated early; malformed bool parsing silently defaults. | Fail fast with clear messages. |
| 32 | `server/draft/draftActor.go` | 704 | Undo pick hardcodes a `3 * time.Hour` expiration reset. | Reuse the configured pick window/expiration logic. |

---

## 🟢 Lower priority / polish

| # | File | Lines | Issue | Recommendation |
|---|------|-------|-------|----------------|
| 33 | `server/database/databaseDriver.go` | 47–49 | Pool settings (`90`, `25`, `30m`) are hardcoded. | Env vars or named constants. |
| 34 | `server/middleware/ratelimit.go` | 59–64 | Login (5), register (3), window (15m), and default general rate (100) are hardcoded. | Config or constants. |
| 35 | `server/server.go` | 234 | `Cache-Control: public, max-age=2592000` is a raw number. | Named constant with comment. |
| 36 | `server/cache/avatarStore.go` | 58 | Redis TTL is `4*7*24*time.Hour`. | `AvatarCacheTTL` constant. |
| 37 | `server/model/user.go` | 35, 171, 239 | bcrypt cost `14`, session expiration `10 days`, and duplicate-session threshold are hardcoded. | Constants/env. |
| 38 | `server/background/cleanup.go` | 59 | Cleanup deletes sessions expiring before `now + 2 hours`. | Constant or env. |
| 39 | `server/utils/utils.go` | 14–25 | `Events()` returns a hardcoded 2026 event list. | Move to config/env. |
| 40 | `server/scorer/scorer.go` | 230–239, 160–166, 72–109 | Alliance-selection scores, playoff match points, and bonus points are hardcoded. | Define constants/season config. |
| 41 | `server/handler/draftPickPageHandler.go` | 207, 227, 241/259 | WebSocket read deadline (120s), ping ticker (30s), and write deadlines (10s) are hardcoded. | Constants. |
| 42 | `server/draft/draftActor.go` | 110, 250, 267 | Inbox buffer (100) and message/reply timeouts (5s) are repeated throughout the actor. | Centralized constants. |
| 43 | `server/draft/pickValidator.go` | 44–61 | O(n×m) event-validity loop (lists are tiny). | Use a map for draft events. |
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
| 51 | `server/draft/draftActor.go` | 734 | `// TODO: Wrap SkipPick and MakePickAvailable in a database transaction` |
| 52 | `server/draft/draftActor.go` | 826 | `// TODO: Wrap DeletePick and ResetPick in a database transaction` |
| 53 | `server/draft/draftActorMap.go` | 12 | `// TODO should we LRU this?` |
| 54 | `server/utils/utils.go` | 109 | `// todo we should make it so this in configurable per draft` |
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
