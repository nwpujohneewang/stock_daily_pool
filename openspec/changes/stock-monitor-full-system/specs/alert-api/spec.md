## ADDED Requirements

### Requirement: Alert Query API
The system SHALL provide API to query alerts.

#### Scenario: Query today's alerts
- **WHEN** GET /api/v1/alerts
- **THEN** the system SHALL return alerts for today
- **AND** SHALL support filtering by topic_id
- **AND** SHALL support sorting by trigger_time

#### Scenario: Query historical alerts
- **WHEN** GET /api/v1/alerts/history with start_date and end_date
- **THEN** the system SHALL return alerts in date range
- **AND** SHALL support pagination

### Requirement: Alert Summary
The system SHALL provide alert summary statistics.

#### Scenario: Today's alert summary
- **WHEN** GET /api/v1/alerts/today
- **THEN** the system SHALL return total count and topic breakdown
- **AND** SHALL return latest alert

### Requirement: Focus Topics Management
The system SHALL provide API to manage daily focus topics.

#### Scenario: Set focus topics
- **WHEN** POST /api/v1/focus with date and topic_ids
- **THEN** the system SHALL upsert to daily_focus_topics
- **AND** SHALL sync to Redis focus:topics:{date}

#### Scenario: Get focus topics
- **WHEN** GET /api/v1/focus with date
- **THEN** the system SHALL return focus topics for that date

#### Scenario: Remove focus topic
- **WHEN** DELETE /api/v1/focus/:topic_id with date
- **THEN** the system SHALL remove the focus topic
- **AND** SHALL update Redis
