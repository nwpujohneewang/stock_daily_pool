## ADDED Requirements

### Requirement: Real-time Market Data Collection
The system SHALL collect real-time market quotes for approximately 5000 A-share stocks every 10 seconds during trading hours (09:25-15:01).

#### Scenario: Successful quote collection
- **WHEN** the 10-second ticker triggers during trading hours
- **THEN** the system SHALL fetch quotes for all active stocks in batches of 50 from Tushare API

#### Scenario: Quote fetch failure handling
- **WHEN** a batch of quotes fails to fetch
- **THEN** the system SHALL retry with exponential backoff (1s→2s→4s→8s)
- **AND** after 4 retries failure, SHALL record the error and continue with next batch

#### Scenario: Non-trading hours skip
- **WHEN** the ticker triggers outside trading hours (before 09:25 or after 15:01)
- **THEN** the system SHALL skip the collection cycle

### Requirement: Stock Sharding
The system SHALL partition approximately 5000 stocks into 16 shards using hash(ts_code) % 16 for parallel processing.

#### Scenario: Shard assignment determinism
- **WHEN** a stock is assigned to a shard
- **THEN** the same stock SHALL always be assigned to the same shard across all cycles

#### Scenario: Shard parallel execution
- **WHEN** processing starts for a cycle
- **THEN** all 16 shards SHALL be processed in parallel using errgroup
- **AND** individual stock failures within a shard SHALL NOT affect other stocks in the same shard

### Requirement: Limit-up Detection
The system SHALL detect when a stock reaches its limit-up price based on board rules.

#### Scenario: Limit-up detection for main board
- **WHEN** a main board stock (board_code=MAIN) price >= pre_close * 1.10
- **THEN** the system SHALL mark the stock as limit-up

#### Scenario: Limit-up detection for GEM board
- **WHEN** a GEM board stock (board_code=GEM, codes starting with 300/301) price >= pre_close * 1.20
- **THEN** the system SHALL mark the stock as limit-up

#### Scenario: Limit-up detection for STAR board
- **WHEN** a STAR board stock (board_code=STAR, codes starting with 688) price >= pre_close * 1.20
- **THEN** the system SHALL mark the stock as limit-up

#### Scenario: Limit-up detection for BSE board
- **WHEN** a BSE board stock (board_code=BSE, codes starting with 8/4) price >= pre_close * 1.30
- **THEN** the system SHALL mark the stock as limit-up

#### Scenario: ST stock filtering
- **WHEN** a stock has is_st = TRUE
- **THEN** the system SHALL skip the stock from all limit-up and above-5% calculations

### Requirement: Limit State Machine
The system SHALL track stock limit-up state using a state machine with抖动 suppression.

#### Scenario: First limit-up event
- **WHEN** a stock transitions from NONE to LIMIT_UP state
- **THEN** the system SHALL record the first_limit_time
- **AND** shall mark IsFirstLimitUp = true

#### Scenario: Limit-up open (炸板)
- **WHEN** a stock in LIMIT_UP state drops below limit-up price
- **THEN** the system SHALL transition to OPENED state
- **AND** SHALL NOT update first_limit_time

#### Scenario: Re-seal detection
- **WHEN** a stock in OPENED state reaches limit-up price again
- **THEN** the system SHALL transition to RE_SEALED state
- **AND** SHALL NOT mark IsFirstLimitUp = true (already marked in first event)

### Requirement: Above-5% Detection
The system SHALL detect stocks with change percentage >= 5% but not at limit-up.

#### Scenario: Above-5% detection
- **WHEN** a stock change_pct >= 5.0 AND is not at limit-up
- **THEN** the system SHALL add the stock to the above-5% pool

#### Scenario: Exclusion from above-5% pool
- **WHEN** a stock is at limit-up
- **THEN** the system SHALL NOT add the stock to the above-5% pool (limit-up takes priority)

### Requirement: Redis Pool Updates
The system SHALL update Redis pools when stock status changes.

#### Scenario: Add to limit-up pool
- **WHEN** a stock reaches limit-up for the first time (IsFirstLimitUp = true)
- **THEN** the system SHALL SADD the stock to pool:limit_up:{date}
- **AND** SHALL ZADD to pool:limit_up:sorted:{date} with first_limit_time as score

#### Scenario: Remove from limit-up pool
- **WHEN** a limit-up stock opens (state becomes OPENED)
- **THEN** the system SHALL SREM the stock from pool:limit_up:{date}
- **AND** SHALL ZREM from pool:limit_up:sorted:{date}

#### Scenario: First limit time recording
- **WHEN** IsFirstLimitUp = true for a stock
- **THEN** the system SHALL HSETNX first_limit:{date} {ts_code} {time} to record first limit-up time (only writes once)
