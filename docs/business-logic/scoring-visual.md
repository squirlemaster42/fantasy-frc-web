# Scoring Algorithm Visual Guide

Visual representations of the Fantasy FRC scoring system.

## 🎯 Complete Score Flow

```mermaid
graph TD
    A[Team Performance] --> B[Match Results]
    B --> C{Match Type?}
    
    C -->|Qualification| D[Qual Score Calculation]
    C -->|Playoff| E[Playoff Score Calculation]
    C -->|Einstein| F[Einstein Score Calculation]
    
    D --> G[Base Points + Bonuses]
    E --> H[Bracket-Based Points]
    F --> I[Doubled Playoff Points]
    
    J[Alliance Selection] --> K[Alliance Score Calculation]
    
    G --> L[Team Total Score]
    H --> L
    I --> L
    K --> L
    
    L --> M[Player Fantasy Score]
    M --> N[Draft Rankings]
```

## 📊 Match Scoring Decision Tree

```mermaid
flowchart TD
    A[Match Result Received] --> B{Match Level?}
    
    B -->|qm| C[Qualification Match]
    B -->|qf| D[Quarterfinal Match]
    B -->|sf| E[Semifinal Match]
    B -->|f| F[Final Match]
    
    C --> G[Base: 3 pts for win]
    G --> H{Bonuses Achieved?}
    H -->|Auto Bonus| I[+1 pt]
    H -->|Barge Bonus| J[+1 pt]
    H -->|Coral Bonus| K[+1 pt]
    
    D --> L[No base points]
    E --> M{Bracket Position?}
    F --> N[18 pts]
    
    M -->|Upper| O[15 pts]
    M -->|Lower| P[9 pts]
    
    Q{Einstein Event?}
    O --> Q
    P --> Q
    N --> Q
    Q -->|Yes| R[×2 Multiplier]
    Q -->|No| S[No Multiplier]
    
    I --> T[Total Match Score]
    J --> T
    K --> T
    L --> T
    R --> T
    S --> T
```

## 🏆 Alliance Selection Scoring

```mermaid
graph LR
    A[Alliance 1] --> B[Captain: 64pts]
    A --> C[Pick 1: 62pts]
    A --> D[Pick 2: 18pts]
    A --> E[Pick 3: 16pts]
    
    F[Alliance 2] --> G[Captain: 60pts]
    F --> H[Pick 1: 58pts]
    F --> I[Pick 2: 20pts]
    F --> J[Pick 3: 14pts]
    
    K[Alliance 8] --> L[Captain: 36pts]
    K --> M[Pick 1: 34pts]
    K --> N[Pick 2: 32pts]
    K --> O[Pick 3: 2pts]
    
    style A fill:#e1f5fe
    style F fill:#e1f5fe
    style K fill:#e1f5fe
```

## 📈 Player Score Composition

```mermaid
pie title Player Score Sources
    "Qualification Scores" : 35
    "Playoff Scores" : 25
    "Alliance Selection" : 30
    "Einstein Multiplier" : 10
```

## 🎮 Draft Impact on Scoring

```mermaid
graph TD
    A[8 Players] --> B[Snake Draft Order]
    B --> C[64 Total Teams Picked]
    C --> D[8 Teams per Player]
    
    D --> E[Player Score = Sum of 8 Team Scores]
    E --> F[Ranking Determination]
    
    G[Draft Strategy] --> H{Team Selection}
    H -->|High Alliance| I[Early Alliance Picks]
    H -->|Strong Qual| J[High-Performing Teams]
    H -->|Playoff Success| K[Deep Playoff Runs]
    
    I --> L[Alliance Score Points]
    J --> M[Match Performance Points]
    K --> N[Playoff Points]
    
    L --> O[Optimal Player Score]
    M --> O
    N --> O
```

## ⏰ Match Processing Priority

```mermaid
gantt
    title Match Processing Order
    dateFormat X
    axisFormat %s
    
    section Event Types
    Qualification Matches :active, qm1, 0, 1
    Quarterfinals :active, qf1, 1, 2
    Semifinals :active, sf1, 2, 3
    Finals :active, f1, 3, 4
    Einstein Special :active, einstein, 4, 5
```

## 🎯 Score Calculation Examples

### Example 1: Championship Team Performance

```mermaid
graph TD
    A[Team X] --> B[Qualification: 5-2 Record]
    B --> C[Qual Points: 15 + 3 bonuses = 18]
    
    A --> D[Quarterfinal: Lost]
    D --> E[QF Points: 0]
    
    A --> F[Alliance: 3rd Captain]
    F --> G[Alliance Points: 56 × 2 = 112]
    
    C --> H[Total: 130 points]
    E --> H
    G --> H
```

### Example 2: Einstein Championship Team

```mermaid
graph TD
    A[Team Y] --> B[Einstein SF: Upper Bracket Win]
    B --> C[Base Points: 15]
    C --> D[Einstein Multiplier: ×2]
    D --> E[Einstein Points: 30]
    
    A --> F[Regular Event Alliance: 5th Pick]
    F --> G[Alliance Points: 26 × 2 = 52]
    
    E --> H[Total: 82 points]
    G --> H
```

## 📊 Score Distribution Analysis

```mermaid
bar-chart
    title Typical Score Distribution by Category
    x-axis [Qualification, Playoff QF, Playoff SF, Playoff F, Alliance]
    y-axis "Points" 0 --> 70
    series [Team A, Team B, Team C]
    data [15, 0, 15, 36, 108]
    data [12, 0, 0, 0, 64]
    data [18, 0, 30, 0, 52]
```

## 🔍 Validation Rules

```mermaid
flowchart TD
    A[Score Calculation] --> B{Validation Checks}
    
    B --> C[Team in Valid Event?]
    B --> D[Match Played?]
    B --> E[Team Not DQed?]
    
    C -->|No| F[Score = 0]
    D -->|No| G[Score = 0]
    E -->|Yes| H[Calculate Points]
    
    F --> I[Log Error]
    G --> I
    H --> J[Apply Multipliers]
    
    J --> K[Final Score]
    I --> K
```

---

*Visual guide complements the detailed scoring algorithm documentation at [scoring.md](./scoring.md)*