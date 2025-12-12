# Data Flow

How data moves through the Fantasy FRC system from external sources to end users.

## 🌊 Data Flow Overview

```mermaid
graph TD
    subgraph "External Sources"
        A[TBA API]
        B[Web Clients]
        C[Admin Users]
    end
    
    subgraph "Ingestion Layer"
        D[TBA Handler]
        E[Web Server]
        F[Admin Console]
    end
    
    subgraph "Processing Layer"
        G[Scorer]
        H[Draft Manager]
        I[Authentication]
    end
    
    subgraph "Storage Layer"
        J[PostgreSQL]
        K[Redis Cache]
    end
    
    subgraph "Output Layer"
        L[WebSocket Hub]
        M[REST Responses]
        N[Static Pages]
    end
    
    A --> D
    B --> E
    C --> F
    D --> G
    E --> H
    E --> I
    F --> H
    G --> J
    H --> J
    I --> J
    G --> K
    H --> L
    I --> K
    J --> M
    K --> M
    J --> N
```

## 📊 Data Types and Sources

### FRC Competition Data
- **Source**: The Blue Alliance API
- **Types**: Teams, matches, events, rankings
- **Frequency**: Real-time webhook + periodic polling
- **Format**: JSON API responses

### User-Generated Data
- **Source**: Web client interactions
- **Types**: Drafts, picks, user accounts, invitations
- **Frequency**: User-driven events
- **Validation**: Server-side validation and sanitization

### Administrative Data
- **Source**: Admin console actions
- **Types**: System configuration, user management
- **Frequency**: Administrative operations
- **Authorization**: Role-based access control

## 🔄 Core Data Flows

### Draft Lifecycle Data Flow
```mermaid
stateDiagram-v2
    [*] --> Filling
    Filling --> Waiting_To_Start: Players ready
    Waiting_To_Start --> Picking: Scheduled time
    Picking --> Teams_Playing: All picks made
    Teams_Playing --> Complete: Event finished
    
    note right of Filling
        Data: Draft config
        Source: Draft creation
        Storage: Drafts table
    end note
    
    note right of Picking
        Data: Team selections
        Source: User picks
        Storage: Picks table
        Real-time: WebSocket updates
    end note
```

### Scoring Data Pipeline
```mermaid
graph LR
    A[TBA Webhook] --> B[Match Processor]
    B --> C[Score Calculator]
    C --> D[Ranking Updater]
    D --> E[Cache Refresher]
    E --> F[Client Notifier]
    
    subgraph "Data Transformations"
        G[Raw Match Data]
        H[Processed Scores]
        I[Player Rankings]
        J[Cached Results]
    end
    
    B --> G
    C --> H
    D --> I
    E --> J
```

### Real-time Notification Flow
```mermaid
sequenceDiagram
    participant S as Scorer
    participant W as WebSocket Hub
    participant C1 as Client 1
    participant C2 as Client 2
    participant C3 as Client 3
    
    S->>W: Score Update Event
    W->>C1: Push Update
    W->>C2: Push Update
    W->>C3: Push Update
    
    Note over C1,C3: Only clients in relevant draft
```

## 🗄️ Data Storage Patterns

### Primary Database (PostgreSQL)
```mermaid
erDiagram
    Users ||--o{ Drafts : owns
    Users ||--o{ DraftPlayers : participates
    Drafts ||--o{ DraftPlayers : contains
    Drafts ||--o{ DraftInvites : sends
    DraftPlayers ||--o{ Picks : makes
    Teams ||--o{ Picks : selected
    Matches ||--o{ Matches_Teams : involves
    Teams ||--o{ Matches_Teams : plays
    Users ||--o{ UserSessions : has
```

### Cache Layer (Redis)
- **Session Storage**: Fast user authentication
- **Score Cache**: Frequently accessed rankings
- **Draft State**: Current draft status for quick lookups
- **TTL Strategy**: Automatic cache expiration

## 📈 Data Volume and Performance

### Data Characteristics
| Data Type | Volume | Update Frequency | Access Pattern |
|-----------|--------|------------------|----------------|
| User Data | Low | Medium | Read-heavy |
| Draft Data | Medium | High (during drafts) | Read-write |
| Match Data | High | Very High (events) | Write-heavy |
| Score Data | Medium | High | Read-heavy |

### Performance Optimizations
- **Database Indexing**: Optimized query performance
- **Connection Pooling**: Efficient resource usage
- **Batch Processing**: Reduced database round trips
- **Async Processing**: Non-blocking operations

## 🔒 Data Security and Privacy

### Data Protection
- **Encryption**: Password hashing with bcrypt
- **Session Security**: SHA256 token generation
- **Input Validation**: SQL injection prevention
- **Data Sanitization**: XSS protection

### Privacy Considerations
- **User Data**: Minimal personal information collection
- **Session Data**: Automatic expiration and cleanup
- **Audit Trail**: Administrative action logging
- **Data Retention**: Configurable cleanup policies

---

*TODO: Add detailed data transformation examples, performance benchmarks, and security audit procedures*