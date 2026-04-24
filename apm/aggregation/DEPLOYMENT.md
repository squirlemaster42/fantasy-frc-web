# Log Aggregation Deployment Guide

*Fantasy FRC Web · Updated: 2026-04-24*

## Overview

This document describes where each component of the log aggregation system should be deployed and how to configure them.

## Architecture Recap

```
┌─────────────────────────────────────────────────────────────┐
│                    CLOUD SERVER                            │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Fantasy FRC Go App (updated)                       │    │
│  │ • Structured JSON logging to stdout → journalctl   │    │
│  │ • Ring buffer stores last 1000 log entries         │    │
│  │ • Exposes /logs endpoint (protected by auth)       │    │
│  │ • Existing /metrics endpoint (unchanged)           │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                              ↕
                    Home Lab → Cloud (pull only)
                              ↕
┌─────────────────────────────────────────────────────────────┐
│                 HOME LAB MONITORING SERVER                 │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Log Poller Script                                  │    │
│  │ • /opt/fantasy-frc-log-poller/poll.sh              │    │
│  │ • Runs every 10s via systemd timer                │    │
│  │ • Polls cloud /logs endpoint                      │    │
│  │ • Maintains state (last_seen timestamp)            │    │
│  │ • Outputs to /var/lib/fantasy-frc-monitoring/     │    │
│  └─────────────────────────────────────────────────────┘    │
│                           ↕                                │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Grafana Alloy                                      │    │
│  │ • Reads polled logs from file                      │    │
│  │ • Parses JSON log entries                          │    │
│  │ • Sends to Loki                                    │    │
│  └─────────────────────────────────────────────────────┘    │
│                           ↕                                │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Loki (port 3100)                                   │    │
│  │ • Stores all application logs                      │    │
│  └─────────────────────────────────────────────────────┘    │
│                           ↕                                │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Grafana                                            │    │
│  │ • Queries Loki for log visualization               │    │
│  │ • Queries Prometheus for metrics (existing)         │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Deployment Locations

### 1. Cloud Server (Your VPS/Cloud Instance)

**Components to Deploy:**

| Component | Location on Server | Action Required |
|-----------|-------------------|-----------------|
| Updated Go Binary | `/home/<user>/fantasy-frc/` or your current deployment path | Replace existing binary |
| Environment Variables | `.env` file or systemd service file | Add if needed (existing vars are sufficient) |

**Files Modified (already done in repo):**
- `server/log/buffer.go` (new)
- `server/log/handler.go` (new)
- `server/log/log.go` (modified)
- `server/main.go` (modified)
- `server/server.go` (modified)
- `server/handler/logsHandler.go` (new)

**Deployment Steps:**

1. **Build the updated app:**
   ```bash
   cd /home/jmisbach/fantasy-frc-web/server
   go build -o fantasy-frc .
   ```

2. **Copy binary to deployment location:**
   ```bash
   # Example - adjust to your deployment path
   sudo systemctl stop fantasy-frc
   cp fantasy-frc /opt/fantasy-frc/
   sudo systemctl start fantasy-frc
   ```

3. **Verify the /logs endpoint is working:**
   ```bash
   # From the cloud server itself
   curl -H "Authorization: Bearer $METRIC_SECRET" \
        http://localhost:<port>/logs
   ```

4. **Check journalctl for JSON logs:**
   ```bash
   journalctl -u fantasy-frc --no-pager -n 5
   # Should show JSON-formatted log entries with time, level, msg, service fields
   ```

**Required Environment Variables (already should be set):**
- `METRIC_SECRET` - Used to protect /logs and /metrics endpoints

---

### 2. Home Lab Monitoring Server

**Components to Deploy:**

| Component | Install Location | Description |
|-----------|-----------------|-------------|
| **Log Poller Script** | `/opt/fantasy-frc-log-poller/poll.sh` | Shell script that polls cloud /logs endpoint |
| **State Directory** | `/var/lib/fantasy-frc-monitoring/` | Stores last_seen timestamp and polled logs |
| **Systemd Service** | `/etc/systemd/system/fantasy-frc-log-poll.service` | Defines the poller as a service |
| **Systemd Timer** | `/etc/systemd/system/fantasy-frc-log-poll.timer` | Runs poller every 10 seconds |
| **Grafana Alloy** | Package install or Docker | Collects and forwards logs to Loki |
| **Loki** | Docker or binary | Stores logs |
| **Grafana** | Existing (already running) | Visualization (add Loki datasource) |

#### 2.1 Deploy Log Poller Script

```bash
# Create directory
sudo mkdir -p /opt/fantasy-frc-log-poller

# Copy script from repo
sudo cp /path/to/repo/apm/aggregation/scripts/poll.sh \
         /opt/fantasy-frc-log-poller/poll.sh

# Make executable
sudo chmod +x /opt/fantasy-frc-log-poller/poll.sh

# Edit the script with your actual values
sudo nano /opt/fantasy-frc-log-poller/poll.sh
```

**Edit these values in the script:**
```bash
CLOUD_URL="http://your-cloud-ip:your-port/logs"
SECRET="your-actual-METRIC_SECRET"
```

#### 2.2 Deploy Systemd Units

```bash
# Copy service and timer files
sudo cp /path/to/repo/apm/aggregation/systemd/fantasy-frc-log-poll.service \
         /etc/systemd/system/
sudo cp /path/to/repo/apm/aggregation/systemd/fantasy-frc-log-poll.timer \
         /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload

# Enable and start the timer
sudo systemctl enable fantasy-frc-log-poll.timer
sudo systemctl start fantasy-frc-log-poll.timer

# Verify timer is active
systemctl list-timers | grep fantasy-frc
```

#### 2.3 Deploy Loki

**Option A: Docker (Recommended)**
```bash
docker run -d \
  --name loki \
  -p 3100:3100 \
  grafana/loki:latest \
  -config.file=/etc/loki/local-config.yaml
```

**Option B: Binary**
```bash
# Download Loki binary
curl -O -L "https://github.com/grafana/loki/releases/download/v2.9.0/loki-linux-amd64.zip"
unzip loki-linux-amd64.zip
sudo mv loki-linux-amd64 /usr/local/bin/loki

# Run Loki
loki --config.file=/etc/loki/local-config.yaml &
```

Verify Loki is running:
```bash
curl http://localhost:3100/ready
# Should return "ready"
```

#### 2.4 Deploy Grafana Alloy

**Install Alloy (Ubuntu/Debian):**
```bash
curl -fsSL https://apt.grafana.com/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/grafana.gpg
echo "deb [signed-by=/etc/apt/keyrings/grafana.gpg] https://apt.grafana.com stable main" | \
  sudo tee /etc/apt/sources.list.d/grafana.list
sudo apt update
sudo apt install alloy
```

**Configure Alloy:**
```bash
# Copy config from repo
sudo cp /path/to/repo/apm/aggregation/alloy/config.yaml \
         /etc/alloy/config.yaml

# Edit config if needed (Loki URL, file paths)
sudo nano /etc/alloy/config.yaml
```

**Start Alloy:**
```bash
sudo systemctl enable alloy
sudo systemctl start alloy
```

#### 2.5 Configure Grafana

1. **Add Loki Datasource:**
   - Navigate to Grafana web UI (http://home-lab-ip:3000)
   - Go to Configuration → Data Sources → Add data source
   - Select "Loki"
   - URL: `http://localhost:3100`
   - Click "Save & Test"

2. **Create Dashboard:**
   - Create new dashboard
   - Add log panel
   - Query: `{job="fantasy-frc"}`
   - Select Loki datasource

---

## Verification Checklist

### Cloud Server:
- [ ] App builds successfully
- [ ] App starts without errors
- [ ] Logs appear in journalctl as JSON
- [ ] `/logs` endpoint returns 200 with auth header
- [ ] `/logs` endpoint returns 401 without auth header

### Home Lab:
- [ ] Poller script runs manually without errors
- [ ] State file (`/var/lib/fantasy-frc-monitoring/last_seen.txt`) updates after poll
- [ ] Log file (`/var/lib/fantasy-frc-monitoring/latest_logs.json`) contains logs
- [ ] Systemd timer fires every 10 seconds
- [ ] Loki responds on port 3100
- [ ] Alloy starts without errors
- [ ] Grafana can query Loki datasource
- [ ] Logs appear in Grafana dashboard

---

## Quick Verification Commands

**Test poller script manually:**
```bash
sudo /opt/fantasy-frc-log-poller/poll.sh
cat /var/lib/fantasy-frc-monitoring/latest_logs.json | head -5
```

**Check systemd timer status:**
```bash
systemctl status fantasy-frc-log-poll.timer
journalctl -u fantasy-frc-log-poll.service -f
```

**Test Loki:**
```bash
curl "http://localhost:3100/loki/api/v1/query?query={job=\"fantasy-frc\"}"
```

**Check Alloy logs:**
```bash
sudo journalctl -u alloy -f
```

---

## File Reference

All deployment files are in the repo at:
- **Poller script:** `apm/aggregation/scripts/poll.sh`
- **Systemd units:** `apm/aggregation/systemd/`
- **Alloy config:** `apm/aggregation/alloy/config.yaml`

Copy these to the appropriate locations on your home lab server as described above.
