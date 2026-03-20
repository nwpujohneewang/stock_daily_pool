## ADDED Requirements

### Requirement: Daily Snapshot Persistence
The system SHALL persist Redis pool data to PostgreSQL at 15:05 daily.

#### Scenario: Snapshot persistence
- **WHEN** closing snapshot triggers at 15:05
- **THEN** the system SHALL read limit-up pool and above-5% pool from Redis
- **AND** SHALL batch INSERT/UPDATE to daily_stock_pool table

#### Scenario: Idempotent persistence
- **WHEN** persisting snapshot data
- **THEN** the system SHALL use ON CONFLICT (date, ts_code, pool_type) DO UPDATE
- **AND** SHALL be safe for crash recovery and restart

### Requirement: Monthly Partition Creation
The system SHALL automatically create next month's partition table.

#### Scenario: Partition creation
- **WHEN** history cleanup triggers on 1st of month
- **THEN** the system SHALL create partition table for next month
- **AND** SHALL use naming convention: daily_stock_pool_YYYY_MM
