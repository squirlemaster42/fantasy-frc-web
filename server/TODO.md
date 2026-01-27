# TODO List

Comprehensive list of pending tasks, bugs, and technical debt for the Fantasy FRC server.

> Last updated: 2026-01-21

---

## Code TODOs

### High Priority (Bugs/Issues)

| File | Line | Description | Status |
|------|------|-------------|--------|
| `picking/pickManager.go` | 60 | Bug on last pick when watching page | Open |
| `model/draft.go` | 649 | Should not crash program on error | Open |
| `draft/draftManager.go` | 245 | Transition doesn't execute due to lock | Open |

### Medium Priority (Validation)

| File | Line | Description | Status |
|------|------|-------------|--------|
| `handler/draftProfilePageHandler.go` | 199 | Check start time > current time | Open |
| `handler/draftProfilePageHandler.go` | 236 | Verify all players accepted draft | Open |
| `handler/draftProfilePageHandler.go` | 238 | Handle UI when players haven't accepted | Open |
| `model/draft.go` | 856 | Change hardcoded `8` to draft size | Open |

### Enhancement

| File | Line | Description | Status |
|------|------|-------------|--------|
| `model/draft.go` | 523 | Allow uninviting from draft | Open |
| `handler/adminPageHandler.go` | 110 | Get notifier for admin console | Open |
| `handler/draftPickPageHandler.go` | 146 | Unregister WebSocket listener | Open |

### Technical Debt

| File | Line | Description | Status |
|------|------|-------------|--------|
| `draft/draftManager.go` | 119 | Make map thread-safe | Open |
| `draft/draftManager.go` | 151 | Check if reload path needs locking | Open |
| `draft/draftManager.go` | 256 | Refactor lock functions | Open |
| `picking/pickManager.go` | 57 | Handle error on all callers | Open |
| `main.go` | 35 | Session secret not in config | Open |
| `tbaHandler/tbaHandler.go` | 129 | Verify caching status | Open |
| `handler/authPageHandler.go` | 62 | Enable secure cookies | Open |
| `background/cleanup.go` | 27 | Verify if cleanup is used | Open |
| `utils/utils.go` | 35 | Add actual error handling | Open |

---

## Documentation TODOs

### WebSocket API

- [ ] Add complete client SDK documentation
- [ ] Add advanced error handling patterns
- [ ] Add performance benchmarking data

### Architecture

**System Overview (`architecture/system-overview.md`)**
- [ ] Add detailed component descriptions
- [ ] Add deployment diagrams
- [ ] Add scaling considerations

**Data Flow (`architecture/data-flow.md`)**
- [ ] Add detailed data transformation examples
- [ ] Add performance benchmarks
- [ ] Add security audit procedures

**Component Interactions (`architecture/component-interactions.md`)**
- [ ] Add detailed interaction diagrams
- [ ] Add error handling flows
- [ ] Add performance metrics

### Database

**Schema (`database/schema.md`)**
- [ ] Add query optimization guide
- [ ] Add backup procedures
- [ ] Add monitoring queries

### Business Logic

**Scoring (`business-logic/scoring.md`)**
- [ ] Add historical scoring examples
- [ ] Add edge case handling details
- [ ] Add performance optimization notes

**Draft States (`business-logic/draft-states.md`)**
- [ ] Add detailed timing configuration examples
- [ ] Add WebSocket message formats
- [ ] Add administrative command documentation

### Development

**Setup (`development/setup.md`)**
- [ ] Add troubleshooting guide
- [ ] Add performance profiling setup
- [ ] Add CI/CD integration instructions

---

## Priority Matrix

```
┌─────────────────────────────────────────────────────────────────┐
│                     PRIORITY MATRIX                              │
├──────────────────┬───────────────────┬──────────────────────────┤
│                  │ Quick Fix         │ Requires Investigation   │
├──────────────────┼───────────────────┼──────────────────────────┤
│ High Impact      │ MODEL-DRAFT-649   │ PICKMGR-60               │
│                  │ DRAFTMGR-245      │ DRAFTMGR-119             │
├──────────────────┼───────────────────┼──────────────────────────┤
│ Low Impact       │ HANDLER-199       │ ADMIN-110                │
│                  │ MODEL-856         │ DRAFTPICK-146            │
└──────────────────┴───────────────────┴──────────────────────────┘
```

---

## Suggested Order of Work

### Phase 1: Critical Bugs (1-2 days)
1. `picking/pickManager.go:60` - Fix last pick bug
2. `model/draft.go:649` - Prevent crash on error

### Phase 2: Validation & Safety (2-3 days)
1. `draft/draftManager.go:119` - Make map thread-safe
2. `handler/draftProfilePageHandler.go:199` - Start time validation
3. `handler/draftProfilePageHandler.go:236` - Player acceptance check

### Phase 3: Technical Debt (1 week)
1. `main.go:35` - Move session secret to config
2. `handler/authPageHandler.go:62` - Enable secure cookies
3. `draft/draftManager.go` - Clean up locking logic
4. `utils/utils.go:35` - Proper error handling

### Phase 4: Features (ongoing)
1. `model/draft.go:523` - Uninvite feature
2. `model/draft.go:856` - Dynamic draft size
3. `handler/adminPageHandler.go:110` - Admin console notifier

---

## How to Use This Document

1. **Track Progress**: Mark items as [In Progress] or [Complete]
2. **Estimate Effort**: Add time estimates when claiming items
3. **Link PRs**: Reference PR numbers when items are addressed
4. **Update Regularly**: Sync with actual code changes

## Adding New TODOs

When adding TODOs to code:

```go
// TODO: [Brief description] - [File:Line]
//       [Detailed explanation if needed]
```

Example:
```go
// TODO: Add rate limiting - handler/api.go:42
//       Need to implement per-user rate limiting for API endpoints
//       to prevent abuse during high-traffic periods.
```

---

*This document is maintained by the development team. Last sync: 2026-01-21*
