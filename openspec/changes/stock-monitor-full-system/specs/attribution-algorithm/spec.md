## ADDED Requirements

### Requirement: Four-Dimensional Attribution Scoring
The system SHALL calculate attribution scores for candidate topics using four dimensions.

#### Scenario: S1 Activity Score calculation
- **WHEN** calculating S1 for a topic
- **THEN** the system SHALL use formula: S_activity = topic_limit_count / max_limit_count_all_topics
- **AND** SHALL normalize to range [0.0, 1.0]

#### Scenario: S2 Bind Strength Score calculation
- **WHEN** calculating S2 for a topic (normal mode only)
- **THEN** the system SHALL use formula: S_bindStrength = hit_count / total_hit_count_for_stock
- **AND** SHALL return 0.0 when total_hit_count = 0

#### Scenario: S3 Time Proximity Score calculation
- **WHEN** calculating S3 for a topic
- **THEN** the system SHALL use formula: S_timeProximity = 1.0 / (1.0 + avg_time_gap_minutes / 15.0)
- **AND** SHALL return 0.5 (neutral) when no prior limit-ups exist for the topic

#### Scenario: S4 Recency Score calculation
- **WHEN** calculating S4 for a topic
- **THEN** the system SHALL use formula: S_recency = 1.0 / (1.0 + days_since_last_seen / 30.0)
- **AND** SHALL return 0.1 when last_seen_date is null

### Requirement: Normal Weight Mode
The system SHALL use normal weight mode when jiuyan mappings exist.

#### Scenario: Normal weight configuration
- **WHEN** attribution is in normal mode
- **THEN** the system SHALL use weights: {activity: 0.25, bindStrength: 0.40, timeProximity: 0.20, recency: 0.15}

### Requirement: Concept Weight Mode
The system SHALL use concept weight mode when only concept tags exist (no jiuyan data).

#### Scenario: Concept weight configuration
- **WHEN** attribution is in concept mode
- **THEN** the system SHALL use weights: {activity: 0.45, bindStrength: 0.00, timeProximity: 0.40, recency: 0.15}
- **AND** SHALL skip S2 calculation (weight = 0)

### Requirement: Total Score Calculation
The system SHALL calculate total score as weighted sum of four dimensions.

#### Scenario: Total score formula
- **WHEN** calculating total score
- **THEN** the system SHALL use: TotalScore = W1*S1 + W2*S2 + W3*S3 + W4*S4
- **AND** SHALL round to 4 decimal places

### Requirement: Attribution Decision
The system SHALL decide attribution based on score gaps.

#### Scenario: Single attribution
- **WHEN** there are >= 2 candidates and Top1.Score - Top2.Score >= 0.1
- **THEN** the system SHALL return only Top1 as attribution

#### Scenario: Dual attribution
- **WHEN** there are >= 2 candidates and Top1.Score - Top2.Score < 0.1
- **THEN** the system SHALL return both Top1 and Top2 as attribution
- **AND** SHALL mark IsDualAttribution = true

#### Scenario: No valid candidates
- **WHEN** all candidates are filtered out or score = 0
- **THEN** the system SHALL return empty attribution
- **AND** SHALL mark as unclassified

### Requirement: Active Topic Filtering
The system SHALL only consider topics with limit-up activity today.

#### Scenario: Topic filtering
- **WHEN** filtering candidate topics
- **THEN** the system SHALL only keep topics with topic_limit_count > 0
- **AND** SHALL exclude inactive topics from scoring
