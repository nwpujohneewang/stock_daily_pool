## Why

A股市场短线交易中，涨停板和强势股（涨幅>5%）是核心选股指标。当前缺乏一个能实时监控、热点归因、策略告警的一体化系统。手动筛选效率低、信息滞后，无法捕捉热点轮动机会。

本系统旨在构建一个**实时涨停板/强势股监控系统**，自动采集行情数据、判断涨停状态、四层级联分类热点归属、触发策略告警，为短线交易提供数据支撑。

## What Changes

从零构建一个完整的 Go 后端系统，包含以下核心功能：

1. **实时行情监控** — 每10秒拉取全市场约5000只股票行情，按16分片并发处理
2. **涨停/强势股判定** — 根据板块规则（主板10%、创业板/科创板20%、北交所30%）计算涨停价，状态机抑制抖动
3. **四层级联分类引擎** — Redis缓存 → PostgreSQL韭研映射 → Tushare概念标签 → LLM兜底
4. **四维加权归因算法** — 热点活跃度、关联强度、时间聚集性、时效性四维度评分
5. **策略告警引擎** — 昨日5%+未涨停 + 今日首次涨停 → 即时告警
6. **韭研公社爬虫** — 全量/增量爬取热点和股票映射，同义词归一化
7. **Tushare概念同步** — 概念板块数据同步，概念→热点映射建立
8. **WebSocket实时推送** — pool_snapshot / pool_diff / strategy_alert 三种消息类型
9. **RESTful API管理** — 热点CRUD、股票标注、告警查询、系统管理
10. **定时调度** — 盘前初始化、实时采集、收盘快照、缓存预热

## Capabilities

### New Capabilities

- `stock-monitor-core`: 核心监控系统（MonitorService），负责实时行情采集、涨停判定、分片并发处理
- `classification-engine`: 四层级联分类引擎（ClassifyService），支持Redis/PG/LLM多层降级
- `attribution-algorithm`: 四维加权归因算法，按热点活跃度/关联强度/时间聚集性/时效性评分
- `alert-engine`: 策略告警引擎（AlertService），判定并推送策略2告警
- `jiuyan-crawler`: 韭研公社爬虫服务（CrawlerService），全量/增量爬取热点数据
- `concept-sync`: Tushare概念同步服务（ConceptSyncService），同步概念板块和成分股
- `snapshot-service`: 收盘快照服务（SnapshotService），Redis数据持久化到PG
- `topic-management`: 热点管理API，CRUD、同义词管理、热点合并
- `stock-classification-api`: 股票分类API，手动标注、重新分类、证据查询
- `pool-api`: 实时池API，涨停池/5%池查询、历史快照
- `alert-api`: 告警API，即时告警、历史查询、汇总统计
- `websocket-push`: WebSocket实时推送，全量快照/增量变更/策略告警
- `cache-layer`: Redis缓存层，行情/池子/映射/概念/活跃度缓存
- `data-layer`: PostgreSQL数据层，12张表完整Schema
- `scheduler`: 定时调度器，8个定时任务管理
- `external-clients`: 外部API客户端（Tushare/韭研/LLM），带熔断和退避重试

### Modified Capabilities

（无，从零开始）

## Impact

- **语言**: Go 1.22+，使用Hertz框架
- **数据库**: PostgreSQL 15+ (主存储) + Redis 7+ (实时缓存)
- **外部依赖**: Tushare Pro API、韭研公社API、OpenAI兼容LLM
- **目录结构**: cmd/server、internal/{config,model,repo,service,handler,middleware,ws,scheduler,cache,external,pkg}
- **数据量级**: ~5000只股票，~400个概念板块，~N个热点（动态增长）
- **性能目标**: 10秒周期完成全市场处理，<1ms Redis查询延迟
