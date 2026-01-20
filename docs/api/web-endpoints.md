# Web API Documentation

## Architecture Overview

The Fantasy FRC Web application is a **traditional web application** that uses **HTML templates with form submissions**, not a REST API. The application serves HTML pages and processes form data using standard HTTP patterns.

### Key Architectural Patterns
- **Server-Side Rendering**: HTML templates rendered on the server using Templ
- **Form-Based Interaction**: POST requests handle form submissions with redirects
- **Session Authentication**: Cookie-based session management
- **Real-Time Updates**: WebSocket connections for live draft updates
- **HTMX Integration**: Enhanced user experience with partial page updates

## Authentication & Session Management

### Session Token Lifecycle

#### Token Creation
- **Generation**: 128-bit random tokens using `crypto/rand`
- **Encoding**: Base32 encoding for URL-safe representation
- **Storage**: SHA-256 hash stored in database (never plain tokens)
- **Duration**: 10-day expiration with automatic extension on access

#### Authentication Flow
1. **Login**: User submits credentials → bcrypt verification → session token created → cookie set
2. **Request**: Cookie extracted → token validated against database → user UUID retrieved
3. **Expiration**: Background cleanup removes sessions expired > 2 hours
4. **Logout**: Session removed from database → cookie cleared on client

#### Cookie Management
- **Name**: `sessionToken`
- **Security**: HttpOnly flag (prevents JavaScript access)
- **Environment**: Secure flag disabled in development, enabled in production
- **Extension**: Valid sessions automatically extend expiration by 10 days

#### Authorization Middleware
- **`Authenticate`**: Validates session token for protected routes (`/u/*`)
- **`CheckAdmin`**: Additional admin verification for admin routes (`/u/admin/*`)
- **Failure Handling**: Invalid sessions redirect to `/login` with HTTP 303

## Public Endpoints (No Authentication Required)

### Authentication Routes

#### GET `/login`
- **Purpose**: Serves the login page
- **Response**: HTML template (Login view)
- **Validation**: None required

#### POST `/login`
- **Purpose**: Handles user authentication
- **Form Data**: 
  - `username`: Existing username
  - `password`: User password
- **Validation Rules**:
  - Username must exist in database
  - Password must match bcrypt hash
- **Response**: 
  - Success: HX-Redirect to `/u/home`
  - Failure: Error page with invalid credentials message
- **Error Handling**: Red alert box displaying "Invalid username or password"

#### GET `/register`
- **Purpose**: Serves user registration form
- **Response**: HTML template (Register view)

#### POST `/register`
- **Purpose**: Creates new user account
- **Form Data**:
  - `username`: Desired username
  - `password`: User password
  - `confirmPassword`: Password confirmation
- **Validation Rules**:
  - Username must be unique (checked via `model.UsernameTaken()`)
  - Passwords must match
  - Password hashed with bcrypt (cost factor 14)
- **Response**: 
  - Success: Redirect to `/u/home`
  - Failure: Error page with specific validation messages
- **Error Handling**: Displays username taken or password mismatch errors

#### POST `/logout`
- **Purpose**: Ends user session
- **Form Data**: None (uses current session)
- **Response**: HX-Redirect to `/login`
- **Process**: Session removed from database, cookie cleared

### Landing Page

#### GET `/`
- **Purpose**: Main application landing page
- **Response**: HTML template (Landing view)
- **Features**: Application overview and navigation

### External Integration

#### POST `/tbaWebhook`
- **Purpose**: Receives webhook events from The Blue Alliance (TBA)
- **Authentication**: HMAC validation using `TbaWekhookSecret`
- **Expected Data**: JSON webhook payload
- **Supported Events**:
  - `upcoming_match`, `match_score`, `match_video`
  - `starting_comp_level`, `alliance_selection`
  - `awards_posted`, `schedule_updated`
  - `ping`, `broadcast`, `verification`
- **Response**: Empty response (processing happens asynchronously)
- **Security**: Validates HMAC signature before processing

## Protected Endpoints (/u/* - Authentication Required)

All routes under `/u` require a valid session token cookie.

### User Dashboard

#### GET `/u/home`
- **Purpose**: User's main dashboard
- **Response**: HTML template displaying user's drafts
- **Data Display**: All drafts associated with authenticated user

### Draft Management

#### GET `/u/createDraft`
- **Purpose**: Serves draft creation form
- **Response**: HTML template (Create Draft view)
- **Features**: Form for new draft configuration

#### POST `/u/createDraft`
- **Purpose**: Creates new fantasy draft
- **Form Data**:
  - `draftName`: Draft display name
  - `description`: Draft description
  - `interval`: Pick time limit in seconds
  - `startTime`: Draft start time (RFC3339 format)
  - `endTime`: Draft end time (RFC3339 format)
- **Validation Rules**:
  - Draft name required
  - Time parsing with RFC3339 format validation
  - Interval must convert to valid integer
- **Response**: HX-Redirect to `/u/draft/{id}/profile`
- **Error Handling**: Displays form validation errors with preserved input

#### GET `/u/draft/:id/profile`
- **Purpose**: Serves draft profile page
- **URL Parameters**: `id` - Draft ID (integer)
- **Response**: HTML template (Draft Profile view)
- **Access Control**: 
  - Draft owner can edit settings
  - Other users can view only
- **Display**: Draft details, players, current settings, scoring

#### POST `/u/draft/:id/updateDraft`
- **Purpose**: Updates draft configuration
- **URL Parameters**: `id` - Draft ID (integer)
- **Form Data**: Same as creation form
- **Validation Rules**: Same as creation + ownership verification
- **Access Control**: Draft owner only
- **Response**: HX-Redirect to `/u/draft/{id}/profile`
- **Security**: Silent failure if user is not draft owner

#### POST `/u/draft/:id/startDraft`
- **Purpose**: Initiates draft start process
- **URL Parameters**: `id` - Draft ID (integer)
- **Response**: Updated start button or error message
- **Validation**:
  - Draft must be in FILLING state
  - User must be draft owner
- **State Transition**: FILLING → WAITING_TO_START

### Player Management

#### POST `/u/searchPlayers`
- **Purpose**: Searches for users to invite to drafts
- **Form Data**:
  - `search`: Username search term
- **Validation Rules**: None (open search)
- **Response**: HTML search results with invite buttons
- **Display**: List of matching users with invitation controls

#### POST `/u/draft/:id/invitePlayer`
- **Purpose**: Invites players to join a draft
- **URL Parameters**: `id` - Draft ID (integer)
- **Form Data**:
  - `userUuid`: UUID of user to invite
  - `search`: Current search term (for UI consistency)
- **Validation Rules**:
  - User must exist
  - User cannot already be in draft
  - Draft must be in FILLING state
- **Access Control**: Draft owner only
- **Response**: Updated player list HTML
- **Error Handling**: Displays appropriate error messages

#### GET `/u/viewInvites`
- **Purpose**: Shows user's pending draft invitations
- **Response**: HTML template (Draft Invites view)
- **Display**: All pending invitations for authenticated user

#### POST `/u/acceptInvite`
- **Purpose**: Accepts a draft invitation
- **Form Data**:
  - `inviteId`: ID of invitation to accept
- **Validation Rules**:
  - Invitation must exist
  - User must be the intended recipient
  - Draft must not be full
- **Response**: Updated invites page
- **Error Handling**: Shows invitation acceptance failures

### Live Drafting Interface

#### GET `/u/draft/:id/pick`
- **Purpose**: Serves live draft picking interface
- **URL Parameters**: `id` - Draft ID (integer)
- **Response**: HTML template (Draft Pick view)
- **Features**:
  - Current pick display
  - Timer countdown
  - Available teams
  - Pick submission form
  - Real-time updates

#### POST `/u/draft/:id/makePick`
- **Purpose**: Handles team selection during draft
- **URL Parameters**: `id` - Draft ID (integer)
- **Form Data**:
  - `pickInput`: Team number (without "frc" prefix)
- **Validation Rules**:
  - Draft must be in PICKING state
  - User must be current picker
  - Team must be available (not already drafted)
  - Team format validation
- **Response**: Updated pick page with success/error
- **Security**: Only current player can submit picks
- **Error Handling**: Detailed validation error messages

#### POST `/u/draft/:id/skipPickToggle`
- **Purpose**: Toggles auto-skip functionality for current pick
- **URL Parameters**: `id` - Draft ID (integer)
- **Form Data**: Contains "skipping" to enable skip
- **Validation Rules**:
  - Draft must be in PICKING state
  - User must be current picker
- **Response**: Success/failure status
- **Purpose**: Allows players to automatically skip their turn

### Scoring & Analytics

#### GET `/u/team/score`
- **Purpose**: Serves team scoring lookup form
- **Response**: HTML template (Team Score view)
- **Features**: Form to look up scoring data for specific teams

#### POST `/u/team/score`
- **Purpose**: Retrieves scoring data for a specific team
- **Form Data**:
  - `teamNumber`: FRC team number
- **Validation Rules**:
  - Team number must be valid integer
  - Team must exist in database
- **Response**: HTML with team scoring report
- **Error Handling**: Shows team not found errors

#### GET `/u/draft/:id/draftScore`
- **Purpose**: Displays scoring leaderboard for draft
- **URL Parameters**: `id` - Draft ID (integer)
- **Response**: HTML template (Draft Score view)
- **Display**: Scoring rankings, points breakdown, match results

## Admin Endpoints (/u/admin/* - Authentication + Admin Required)

### Administrative Console

#### GET `/u/admin/console`
- **Purpose**: Serves administrative interface
- **Response**: HTML template (Admin Console view)
- **Access Control**: Admin users only
- **Features**: System management tools and command interface

#### POST `/u/admin/processCommand`
- **Purpose**: Executes administrative commands
- **Form Data**:
  - `command`: Admin command to execute
- **Available Commands**:
  - `ping` - Test system connectivity
  - `listdraft -s <search>` - List drafts with optional search filter
  - `startdraft -id <draftId>` - Force start a specific draft
  - `skippick -id <draftId>` - Force skip current pick in draft
- **Validation Rules**:
  - Command must be in allowed list
  - Draft ID validation for draft-specific commands
- **Access Control**: Admin users only
- **Response**: Command output and results
- **Security**: All admin actions logged with user context

## WebSocket Endpoints

### Real-Time Draft Updates

#### WS `/u/draft/:id/pickNotifier`
- **Purpose**: Real-time draft updates during live picking
- **URL Parameters**: `id` - Draft ID (integer)
- **Authentication**: Requires valid session token
- **Protocol**: Custom message format for UI updates
- **Events**:
  - New picks made
  - Draft state changes
  - Timer updates
  - Player status changes
- **Message Format**: HTML fragments for HTMX integration
- **Connection Management**: Automatic cleanup on disconnect

## Form Validation & Error Handling

### Common Validation Patterns

#### Input Sanitization
- **SQL Injection Prevention**: All database queries use prepared statements
- **XSS Prevention**: Templates automatically escape HTML content
- **Input Validation**: Server-side validation for all form submissions

#### Security Validation Tests
The application includes security tests that validate rejection of:
- **Script Tags**: `<script>alert('xss')</script>`
- **Event Handlers**: `<div onclick=alert(1)>`
- **SQL Injection**: `'; DROP TABLE users; --`
- **Path Traversal**: `../../../etc/passwd`

### HTTP Status Codes Used

#### Success Codes
- **200 OK**: Successful GET requests and form submissions
- **303 See Other**: Redirects after login/logout and form processing

#### Error Codes
- **400 Bad Request**: Invalid parameters (malformed draft IDs, invalid input)
- **401 Unauthorized**: Invalid authentication or insufficient permissions
- **500 Internal Server Error**: Database errors or unexpected server failures

### Error Display Patterns

#### Form Errors
```html
<div class="bg-red-900/20 border border-red-700 text-red-300 px-4 py-3 rounded-lg mb-6 text-center">
    <div class="flex items-center justify-center gap-2">
        <!-- Error icon -->
        {errorMessage}
    </div>
</div>
```

#### Validation Error Messages
- **Login**: "Invalid username or password"
- **Registration**: "Username is already taken" or "Passwords do not match"
- **Draft Operations**: Context-specific messages for validation failures
- **Authorization**: Silent failures with redirects (security measure)

### Error Handling Implementation

#### Assert Package Usage
```go
assert := assert.CreateAssertWithContext("Function Name")
assert.NoError(err, "Error message with context")
assert.AddContext("Key", value)
```

#### Security Considerations
- **Authentication Failures**: Logged with IP addresses for monitoring
- **Authorization Failures**: Silent redirects to prevent information disclosure
- **Input Validation**: Comprehensive validation before database operations

## Security Considerations

### Current Security Measures
- **Password Security**: bcrypt with high cost factor (14)
- **Session Security**: HttpOnly cookies, token hashing, automatic cleanup
- **Database Security**: Prepared statements, connection pooling
- **Input Validation**: Server-side validation, XSS prevention
- **Access Control**: Role-based middleware, resource ownership verification

### Security Enhancement Opportunities
- **CSRF Protection**: Currently not implemented (recommended for state-changing operations)
- **Rate Limiting**: Not currently implemented (recommended for authentication endpoints)
- **Content Security Policy**: Additional hardening against XSS attacks

### Session Security Implementation
- **Token Storage**: SHA-256 hash stored, never plain tokens
- **Cookie Configuration**: HttpOnly, Secure in production
- **Automatic Cleanup**: Expired sessions removed every 2 hours
- **IP Logging**: All authentication attempts logged with source IP

---

*This documentation covers all HTTP endpoints in the Fantasy FRC Web application. For WebSocket API details, see [WebSocket API](./websocket-api.md).*