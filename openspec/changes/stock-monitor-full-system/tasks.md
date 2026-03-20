## 1. Project Setup

- [x] 1.1 Initialize Go module with go mod init
- [x] 1.2 Create directory structure (cmd, internal, migrations, config)
- [x] 1.3 Add dependencies (pgx/v5, go-redis, gin, Hertz, viper, zap, cron, websocket)
- [x] 1.4 Create config.yaml and config loader
- [x] 1.5 Setup logging with zap
- [x] 1.6 Create Makefile with build/run/test commands

## 2. Data Layer (PostgreSQL)

- [x] 2.1 Create migrations/001_create_stock_basic_info.sql
- [x] 2.2 Create migrations/002_create_topics.sql
- [x] 2.3 Create migrations/003_create_topic_synonyms.sql
- [x] 2.4 Create migrations/004_create_tushare_concepts.sql
- [x] 2.5 Create migrations/005_create_topic_concepts.sql
- [x] 2.6 Create migrations/006_create_stock_topic_relations.sql
- [x] 2.7 Create migrations/007_create_tushare_concept_details.sql
- [x] 2.8 Create migrations/008_create_focus_topics.sql
- [x] 2.9 Create migrations/009_create_daily_stock_pool.sql (with partitioning)
- [x] 2.10 Create migrations/010_create_strategy_alerts.sql
- [x] 2.11 Create migrations/011_create_classification_audit_log.sql
- [x] 2.12 Create migrations/012_create_market_snapshots.sql
- [x] 2.13 Implement model structs for all tables
- [x] 2.14 Implement repository layer (stock_repo, topic_repo, mapping_repo, etc.)

## 3. Cache Layer (Redis)

- [x] 3.1 Setup Redis client connection
- [x] 3.2 Implement quote_cache.go (rt:quote:{ts_code})
- [x] 3.3 Implement pool_cache.go (pool:limit_up, pool:above5, sorted sets)
- [x] 3.4 Implement mapping_cache.go (cache:stock_topics, cache:bind_strength)
- [x] 3.5 Implement concept_cache.go (cache:stock_concepts, cache:concept_to_topics)
- [x] 3.6 Implement focus_cache.go (focus:topics:{date})
- [x] 3.7 Implement activity_cache.go (topic:activity:limit:{date})
- [x] 3.8 Implement snapshot_cache.go (snapshot:current)
- [x] 3.9 Implement alert_dedup_cache.go (alerted:{date})
- [x] 3.10 Implement cache warmup service

## 4. External Clients

- [x] 4.1 Implement Tushare client with retry and rate limiting
- [x] 4.2 Implement Jiuyan client with code format conversion
- [x] 4.3 Implement LLM client with timeout and concurrency control
- [x] 4.4 Implement circuit breaker pattern
- [x] 4.5 Implement exponential backoff retry

## 5. Core Services

- [x] 5.1 Implement board rule detection and limit price calculation
- [x] 5.2 Implement limit state machine with抖动 suppression
- [x] 5.3 Implement MonitorService with 16-shard parallel processing
- [x] 5.4 Implement ClassifyService (four-layer cascade)
- [x] 5.5 Implement AttributionEngine (four-dimensional scoring)
- [x] 5.6 Implement AlertService (strategy 2 detection)
- [x] 5.7 Implement CrawlerService (Jiuyan full/incremental crawl)
- [x] 5.8 Implement ConceptSyncService (Tushare concept sync)
- [x] 5.9 Implement SnapshotService (closing snapshot persistence)
- [x] 5.10 Implement TopicService (CRUD, synonyms, merge)
- [x] 5.11 Implement synonym normalization

## 6. WebSocket

- [x] 6.1 Implement Hub (connection management, broadcast)
- [x] 6.2 Implement Client (read/write pumps, ping/pong)
- [x] 6.3 Implement message types (pool_snapshot, pool_diff, strategy_alert)
- [x] 6.4 Implement backpressure handling
- [x] 6.5 Implement snapshot push on connection

## 7. HTTP API

- [x] 7.1 Setup Gin/Hertz router with middleware
- [x] 7.2 Implement auth middleware (X-API-Key)
- [x] 7.3 Implement pool handlers (limit-up, above5, snapshot, history)
- [x] 7.4 Implement topic handlers (CRUD, synonyms, concepts)
- [x] 7.5 Implement stock classification handlers
- [x] 7.6 Implement alert handlers
- [x] 7.7 Implement focus handlers
- [x] 7.8 Implement admin handlers (crawl, cache refresh, stats)
- [x] 7.9 Implement health check endpoint

## 8. Scheduler

- [x] 8.1 Setup cron scheduler with 6-field expressions
- [x] 8.2 Implement pre-market initialization task
- [x] 8.3 Implement real-time collection task (10-second ticker)
- [x] 8.4 Implement Jiuyan sync task (daily at 09:00)
- [x] 8.5 Implement concept sync task (weekly on Sunday)
- [x] 8.6 Implement closing snapshot task (15:05)
- [x] 8.7 Implement cache warmup task (09:10)
- [x] 8.8 Implement history cleanup task (monthly)
- [x] 8.9 Implement partition auto-creation

## 9. Initialization & Main

- [x] 9.1 Implement database migration runner
- [x] 9.2 Implement system initialization sequence (8 steps)
- [x] 9.3 Implement graceful shutdown
- [x] 9.4 Wire up all services in main.go
- [x] 9.5 Add trading calendar detection

## 10. Testing & Documentation

- [x] 10.1 Write unit tests for AttributionEngine
- [x] 10.2 Write unit tests for limit price calculation
- [x] 10.3 Write unit tests for code format conversion
- [x] 10.4 Write unit tests for state machine
- [x] 10.5 Create README.md with setup instructions
- [x] 10.6 Add sample config.yaml
