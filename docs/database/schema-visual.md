# Database Schema Visual Guide

Visual representations of Fantasy FRC database structure and relationships.

## 🗄️ Complete Entity Relationship Diagram

```mermaid
erDiagram
    Users {
        uuid UserUuid PK
        string Username UK
        string Password
    }
    
    Teams {
        string tbaId PK
        string name
        smallint AllianceScore
    }
    
    Drafts {
        int Id PK
        string DisplayName
        text Description
        timestamp StartTime
        timestamp EndTime
        interval Interval
        string Status
        uuid OwnerUserUuid FK
    }
    
    DraftPlayers {
        int Id PK
        int draftId FK
        uuid UserUuid FK
        smallint playerOrder
        boolean Pending
        boolean skipPicks
    }
    
    Picks {
        int Id PK
        int player FK
        string pick FK
        timestamp pickTime
        timestamp AvailableTime
        timestamp ExpirationTime
        boolean Skipped
    }
    
    Matches {
        string tbaId PK
        boolean played
        smallint redScore
        smallint blueScore
    }
    
    Matches_Teams {
        string team_tbaId PK,FK
        string match_tbaId PK,FK
        varchar alliance
        boolean isDqed
    }
    
    DraftInvites {
        int Id PK
        int draftId FK
        uuid invitedUserUuid FK
        uuid invitingUserUuid FK
        string draftName
        string invitingPlayerName
        timestamp sentTime
        timestamp acceptedTime
        boolean accepted
        boolean canceled
    }
    
    UserSessions {
        int Id PK
        uuid UserUuid FK
        bytea sessionToken
        timestamp expirationTime
    }
    
    DraftReaders {
        int Id PK
        uuid UserUuid FK
        int draft FK
    }
    
    TbaCache {
        text url PK
        varchar etag
        bytea responseBody
    }
    
    Users ||--o{ Drafts : owns
    Users ||--o{ DraftPlayers : participates
    Users ||--o{ DraftInvites : sends
    Users ||--o{ DraftInvites : receives
    Users ||--o{ UserSessions : authenticates
    Drafts ||--o{ DraftPlayers : contains
    Drafts ||--o{ DraftInvites : creates
    Drafts ||--o{ DraftReaders : visible_to
    DraftPlayers ||--o{ Picks : makes
    Teams ||--o{ Picks : selected_as
    Matches ||--o{ Matches_Teams : involves
    Teams ||--o{ Matches_Teams : plays_in
```

## 📊 Table Relationship Overview

### Core Entity Relationships
```mermaid
graph TD
    subgraph "User Management"
        A[Users] --> B[UserSessions]
        A --> C[DraftPlayers]
        A --> D[DraftInvites]
        A --> E[DraftReaders]
    end
    
    subgraph "Draft System"
        F[Drafts] --> C
        F --> D
        F --> E
        F --> G[Picks]
    end
    
    subgraph "Team & Match Data"
        H[Teams] --> G
        H --> I[Matches_Teams]
        I --> J[Matches]
    end
    
    C --> G
    G --> H
```

### Draft Lifecycle Flow
```mermaid
sequenceDiagram
    participant U as Users
    participant D as Drafts
    participant DP as DraftPlayers
    participant P as Picks
    participant T as Teams
    
    U->>D: Create Draft
    D->>DP: Add Players
    DP->>P: Create Pick Slots
    U->>DP: Accept Invitation
    DP->>P: Activate Player
    P->>T: Select Teams
    T->>P: Record Selection
```

## 🏗️ Schema Evolution Timeline

```mermaid
gantt
    title Database Schema Evolution
    dateFormat X
    axisFormat %s
    
    section Version 1.0
    Basic Tables :active, v1, 0, 2
    User Management :active, v1, 2, 4
    Match Tracking :active, v1, 4, 6
    
    section Version 1.1
    UUID Migration :active, v11, 6, 8
    FK Updates :active, v11, 8, 10
    
    section Version 1.2
    Draft Enhancements :active, v12, 10, 12
    Timing Columns :active, v12, 12, 14
    
    section Version 1.3
    Skip Feature :active, v13, 14, 15
    
    section Version 1.4
    Performance Cache :active, v14, 15, 16
```

## 📈 Data Flow Patterns

### User Registration Flow
```mermaid
flowchart TD
    A[New User] --> B[Create User Record]
    B --> C[Generate UUID]
    C --> D[Hash Password]
    D --> E[Store in Users Table]
    E --> F[Create Session]
    F --> G[Store in UserSessions]
```

### Draft Creation Flow
```mermaid
flowchart TD
    A[User Creates Draft] --> B[Insert into Drafts]
    B --> C[Owner as DraftPlayers]
    C --> D[Initial Status: FILLING]
    D --> E[Create Pick Slots]
    E --> F[64 Pick Records]
```

### Pick Processing Flow
```mermaid
flowchart TD
    A[Player Turn] --> B[Find Current Pick]
    B --> C[Validate Team Selection]
    C --> D{Team Valid?}
    D -->|Yes| E[Update Pick Record]
    D -->|No| F[Return Error]
    E --> G[Create Next Pick]
    G --> H[Notify Players]
    F --> H
```

## 🔍 Key Index Patterns

### Primary Key Distribution
```mermaid
pie title Primary Key Types
    "UUID (Users)" : 25
    "SERIAL (Auto-increment)" : 50
    "Natural Keys (tbaId)" : 15
    "Composite Keys" : 10
```

### Foreign Key Relationships
```mermaid
graph LR
    subgraph "User-Centric"
        A[Users] --> B[DraftPlayers]
        A --> C[DraftInvites]
        A --> D[UserSessions]
        A --> E[DraftReaders]
    end
    
    subgraph "Draft-Centric"
        F[Drafts] --> B
        F --> C
        F --> E
        F --> G[Picks]
    end
    
    subgraph "Data-Centric"
        H[Teams] --> G
        H --> I[Matches_Teams]
        J[Matches] --> I
    end
```

## 📊 Table Size Analysis

### Row Count Distribution
```mermaid
bar-chart
    title Estimated Table Sizes
    x-axis [Users, Teams, Drafts, Players, Picks, Matches]
    y-axis "Rows (thousands)" 0 --> 200
    series [Current]
    data [1, 3.5, 0.5, 4, 32, 50]
```

### Growth Rate Projection
```mermaid
line-chart
    title Monthly Growth Projections
    x-axis [Jan, Feb, Mar, Apr, May, Jun]
    y-axis "Rows" 0 --> 10000
    series [Users, Drafts, Picks, Matches]
    data [1000, 1100, 1200, 1300, 1400, 1500]
    data [50, 100, 150, 200, 250, 300]
    data [3200, 6400, 9600, 12800, 16000, 19200]
    data [5000, 10000, 15000, 20000, 25000, 30000]
```

## 🔒 Security Architecture

### Authentication Flow
```mermaid
sequenceDiagram
    participant C as Client
    participant A as Application
    participant DB as Database
    participant S as Sessions
    
    C->>A: Login Request
    A->>DB: Find User by Username
    DB-->>A: User Record
    A->>A: Verify Password Hash
    A->>S: Create Session Token
    S->>DB: Store Session
    DB-->>S: Session ID
    S-->>A: Session Token
    A-->>C: Authentication Success
```

### Data Protection Layers
```mermaid
graph TD
    A[Application Layer] --> B[Input Validation]
    B --> C[Prepared Statements]
    C --> D[Database Constraints]
    D --> E[Foreign Key Rules]
    E --> F[Data Integrity]
    
    G[Security Features] --> H[bcrypt Password Hashing]
    G --> I[SHA256 Session Tokens]
    G --> J[UUID Primary Keys]
    G --> K[SQL Injection Prevention]
```

## 🚀 Performance Optimization

### Query Optimization Patterns
```mermaid
graph LR
    subgraph "Read Optimization"
        A[Prepared Statements]
        B[Connection Pooling]
        C[Result Caching]
    end
    
    subgraph "Write Optimization"
        D[Batch Operations]
        E[Transaction Batching]
        F[Minimal Locking]
    end
    
    subgraph "Index Strategy"
        G[Primary Keys]
        H[Foreign Keys]
        I[Unique Constraints]
    end
```

### Cache Implementation
```mermaid
sequenceDiagram
    participant A as Application
    participant T as TBA API
    participant C as TbaCache
    participant D as Database
    
    A->>C: Check Cache
    alt Cache Hit
        C-->>A: Cached Response
    else Cache Miss
        A->>T: API Request
        T-->>A: Response Data
        A->>C: Store in Cache
        A->>D: Process Data
    end
```

## 📋 Migration Process

### UUID Migration Steps
```mermaid
flowchart TD
    A[Start Migration] --> B[Add UUID Columns]
    B --> C[Generate UUIDs for Existing Records]
    C --> D[Update Foreign Key References]
    D --> E[Create New Constraints]
    E --> F[Drop Legacy ID Columns]
    F --> G[Update Primary Keys]
    G --> H[Verify Data Integrity]
    H --> I[Complete Migration]
```

### Schema Versioning
```mermaid
stateDiagram-v2
    [*] --> v1_0: Initial Schema
    v1_0 --> v1_1: UUID Migration
    v1_1 --> v1_2: Draft Enhancements
    v1_2 --> v1_3: Skip Feature
    v1_3 --> v1_4: Performance Cache
    v1_4 --> [*]
```

---

*Visual guide complements detailed schema documentation at [schema.md](./schema.md)*