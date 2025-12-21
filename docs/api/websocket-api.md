# WebSocket API

Real-time communication protocol for live draft updates and notifications.

## 🔌 WebSocket Connection

### Endpoint
```
wss://your-domain.com/u/draft/:id/pickNotifier
```

### Authentication
WebSocket connections inherit authentication from the HTTP session that establishes them.

### Connection Flow
```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant H as WebSocket Hub
    
    C->>S: HTTP Upgrade Request
    S->>H: Register Connection
    H-->>C: Connection Established
    
    Note over C,H: Real-time communication
    
    C->>H: Subscribe to Draft
    H-->>C: Confirmation
    
    Note over H: Draft Event Occurs
    H->>C: Broadcast Update
```

## 🎯 Use Cases

### Draft Room Interface
```mermaid
graph TD
    A[Player Connects] --> B[Subscribe to Draft]
    B --> C[Receive Current State]
    C --> D[Listen for Updates]
    D --> E[Handle Pick Events]
    E --> F[Update UI]
    F --> D
```

### Live Scoreboard
```mermaid
graph LR
    A[Match Ends] --> B[Score Calculated]
    B --> C[WebSocket Broadcast]
    C --> D[Client Updates]
    D --> E[Ranking Refresh]
```

### Admin Monitoring
```mermaid
graph TD
    A[Admin Connects] --> B[Subscribe to All Drafts]
    B --> C[Monitor System Events]
    C --> D[Handle Alerts]
    D --> E[Take Action]
```

*TODO: Add complete client SDK documentation, advanced error handling patterns, and performance benchmarking data*
