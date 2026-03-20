## ADDED Requirements

### Requirement: Database Schema Creation
The system SHALL create 12 PostgreSQL tables as defined in AGENTS.md.

#### Scenario: Tables creation order
- **WHEN** running migrations
- **THEN** the system SHALL create tables in dependency order
- **AND** SHALL create indexes after table creation

### Requirement: Stock Basic Info Table
The system SHALL store stock basic information.

#### Scenario: Stock data upsert
- **WHEN** syncing stock data from Tushare
- **THEN** the system SHALL use ON CONFLICT (ts_code) DO UPDATE
- **AND** SHALL auto-detect board_code from stock code

### Requirement: Topics Table
The system SHALL store hot topic information.

#### Scenario: Topic upsert
- **WHEN** creating or updating topic
- **THEN** the system SHALL use ON CONFLICT (name) DO UPDATE
- **AND** SHALL increment occurrence_count

### Requirement: Stock-Topic Relations Table
The system SHALL store stock-topic mappings.

#### Scenario: Mapping upsert with manual protection
- **WHEN** upserting mapping
- **THEN** the system SHALL NOT update mappings with source = 'manual'
- **AND** SHALL use WHERE source != 'manual'

### Requirement: Daily Stock Pool Partitioning
The system SHALL use monthly range partitioning on daily_stock_pool.

#### Scenario: Partition query
- **WHEN** querying historical pool data
- **THEN** PostgreSQL SHALL use partition pruning automatically

### Requirement: JSONB Fields
The system SHALL use JSONB for flexible data storage.

#### Scenario: Extra info storage
- **WHEN** storing alert extra info
- **THEN** the system SHALL use JSONB type
- **AND** SHALL support efficient queries on JSON fields

### Requirement: Array Fields
The system SHALL use PostgreSQL arrays for topic IDs.

#### Scenario: Topic IDs array
- **WHEN** storing attributed topic IDs
- **THEN** the system SHALL use BIGINT[] type
- **AND** SHALL support GIN indexing for containment queries
