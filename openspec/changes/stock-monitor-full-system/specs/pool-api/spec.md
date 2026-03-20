## ADDED Requirements

### Requirement: Real-time Pool Query
The system SHALL provide API to query real-time pools.

#### Scenario: Query limit-up pool
- **WHEN** GET /api/v1/pool/limit-up
- **THEN** the system SHALL return stocks grouped by topics
- **AND** SHALL include unclassified stocks in separate section
- **AND** SHALL support sorting by first_limit_time

#### Scenario: Query above-5% pool
- **WHEN** GET /api/v1/pool/above5
- **THEN** the system SHALL return stocks with change_pct >= 5%
- **AND** SHALL exclude limit-up stocks

### Requirement: Snapshot Query
The system SHALL provide snapshot API for current state.

#### Scenario: Get current snapshot
- **WHEN** GET /api/v1/pool/limit-up/snapshot
- **THEN** the system SHALL return Redis snapshot:current data
- **AND** SHALL include both limit-up and above-5% sections

### Requirement: Historical Pool Query
The system SHALL provide API to query historical pool data.

#### Scenario: Query historical pool
- **WHEN** GET /api/v1/pool/history with date
- **THEN** the system SHALL query daily_stock_pool for that date
- **AND** SHALL support filtering by pool_type and topic_id
- **AND** SHALL return paginated results
