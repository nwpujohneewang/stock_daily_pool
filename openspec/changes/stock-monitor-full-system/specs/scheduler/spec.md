## ADDED Requirements

### Requirement: Pre-Market Initialization
The system SHALL run pre-market initialization task.

#### Scenario: Daily initialization at 09:15
- **WHEN** scheduler triggers at 09:15 on trading day
- **THEN** the system SHALL check if it's a trading day
- **AND** SHALL clean up previous day's temporary Redis keys
- **AND** SHALL initialize today's Redis data structures
- **AND** SHALL load board rules to memory cache

### Requirement: Real-time Collection
The system SHALL run real-time collection every 10 seconds.

#### Scenario: 10-second collection
- **WHEN** scheduler triggers every 10 seconds between 09:25-15:01
- **THEN** the system SHALL process all stocks in parallel shards
- **AND** SHALL compute diff and push WebSocket after all shards complete

### Requirement: Jiuyan Sync
The system SHALL sync jiuyan data daily.

#### Scenario: Daily jiuyan sync at 09:00
- **WHEN** scheduler triggers at 09:00 on trading day
- **THEN** the system SHALL crawl previous trading day's data

### Requirement: Concept Sync
The system SHALL sync concepts weekly.

#### Scenario: Weekly concept sync on Sunday 02:00
- **WHEN** scheduler triggers on Sunday at 02:00
- **THEN** the system SHALL sync new concepts from Tushare
- **AND** SHALL update concept_topic_mappings

### Requirement: Closing Snapshot
The system SHALL save snapshot at 15:05.

#### Scenario: Daily snapshot at 15:05
- **WHEN** scheduler triggers at 15:05 on trading day
- **THEN** the system SHALL persist Redis data to PostgreSQL
- **AND** SHALL stop real-time monitoring

### Requirement: Cache Warmup
The system SHALL warmup cache at 09:10.

#### Scenario: Cache warmup
- **WHEN** scheduler triggers at 09:10 on trading day
- **THEN** the system SHALL load all mappings to Redis cache
- **AND** SHALL load concepts and synonyms

### Requirement: History Cleanup
The system SHALL run history cleanup monthly.

#### Scenario: Monthly cleanup on 1st at 03:00
- **WHEN** scheduler triggers on 1st of month at 03:00
- **THEN** the system SHALL create next month's partition
- **AND** SHALL cleanup expired Redis keys
