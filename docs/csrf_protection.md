# CSRF Protection Implementation

## Overview

This application implements Cross-Site Request Forgery (CSRF) protection using the Echo framework's built-in middleware and template integration.

## Implementation Details

### Middleware Configuration

**Global CSRF Protection** (server.go:40-44):
```go
app.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
    Skipper: func(c echo.Context) bool {
        return c.Request().URL.Path == "/tbaWebhook"
    },
}))
```

**Protected Routes** (server.go:55-56):
```go
protected := app.Group("/u", auth.Authenticate)
protected.Use(middleware.CSRF())
```

### Token Generation and Storage

- **Token Source**: Echo framework generates CSRF tokens automatically
- **Token Storage**: Stored in Echo context as `c.Get("csrf")`
- **Token Format**: Random string, unique per session
- **Token Validation**: Automatic validation by middleware

### Template Integration

**CSRF Token Component** (view/partials/csrf.templ):
```templ
templ CSRFToken(c echo.Context) {
    <input type="hidden" name="csrf_token" value={ c.Get("csrf").(string) } />
}
```

**Form Integration**:
All protected forms include `@partials.CSRFToken(c)` which renders:
```html
<input type="hidden" name="csrf_token" value="generated_token_here">
```

### Protected Endpoints

**Public Routes with CSRF**:
- `POST /login` - User authentication
- `POST /register` - User registration
- `POST /logout` - User logout

**Protected Routes** (`/u/*` prefix):
- `POST /u/createDraft` - Create new draft
- `POST /u/draft/:id/updateDraft` - Update draft
- `POST /u/draft/:id/startDraft` - Start draft
- `POST /u/draft/:id/makePick` - Make team selection
- `POST /u/draft/:id/invitePlayer` - Invite player
- `POST /u/acceptInvite` - Accept draft invitation
- `POST /u/team/score` - Get team score
- `POST /u/searchPlayers` - Search for players
- `POST /u/draft/:id/skipPickToggle` - Toggle skip pick
- `POST /u/admin/processCommand` - Admin commands

**Exempted Routes**:
- `POST /tbaWebhook` - External webhook (no CSRF required)

### Security Features

1. **Session-Based Tokens**: Each session gets unique CSRF token
2. **Automatic Validation**: Middleware validates tokens on POST requests
3. **Origin Validation**: Cross-origin requests blocked
4. **Token Expiration**: Tokens expire with session
5. **Form Integration**: All forms automatically include tokens

## Testing Procedures

### Automated Testing

**Run Go Tests**:
```bash
cd server
make test-csrf
```

**Test Coverage**:
- Valid CSRF token acceptance
- Missing CSRF token rejection
- Invalid CSRF token rejection
- Cross-origin request blocking
- Token uniqueness verification
- Token expiration handling

### Manual Testing

**Run Bash Script**:
```bash
cd server
make test-csrf-manual
```

**Browser Testing**:
Follow checklist in `docs/csrf_testing_checklist.md`

### Test Files

- `server/csrf_test.go` - Go integration tests
- `scripts/test_csrf.sh` - Bash automation script
- `docs/csrf_testing_checklist.md` - Manual testing checklist

## Security Considerations

### Current Implementation Strengths
✅ All POST endpoints protected
✅ Automatic token generation and validation
✅ Template integration prevents missing tokens
✅ Session-based token uniqueness
✅ Cross-origin request protection
✅ Webhook exemption for external integrations

### Security Recommendations
🔍 **Consider implementing**:
- Double-submit cookie pattern for additional protection
- Custom token rotation for long-lived sessions
- Request origin validation for sensitive operations
- CSRF token refresh on authentication

🔒 **Monitor for**:
- CSRF token leakage in logs
- Token reuse across sessions
- Missing tokens in new forms
- Cross-origin request attempts

## Troubleshooting

### Common Issues

**"403 Forbidden on valid request"**:
1. Check if CSRF token is present in form
2. Verify token is correctly extracted from context
3. Ensure cookies are being sent with request
4. Check token format (no URL encoding issues)

**"CSRF token not found"**:
1. Verify page includes `@partials.CSRFToken(c)`
2. Check if context is properly passed to template
3. Ensure middleware is applied to route
4. Verify user is authenticated for protected routes

**"Webhook endpoint blocked"**:
1. Verify webhook is in CSRF skipper
2. Check route registration order
3. Ensure middleware configuration is correct

### Debug Commands

```bash
# Check form rendering
curl -s http://localhost:3000/login | grep csrf_token

# Test with verbose output
curl -v -X POST -d "username=test&password=test&csrf_token=TOKEN" \
  http://localhost:3000/login

# Run specific test
go test -v ./... -run TestCSRFProtection
```

## Compliance

This CSRF implementation follows security best practices:
- OWASP CSRF Prevention Cheat Sheet
- SameSite cookie attributes
- Secure token generation
- Proper validation and error handling
- Defense in depth (multiple protection layers)

## Maintenance

**Regular Tasks**:
- Run automated tests after code changes
- Verify new forms include CSRF tokens
- Check middleware configuration updates
- Monitor security advisories for CSRF vulnerabilities

**Update Procedures**:
1. Test all endpoints after framework updates
2. Verify template integration still works
3. Run full test suite
4. Update documentation as needed