# System Overview

High-level architecture of the Fantasy FRC web application.

## 🏗️ Architecture Components

```mermaid
graph TB
    subgraph "Client Layer"
        A[Web Browser]
        B[Mobile App]
    end
    
    subgraph "Application Layer"
        C[Echo Web Server]
        D[Authentication Middleware]
        E[Handler Layer]
    end
    
    subgraph "Business Logic Layer"
        F[Draft Manager]
        G[Pick Manager]
        H[Scorer]
        I[TBA Handler]
    end
    
    subgraph "Data Layer"
        J[PostgreSQL Database]
        K[Redis Cache]
    end
    
    subgraph "External Services"
        L[The Blue Alliance API]
        M[Email Service]
    end
    
    A --> C
    B --> C
    C --> D
    D --> E
    E --> F
    E --> G
    E --> H
    E --> I
    F --> J
    G --> J
    H --> J
    I --> J
    H --> K
    I --> L
    F --> M
```

## 📋 Core Components

### Web Server
- **Framework**: Echo v4
- **Purpose**: HTTP request handling and routing
- **Features**: Static assets, middleware, WebSocket support

### Authentication System
- **Method**: Session-based authentication
- **Security**: bcrypt password hashing, SHA256 session tokens
- **Features**: Role-based access control, admin permissions

### Draft Management
- **State Machine**: Draft lifecycle management
- **Real-time**: WebSocket notifications for draft events
- **Validation**: Pick validation and timing enforcement

### Scoring System
- **Background Service**: Continuous match result processing
- **Algorithm**: Complex scoring based on match types and alliance selection
- **Real-time**: Live score updates and rankings

## 🔄 Data Flow Overview

```mermaid
sequenceDiagram
    participant C as Client
    participant W as Web Server
    participant D as Draft Manager
    participant S as Scorer
    participant DB as Database
    participant TBA as TBA API
    
    C->>W: Create Draft
    W->>D: Initialize Draft
    D->>DB: Save Draft
    
    C->>W: Make Pick
    W->>D: Process Pick
    D->>DB: Save Pick
    D->>C: WebSocket Update
    
    TBA->>S: Match Results
    S->>DB: Update Scores
    S->>C: Live Score Updates
```

## 🎯 Design Principles

- **Modularity**: Clear separation of concerns
- **Scalability**: Horizontal scaling support
- **Reliability**: Error handling and recovery
- **Security**: Authentication and data protection
- **Performance**: Efficient data access and caching

## 📊 Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Web Framework | Echo v4 | HTTP server and routing |
| Database | PostgreSQL | Primary data storage |
| Caching | Redis | Session storage and caching |
| Frontend | Templ + HTMX | Server-rendered UI |
| Real-time | WebSocket | Live updates |
| External API | The Blue Alliance | FRC data source |

---

*TODO: Add detailed component descriptions, deployment diagrams, and scaling considerations*