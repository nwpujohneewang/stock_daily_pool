## ADDED Requirements

### Requirement: Jiuyan API Crawling
The system SHALL crawl hot topic data from Jiuyan Gongshe API.

#### Scenario: Single date crawl
- **WHEN** crawling data for a specific date
- **THEN** the system SHALL call GET /action/field?date={YYYY-MM-DD}
- **AND** SHALL convert jiuyan code format (sz000001) to tushare format (000001.SZ)

#### Scenario: Synonym normalization
- **WHEN** processing crawled topic names
- **THEN** the system SHALL check topic_synonyms table first
- **AND** SHALL normalize aliases to main topic

#### Scenario: New topic creation
- **WHEN** a crawled topic does not match existing topic or synonym
- **THEN** the system SHALL create a new topic with source = 'jiuyan'

### Requirement: Mapping Upsert
The system SHALL upsert stock-topic mappings from crawl data.

#### Scenario: Mapping upsert with hit count increment
- **WHEN** stock-topic mapping exists from previous crawl
- **THEN** the system SHALL increment hit_count by 1
- **AND** SHALL update last_seen_date

#### Scenario: New mapping creation
- **WHEN** stock-topic mapping does not exist
- **THEN** the system SHALL INSERT with hit_count = 1
- **AND** SHALL set first_seen_date and last_seen_date

### Requirement: Crawl Retry Logic
The system SHALL retry failed crawl operations with exponential backoff.

#### Scenario: Crawl failure retry
- **WHEN** API call fails
- **THEN** the system SHALL retry with delays: 1s → 2s → 4s → 8s
- **AND** SHALL mark date as failed in jiuyan_crawl_logs after 4 failures

### Requirement: Historical Crawl
The system SHALL crawl historical data from crawl_start_date to current date.

#### Scenario: Historical crawl skipping completed dates
- **WHEN** crawling historical data
- **THEN** the system SHALL skip dates with status=1 in jiuyan_crawl_logs
- **AND** SHALL only crawl dates without successful completion

### Requirement: Evidence Text Recording
The system SHALL record the explanation text from jiuyan API.

#### Scenario: Evidence recording
- **WHEN** crawling stock-topic relation with explanation
- **THEN** the system SHALL store article.action_info.expound as evidence_text
- **AND** SHALL include it in classify_evidence record
