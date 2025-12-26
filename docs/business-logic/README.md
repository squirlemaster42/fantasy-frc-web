# Business Logic Documentation

Core business rules and algorithms that drive the Fantasy FRC system.

## 📚 Documentation

### [🎯 Scoring Algorithm](./scoring.md)
Complete breakdown of how team and player scores are calculated, including:
- Qualification match scoring with bonuses
- Playoff match scoring by bracket position
- Alliance selection scoring tables
- Einstein championship multipliers
- Real-time score processing

### [📊 Scoring Visual Guide](./scoring-visual.md)
Visual representations of the scoring system:
- Flow charts for score calculation
- Decision trees for match processing
- Alliance selection scoring diagrams
- Player score composition charts

### [🏗️ Draft State Machine](./draft-states.md)
Draft lifecycle management and state transitions:
- 5 draft states and transition rules
- Pick validation and timing logic
- Snake draft ordering algorithm
- Real-time notification flows

### [📊 Draft State Visual Guide](./draft-states-visual.md)
Visual representations of draft lifecycle:
- State transition diagrams
- Pick management flowcharts
- Snake draft pattern visualization
- Time management system

### [✅ Pick Validation](./pick-validation.md) *(Coming Soon)*
Rules governing valid team selections:
- Team eligibility requirements
- Event participation validation
- Duplicate pick prevention
- Business hour restrictions

## 🎯 Key Concepts

### Scoring Components
- **Qualification Score**: Points from qualification matches
- **Playoff Score**: Points from elimination matches
- **Alliance Score**: Points from alliance selection position
- **Einstein Score**: Doubled playoff points at championship

### Draft Mechanics
- **Snake Draft**: Alternating pick order for fairness
- **Time Limits**: Configurable pick expiration with business hours
- **Real-time Updates**: WebSocket notifications for live drafts
- **Auto-skip**: Automatic progression on expired picks

### Player Experience
- **8 Players**: Standard draft size -- May be configurable in the cuture
- **8 Teams**: Each player drafts 8 teams -- May be configurable in the cuture
- **64 Total Picks**: Complete draft coverage -- May be configurable in the cuture
- **Live Rankings**: Real-time score updates

## 🔗 Related Documentation

- [Architecture Overview](../architecture/system-overview.md)
- [Data Flow](../architecture/data-flow.md)
- [Database Schema](../database/schema.md)
- [API Routes](../api/rest-api.md)

## 🛠️ Implementation Details

### Core Services
- **Scorer**: Background service processing match results
- **Draft Manager**: State machine for draft lifecycle
- **Pick Manager**: Turn-based pick validation and processing
- **WebSocket Hub**: Real-time event broadcasting

### Data Sources
- **The Blue Alliance API**: Official FRC competition data
- **Webhook Integration**: Real-time match result updates
- **Database**: Persistent storage for scores and drafts

---

*Business logic documentation focuses on the rules and algorithms that make Fantasy FRC engaging and fair for all participants.*
