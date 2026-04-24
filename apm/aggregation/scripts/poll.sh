#!/bin/bash
STATE_DIR="/var/lib/fantasy-frc-monitoring"
STATE_FILE="$STATE_DIR/last_seen.txt"
OUTPUT_FILE="$STATE_DIR/latest_logs.json"
CLOUD_URL="http://<cloud-ip>:<port>/logs"
SECRET="<METRIC_SECRET>"

mkdir -p "$STATE_DIR"

LAST_SEEN=""
if [ -f "$STATE_FILE" ]; then
    LAST_SEEN=$(cat "$STATE_FILE")
fi

URL="$CLOUD_URL"
if [ -n "$LAST_SEEN" ]; then
    ESCAPED=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$LAST_SEEN'))" 2>/dev/null || echo "$LAST_SEEN")
    URL="${CLOUD_URL}?last_seen=${ESCAPED}"
fi

RESPONSE=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $SECRET" "$URL")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ] && [ -n "$BODY" ]; then
    echo "$BODY" > "$OUTPUT_FILE"
    
    NEWEST=$(echo "$BODY" | grep -v '^$' | tail -1 | jq -r '.time' 2>/dev/null)
    if [ -n "$NEWEST" ] && [ "$NEWEST" != "null" ]; then
        echo "$NEWEST" > "$STATE_FILE"
    fi
fi
