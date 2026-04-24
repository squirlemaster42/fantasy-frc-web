# Log-Only Pull-Based Aggregation Plan
*Fantasy FRC Web · Updated: 2026-04-24*

## Overview
This plan implements **log aggregation only** (no traces) using a pull-based architecture that complies with all network constraints:
- No extra software deployed to the cloud server
- All monitoring tools remain in the home lab
- Uses Grafana Alloy's pull-based `http.poller` to collect logs from the cloud app
- Preserves existing Prometheus metrics scraping (already working)

## Constraints Recap
| Constraint | Status |
|------------|--------|
| Cloud → Home Lab traffic blocked | ✅ Respected (no push from cloud) |
| Home Lab → Cloud traffic allowed | ✅ All log collection is home lab polling cloud |
| No extra cloud software | ✅ Only app code changes on cloud server |
| Keep monitoring in home lab | ✅ Grafana/Prometheus/Alloy/Loki all run locally |
| No traces (for now) | ✅ Excluded entirely |

## Architecture
```
CLOUD SERVER (existing app only)
┌─────────────────────────────────────────┐
│ Fantasy FRC Go App                      │
│ • Exposes /logs endpoint (JSON logs)    │
│ • Protected by IP + METRIC_SECRET auth  │
│ • Existing /metrics endpoint (unchanged)│
└─────────────────────────────────────────┘
            ↕ Home Lab → Cloud Pull (Allowed)

HOME LAB MONITORING SERVER
┌─────────────────────────────────────────┐
│ Grafana Alloy                           │
│ • Polls cloud /logs every 10s           │
│ • Sends logs to local Loki               │
├─────────────────────────────────────────┤
│ Loki (local)                            │
│ • Stores all application logs            │
├─────────────────────────────────────────┤
│ Grafana (local)                         │
│ • Queries local Loki for log visualization│
│ • Queries local Prometheus for metrics   │
├─────────────────────────────────────────┤
│ Prometheus (local, existing)            │
│ • Continues scraping cloud /metrics      │
└─────────────────────────────────────────┘
```

## Network Flow
All log traffic is **home lab → cloud pull** (no cloud → home lab traffic):

| Source | Destination | Protocol/Port | Purpose |
|--------|-------------|---------------|---------|
| Home Lab Alloy | Cloud App | HTTP/{app-port}/logs | Pull JSON logs |
| Home Lab Grafana | Cloud App | HTTP/{app-port}/metrics | Scrape Prometheus metrics (existing) |

## Implementation Steps

### Part 1: Cloud App Changes (No Extra Software)
Only code updates to the existing Go app in `/home/jmisbach/fantasy-frc-web/server`:

#### 1.1 Switch to Structured JSON Logging
Update `server/log/log.go` to configure `slog` to output JSON with consistent fields:
- Required fields: `time` (RFC3339), `level`, `msg`, `service=fantasy-frc`
- Optional: Add existing context fields (user ID, request ID from correlation middleware)

#### 1.2 Add In-Memory Log Buffer
- Add a thread-safe ring buffer (size ~1000 entries) to store recent structured log entries
- Buffer evicts oldest entries when full
- Populated by overriding the default `slog` handler to write to both existing output and the buffer

#### 1.3 Add Protected `/logs` Endpoint
Add a new route in `server/server.go`:
- **Path**: `/logs`
- **Auth**: Reuse existing `METRIC_SECRET` (same as `/metrics` endpoint)
- **IP Restriction**: Only accept requests from home lab's public IP
- **Query Param**: `?last_seen=<RFC3339-timestamp>` to return only logs after that time (avoids duplicates for Alloy polling)
- **Response**: JSON lines (one JSON object per line) of log entries

Example `/logs` response:
```json
{"time":"2026-04-24T14:00:00Z","level":"INFO","msg":"Starting Server","service":"fantasy-frc"}
{"time":"2026-04-24T14:00:01Z","level":"DEBUG","msg":"Draft daemon started","draft_count":3}
```

### Part 2: Home Lab Monitoring Server Changes
Deploy only two new components (both in home lab, no cloud changes):

#### 2.1 Deploy Loki (if not present)
- Run Loki locally on home lab (no cross-network traffic)
- Default local config is sufficient for initial setup
- Expose port `3100` (localhost only)

#### 2.2 Deploy Grafana Alloy
Install Alloy on home lab (only extra software, kept locally):
- Configure `http.poller` to scrape cloud app's `/logs` endpoint:
  ```yaml
  # alloy-config.yaml
  http.poller:
    - name: fantasy-frc-logs
      url: http://<cloud-server-ip>:<app-port>/logs?last_seen={{ .lastSeen }}
      method: GET
      headers:
        - name: Authorization
          value: Bearer <your-METRIC_SECRET>
      poll_interval: 10s
      transformers:
        - json:
            expressions:
              time: time
              level: level
              message: msg
      output:
        - loki.write

  loki.write:
    endpoint:
      url: http://localhost:3100/loki/api/v1/push
  ```

#### 2.3 Update Grafana Datasources
- Add local Loki as a datasource in home lab Grafana (no cross-network configuration needed)
- Existing Prometheus datasource remains unchanged

## Tradeoffs
| Pros | Cons |
|------|------|
| No extra cloud software | Requires Alloy installation on home lab |
| All monitoring in home lab | Non-standard OTel approach (bypasses OTel logs SDK) |
| Complies with all network constraints | No trace support (excluded by design) |
| Preserves existing Prometheus setup | Custom /logs endpoint requires maintenance |

## Approval Status
- [ ] Plan approved for implementation
- [ ] Pending user sign-off
