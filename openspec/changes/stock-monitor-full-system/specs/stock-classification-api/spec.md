## ADDED Requirements

### Requirement: Manual Classification
The system SHALL allow manual topic annotation for stocks with highest priority.

#### Scenario: Manual classification creation
- **WHEN** POST /api/v1/stock/:ts_code/topics with topic_id
- **THEN** the system SHALL create/update mapping with source = 'manual'
- **AND** SHALL set confidence = 1.0

#### Scenario: Manual classification cannot be overwritten
- **WHEN** automatic process tries to update manual mapping
- **THEN** the system SHALL skip the update (WHERE source != 'manual')

### Requirement: Classification Removal
The system SHALL allow removal of topic mappings.

#### Scenario: Remove mapping
- **WHEN** DELETE /api/v1/stock/:ts_code/topics/:topic_id
- **THEN** the system SHALL delete the mapping
- **AND** SHALL only allow deletion of source = 'manual' mappings

### Requirement: Reclassification Trigger
The system SHALL trigger reclassification on demand.

#### Scenario: Manual reclassification
- **WHEN** POST /api/v1/stock/:ts_code/reclassify
- **THEN** the system SHALL run full four-layer classification
- **AND** SHALL return all candidate scores and final attribution

### Requirement: Evidence Query
The system SHALL provide query interface for classification evidence.

#### Scenario: Query evidence by stock and date
- **WHEN** GET /api/v1/stock/:ts_code/evidence
- **THEN** the system SHALL return evidence records for the stock
- **AND** SHALL include candidate_scores JSON with all scoring details

### Requirement: Evidence Correction
The system SHALL allow manual correction of classification results.

#### Scenario: Correct attribution
- **WHEN** PUT /api/v1/stock/:ts_code/evidence/:id/correct with corrected_topic_id
- **THEN** the system SHALL update corrected_topic_id in evidence record
- **AND** SHALL use the correction for future training data
