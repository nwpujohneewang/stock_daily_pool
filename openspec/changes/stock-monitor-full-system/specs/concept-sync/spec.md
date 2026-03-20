## ADDED Requirements

### Requirement: Tushare Concept List Sync
The system SHALL sync concept board list from Tushare API.

#### Scenario: Full concept list sync
- **WHEN** syncing all concepts
- **THEN** the system SHALL call Tushare concept() API
- **AND** SHALL upsert all concepts to tushare_concepts table

### Requirement: Tushare Concept Detail Sync
The system SHALL sync concept stock members from Tushare API.

#### Scenario: Concept detail sync
- **WHEN** syncing concept details for a concept
- **THEN** the system SHALL call Tushare concept_detail() API with concept ID
- **AND** SHALL upsert stock-concept relations to tushare_concept_details table

#### Scenario: Rate limit handling
- **WHEN** syncing concept details
- **THEN** the system SHALL wait 120ms between calls to respect 500/minute limit
- **AND** SHALL use exponential backoff on failures

### Requirement: Concept-Topic Mapping Building
The system SHALL build mappings between Tushare concepts and system topics.

#### Scenario: Exact match mapping
- **WHEN** concept name exactly matches topic name
- **THEN** the system SHALL create mapping with match_type = 'exact'

#### Scenario: Synonym match mapping
- **WHEN** concept name matches a synonym
- **THEN** the system SHALL create mapping with match_type = 'synonym'

#### Scenario: LLM assisted mapping
- **WHEN** concept name does not match exactly
- **THEN** the system SHALL use LLM to find best matching topic
- **AND** SHALL create mapping with match_type = 'llm'
