## ADDED Requirements

### Requirement: Strategy 2 Alert Triggering
The system SHALL trigger alerts when all five conditions are met.

#### Scenario: C1 - Topic is in focus list
- **WHEN** checking alert conditions
- **THEN** at least one attributed topic SHALL be in daily_focus_topics
- **AND** the system SHALL check using Redis SISMEMBER focus:topics:{date}

#### Scenario: C2 - Stock was in above-5% pool yesterday
- **WHEN** checking alert conditions
- **THEN** the stock SHALL have pool_type=2 entry in daily_stock_pool for yesterday
- **AND** prev_day_pct SHALL be >= 5.0

#### Scenario: C3 - Stock was not limit-up yesterday
- **WHEN** checking alert conditions
- **THEN** the stock SHALL NOT have pool_type=1 entry in daily_stock_pool for yesterday

#### Scenario: C4 - First limit-up within trading hours
- **WHEN** checking alert conditions
- **THEN** the first_limit_time SHALL be between 09:25 and 15:00

#### Scenario: C5 - Alert idempotency
- **WHEN** checking alert conditions
- **THEN** the system SHALL check Redis SISMEMBER alerted:{date} {ts_code}:{topic_id}
- **AND** SHALL return already_alerted if the key exists

### Requirement: Alert Deduplication
The system SHALL ensure each stock-topic combination alerts only once per day.

#### Scenario: Alert deduplication
- **WHEN** alert triggers successfully
- **THEN** the system SHALL SADD alerted:{date} {ts_code}:{topic_id}
- **AND** SHALL use ON CONFLICT DO NOTHING for database insert

### Requirement: Alert Persistence
The system SHALL persist alert records to database.

#### Scenario: Alert record creation
- **WHEN** alert triggers successfully
- **THEN** the system SHALL INSERT into strategy_alerts table
- **AND** SHALL include trigger_price, trigger_time, prev_day_pct, extra_info (JSONB)

### Requirement: WebSocket Alert Push
The system SHALL push strategy_alert messages immediately when alerts trigger.

#### Scenario: Immediate alert push
- **WHEN** alert triggers
- **THEN** the system SHALL immediately broadcast strategy_alert via WebSocket
- **AND** SHALL NOT wait for 10-second cycle

### Requirement: Dual Attribution Alert Handling
The system SHALL check focus list for both topics in dual attribution.

#### Scenario: Dual attribution alert
- **WHEN** stock has dual attribution (Topic1 and Topic2)
- **THEN** the system SHALL check focus for Topic1 first
- **AND** SHALL check focus for Topic2 if Topic1 not in focus
- **AND** SHALL trigger alert for the first topic found in focus
