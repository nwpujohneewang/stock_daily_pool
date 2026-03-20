## ADDED Requirements

### Requirement: Topic CRUD Operations
The system SHALL provide CRUD operations for hot topics.

#### Scenario: Create topic
- **WHEN** POST /api/v1/topics with topic name
- **THEN** the system SHALL create topic with source = 'manual'
- **AND** SHALL return conflict error if name already exists

#### Scenario: Update topic
- **WHEN** PUT /api/v1/topics/:id
- **THEN** the system SHALL update name, is_active, priority
- **AND** SHALL return 404 if topic not found

#### Scenario: Soft delete topic
- **WHEN** DELETE /api/v1/topics/:id
- **THEN** the system SHALL set is_active = FALSE
- **AND** SHALL NOT physically delete the record

#### Scenario: List topics with pagination
- **WHEN** GET /api/v1/topics with pagination params
- **THEN** the system SHALL return topics sorted by priority DESC
- **AND** SHALL support keyword search on name

### Requirement: Synonym Management
The system SHALL provide synonym management for topics.

#### Scenario: Add synonym
- **WHEN** POST /api/v1/topics/:id/synonyms
- **THEN** the system SHALL add synonym to topic_synonyms table
- **AND** SHALL return conflict error if synonym already exists

#### Scenario: Delete synonym
- **WHEN** DELETE /api/v1/topics/synonyms/:synonym_id
- **THEN** the system SHALL physically delete the synonym record

### Requirement: Topic Merge
The system SHALL merge one topic into another.

#### Scenario: Merge topics
- **WHEN** PUT /api/v1/topics/:id/merge with target_topic_id
- **THEN** the system SHALL migrate all stock_topic_mappings to target
- **AND** SHALL migrate all synonyms to target
- **AND** SHALL add source topic name as synonym of target
- **AND** SHALL set source topic is_active = FALSE
- **AND** SHALL NOT allow merging topic into itself
