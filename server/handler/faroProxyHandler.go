package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"server/log"
	"server/model"

	"github.com/labstack/echo/v4"
)

type faroBatch struct {
	Batch []faroItem `json:"batch"`
}

type faroItem struct {
	Type       string          `json:"type"`
	TimestampMs int64           `json:"timestampMs"`
	Meta       faroMeta        `json:"meta"`
	Payload    json.RawMessage `json:"payload"`
}

type faroMeta struct {
	SDK  faroSDKMeta  `json:"sdk"`
	App  faroAppMeta  `json:"app"`
	Page faroPageMeta `json:"page,omitempty"`
}

type faroSDKMeta struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type faroAppMeta struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type faroPageMeta struct {
	Url string `json:"url,omitempty"`
}

var allowedEventTypes = map[string]bool{
	"exception":     true,
	"log":           true,
	"event":         true,
	"measurement":   true,
	"session_start": true,
	"view":          true,
}

var (
	rateLimiter     = &sync.Map{}
	rateLimitWindow  = 1 * time.Minute
	rateLimitMaxReqs = 10
)

type rateLimitEntry struct {
	count    int
	resetAt  time.Time
}

var (
	emailRegex    = regexp.MustCompile(`(?i)[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}`)
	phoneRegex    = regexp.MustCompile(`(?i)(\+?1[\s.-]?)?(\([0-9]{3}\)|[0-9]{3})[\s.-]?[0-9]{3}[\s.-]?[0-9]{4}`)
	ipv4Regex    = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
)

func (h *Handler) HandleFaroProxy(c echo.Context) error {
	if c.Request().Method != http.MethodPost && c.Request().Method != http.MethodOptions {
		return c.NoContent(http.StatusMethodNotAllowed)
	}

	if c.Request().Method == http.MethodOptions {
		return c.NoContent(http.StatusNoContent)
	}

	contentType := c.Request().Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return c.NoContent(http.StatusUnsupportedMediaType)
	}

	faroToken := c.Request().Header.Get("X-Faro-Token")
	if faroToken == "" {
		log.Warn(c.Request().Context(), "Missing Faro token")
		return c.NoContent(http.StatusUnauthorized)
	}

	payload, ok := h.validateFaroToken(faroToken)
	if !ok {
		log.Warn(c.Request().Context(), "Invalid Faro token")
		return c.NoContent(http.StatusUnauthorized)
	}

	if payload.SessionToken != "" {
		isValid := model.ValidateSessionToken(h.Database, payload.SessionToken)
		if !isValid {
			log.Warn(c.Request().Context(), "Invalid session token in Faro token")
			return c.NoContent(http.StatusUnauthorized)
		}
	}

	ip := c.RealIP()
	if !checkRateLimit(ip) {
		log.Warn(c.Request().Context(), "Rate limit exceeded for Faro proxy", "ip", ip)
		return c.NoContent(http.StatusTooManyRequests)
	}

	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil || len(bodyBytes) > 1024*1024 {
		return c.NoContent(http.StatusRequestEntityTooLarge)
	}

	var batch faroBatch
	if err := json.Unmarshal(bodyBytes, &batch); err != nil {
		log.Warn(c.Request().Context(), "Failed to decode Faro batch", "error", err)
		return c.NoContent(http.StatusBadRequest)
	}

	if len(batch.Batch) == 0 || len(batch.Batch) > 50 {
		return c.NoContent(http.StatusBadRequest)
	}

	for i := range batch.Batch {
		item := &batch.Batch[i]

		if item.TimestampMs <= 0 || time.Now().UnixMilli()-item.TimestampMs > 3600000 {
			return c.NoContent(http.StatusBadRequest)
		}

		if !allowedEventTypes[item.Type] {
			return c.NoContent(http.StatusBadRequest)
		}

		if item.Meta.SDK.Name != "faro-web" {
			return c.NoContent(http.StatusBadRequest)
		}

		if item.Meta.App.Name != "fantasy-frc-web" {
			return c.NoContent(http.StatusBadRequest)
		}

		item.Payload = json.RawMessage(redactPII(string(item.Payload)))
		item.Payload = json.RawMessage(truncateStrings(string(item.Payload), 10000))
	}

	validatedBytes, err := json.Marshal(batch)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	go forwardToAlloy(h.FaroAlloyInternalURL, h.FaroAlloyBearerToken, validatedBytes)

	return c.NoContent(http.StatusAccepted)
}

func checkRateLimit(ip string) bool {
	now := time.Now()

	val, _ := rateLimiter.LoadOrStore(ip, &rateLimitEntry{
		count:   0,
		resetAt: now.Add(rateLimitWindow),
	})

	entry := val.(*rateLimitEntry)

	if now.After(entry.resetAt) {
		entry.count = 0
		entry.resetAt = now.Add(rateLimitWindow)
	}

	entry.count++
	return entry.count <= rateLimitMaxReqs
}

func redactPII(input string) string {
	result := emailRegex.ReplaceAllString(input, "[REDACTED_EMAIL]")
	result = phoneRegex.ReplaceAllString(result, "[REDACTED_PHONE]")
	result = ipv4Regex.ReplaceAllString(result, "[REDACTED_IP]")
	return result
}

func truncateStrings(input string, maxLen int) string {
	var result bytes.Buffer
	inString := false
	stringStart := 0

	for i := 0; i < len(input); i++ {
		if input[i] == '"' {
			if !inString {
				inString = true
				stringStart = i + 1
			} else {
				strLen := i - stringStart
				if strLen > maxLen {
					result.WriteString(input[stringStart : stringStart+maxLen])
					result.WriteString("...[TRUNCATED]")
					inString = false
					continue
				}
				inString = false
			}
		}
		if !inString || i < stringStart+maxLen {
			result.WriteByte(input[i])
		}
	}

	return result.String()
}

func forwardToAlloy(url, bearerToken string, payload []byte) {
	if url == "" {
		log.WarnNoContext("FARO_ALLOY_INTERNAL_URL not set, skipping forward")
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		log.ErrorNoContext("Failed to create Alloy forward request", "error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.ErrorNoContext("Failed to forward to Alloy", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.WarnNoContext("Alloy returned error", "status", resp.StatusCode, "body", string(body))
	}
}
