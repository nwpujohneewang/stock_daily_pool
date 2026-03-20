## ADDED Requirements

### Requirement: Tushare Client
The system SHALL provide Tushare API client with retry logic.

#### Scenario: API request with retry
- **WHEN** calling Tushare API
- **THEN** the system SHALL use exponential backoff on failure
- **AND** SHALL retry up to 4 times

#### Scenario: Rate limiting
- **WHEN** calling Tushare API
- **THEN** the system SHALL respect 500 requests/minute limit
- **AND** SHALL use token bucket for rate control

### Requirement: Jiuyan Client
The system SHALL provide Jiuyan API client.

#### Scenario: API request with retry
- **WHEN** calling Jiuyan API
- **THEN** the system SHALL use exponential backoff on failure
- **AND** SHALL convert code formats appropriately

### Requirement: LLM Client
The system SHALL provide LLM API client with concurrency control.

#### Scenario: LLM request with timeout
- **WHEN** calling LLM API
- **THEN** the system SHALL set 30-second timeout
- **AND** SHALL limit concurrent requests to 5

#### Scenario: Response parsing
- **WHEN** LLM returns classification result
- **THEN** the system SHALL parse JSON response
- **AND** SHALL handle parsing errors gracefully

### Requirement: Circuit Breaker
The system SHALL implement circuit breaker pattern.

#### Scenario: Circuit open on consecutive failures
- **WHEN** external API fails 5 times consecutively
- **THEN** the system SHALL open circuit breaker
- **AND** SHALL skip API calls for 60 seconds

#### Scenario: Circuit half-open
- **WHEN** circuit has been open for cooldown period
- **THEN** the system SHALL allow 3 test requests
- **AND** SHALL close circuit on success
