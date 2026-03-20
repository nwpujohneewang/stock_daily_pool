## ADDED Requirements

### Requirement: Quote Cache
The system SHALL cache real-time quotes in Redis.

#### Scenario: Quote caching
- **WHEN** fetching new quotes
- **THEN** the system SHALL store in rt:quote:{ts_code} hash
- **AND** SHALL set TTL to end of trading day

### Requirement: Pool Caching
The system SHALL cache pool data in Redis.

#### Scenario: Limit-up pool caching
- **WHEN** stock reaches limit-up
- **THEN** the system SHALL SADD to pool:limit_up:{date}
- **AND** SHALL ZADD to pool:limit_up:sorted:{date} with time as score

#### Scenario: Above-5% pool caching
- **WHEN** stock change >= 5%
- **THEN** the system SHALL SADD to pool:above5:{date}

### Requirement: Mapping Cache
The system SHALL cache stock-topic mappings for fast L1 lookup.

#### Scenario: Cache population
- **WHEN** cache warmup runs
- **THEN** the system SHALL load all stock_topic_mappings to cache:stock_topics
- **AND** SHALL load bind_strength data to cache:bind_strength

#### Scenario: Cache update
- **WHEN** L2 hit occurs
- **THEN** the system SHALL asynchronously update cache:stock_topics

### Requirement: Focus Topics Cache
The system SHALL cache focus topics for fast C1 check.

#### Scenario: Focus cache update
- **WHEN** focus topics are set
- **THEN** the system SHALL SADD to focus:topics:{date}

### Requirement: Alert Deduplication Cache
The system SHALL cache alerted status for idempotency.

#### Scenario: Alert deduplication
- **WHEN** alert triggers
- **THEN** the system SHALL SADD to alerted:{date}

### Requirement: Topic Activity Tracking
The system SHALL track topic limit-up activity in real-time.

#### Scenario: Activity increment
- **WHEN** stock reaches limit-up for a topic
- **THEN** the system SHALL ZINCRBY topic:activity:limit:{date} {topic_id} 1

#### Scenario: Activity decrement
- **WHEN** limit-up stock opens
- **THEN** the system SHALL ZINCRBY topic:activity:limit:{date} {topic_id} -1

### Requirement: Snapshot Caching
The system SHALL maintain current snapshot in Redis.

#### Scenario: Snapshot update
- **WHEN** 10-second cycle completes
- **THEN** the system SHALL update snapshot:current with full state
