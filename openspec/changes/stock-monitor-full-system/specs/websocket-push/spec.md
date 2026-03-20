## ADDED Requirements

### Requirement: WebSocket Connection
The system SHALL accept WebSocket connections at /ws endpoint.

#### Scenario: Connection establishment
- **WHEN** client connects to /ws?token={api_key}
- **THEN** the system SHALL validate the API key
- **AND** SHALL send pool_snapshot immediately after connection

#### Scenario: Heartbeat mechanism
- **WHEN** connection is established
- **THEN** the system SHALL send ping every 50 seconds
- **AND** SHALL disconnect if no pong received within 60 seconds

### Requirement: Pool Snapshot Push
The system SHALL push full pool snapshot on connection and large changes.

#### Scenario: Initial snapshot push
- **WHEN** client connects
- **THEN** the system SHALL send pool_snapshot with current state

#### Scenario: Large change push
- **WHEN** pool change rate >= 30%
- **THEN** the system SHALL send pool_snapshot instead of pool_diff

### Requirement: Pool Diff Push
The system SHALL push incremental changes when change rate < 30%.

#### Scenario: Diff push
- **WHEN** pool changes with rate < 30%
- **THEN** the system SHALL send pool_diff with added and removed stocks

### Requirement: Strategy Alert Push
The system SHALL immediately push strategy alerts.

#### Scenario: Alert push
- **WHEN** strategy alert triggers
- **THEN** the system SHALL immediately send strategy_alert message
- **AND** SHALL NOT wait for 10-second cycle

### Requirement: Backpressure Handling
The system SHALL handle slow clients gracefully.

#### Scenario: Buffer full handling
- **WHEN** client's send buffer reaches 256 messages
- **THEN** the system SHALL close the client connection
- **AND** SHALL log warning with client_id
