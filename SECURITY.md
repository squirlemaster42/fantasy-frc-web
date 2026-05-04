# Security Policy

## Supported Versions

The following versions of Fantasy FRC Web are currently supported with security updates:

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in Fantasy FRC Web, please report it responsibly.

**Please do not open public issues for security vulnerabilities.**

Instead, contact the maintainer directly:

- **Jakob** (project maintainer)

Include the following details in your report:
- Description of the vulnerability
- Steps to reproduce (if applicable)
- Potential impact
- Suggested fix (if you have one)

## Security Measures

Fantasy FRC Web implements the following security practices:

- **Password Security**: bcrypt hashing with cost factor 14
- **Session Management**: SHA-256 hashed tokens with automatic expiration
- **Database Security**: Prepared statements for all queries
- **Input Validation**: Server-side validation and sanitization
- **XSS Prevention**: HTML template auto-escaping
- **Webhook Validation**: HMAC signature verification for TBA webhooks

## Known Limitations

The following security enhancements are identified for future implementation:

- **CSRF Protection**: Not currently implemented
- **Rate Limiting**: Not currently implemented
- **Content Security Policy**: Additional hardening recommended

## Disclosure Policy

We will acknowledge receipt of your vulnerability report within 5 business days and will send a more detailed response within 10 business days indicating the next steps in handling your report.

After the initial reply to your report, we will endeavor to keep you informed of the progress towards a fix and full announcement, and may ask for additional information or guidance.

## Security Updates

Security updates will be released as promptly as possible and announced via the project's communication channels.
