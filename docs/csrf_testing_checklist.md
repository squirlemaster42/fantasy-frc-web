# CSRF Testing Checklist

## Pre-Test Setup
- [ ] Server running on localhost:3000
- [ ] Test database populated with test users
- [ ] Browser dev tools open
- [ ] Network tab monitoring enabled

## Automated Testing
- [ ] Run `./scripts/test_csrf.sh`
- [ ] Run `go test -v ./... -run TestCSRF`
- [ ] All automated tests pass

## Form Rendering Tests
- [ ] Login form contains CSRF token
- [ ] Register form contains CSRF token
- [ ] Create draft form contains CSRF token
- [ ] Update draft form contains CSRF token
- [ ] Player invite form contains CSRF token
- [ ] Team score form contains CSRF token
- [ ] Admin console form contains CSRF token

## Positive Tests (Valid CSRF)
- [ ] Login with valid token succeeds (200/302)
- [ ] Registration with valid token succeeds (200/302)
- [ ] Draft creation with valid token succeeds (200/302)
- [ ] Draft update with valid token succeeds (200/302)
- [ ] Player invitation with valid token succeeds (200/302)
- [ ] Team score lookup with valid token succeeds (200/302)

## Negative Tests (Invalid/Missing CSRF)
- [ ] Login without CSRF token fails (403)
- [ ] Login with invalid CSRF token fails (403)
- [ ] Registration without CSRF token fails (403)
- [ ] Registration with invalid CSRF token fails (403)
- [ ] Draft creation without CSRF token fails (403)
- [ ] Draft creation with invalid CSRF token fails (403)
- [ ] Player invitation without CSRF token fails (403)
- [ ] Team score lookup without CSRF token fails (403)

## Edge Cases
- [ ] Webhook endpoint works without CSRF (200/204)
- [ ] Cross-origin request rejected (403)
- [ ] Expired session rejected (401/403)
- [ ] Token reuse rejected (403)
- [ ] AJAX/HTMX requests protected
- [ ] Multiple browser tabs work correctly

## Security Headers Verification
- [ ] CSRF tokens are unique per session
- [ ] Session cookies have SameSite attribute
- [ ] Session cookies have Secure attribute (when using HTTPS)
- [ ] CORS headers properly configured
- [ ] X-Frame-Options header present

## Browser Testing Steps

### Manual CSRF Testing
1. **Open application in browser**
   - Navigate to http://localhost:3000
   - Open browser dev tools (F12)
   - Go to Network tab

2. **Test login form**
   - Go to /login
   - Inspect form element in dev tools
   - Verify `<input type="hidden" name="csrf_token" value="...">` exists
   - Submit form with valid credentials
   - Check network request includes csrf_token parameter

3. **Test CSRF token removal**
   - Refresh login page
   - Use dev tools to remove CSRF token from form
   - Submit form
   - Verify request fails (403 Forbidden)

4. **Test invalid CSRF token**
   - Refresh login page
   - Use dev tools to change CSRF token value
   - Submit form
   - Verify request fails (403 Forbidden)

5. **Test cross-origin request**
   - Open browser console
   - Execute: `fetch('http://localhost:3000/login', {method: 'POST', headers: {'Content-Type': 'application/x-www-form-urlencoded'}, body: 'username=test&password=test', credentials: 'include'})`
   - Verify request fails (CORS or CSRF error)

6. **Test authenticated endpoints**
   - Login successfully
   - Navigate to /u/createDraft
   - Verify CSRF token in form
   - Test form submission with/without token

### AJAX/HTMX Testing
1. **Test HTMX requests**
   - Trigger any HTMX-powered action
   - Check network request includes CSRF token
   - Verify request succeeds with valid token

2. **Test AJAX without CSRF**
   - Use browser console to make AJAX request
   - Omit CSRF token
   - Verify request fails

## Security Scenarios

### Scenario 1: Basic CSRF Attack
1. Attacker creates malicious site: https://evil-site.com
2. Victim is logged into http://localhost:3000
3. Victim visits evil-site.com
4. Site submits form to localhost:3000
5. **Expected**: Request blocked by CSRF protection

### Scenario 2: Token Extraction
1. Attacker tries to read CSRF token from page
2. **Expected**: Same-origin policy prevents access

### Scenario 3: Session Fixation
1. Attacker provides victim with session ID
2. Victim logs in with provided session
3. **Expected**: Session regeneration prevents fixation

## Test Results Documentation

### Automated Test Results
```
Date: [DATE]
Tester: [NAME]
Environment: [DEV/STAGING/PROD]

Script Results:
- Form Rendering: [PASS/FAIL]
- Login Endpoint: [PASS/FAIL]
- Register Endpoint: [PASS/FAIL]
- Protected Endpoints: [PASS/FAIL]
- Webhook Exemption: [PASS/FAIL]

Go Test Results:
- TestCSRFProtection: [PASS/FAIL]
- TestCSRFTokenUniqueness: [PASS/FAIL]
- TestCSRFTokenExpiration: [PASS/FAIL]
```

### Manual Test Results
```
Browser: [Chrome/Firefox/Safari]
OS: [Windows/Mac/Linux]

Form Tests: [PASS/FAIL]
Positive Tests: [PASS/FAIL]
Negative Tests: [PASS/FAIL]
Edge Cases: [PASS/FAIL]
Security Headers: [PASS/FAIL]
```

## Troubleshooting

### Common Issues
1. **"No CSRF token found"**
   - Check if server is running
   - Verify page contains form with CSRF token
   - Check network connectivity

2. **"403 Forbidden on valid request"**
   - Verify CSRF token is correctly extracted
   - Check token format (no URL encoding issues)
   - Ensure cookies are being sent

3. **"Authentication fails"**
   - Verify test user exists in database
   - Check password is correct
   - Ensure database is accessible

4. **"Cross-origin test fails"**
   - Verify CORS headers are properly set
   - Check if browser blocks cross-origin requests
   - Test with different browsers

### Debug Commands
```bash
# Check server response
curl -v http://localhost:3000/login

# Extract CSRF token manually
curl -s http://localhost:3000/login | grep csrf_token

# Test with verbose output
curl -v -X POST -d "username=test&password=test&csrf_token=TOKEN" http://localhost:3000/login
```

## Success Criteria
- [ ] All automated tests pass
- [ ] All manual tests pass
- [ ] No security vulnerabilities found
- [ ] All forms include CSRF tokens
- [ ] All endpoints properly validate tokens
- [ ] Webhook exemption works correctly
- [ ] Cross-origin requests blocked