# Draft State Machine Visual Guide

Visual representations of Fantasy FRC draft lifecycle and state management.

## 🎯 Complete State Flow

```mermaid
stateDiagram-v2
    [*] --> FILLING: Create Draft
    FILLING --> WAITING_TO_START: Start Draft
    WAITING_TO_START --> PICKING: Scheduled Time
    PICKING --> TEAMS_PLAYING: All Picks Complete
    TEAMS_PLAYING --> COMPLETE: Event Finished
    COMPLETE --> [*]
```

## 📊 State Transition Matrix

| Current State | Next State | Trigger | Auto? |
|---------------|-------------|----------|---------|
| FILLING | WAITING_TO_START | Owner starts draft | ❌ |
| WAITING_TO_START | PICKING | Scheduled time reached | ✅ |
| PICKING | TEAMS_PLAYING | 64 picks completed | ✅ |
| TEAMS_PLAYING | COMPLETE | All events finished | ✅ |

## 🎮 Draft Lifecycle Timeline

```mermaid
gantt
    title Typical Draft Timeline
    dateFormat X
    axisFormat %s
    
    section Draft States
    Setup Phase :active, filling, 0, 3
    Waiting Period :active, waiting, 3, 5
    Picking Phase :active, picking, 5, 9
    Competition :active, playing, 9, 13
    Completed :done, complete, 13, 14
```

## 🔄 Pick Management Flow

```mermaid
flowchart TD
    A[Draft Starts] --> B[Randomize Order]
    B --> C[Player 1 Turn]
    C --> D{Pick Made?}
    D -->|Yes| E[Validate Pick]
    D -->|No/Skip| F[Next Player]
    
    E --> G{Valid?}
    G -->|Yes| H[Record Pick]
    G -->|No| I[Error Message]
    
    H --> J{All Picks Done?}
    I --> F
    F --> K{More Players?}
    K -->|Yes| L[Continue Snake Order]
    K -->|No| M[End Picking Phase]
    J -->|Yes| M
    J -->|No| F
    L --> C
```

## 🐍 Snake Draft Pattern

```mermaid
graph TD
    subgraph "Round 1 (Forward)"
        A1[Player 1] --> A2[Player 2]
        A2 --> A3[Player 3]
        A3 --> A4[Player 4]
        A4 --> A5[Player 5]
        A5 --> A6[Player 6]
        A6 --> A7[Player 7]
        A7 --> A8[Player 8]
    end
    
    subgraph "Round 2 (Reverse)"
        B1[Player 8] --> B2[Player 7]
        B2 --> B3[Player 6]
        B3 --> B4[Player 5]
        B4 --> B5[Player 4]
        B5 --> B6[Player 3]
        B6 --> B7[Player 2]
        B7 --> B8[Player 1]
    end
    
    subgraph "Round 3 (Forward)"
        C1[Player 1] --> C2[Player 2]
        C2 --> C3[Player 3]
        C3 --> C4[Player 4]
        C4 --> C5[Player 5]
        C5 --> C6[Player 6]
        C6 --> C7[Player 7]
        C7 --> C8[Player 8]
    end
    
    A8 --> B1
    B8 --> C1
```

## ⏰ Time Management System

```mermaid
stateDiagram-v2
    [*] --> PickAvailable
    PickAvailable --> PickMade: Player selects team
    PickAvailable --> PickExpired: Time limit reached
    PickMade --> NextPick: Process selection
    PickExpired --> NextPick: Auto-skip
    NextPick --> PickAvailable: Next player turn
    NextPick --> DraftComplete: All picks done
    DraftComplete --> [*]
```

### Business Hours Schedule

```mermaid
graph LR
    subgraph "Weekday Schedule"
        W1[17:00] --> W2[22:00]
    end
    
    subgraph "Weekend Schedule"
        WE1[08:00] --> WE2[22:00]
    end
    
    style W1 fill:#ffeb3b
    style W2 fill:#ffeb3b
    style WE1 fill:#4caf50
    style WE2 fill:#4caf50
```

## 🔒 Concurrency and Locking

```mermaid
sequenceDiagram
    participant D1 as Draft Manager 1
    participant D2 as Draft Manager 2
    participant DB as Database
    participant L as Lock Manager
    
    D1->>L: Request Load Lock
    L-->>D1: Lock Granted
    D2->>L: Request Load Lock
    L-->>D2: Wait
    
    D1->>DB: Load Draft Data
    DB-->>D1: Draft Loaded
    D1->>L: Release Load Lock
    
    L-->>D2: Lock Granted
    D2->>DB: Load Draft Data
```

## 📡 Real-time Notification Flow

```mermaid
sequenceDiagram
    participant P as Player
    participant DM as Draft Manager
    participant PM as Pick Manager
    participant WS as WebSocket Hub
    participant C as All Clients
    
    P->>DM: Make Pick
    DM->>PM: Process Pick
    PM->>PM: Validate Pick
    PM->>PM: Record in Database
    
    PM->>WS: Pick Event
    WS->>C: Broadcast Update
    
    Note over C: All players see live updates
```

## 🎯 State-Specific Operations

### FILLING State Operations
```mermaid
graph TD
    A[FILLING] --> B[Owner Actions]
    A --> C[Player Actions]
    
    B --> D[Update Settings]
    B --> E[Invite Players]
    B --> F[Remove Players]
    
    C --> G[Accept Invite]
    C --> H[Decline Invite]
    
    style A fill:#e3f2fd
    style D fill:#bbdefb
    style E fill:#bbdefb
    style F fill:#bbdefb
    style G fill:#c8e6c9
    style H fill:#ffcdd2
```

### PICKING State Operations
```mermaid
graph TD
    A[PICKING] --> B[Current Player]
    A --> C[Other Players]
    
    B --> D[Make Pick]
    B --> E[Skip Turn]
    
    C --> F[Watch Live]
    C --> G[Receive Notifications]
    
    style A fill:#fff3e0
    style D fill:#ffe0b2
    style E fill:#ffccbc
    style F fill:#e8f5e8
    style G fill:#e8f5e8
```

## 📈 Performance Metrics

### Draft Completion Rates
```mermaid
pie title Draft Outcomes
    "Completed Successfully" : 85
    "Player Shortage" : 10
    "Technical Issues" : 5
```

### Pick Speed Analysis
```mermaid
bar-chart
    title Average Pick Time by Round
    x-axis [Round 1, Round 2, Round 3, Round 4]
    y-axis "Minutes" 0 --> 60
    series [Average Time]
    data [45, 52, 48, 55]
```

## 🚨 Error Handling Scenarios

### Invalid State Transitions
```mermaid
flowchart TD
    A[State Transition Request] --> B{Valid Transition?}
    B -->|Yes| C[Execute Transition]
    B -->|No| D[Return Error]
    
    C --> E{Success?}
    E -->|Yes| F[Update Database]
    E -->|No| G[Rollback State]
    
    F --> H[Notify Clients]
    G --> H
    D --> I[Log Error]
```

### Pick Validation Failures
```mermaid
graph TD
    A[Pick Attempt] --> B{Team Exists?}
    B -->|No| C[Error: Invalid Team]
    B -->|Yes| D{Team Available?}
    D -->|No| E[Error: Already Picked]
    D -->|Yes| F{Valid Event?}
    F -->|No| G[Error: Wrong Event]
    F -->|Yes| H{Current Player?}
    H -->|No| I[Error: Not Your Turn]
    H -->|Yes| J[Accept Pick]
    
    style C fill:#ffcdd2
    style E fill:#ffcdd2
    style G fill:#ffcdd2
    style I fill:#ffcdd2
    style J fill:#c8e6c9
```

## 🎮 User Experience Flow

### Complete Draft Journey
```mermaid
journey
    title Player Draft Experience
    section Draft Creation
      Create Draft: 5: Player
      Invite Friends: 4: Player
      Configure Settings: 5: Player
    section Draft Waiting
      Receive Notification: 4: Player
      Join Draft: 5: Player
      View Player List: 3: Player
    section Draft Picking
      Get Turn Notification: 4: Player
      Research Teams: 3: Player
      Make Selection: 5: Player
      Watch Others Pick: 4: Player
    section Competition
      View Scores: 5: Player
      Track Rankings: 4: Player
      Celebrate Win: 5: Player
```

---

*Visual guide complements detailed state machine documentation at [draft-states.md](./draft-states.md)*