# Error Handling

Comprehensive guide to error codes, responses, and troubleshooting.

## 🚨 Error Response Format

### Standard Error Structure
```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error description",
    "details": {
      "field": "specific_field",
      "issue": "detailed explanation",
      "suggestion": "how to fix"
    },
    "timestamp": "2025-12-11T10:30:00Z",
    "requestId": "req_abc123"
  }
}
```

### HTTP Status Codes
| Status | Category | Usage |
|--------|----------|-------|
| 400 | Client Error | Invalid requests, validation failures |
| 401 | Authentication | Missing or invalid authentication |
| 403 | Authorization | Insufficient permissions |
| 404 | Not Found | Resource does not exist |
| 409 | Conflict | Resource state conflicts |
| 429 | Rate Limit | Too many requests |
| 500 | Server Error | Internal server problems |

## 🔐 Authentication Errors

### `UNAUTHORIZED`
User not authenticated or session expired.

**HTTP Status:** 401
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Authentication required",
    "details": {
      "suggestion": "Please log in to access this resource"
    }
  }
}
```

**Causes:**
- Missing session cookie
- Expired session
- Invalid session token

**Solutions:**
- Redirect to login page
- Clear invalid session data
- Prompt user to re-authenticate

### `INVALID_CREDENTIALS`
Username or password incorrect.

**HTTP Status:** 401
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Invalid username or password",
    "details": {
      "suggestion": "Check your credentials and try again"
    }
  }
}
```

## 🚫 Authorization Errors

### `FORBIDDEN`
User lacks permission for resource.

**HTTP Status:** 403
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "FORBIDDEN",
    "message": "Access denied",
    "details": {
      "resource": "draft_123",
      "requiredRole": "owner",
      "userRole": "player"
    }
  }
}
```

### `NOT_DRAFT_OWNER`
User is not the draft owner.

**HTTP Status:** 403
**Common Scenarios:**
- Attempting to modify draft settings
- Starting or stopping drafts
- Inviting players (if not owner)

## 📋 Validation Errors

### `VALIDATION_ERROR`
Request data fails validation rules.

**HTTP Status:** 400
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input data",
    "details": {
      "fields": [
        {
          "name": "username",
          "issue": "Username must be 3-20 characters",
          "value": "ab"
        },
        {
          "name": "email",
          "issue": "Invalid email format",
          "value": "not-an-email"
        }
      ]
    }
  }
}
```

### Common Validation Rules
| Field | Rules | Error Message |
|-------|-------|---------------|
| `username` | 3-20 chars, alphanumeric | Username must be 3-20 characters |
| `password` | 8+ chars, special char | Password must be at least 8 characters |
| `displayName` | 1-100 chars | Display name required |
| `startTime` | Future date | Start time must be in the future |
| `interval` | 30-300 seconds | Pick interval must be 30-300 seconds |

## 🏗️ Resource Errors

### `NOT_FOUND`
Requested resource does not exist.

**HTTP Status:** 404
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Resource not found",
    "details": {
      "resourceType": "draft",
      "resourceId": "999999",
      "suggestion": "Check the resource ID and try again"
    }
  }
}
```

**Resource Types:**
- `draft` - Draft not found
- `user` - User not found
- `team` - Team not found
- `match` - Match not found

### `DRAFT_NOT_FOUND`
Specific to draft-related operations.

**HTTP Status:** 404
**Common Scenarios:**
- Accessing draft profile
- Making picks in non-existent draft
- Viewing draft scores

## ⚔️ Conflict Errors

### `CONFLICT`
Resource state conflicts with operation.

**HTTP Status:** 409
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "CONFLICT",
    "message": "Resource state conflict",
    "details": {
      "currentState": "PICKING",
      "requiredState": "FILLING",
      "operation": "update_draft_settings"
    }
  }
}
```

### `DRAFT_ALREADY_STARTED`
Cannot modify started draft.

**HTTP Status:** 409
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "DRAFT_ALREADY_STARTED",
    "message": "Draft has already started",
    "details": {
      "currentState": "PICKING",
      "suggestion": "Draft settings can only be modified before starting"
    }
  }
}
```

### `USERNAME_TAKEN`
Username already exists.

**HTTP Status:** 409
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "USERNAME_TAKEN",
    "message": "Username already taken",
    "details": {
      "username": "player1",
      "suggestion": "Choose a different username"
    }
  }
}
```

## 🎯 Draft-Specific Errors

### `INVALID_PICK`
Team selection violates rules.

**HTTP Status:** 400
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "INVALID_PICK",
    "message": "Invalid team selection",
    "details": {
      "teamTbaId": "frc1234",
      "reason": "Team already picked",
      "pickedBy": "player2",
      "pickTime": "2025-12-11T10:05:00Z"
    }
  }
}
```

**Pick Validation Rules:**
- Team exists in database
- Team not already picked
- Team is at valid event
- It's player's turn to pick
- Pick time hasn't expired

### `PICK_EXPIRED`
Player missed pick time limit.

**HTTP Status:** 400
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "PICK_EXPIRED",
    "message": "Pick time expired",
    "details": {
      "expiredAt": "2025-12-11T10:07:00Z",
      "playerId": 456,
      "pickNumber": 5,
      "autoSkipped": true
    }
  }
}
```

### `NOT_YOUR_TURN`
Player attempting to pick out of turn.

**HTTP Status:** 400
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "NOT_YOUR_TURN",
    "message": "It's not your turn to pick",
    "details": {
      "currentPlayerId": 789,
      "currentPlayerUsername": "player2",
      "yourPlayerId": 456,
      "yourPickOrder": 3
    }
  }
}
```

## 🚨 Server Errors

### `INTERNAL_ERROR`
Unexpected server error.

**HTTP Status:** 500
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "An unexpected error occurred",
    "details": {
      "requestId": "req_abc123",
      "timestamp": "2025-12-11T10:30:00Z",
      "suggestion": "Please try again later"
    }
  }
}
```

### `DATABASE_ERROR`
Database operation failed.

**HTTP Status:** 500
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "DATABASE_ERROR",
    "message": "Database operation failed",
    "details": {
      "operation": "INSERT",
      "table": "picks",
      "suggestion": "Please try again"
    }
  }
}
```

### `EXTERNAL_API_ERROR`
External service failure.

**HTTP Status:** 502
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "EXTERNAL_API_ERROR",
    "message": "External service unavailable",
    "details": {
      "service": "The Blue Alliance API",
      "endpoint": "/team/frc1234",
      "httpStatus": 503
    }
  }
}
```

## 📊 Rate Limiting

### `RATE_LIMIT_EXCEEDED`
Too many requests in time window.

**HTTP Status:** 429
**Example Response:**
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded",
    "details": {
      "limit": 60,
      "window": "1 minute",
      "retryAfter": 45,
      "suggestion": "Please wait before making more requests"
    }
  }
}
```

**Rate Limits by Endpoint:**
| Endpoint | Limit | Window |
|----------|-------|--------|
| `/login` | 5 requests | 1 minute |
| `/register` | 3 requests | 5 minutes |
| `/u/draft/*/makePick` | 10 requests | 1 minute |
| `/u/searchPlayers` | 20 requests | 1 minute |

## 🛠️ Client Error Handling

### JavaScript Example
```javascript
async function makeApiCall(url, options = {}) {
  try {
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers
      }
    });

    const data = await response.json();

    if (!response.ok) {
      handleApiError(response.status, data.error);
      return null;
    }

    return data;
  } catch (error) {
    handleNetworkError(error);
    return null;
  }
}

function handleApiError(status, error) {
  switch (error.code) {
    case 'UNAUTHORIZED':
      redirectToLogin();
      break;
    case 'VALIDATION_ERROR':
      showValidationErrors(error.details.fields);
      break;
    case 'DRAFT_ALREADY_STARTED':
      showErrorMessage('Cannot modify started draft');
      break;
    case 'RATE_LIMIT_EXCEEDED':
      showRateLimitMessage(error.details.retryAfter);
      break;
    default:
      showGenericError(error.message);
  }
}
```

### Retry Logic
```javascript
async function retryApiCall(url, options, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const result = await makeApiCall(url, options);
      if (result) return result;
    } catch (error) {
      if (i === maxRetries - 1) throw error;
      
      // Exponential backoff
      const delay = Math.pow(2, i) * 1000;
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }
}
```

## 📝 Error Logging

### Server-Side Logging
```go
// Log error with context
slog.Error("API error occurred",
    "code", "VALIDATION_ERROR",
    "message", "Invalid input data",
    "requestId", "req_abc123",
    "userId", 123,
    "endpoint", "/u/createDraft",
    "fields", validationErrors,
)
```

### Client-Side Logging
```javascript
// Log error for debugging
console.error('API Error:', {
  code: error.code,
  message: error.message,
  requestId: error.requestId,
  timestamp: new Date().toISOString(),
  userAgent: navigator.userAgent,
  url: window.location.href
});
```

---

*TODO: Add error code reference table, troubleshooting guide, and monitoring dashboard documentation*