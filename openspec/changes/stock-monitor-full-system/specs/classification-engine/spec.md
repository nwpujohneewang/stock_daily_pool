## ADDED Requirements

### Requirement: Four-Layer Cascade Classification
The system SHALL classify stocks into hot topics using a four-layer cascade engine that stops at the first hit.

#### Scenario: L1 Redis cache hit
- **WHEN** cache:stock_topics contains the stock's topic mappings
- **THEN** the system SHALL use these mappings directly (latency < 1ms)
- **AND** SHALL check for manual source mappings

#### Scenario: L1 Redis manual source priority
- **WHEN** L1 cache hit contains a mapping with source = 'manual'
- **THEN** the system SHALL directly adopt this topic
- **AND** SHALL skip attribution algorithm

#### Scenario: L2 PostgreSQL jiuyan mapping hit
- **WHEN** L1 misses but stock_topic_mappings contains entries for the stock
- **THEN** the system SHALL use these mappings
- **AND** SHALL asynchronously backfill Redis cache

#### Scenario: L3 PostgreSQL concept tag hit
- **WHEN** L1 and L2 miss but stock_concept_tags contains concept names for the stock
- **THEN** the system SHALL look up concept_topic_mappings to get topic IDs
- **AND** SHALL use concept weight mode for attribution

#### Scenario: L4 LLM async fallback
- **WHEN** L1, L2, and L3 all miss
- **THEN** the system SHALL asynchronously call LLM API
- **AND** SHALL mark the stock as unclassified for current cycle
- **AND** SHALL write result to cache in next cycle

#### Scenario: LLM low confidence rejection
- **WHEN** LLM returns confidence < 0.7
- **THEN** the system SHALL NOT adopt the classification result
- **AND** SHALL keep the stock as unclassified

### Requirement: Classification Evidence Recording
The system SHALL record classification evidence for every classification attempt.

#### Scenario: Evidence recording for classified stock
- **WHEN** a stock is classified into a topic
- **THEN** the system SHALL write a record to classify_evidence table
- **AND** SHALL include classify_layer, strategy, candidate_scores, confidence

#### Scenario: Evidence recording for unclassified stock
- **WHEN** a stock cannot be classified (all layers miss or LLM low confidence)
- **THEN** the system SHALL write a record with topic_id = 0
- **AND** SHALL record the layer that was reached

### Requirement: Classification Layer Enum
The system SHALL use the following classification layer values:
- L1_REDIS: Redis cache hit
- L2_PG_JIUYAN: PostgreSQL jiuyan mapping hit
- L3_PG_CONCEPT: PostgreSQL concept tag hit
- L4_LLM: LLM async fallback

### Requirement: Classification Strategy Enum
The system SHALL use the following classification strategy values:
- MANUAL: Manual annotation (highest priority)
- JIUYAN_ATTRIBUTION: Jiuyan data with attribution algorithm
- CONCEPT_ATTRIBUTION: Concept data with attribution algorithm (concept weight mode)
- LLM_V1: LLM classification
