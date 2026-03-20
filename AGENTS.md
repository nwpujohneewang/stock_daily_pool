# A股实时热点监控系统 — AI Coding 技术规范

> **用途**：本文件为 AI 编码工具（Cursor / Claude Code / Copilot）提供完整的项目上下文。
> 包含项目规则、数据库 Schema、业务逻辑、API 契约、配置与调度全部内容。
> 所有代码均为 Go 语言，可直接复制使用。

---

**目录**

| 章节 | 说明 | 快速跳转 |
|------|------|----------|
| 一、项目总览与开发规则 | 技术栈、目录结构、10条核心RULE、编码约定、架构图 | [跳转](#一项目总览与开发规则) |
| 二、数据库 Schema | 12张PG表DDL + Go struct + Redis 15 key | [跳转](#二数据库-schemapostgresql--redis) |
| 三、业务规则 | 涨停判定、四层分类、四维归因、策略告警等9大模块 | [跳转](#三业务规则business-rules) |
| 四、API 契约与 WebSocket | 37个REST端点 + WebSocket协议 + 错误码 | [跳转](#四api-契约与-websocket-协议) |
| 五、配置、调度与运维 | config.yaml、8个定时任务、初始化流程、容错规则 | [跳转](#五配置调度与运维) |

---

## 一、项目总览与开发规则

### 1.1 Project Identity

| 属性 | 值                                      |
|---|----------------------------------------|
| 项目名 | stock-monitor                          |
| 一句话描述 | A股涨停板/强势股实时监控 + 热点归因 + 策略告警系统          |
| 语言 | Go 1.22+                               |
| HTTP 框架 | Hertz                                  |
| 数据库 | PostgreSQL 15+ (主存储) + Redis 7+ (实时缓存) |
| ORM / Driver | pgx/v5 (原生 SQL，不使用 ORM)                |
| 前端 | React（后期，不在本仓库）                        |
| 实时推送 | WebSocket (gorilla/websocket)          |
| 定时调度 | robfig/cron/v3                         |
| 日志 | go.uber.org/zap (结构化 JSON)             |
| 配置 | spf13/viper (YAML + 环境变量)              |
| 并发控制 | golang.org/x/sync/errgroup             |
| 外部行情源 | Tushare Pro API                        |
| 外部热点源 | 韭研公社 API                               |
| 外部概念源 | Tushare 概念板块 API                       |
| LLM 兜底 | OpenAI-compatible API (gpt-4o-mini)    |
| 时区 | Asia/Shanghai（全局强制）                    |
| 股票代码格式 | Tushare 格式，如 `000001.SZ`、`688256.SH`   |

---

### 1.2 推荐目录结构

```text
stock-monitor/
├── cmd/
│   └── server/
│       └── main.go                     # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go                   # Viper 配置管理
│   ├── model/                          # 数据模型（纯结构体，无业务逻辑）
│   │   ├── stock.go                    # Stock / StockQuote
│   │   ├── board.go                    # BoardRule
│   │   ├── topic.go                    # HotTopic
│   │   ├── synonym.go                  # TopicSynonym
│   │   ├── mapping.go                  # StockTopicMapping
│   │   ├── concept.go                  # StockConceptTag / ConceptTopicMapping
│   │   ├── pool.go                     # DailyStockPool
│   │   ├── alert.go                    # StrategyAlert
│   │   └── evidence.go                 # ClassifyEvidence
│   ├── repo/                           # 数据访问层（PostgreSQL CRUD，纯 SQL）
│   │   ├── stock_repo.go
│   │   ├── board_repo.go
│   │   ├── topic_repo.go
│   │   ├── synonym_repo.go
│   │   ├── mapping_repo.go
│   │   ├── concept_repo.go
│   │   ├── pool_repo.go
│   │   ├── alert_repo.go
│   │   └── evidence_repo.go
│   ├── service/                        # 业务逻辑层
│   │   ├── monitor.go                  # MonitorService — 实时监控引擎 + 分片
│   │   ├── classify.go                 # ClassifyService — 四层分类 + 归因
│   │   ├── alert.go                    # AlertService — 策略告警
│   │   ├── crawler.go                  # CrawlerService — 韭研爬虫
│   │   ├── concept_sync.go            # ConceptSyncService — 概念同步
│   │   ├── snapshot.go                 # SnapshotService — 每日快照
│   │   └── stock.go                    # StockService — 股票基础
│   ├── handler/                        # HTTP Handler（Gin handler func）
│   │   ├── topic_handler.go            # 热点 CRUD
│   │   ├── synonym_handler.go          # 同义词管理
│   │   ├── pool_handler.go             # 池子查询
│   │   ├── alert_handler.go            # 告警查询
│   │   ├── focus_handler.go            # 关注热点
│   │   ├── evidence_handler.go         # 分类证据查询/修正
│   │   ├── concept_handler.go          # 概念映射管理
│   │   └── ws_handler.go              # WebSocket 入口
│   ├── middleware/                      # Gin 中间件
│   │   ├── auth.go                     # X-API-Key 鉴权
│   │   └── audit.go                    # 写操作审计日志
│   ├── ws/                             # WebSocket 核心
│   │   ├── hub.go                      # Hub — 连接管理 + 广播
│   │   └── client.go                   # Client — 单连接读写
│   ├── scheduler/                      # 定时任务
│   │   ├── ticker.go                   # 10s 轮询调度（含分片分发）
│   │   ├── daily_task.go              # 每日爬虫/快照/缓存刷新
│   │   └── partition.go                # PG 按月分区表自动创建
│   ├── cache/                          # Redis 缓存操作封装
│   │   ├── quote_cache.go             # rt:quote:{ts_code}
│   │   ├── pool_cache.go              # pool:limit_up / pool:above5
│   │   ├── mapping_cache.go           # cache:stock_topics / cache:bind_strength
│   │   └── concept_cache.go           # cache:stock_concepts / cache:concept_to_topics
│   ├── external/                       # 外部 API 客户端
│   │   ├── tushare/
│   │   │   ├── client.go              # HTTP 客户端 + token
│   │   │   ├── realtime.go            # 实时行情接口
│   │   │   └── concept.go             # concept() + concept_detail()
│   │   ├── jiuyan/
│   │   │   ├── client.go              # HTTP 客户端 + 代码格式转换
│   │   │   └── types.go               # 响应结构体
│   │   └── llm/
│   │       ├── client.go              # OpenAI-compatible 客户端
│   │       └── classify.go            # 分类 prompt + 解析
│   └── pkg/                            # 通用工具包
│       ├── limiter/
│       │   └── limit_calc.go          # 涨停价计算（从 boards 表读取比例）
│       ├── converter/
│       │   └── code_converter.go      # sz002445 <-> 002445.SZ 互转
│       ├── retry/
│       │   └── backoff.go             # 指数退避重试中间件
│       └── shard/
│           └── shard.go               # hash(ts_code) % N 分片
├── migrations/                         # SQL 迁移文件（顺序执行）
│   ├── 001_create_boards.sql
│   ├── 002_create_stocks.sql
│   ├── 003_create_topics_and_synonyms.sql
│   ├── 004_create_mappings.sql
│   ├── 005_create_concepts.sql
│   ├── 006_create_pools.sql
│   ├── 007_create_alerts.sql
│   ├── 008_create_evidence.sql
│   └── 009_create_crawl_logs.sql
├── config/
│   └── config.yaml                     # 配置文件模板
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

### 1.3 技术栈明细

| 层级 | 技术 | 版本 | 用途 |
|---|---|---|---|
| 语言 | Go | 1.22+ | 后端主语言 |
| HTTP 框架 | gin-gonic/gin | v1.9+ | REST API 路由、中间件 |
| PostgreSQL 驱动 | jackc/pgx/v5 | v5.x | 原生 SQL 查询，连接池管理 |
| Redis 客户端 | redis/go-redis/v9 | v9.x | 缓存读写、Pub/Sub |
| WebSocket | gorilla/websocket | v1.5+ | 实时推送 pool_snapshot / pool_diff / strategy_alert |
| 定时调度 | robfig/cron/v3 | v3.x | 每日爬虫、快照、概念同步 |
| 配置管理 | spf13/viper | v1.18+ | YAML + 环境变量，支持热更新 |
| 结构化日志 | go.uber.org/zap | v1.27+ | JSON 格式日志，全链路 trace |
| 并发控制 | golang.org/x/sync/errgroup | latest | 分片并行 + 统一错误收集 |
| 前端（后期） | React | 18+ | SPA 前端 |
| 数据库 | PostgreSQL | 15+ | 主存储，JSONB + 数组 + 表分区 |
| 缓存 | Redis | 7+ | 实时行情、池子、映射缓存 |
| 行情数据源 | Tushare Pro API | — | 实时/历史行情，概念板块 |
| 热点数据源 | 韭研公社 API | — | 热点 + 股票-热点映射 |
| LLM 兜底 | OpenAI-compatible | gpt-4o-mini | 第4层分类兜底 |

---

### 1.4 构建与运行命令

```makefile
# ---- 构建 ----
make build          # go build -o bin/stock-monitor ./cmd/server

# ---- 运行 ----
make run            # go run ./cmd/server
make run-dev        # GIN_MODE=debug go run ./cmd/server

# ---- 测试 ----
make test           # go test ./... -v -count=1 -race
make test-cover     # go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# ---- 代码质量 ----
make lint           # golangci-lint run ./...
make fmt            # gofmt -w .

# ---- 数据库迁移 ----
make migrate-up     # 按序号顺序执行 migrations/*.sql
make migrate-down   # 回滚最近一次迁移
make migrate-status # 查看当前迁移状态

# ---- 初始化数据 ----
make init-boards    # 初始化板块涨停规则 (boards 表)
make init-stocks    # 从 Tushare stock_basic 初始化 stocks 表
make init-concepts  # 同步 Tushare 概念板块数据
make crawl-history  # 韭研公社全量历史爬取 (2023-01-01 至今)
```

---

### 1.5 核心开发规则（RULES）

> 以下规则是**硬性约束**，所有代码生成和修改必须遵守。违反任何一条视为 BUG。

**RULE-01: 时区强制 Asia/Shanghai，时间列一律 TIMESTAMPTZ**
- 数据库连接串必须包含 `timezone=Asia/Shanghai`
- 所有 `created_at`、`updated_at`、`trigger_time`、`snapshot_time` 等时间列类型为 `TIMESTAMPTZ`，禁止使用 `TIMESTAMP` (without time zone)
- Go 代码中获取当前时间：`time.Now().In(loc)`，其中 `loc` 从 `time.LoadLocation("Asia/Shanghai")` 获取
- 禁止使用 `time.Now()` 裸调用（它取决于服务器时区设置，不可靠）

**RULE-02: 股票代码统一 Tushare 格式 `{symbol}.{exchange}`**
- 系统内部、数据库、Redis key 全部使用 Tushare 格式：`000001.SZ`、`688256.SH`、`301001.SZ`、`430047.BJ`
- 韭研公社返回 `sz002445` 格式，入库前必须调用 `converter.ToTushareCode()` 转换
- 前端展示时可按需转换，但 API 请求参数和响应中一律用 Tushare 格式
- ts_code 字段类型为 `VARCHAR(16)`，字段名统一为 `ts_code`

**RULE-03: 外部 API 调用必须走 retry + exponential backoff**
- 所有对 Tushare、韭研公社、LLM 的 HTTP 调用必须包裹 `pkg/retry.WithBackoff()`
- 退避参数：初始 1s，倍数 2.0，上限 8s，最大重试 4 次
- 单个股票/单个日期的失败不阻塞整体流程（个股级隔离）
- 失败后写入日志并在 Redis 中累计 `stock:error_count:{date}:{ts_code}`
- 单轮 10s 内失败率超过 30% 触发整体异常告警

**RULE-04: 数据库写入一律 ON CONFLICT upsert，禁止先查后插**
- 所有可能重复写入的表，INSERT 语句必须带 `ON CONFLICT (...) DO UPDATE SET ...` 或 `DO NOTHING`
- 幂等键参照下表：

| 表 | 幂等键 | 冲突策略 |
|---|---|---|
| stock_topic_mappings | (ts_code, topic_id) | `DO UPDATE SET hit_count = hit_count + 1, last_seen_date = GREATEST(...)` |
| daily_stock_pool | (date, ts_code, pool_type) | `DO UPDATE` 更新最新行情状态 |
| daily_focus_topics | (date, topic_id) | `DO NOTHING` |
| stock_concept_tags | (ts_code, concept_name) | `DO NOTHING` |
| concept_topic_mappings | (concept_name, topic_id) | `DO NOTHING` |
| jiuyan_crawl_logs | (date) | `DO UPDATE` 更新状态和重试次数 |

- 禁止 `SELECT ... IF NOT EXISTS ... INSERT` 模式（有竞态条件）
- 支持崩溃后安全重启重写

**RULE-05: 分类走四层级联：Redis → PG韭研 → PG概念 → LLM**
- 分类引擎严格按层级顺序查找，命中即停止，不继续查后续层
- 第1层 Redis 缓存：延迟 < 1ms，启动时预热
- 第2层 PG 韭研映射表（stock_topic_mappings）：延迟约 5ms，命中后回填 Redis
- 第3层 PG 概念标签（stock_concept_tags → concept_topic_mappings）：延迟约 5ms
- 第4层 LLM 异步调用：延迟约 3s，**不阻塞**实时监控流程，结果异步回写，下轮 10s 命中
- 每次分类必须写入 `classify_evidence` 表，记录 classify_layer / strategy / candidate_scores / confidence

**RULE-06: source=manual 最高优先级，不可被任何自动流程覆盖**
- `stock_topic_mappings` 表中 `source='manual'` 的记录，分类引擎直接采信，跳过归因算法
- 自动流程（韭研爬虫、LLM、概念同步）禁止更新 `source='manual'` 的行
- 手动标注的删除只能通过 `DELETE /api/v1/stock/:ts_code/topics/:topic_id` 接口
- upsert 时必须检查：`WHERE source != 'manual'`，避免覆盖手动标注

**RULE-07: 并发 16 分片，hash(ts_code) % 16**
- MonitorService 将全市场约 5000 只股票按 `hash(ts_code) % 16` 分成 16 个 shard
- 每个 shard 由独立 goroutine 处理，shard 内部串行，天然无锁
- 使用 `errgroup.Group` 并发执行 16 个 shard，等待全部完成后统一推送
- 同一只股票永远落在同一个 shard，避免并发写入冲突
- 分片数量从配置文件 `monitor.shard_count` 读取，默认 16

**RULE-08: WebSocket 只推三种消息类型**
- `pool_snapshot`：全量快照（首次连接 + 变化量 >= 30%）
- `pool_diff`：增量变更（变化量 < 30%）
- `strategy_alert`：策略告警（即时推送，不等 10s 周期）
- 禁止发明第四种消息类型，前端只解析这三种 `type` 字段
- 心跳：服务端每 50s 发 Ping，客户端 60s 内回 Pong，否则断开
- 发送缓冲区 256 条，满则主动断开连接（背压保护）

**RULE-09: LLM confidence < 0.7 拒绝，退回"未分类"**
- LLM 分类结果必须包含 `confidence` 字段（0.0 ~ 1.0）
- `confidence < 0.7` 的结果不写入映射表，该股票本轮标记为"未分类"
- LLM 调用是异步的，不阻塞实时监控主循环
- 最大并发 5 个 LLM 请求（配置项 `llm.max_concurrent`）
- 单次调用超时 30s（配置项 `llm.timeout_sec`）

**RULE-10: ST 股票直接过滤，不参与任何计算**
- `stocks` 表中 `is_st = TRUE` 的股票，在 MonitorService 行情处理入口即跳过
- 不进入涨停判定、不进入 5% 判定、不进入分类、不进入告警
- ST 状态通过 Tushare stock_basic 接口同步，每日盘前更新

---

### 1.6 编码约定

#### 错误处理

```go
// [MUST] 所有错误必须处理，禁止 _ 丢弃
// [MUST] repo 层返回原始 error，service 层包装上下文后返回
// [MUST] handler 层统一用 helper 函数响应错误

// repo 层 — 返回原始错误
func (r *StockRepo) GetByTsCode(ctx context.Context, tsCode string) (*model.Stock, error) {
    var s model.Stock
    err := r.db.QueryRow(ctx, sql, tsCode).Scan(&s.ID, &s.TsCode, &s.Name /* ... */)
    if err != nil {
        return nil, err // 原始错误，不包装
    }
    return &s, nil
}

// service 层 — 包装上下文
func (s *StockService) GetStock(ctx context.Context, tsCode string) (*model.Stock, error) {
    stock, err := s.repo.GetByTsCode(ctx, tsCode)
    if err != nil {
        return nil, fmt.Errorf("get stock %s: %w", tsCode, err) // 用 %w 包装
    }
    return stock, nil
}

// handler 层 — 统一响应
func (h *StockHandler) GetStock(c *gin.Context) {
    tsCode := c.Param("ts_code")
    stock, err := h.svc.GetStock(c.Request.Context(), tsCode)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            respondError(c, http.StatusNotFound, "stock not found")
            return
        }
        respondError(c, http.StatusInternalServerError, "internal error")
        return
    }
    respondOK(c, stock)
}
```

#### 日志规范

```go
// [MUST] 使用 zap.Logger，禁止 fmt.Println / log.Println
// [MUST] 日志必须包含结构化字段，禁止字符串拼接
// [MUST] 从 context 中提取 logger（支持链路追踪）

// 正确
logger.Info("stock classified",
    zap.String("ts_code", tsCode),
    zap.String("layer", "L1_REDIS"),
    zap.Int64("topic_id", topicID),
    zap.Float64("confidence", score),
)

// 错误 — 禁止
log.Printf("stock %s classified to topic %d", tsCode, topicID)
```

#### Context 传递

```go
// [MUST] 所有函数第一个参数为 ctx context.Context
// [MUST] 数据库查询、Redis 操作、HTTP 请求全部传入 ctx
// [MUST] 长时间运行的 goroutine 监听 ctx.Done() 实现优雅退出

func (s *MonitorService) ProcessShard(ctx context.Context, shardID int, stocks []model.Stock) error {
    for _, stock := range stocks {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        if err := s.processStock(ctx, stock); err != nil {
            s.logger.Warn("process stock failed",
                zap.String("ts_code", stock.TsCode),
                zap.Int("shard", shardID),
                zap.Error(err),
            )
            continue // 个股失败不阻塞分片
        }
    }
    return nil
}
```

#### 命名约定

| 类别 | 约定 | 示例 |
|---|---|---|
| 包名 | 全小写单词，不用下划线 | `repo`, `service`, `handler`, `cache` |
| 文件名 | 小写 + 下划线 | `stock_repo.go`, `limit_calc.go` |
| 结构体 | 大驼峰 | `MonitorService`, `ClassifyEvidence` |
| 接口 | 大驼峰 + 动词/er 后缀 | `StockRepository`, `Classifier` |
| 方法 | 大驼峰 | `GetByTsCode`, `ClassifyStock` |
| 常量 | 全大写 + 下划线 | `CLASSIFY_LAYER_L1_REDIS`, `SOURCE_MANUAL` |
| 数据库列 | 小写 + 下划线 | `ts_code`, `topic_id`, `hit_count` |
| Redis key | 冒号分隔 | `rt:quote:{ts_code}`, `pool:limit_up:{date}` |

#### 测试约定

```go
// [MUST] 每个 service 必须有对应的 _test.go
// [MUST] repo 层测试使用 testcontainers 启动真实 PG/Redis（不 mock 数据库）
// [MUST] service 层测试 mock repo 接口
// [MUST] 测试函数命名：Test{Method}_{Scenario}_{Expected}

func TestClassifyService_ClassifyStock_HitsRedisCache(t *testing.T)     { /* ... */ }
func TestClassifyService_ClassifyStock_FallbackToLLM(t *testing.T)      { /* ... */ }
func TestAttributionEngine_Score_SingleCandidate(t *testing.T)          { /* ... */ }
func TestAttributionEngine_Score_DualAttribution(t *testing.T)          { /* ... */ }
func TestLimitCalc_Calculate_MainBoard10Pct(t *testing.T)               { /* ... */ }
func TestCodeConverter_ToTushareCode_SzPrefix(t *testing.T)             { /* ... */ }
```

---

### 1.7 系统架构

#### 三层架构 + 数据流（ASCII）

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                         EXTERNAL DATA SOURCES                               │
│                                                                             │
│   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐                   │
│   │ Tushare Pro  │   │  韭研公社     │   │   LLM API    │                   │
│   │  (行情+概念)  │   │  (热点+映射)  │   │  (分类兜底)   │                   │
│   └──────┬───────┘   └──────┬───────┘   └──────┬───────┘                   │
│          │                  │                  │                             │
└──────────┼──────────────────┼──────────────────┼────────────────────────────┘
           │ HTTP+Retry       │ HTTP+Retry       │ HTTP+Retry
           ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                       SERVICE LAYER (Go / Gin)                              │
│                                                                             │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐  ┌───────────────┐   │
│  │  Scheduler   │  │MonitorService│  │ClassifyService│  │ AlertService  │   │
│  │ (10s/daily/  │─▶│  (行情拉取    │─▶│ (四层分类     │─▶│ (策略判定     │   │
│  │  weekly)     │  │   涨停判定)   │  │  归因算法)    │  │  即时推送)    │   │
│  └─────────────┘  └──────────────┘  └──────────────┘  └───────┬───────┘   │
│                                                                │           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │           │
│  │CrawlerService│  │ConceptSync   │  │SnapshotService│         │           │
│  │ (韭研爬虫)    │  │ (概念同步)    │  │ (收盘快照)    │         │           │
│  └──────────────┘  └──────────────┘  └──────────────┘         │           │
│                                                                │           │
│  ┌──────────────┐  ┌──────────────┐                            │           │
│  │  REST API    │  │ WebSocket    │◀───────────────────────────┘           │
│  │  (Gin)       │  │  Hub/Client  │──────────┐                             │
│  └──────┬───────┘  └──────┬───────┘          │                             │
│         │                 │                  │                             │
└─────────┼─────────────────┼──────────────────┼─────────────────────────────┘
          │                 │                  │
          ▼                 ▼                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        STORAGE LAYER                                        │
│                                                                             │
│  ┌─────────────────────────────┐  ┌─────────────────────────────────────┐  │
│  │       PostgreSQL 15+        │  │            Redis 7+                 │  │
│  │                             │  │                                     │  │
│  │  boards (板块规则)           │  │  rt:quote:{ts_code}   实时行情     │  │
│  │  stocks (股票基础)           │  │  pool:limit_up:{date} 涨停池      │  │
│  │  hot_topics (热点库)         │  │  pool:above5:{date}   5%池        │  │
│  │  topic_synonyms (同义词)     │  │  cache:stock_topics   映射缓存    │  │
│  │  stock_topic_mappings       │  │  cache:bind_strength  关联强度     │  │
│  │  stock_concept_tags         │  │  cache:stock_concepts 概念标签     │  │
│  │  concept_topic_mappings     │  │  cache:concept_to_topics 概念→热点 │  │
│  │  daily_stock_pool (按月分区) │  │  focus:topics:{date}  关注热点    │  │
│  │  strategy_alerts            │  │  first_limit:{date}   首次涨停    │  │
│  │  classify_evidence          │  │  alerted:{date}       告警去重    │  │
│  │  daily_focus_topics         │  │  topic:activity:*      活跃度     │  │
│  │  jiuyan_crawl_logs          │  │  snapshot:current      当前快照   │  │
│  └─────────────────────────────┘  └─────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
          ▲                 ▲
          │                 │
          ▼                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        CLIENT LAYER                                         │
│                                                                             │
│   ┌──────────────────────────────────────────────────────────────────┐      │
│   │                    React SPA (后期)                               │      │
│   │    REST API ◀──────▶ 池子/热点/告警/管理                          │      │
│   │    WebSocket ◀─────── pool_snapshot / pool_diff / strategy_alert │      │
│   └──────────────────────────────────────────────────────────────────┘      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### 数据流概览

```text
每 10 秒循环:

  Tushare ──HTTP──▶ MonitorService ──hash分片──▶ 16 goroutines
                         │
                         ▼
                  涨停/5% 判定 ──▶ Redis (pool:limit_up / pool:above5)
                         │
                         ▼
                  ClassifyService
                    │ L1: Redis cache ──命中──▶ 归因算法
                    │ L2: PG jiuyan   ──命中──▶ 归因算法, 回填Redis
                    │ L3: PG concept  ──命中──▶ 归因算法(概念权重)
                    │ L4: LLM async   ──异步──▶ 回写PG+Redis, 下轮命中
                    ▼
                  归因算法 (4 维加权评分)
                    ▼
                  AlertService ──命中策略──▶ WebSocket (strategy_alert)
                    ▼
                  WebSocket Hub ──广播──▶ pool_snapshot / pool_diff

每日定时任务:

  15:05  SnapshotService ── Redis → PG daily_stock_pool
  15:30  CrawlerService  ── 韭研公社 → PG hot_topics + stock_topic_mappings
  每周日  ConceptSync     ── Tushare → PG stock_concept_tags
```

---

#### 九大核心模块 — 职责与 Go Interface

| # | 模块 | 职责 | 运行方式 |
|---|---|---|---|
| 1 | MonitorService | 每 10s 拉取全市场行情，分片并发处理，计算涨停/5%，触发分类和告警 | 10s Ticker，16 分片 goroutine |
| 2 | ClassifyService | 四层级联分类 + 四维归因算法，每次记录证据 | 被 MonitorService 同步调用 |
| 3 | AttributionEngine | 四维加权归因评分，解决多热点归因 | 被 ClassifyService 调用 |
| 4 | AlertService | 策略 2 判定 + 告警记录 + 即时推送 | 被 MonitorService 触发 |
| 5 | CrawlerService | 韭研公社全量/增量爬取，维护热点库 + 同义词 + 映射 | 初始化全量 + 每日 15:30 增量 |
| 6 | ConceptSyncService | Tushare 概念板块同步，维护概念标签 + 概念→热点映射 | 初始化全量 + 每周日凌晨增量 |
| 7 | SnapshotService | 收盘后 Redis 实时数据持久化到 PG | 每日 15:05 定时 |
| 8 | TopicService | 热点 CRUD + 同义词管理 + 关注热点管理 | HTTP 请求驱动 |
| 9 | WSHub | WebSocket 连接管理 + 消息广播 + 背压处理 | 常驻 goroutine |

以下为每个模块的完整 Go interface 签名：

```go
// ===========================================================================
// 1. MonitorService — 实时监控引擎
// ===========================================================================
type MonitorService interface {
    // Start 启动 10s 轮询监控，阻塞直到 ctx 取消
    Start(ctx context.Context) error

    // Stop 优雅停止，等待当前轮次处理完成
    Stop() error

    // ProcessTick 处理单个 10s 轮次：拉行情→分片→判定→分类→告警→推送
    ProcessTick(ctx context.Context) error

    // ProcessShard 处理单个分片内的所有股票
    ProcessShard(ctx context.Context, shardID int, stocks []model.Stock) error
}

// ===========================================================================
// 2. ClassifyService — 四层分类引擎
// ===========================================================================
type ClassifyService interface {
    // ClassifyStock 对单只股票执行四层级联分类，返回归因热点列表
    // 自动记录 classify_evidence
    ClassifyStock(ctx context.Context, tsCode string, quote model.StockQuote) ([]model.TopicAttribution, error)

    // BatchClassify 批量分类（分片内串行调用）
    BatchClassify(ctx context.Context, stocks []model.StockQuote) (map[string][]model.TopicAttribution, error)

    // PrewarmCache 启动时预热 Redis 缓存（从 PG 加载映射和概念）
    PrewarmCache(ctx context.Context) error

    // RefreshCache 定时刷新 Redis 缓存
    RefreshCache(ctx context.Context) error
}

// ===========================================================================
// 3. AttributionEngine — 四维加权归因
// ===========================================================================
type AttributionEngine interface {
    // Score 对候选热点列表计算四维加权得分，返回排序后的结果
    // weights: {activity: 0.25, bindStrength: 0.40, timeProximity: 0.20, recency: 0.15}
    // conceptMode 下权重调整为 {activity: 0.45, bindStrength: 0.00, timeProximity: 0.40, recency: 0.15}
    Score(ctx context.Context, tsCode string, candidates []model.CandidateTopic, mode AttributionMode) ([]model.ScoredTopic, error)

    // Attribute 在 Score 基础上执行决策：单归因 / 双归因 / 未分类
    // Top1 - Top2 < 0.1 → 双归因
    Attribute(ctx context.Context, tsCode string, candidates []model.CandidateTopic, mode AttributionMode) ([]model.TopicAttribution, error)
}

// AttributionMode 归因模式
type AttributionMode int

const (
    ModeJiuyan  AttributionMode = iota // 有韭研标签，正常权重
    ModeConcept                        // 仅概念标签，兜底权重
)

// ===========================================================================
// 4. AlertService — 策略告警
// ===========================================================================
type AlertService interface {
    // CheckAndAlert 检查股票是否满足策略 2 告警条件，满足则保存并推送
    // 策略2：昨日关注热点下，昨日涨幅>5%未涨停 + 今日首次涨停 → 告警
    CheckAndAlert(ctx context.Context, tsCode string, quote model.StockQuote, topics []model.TopicAttribution) error

    // GetTodayAlerts 查询今日所有告警记录
    GetTodayAlerts(ctx context.Context, date string) ([]model.StrategyAlert, error)

    // GetHistoryAlerts 分页查询历史告警
    GetHistoryAlerts(ctx context.Context, startDate, endDate string, topicID *int64, page, pageSize int) ([]model.StrategyAlert, int64, error)
}

// ===========================================================================
// 5. CrawlerService — 韭研公社爬虫
// ===========================================================================
type CrawlerService interface {
    // CrawlDate 爬取指定日期的韭研数据并入库
    // 自动同义词归一化、代码格式转换、upsert 写入
    CrawlDate(ctx context.Context, date string) error

    // CrawlHistory 全量历史爬取，从 startDate 到今天
    // 跳过 jiuyan_crawl_logs 中 status=1 的已成功日期
    CrawlHistory(ctx context.Context, startDate string) error

    // CrawlToday 增量爬取当日数据（每日 15:30 调度）
    CrawlToday(ctx context.Context) error

    // GetCrawlLogs 查询爬取日志
    GetCrawlLogs(ctx context.Context, page, pageSize int) ([]model.JiuyanCrawlLog, int64, error)
}

// ===========================================================================
// 6. ConceptSyncService — Tushare 概念同步
// ===========================================================================
type ConceptSyncService interface {
    // SyncAll 全量同步：concept() 获取列表 → concept_detail() 逐个获取股票 → 写入 stock_concept_tags
    SyncAll(ctx context.Context) error

    // SyncIncremental 增量同步（每周）
    SyncIncremental(ctx context.Context) error

    // BuildMappings 自动建立 concept_topic_mappings（精确匹配 + LLM 批量匹配）
    BuildMappings(ctx context.Context) error

    // GetUnmappedConcepts 查询未建立映射的概念列表
    GetUnmappedConcepts(ctx context.Context) ([]model.StockConceptTag, error)
}

// ===========================================================================
// 7. SnapshotService — 每日快照
// ===========================================================================
type SnapshotService interface {
    // TakeSnapshot 将 Redis 中的实时涨停池和 5% 池数据持久化到 PG daily_stock_pool
    // 使用 ON CONFLICT upsert 保证幂等
    TakeSnapshot(ctx context.Context, date string) error

    // GetSnapshot 查询某日的快照数据（按热点分组）
    GetSnapshot(ctx context.Context, date string, poolType int) ([]model.PoolGroup, error)
}

// ===========================================================================
// 8. TopicService — 热点管理
// ===========================================================================
type TopicService interface {
    // ListTopics 分页查询热点列表，支持关键词搜索
    ListTopics(ctx context.Context, keyword string, page, pageSize int) ([]model.HotTopic, int64, error)

    // CreateTopic 新增热点（source=manual）
    CreateTopic(ctx context.Context, name string) (*model.HotTopic, error)

    // UpdateTopic 编辑热点名称、状态、优先级
    UpdateTopic(ctx context.Context, id int64, name string, isActive bool, priority int) error

    // MergeTopic 将 sourceID 热点合并到 targetID，迁移映射 + 同义词
    MergeTopic(ctx context.Context, sourceID, targetID int64) error

    // ListSynonyms 查询热点同义词
    ListSynonyms(ctx context.Context, topicID int64) ([]model.TopicSynonym, error)

    // AddSynonym 新增同义词
    AddSynonym(ctx context.Context, topicID int64, synonym string) error

    // RemoveSynonym 删除同义词
    RemoveSynonym(ctx context.Context, synonymID int64) error

    // SetFocusTopics 设置某日关注热点
    SetFocusTopics(ctx context.Context, date string, topicIDs []int64) error

    // GetFocusTopics 查询某日关注热点
    GetFocusTopics(ctx context.Context, date string) ([]model.HotTopic, error)
}

// ===========================================================================
// 9. WSHub — WebSocket 连接管理
// ===========================================================================
type WSHub interface {
    // Run 启动 Hub 事件循环，处理注册/注销/广播
    Run(ctx context.Context)

    // Register 注册新的 WebSocket 连接
    Register(client *WSClient)

    // Unregister 注销断开的连接
    Unregister(client *WSClient)

    // BroadcastSnapshot 广播全量快照
    BroadcastSnapshot(snapshot *model.PoolSnapshot)

    // BroadcastDiff 广播增量变更
    BroadcastDiff(diff *model.PoolDiff)

    // BroadcastAlert 广播策略告警（即时推送）
    BroadcastAlert(alert *model.StrategyAlert)

    // ClientCount 返回当前活跃连接数
    ClientCount() int
}
```

#### 关键 Model 结构体参考

```go
// model/topic.go
type TopicAttribution struct {
    TopicID   int64   `json:"topic_id"`
    TopicName string  `json:"topic_name"`
    Score     float64 `json:"score"`
    Source    string  `json:"source"` // jiuyan / tushare / llm / manual
}

type CandidateTopic struct {
    TopicID       int64   `json:"topic_id"`
    TopicName     string  `json:"topic_name"`
    HitCount      int     `json:"hit_count"`       // 韭研标注频次
    LastSeenDate  string  `json:"last_seen_date"`
    LimitCount    int     `json:"limit_count"`      // 今日该热点涨停数
    Source        string  `json:"source"`
}

type ScoredTopic struct {
    TopicID        int64   `json:"topic_id"`
    TopicName      string  `json:"topic_name"`
    ActivityScore  float64 `json:"s_activity"`       // S1
    BindStrength   float64 `json:"s_bind_strength"`  // S2
    TimeProximity  float64 `json:"s_time_proximity"` // S3
    RecencyScore   float64 `json:"s_recency"`        // S4
    TotalScore     float64 `json:"total_score"`
}

// model/pool.go
type PoolSnapshot struct {
    Date         string      `json:"date"`
    SnapshotTime string      `json:"snapshot_time"`
    LimitUp      PoolSection `json:"limit_up"`
    Above5Pct    PoolSection `json:"above_5pct"`
}

type PoolSection struct {
    TotalCount   int          `json:"total_count"`
    Groups       []PoolGroup  `json:"groups"`
    Unclassified []PoolStock  `json:"unclassified"`
}

type PoolGroup struct {
    TopicID   int64       `json:"topic_id"`
    TopicName string      `json:"topic_name"`
    Stocks    []PoolStock `json:"stocks"`
}

type PoolStock struct {
    TsCode           string  `json:"ts_code"`
    Name             string  `json:"name"`
    ChangePct        float64 `json:"change_pct"`
    CurrentPrice     float64 `json:"current_price"`
    FirstLimitTime   string  `json:"first_limit_time,omitempty"`
    AttributionScore float64 `json:"attribution_score,omitempty"`
}

type PoolDiff struct {
    AddedLimitUp   []PoolStock `json:"added_limit_up"`
    RemovedLimitUp []PoolStock `json:"removed_limit_up"`
    AddedAbove5    []PoolStock `json:"added_above5"`
    RemovedAbove5  []PoolStock `json:"removed_above5"`
}

// model/alert.go
type StrategyAlert struct {
    ID           int64           `json:"id"`
    Date         string          `json:"date"`
    TsCode       string          `json:"ts_code"`
    StockName    string          `json:"stock_name"`
    TopicID      int64           `json:"topic_id"`
    TopicName    string          `json:"topic_name"`
    AlertType    int             `json:"alert_type"`
    TriggerPrice float64         `json:"trigger_price"`
    TriggerTime  string          `json:"trigger_time"`
    PrevDayPct   float64         `json:"prev_day_pct"`
    ExtraInfo    json.RawMessage `json:"extra_info"`
    Notified     bool            `json:"notified"`
}

// model/evidence.go
type ClassifyEvidence struct {
    ID               int64           `json:"id"`
    Date             string          `json:"date"`
    TsCode           string          `json:"ts_code"`
    TopicID          int64           `json:"topic_id"`
    ClassifyLayer    string          `json:"classify_layer"`    // L1_REDIS / L2_PG_JIUYAN / L3_PG_CONCEPT / L4_LLM
    Strategy         string          `json:"strategy"`          // JIUYAN_ATTR / CONCEPT_ATTR / LLM_V1 / MANUAL
    CandidateScores  json.RawMessage `json:"candidate_scores"`
    EvidenceText     string          `json:"evidence_text"`
    Confidence       float64         `json:"confidence"`
    CorrectedTopicID *int64          `json:"corrected_topic_id,omitempty"`
}
```

---

### 1.8 配置文件参考 (config/config.yaml)

```yaml
server:
  port: 8080
  mode: release                    # release / debug
  api_key: ${API_KEY}              # X-API-Key 鉴权

database:
  host: localhost
  port: 5432
  user: stock_monitor
  password: ${DB_PASSWORD}
  dbname: stock_monitor
  max_open_conns: 50
  max_idle_conns: 10
  timezone: Asia/Shanghai          # RULE-01: 强制时区

redis:
  addr: localhost:6379
  password: ${REDIS_PASSWORD}
  db: 0

tushare:
  token: ${TUSHARE_TOKEN}
  rate_limit_per_min: 500          # Tushare 限流

jiuyan:
  base_url: https://app.jiuyangongshe.com/jystock-app/api/v1
  token: ${JIUYAN_TOKEN}
  crawl_start_date: "2023-01-01"   # 全量爬取起始日

llm:
  enabled: true
  api_url: ${LLM_API_URL}
  api_key: ${LLM_API_KEY}
  model: gpt-4o-mini
  max_concurrent: 5                # 最大并发 LLM 请求数
  timeout_sec: 30                  # 单次超时
  min_confidence: 0.7              # RULE-09: 低于此值拒绝

monitor:
  interval_sec: 10                 # 轮询间隔
  trading_start: "09:25"           # 交易开始时间
  trading_end: "15:01"             # 交易结束时间
  shard_count: 16                  # RULE-07: 分片数
  error_threshold_pct: 30          # 单轮失败率阈值

retry:
  max_retries: 4                   # RULE-03: 最大重试
  initial_delay_ms: 1000           # 初始退避 1s
  multiplier: 2.0                  # 退避倍数
  max_delay_ms: 8000               # 最大退避 8s

scheduler:
  daily_crawl_time: "15:30"        # 韭研日增量
  daily_snapshot_time: "15:05"     # 每日快照
  concept_sync_cron: "0 2 * * 0"   # 概念同步: 每周日凌晨 2 点

websocket:
  write_wait_sec: 10               # 写超时
  pong_wait_sec: 60                # Pong 等待超时
  ping_period_sec: 50              # Ping 发送间隔
  max_message_size: 4096           # 最大消息体
  send_buf_size: 256               # RULE-08: 发送缓冲区
```

---

## 二、数据库 Schema（PostgreSQL + Redis）

本节定义系统全部 12 张 PostgreSQL 表的完整 DDL、Go struct 映射、Upsert SQL，
以及 Redis 15 个 Key 的结构化声明。所有代码可直接复制到项目中使用。

---

### 2.1 总览

#### 2.1.1 数据库选型

| 组件 | 选型 | 关键理由 |
|------|------|----------|
| 主存储 | PostgreSQL 15+ | JSONB、数组类型(`BIGINT[]`)、表分区、GIN 索引 |
| 实时缓存 | Redis 7+ | 实时行情快照(10s)、涨停池/5%池(Set/SortedSet)、映射缓存、Pub/Sub |
| Go PG 驱动 | `github.com/jackc/pgx/v5` | 原生支持 `pgtype`、连接池 |
| Go Redis 驱动 | `github.com/redis/go-redis/v9` | 上下文支持、Pipeline |

#### 2.1.2 通用约定

- 时区：服务器统一 `Asia/Shanghai`，存储层统一 `TIMESTAMPTZ`。
- 时间戳来源：优先使用行情源时间（交易所时间），非服务器本地时间。
- 幂等写入：所有可能重复写入的表均通过 **唯一索引 + `ON CONFLICT`** 实现幂等。
- 软删除：`is_active = FALSE` 表示逻辑删除，不做物理删除。
- 命名：表名 `snake_case`，Go struct `PascalCase`，JSON 字段 `snake_case`。

#### 2.1.3 表创建顺序（考虑外键依赖）

```text
第 1 批（无外键依赖）:
  1. stock_basic_info          -- 股票基础信息
  2. topics                    -- 热点主题
  3. tushare_concepts          -- Tushare 概念板块

第 2 批（依赖第 1 批）:
  4. topic_synonyms            -- 依赖 topics
  5. topic_concepts            -- 依赖 topics + tushare_concepts
  6. stock_topic_relations     -- 依赖 topics + stock_basic_info
  7. tushare_concept_details   -- 依赖 tushare_concepts + stock_basic_info
  8. focus_topics              -- 依赖 topics

第 3 批（依赖第 1-2 批）:
  9.  daily_stock_pool          -- 分区表，引用 topics(逻辑)
  10. strategy_alerts           -- 依赖 topics
  11. classification_audit_log  -- 依赖 topics
  12. market_snapshots          -- 无外键，但逻辑上在爬虫之后
```

#### 2.1.4 Go 包结构建议

```text
internal/
├── model/                          # 所有 Go struct 定义
│   ├── stock_basic_info.go
│   ├── topic.go                    # topics + topic_synonyms
│   ├── topic_concepts.go
│   ├── stock_topic_relations.go
│   ├── tushare_concept.go          # tushare_concepts + tushare_concept_details
│   ├── focus_topics.go
│   ├── daily_stock_pool.go
│   ├── strategy_alerts.go
│   ├── classification_audit_log.go
│   └── market_snapshots.go
├── repo/                           # 数据访问层 (PostgreSQL CRUD)
│   ├── stock_repo.go
│   ├── topic_repo.go
│   ├── mapping_repo.go
│   ├── concept_repo.go
│   ├── pool_repo.go
│   ├── alert_repo.go
│   ├── evidence_repo.go
│   └── crawl_repo.go
├── cache/                          # Redis 缓存操作
│   ├── quote_cache.go              # rt:quote:*
│   ├── pool_cache.go               # pool:*
│   ├── mapping_cache.go            # cache:stock_topics, cache:bind_strength
│   └── concept_cache.go            # cache:stock_concepts, cache:concept_to_topics
└── migration/                      # SQL 迁移文件
    ├── 001_create_stock_basic_info.sql
    ├── 002_create_topics.sql
    ├── 003_create_topic_synonyms.sql
    ├── 004_create_tushare_concepts.sql
    ├── 005_create_topic_concepts.sql
    ├── 006_create_stock_topic_relations.sql
    ├── 007_create_tushare_concept_details.sql
    ├── 008_create_focus_topics.sql
    ├── 009_create_daily_stock_pool.sql
    ├── 010_create_strategy_alerts.sql
    ├── 011_create_classification_audit_log.sql
    └── 012_create_market_snapshots.sql
```

Go struct 通用 import：

```go
import (
    "time"

    "github.com/jackc/pgx/v5/pgtype"
    "github.com/shopspring/decimal"
)
```

---

### 2.2 表 1：stock_basic_info（股票基础信息）

存储全市场约 5300 只股票的基础信息，数据来源为 Tushare `stock_basic` 接口。
`board_code` 字段用于关联涨跌停规则：创业板(300/301)=GEM，科创板(688)=STAR，北交所(8/4)=BSE，其他=MAIN。
ST 股票 `is_st = TRUE`，系统直接忽略不参与任何计算。

#### DDL

```sql
CREATE TABLE stock_basic_info (
    id              BIGSERIAL       PRIMARY KEY,
    ts_code         VARCHAR(16)     NOT NULL,           -- Tushare 代码，如 000001.SZ
    symbol          VARCHAR(10)     NOT NULL,           -- 纯数字代码，如 000001
    name            VARCHAR(64)     NOT NULL,           -- 股票名称
    exchange        VARCHAR(8)      NOT NULL,           -- 交易所：SSE / SZSE / BSE
    board_code      VARCHAR(16)     NOT NULL DEFAULT 'MAIN', -- 板块：MAIN / GEM / STAR / BSE
    industry        VARCHAR(64),                        -- 所属行业
    is_st           BOOLEAN         NOT NULL DEFAULT FALSE, -- 是否 ST
    list_date       DATE,                               -- 上市日期
    status          SMALLINT        NOT NULL DEFAULT 1, -- 1=正常 0=停牌/退市
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_stock_basic_info_ts_code UNIQUE (ts_code)
);

COMMENT ON TABLE  stock_basic_info                   IS '股票基础信息，来源 Tushare stock_basic';
COMMENT ON COLUMN stock_basic_info.ts_code           IS 'Tushare 股票代码，如 000001.SZ';
COMMENT ON COLUMN stock_basic_info.symbol            IS '纯数字代码，如 000001';
COMMENT ON COLUMN stock_basic_info.exchange          IS '交易所编码：SSE=上交所 SZSE=深交所 BSE=北交所';
COMMENT ON COLUMN stock_basic_info.board_code        IS '板块编码，用于涨跌停规则: MAIN/GEM/STAR/BSE';
COMMENT ON COLUMN stock_basic_info.is_st             IS '是否ST股，ST股不参与涨停监控';
COMMENT ON COLUMN stock_basic_info.status            IS '1=正常上市 0=停牌或退市';

CREATE INDEX idx_stock_basic_info_board_code ON stock_basic_info (board_code);
CREATE INDEX idx_stock_basic_info_exchange   ON stock_basic_info (exchange);
CREATE INDEX idx_stock_basic_info_status     ON stock_basic_info (status) WHERE status = 1;
```

#### Go Struct

```go
// StockBasicInfo 股票基础信息
type StockBasicInfo struct {
    ID        int64      `db:"id"         json:"id"`
    TsCode    string     `db:"ts_code"    json:"ts_code"`
    Symbol    string     `db:"symbol"     json:"symbol"`
    Name      string     `db:"name"       json:"name"`
    Exchange  string     `db:"exchange"   json:"exchange"`
    BoardCode string     `db:"board_code" json:"board_code"`
    Industry  *string    `db:"industry"   json:"industry,omitempty"`
    IsST      bool       `db:"is_st"      json:"is_st"`
    ListDate  *time.Time `db:"list_date"  json:"list_date,omitempty"`
    Status    int16      `db:"status"     json:"status"`
    CreatedAt time.Time  `db:"created_at" json:"created_at"`
    UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}
```

#### Upsert SQL

```sql
INSERT INTO stock_basic_info (ts_code, symbol, name, exchange, board_code, industry, is_st, list_date, status, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
ON CONFLICT (ts_code) DO UPDATE SET
    name       = EXCLUDED.name,
    exchange   = EXCLUDED.exchange,
    board_code = EXCLUDED.board_code,
    industry   = EXCLUDED.industry,
    is_st      = EXCLUDED.is_st,
    list_date  = EXCLUDED.list_date,
    status     = EXCLUDED.status,
    updated_at = NOW();
```

---

### 2.3 表 2：topics（热点主题）

系统核心实体——热点主题。数据来源为韭研公社爬虫和人工手动创建。
每个热点有唯一名称，通过 `topic_synonyms` 表做同义词归一化。
`is_active = FALSE` 表示逻辑删除（软删除），系统不再使用但保留历史数据。

#### DDL

```sql
CREATE TABLE topics (
    id                BIGSERIAL       PRIMARY KEY,
    name              VARCHAR(128)    NOT NULL,           -- 热点名称
    source            VARCHAR(16)     NOT NULL DEFAULT 'jiuyan', -- 来源：jiuyan / manual
    jiuyan_field_id   VARCHAR(64),                        -- 韭研公社 action_field_id
    first_seen_date   DATE,                               -- 首次出现日期
    last_seen_date    DATE,                               -- 最后出现日期
    occurrence_count  INT             NOT NULL DEFAULT 0, -- 历史出现天数
    priority          INT             NOT NULL DEFAULT 0, -- 展示/处理优先级，数值越大越优先
    is_active         BOOLEAN         NOT NULL DEFAULT TRUE, -- 是否启用
    created_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_topics_name UNIQUE (name)
);

COMMENT ON TABLE  topics                        IS '热点主题库，核心实体';
COMMENT ON COLUMN topics.name                   IS '热点名称，全局唯一';
COMMENT ON COLUMN topics.source                 IS '数据来源：jiuyan=韭研公社 manual=人工创建';
COMMENT ON COLUMN topics.jiuyan_field_id        IS '韭研公社的 action_field_id，用于数据溯源';
COMMENT ON COLUMN topics.first_seen_date        IS '该热点首次出现的日期';
COMMENT ON COLUMN topics.last_seen_date         IS '该热点最近一次出现的日期';
COMMENT ON COLUMN topics.occurrence_count       IS '历史出现天数，用于热点活跃度评估';
COMMENT ON COLUMN topics.priority               IS '展示和处理优先级，数值越大越优先';
COMMENT ON COLUMN topics.is_active              IS 'FALSE=软删除，系统不再使用但保留历史数据';

CREATE INDEX idx_topics_source    ON topics (source);
CREATE INDEX idx_topics_active    ON topics (is_active) WHERE is_active = TRUE;
CREATE INDEX idx_topics_priority  ON topics (priority DESC) WHERE is_active = TRUE;
```

#### Go Struct

```go
// Topic 热点主题
type Topic struct {
    ID              int64      `db:"id"               json:"id"`
    Name            string     `db:"name"             json:"name"`
    Source          string     `db:"source"            json:"source"`
    JiuyanFieldID   *string    `db:"jiuyan_field_id"  json:"jiuyan_field_id,omitempty"`
    FirstSeenDate   *time.Time `db:"first_seen_date"  json:"first_seen_date,omitempty"`
    LastSeenDate    *time.Time `db:"last_seen_date"   json:"last_seen_date,omitempty"`
    OccurrenceCount int        `db:"occurrence_count"  json:"occurrence_count"`
    Priority        int        `db:"priority"          json:"priority"`
    IsActive        bool       `db:"is_active"         json:"is_active"`
    CreatedAt       time.Time  `db:"created_at"        json:"created_at"`
    UpdatedAt       time.Time  `db:"updated_at"        json:"updated_at"`
}
```

#### Upsert SQL

```sql
INSERT INTO topics (name, source, jiuyan_field_id, first_seen_date, last_seen_date, occurrence_count, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (name) DO UPDATE SET
    jiuyan_field_id  = COALESCE(EXCLUDED.jiuyan_field_id, topics.jiuyan_field_id),
    first_seen_date  = LEAST(topics.first_seen_date, EXCLUDED.first_seen_date),
    last_seen_date   = GREATEST(topics.last_seen_date, EXCLUDED.last_seen_date),
    occurrence_count = topics.occurrence_count + EXCLUDED.occurrence_count,
    updated_at       = NOW();
```

---

### 2.4 表 3：topic_synonyms（热点同义词）

实现热点名称归一化。爬虫入库和分类引擎查询时先查本表做同义词匹配，
例如"算力概念" -> 归一到"算力"对应的 `topic_id`。

#### DDL

```sql
CREATE TABLE topic_synonyms (
    id          BIGSERIAL       PRIMARY KEY,
    topic_id    BIGINT          NOT NULL,           -- 归并目标热点
    synonym     VARCHAR(128)    NOT NULL,           -- 同义词文本
    source      VARCHAR(16)     NOT NULL DEFAULT 'jiuyan', -- 来源：jiuyan / manual / auto
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_topic_synonyms_synonym UNIQUE (synonym),
    CONSTRAINT fk_topic_synonyms_topic   FOREIGN KEY (topic_id)
        REFERENCES topics (id) ON DELETE CASCADE
);

COMMENT ON TABLE  topic_synonyms                IS '热点同义词，用于名称归一化';
COMMENT ON COLUMN topic_synonyms.topic_id       IS '归并到的目标热点 ID';
COMMENT ON COLUMN topic_synonyms.synonym        IS '同义词文本，全局唯一';
COMMENT ON COLUMN topic_synonyms.source         IS '来源：jiuyan=爬虫识别 manual=手动添加 auto=自动检测';

CREATE INDEX idx_topic_synonyms_topic_id ON topic_synonyms (topic_id);
```

#### Go Struct

```go
// TopicSynonym 热点同义词
type TopicSynonym struct {
    ID        int64     `db:"id"         json:"id"`
    TopicID   int64     `db:"topic_id"   json:"topic_id"`
    Synonym   string    `db:"synonym"    json:"synonym"`
    Source    string    `db:"source"     json:"source"`
    CreatedAt time.Time `db:"created_at" json:"created_at"`
}
```

#### Upsert SQL

```sql
-- 同义词不应重复，冲突时忽略
INSERT INTO topic_synonyms (topic_id, synonym, source)
VALUES ($1, $2, $3)
ON CONFLICT (synonym) DO NOTHING;
```

---

### 2.5 表 4：tushare_concepts（Tushare 概念板块）

存储 Tushare `concept()` 接口返回的概念板块列表（约 400 个）。
每个概念板块可通过 `topic_concepts` 映射到一个或多个系统热点。

#### DDL

```sql
CREATE TABLE tushare_concepts (
    id              BIGSERIAL       PRIMARY KEY,
    concept_code    VARCHAR(16)     NOT NULL,           -- Tushare 概念代码
    concept_name    VARCHAR(128)    NOT NULL,           -- 概念名称，如"人工智能"
    source          VARCHAR(16)     NOT NULL DEFAULT 'tushare',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_tushare_concepts_code UNIQUE (concept_code),
    CONSTRAINT uq_tushare_concepts_name UNIQUE (concept_name)
);

COMMENT ON TABLE  tushare_concepts                   IS 'Tushare 概念板块列表';
COMMENT ON COLUMN tushare_concepts.concept_code      IS 'Tushare 概念代码，如 TS0001';
COMMENT ON COLUMN tushare_concepts.concept_name      IS '概念名称，如"人工智能"、"芯片概念"';
```

#### Go Struct

```go
// TushareConcept Tushare 概念板块
type TushareConcept struct {
    ID          int64     `db:"id"            json:"id"`
    ConceptCode string    `db:"concept_code"  json:"concept_code"`
    ConceptName string    `db:"concept_name"  json:"concept_name"`
    Source      string    `db:"source"        json:"source"`
    CreatedAt   time.Time `db:"created_at"    json:"created_at"`
    UpdatedAt   time.Time `db:"updated_at"    json:"updated_at"`
}
```

#### Upsert SQL

```sql
INSERT INTO tushare_concepts (concept_code, concept_name, source, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (concept_code) DO UPDATE SET
    concept_name = EXCLUDED.concept_name,
    updated_at   = NOW();
```

---

### 2.6 表 5：topic_concepts（热点-概念关联）

建立 Tushare 概念板块到系统热点的映射关系。
通过自动精确匹配 + LLM 批量匹配 + 人工补充三种方式建立。
多对多关系：一个概念可映射多个热点，一个热点可对应多个概念。

#### DDL

```sql
CREATE TABLE topic_concepts (
    id              BIGSERIAL       PRIMARY KEY,
    concept_name    VARCHAR(128)    NOT NULL,           -- Tushare 概念名称
    concept_code    VARCHAR(16)     NOT NULL,           -- Tushare 概念代码
    topic_id        BIGINT          NOT NULL,           -- 对应的系统热点 ID
    match_type      VARCHAR(16)     NOT NULL DEFAULT 'exact', -- 匹配类型
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_topic_concepts_name_topic UNIQUE (concept_name, topic_id),
    CONSTRAINT fk_topic_concepts_topic      FOREIGN KEY (topic_id)
        REFERENCES topics (id) ON DELETE CASCADE
);

COMMENT ON TABLE  topic_concepts                    IS '概念板块 → 热点映射，多对多';
COMMENT ON COLUMN topic_concepts.concept_name       IS 'Tushare 概念名称';
COMMENT ON COLUMN topic_concepts.concept_code       IS 'Tushare 概念代码';
COMMENT ON COLUMN topic_concepts.topic_id           IS '映射到的系统热点 ID';
COMMENT ON COLUMN topic_concepts.match_type         IS '匹配方式：exact=精确匹配 synonym=同义词 llm=LLM匹配 manual=人工标注';

CREATE INDEX idx_topic_concepts_topic_id     ON topic_concepts (topic_id);
CREATE INDEX idx_topic_concepts_concept_code ON topic_concepts (concept_code);
```

#### Go Struct

```go
// TopicConcept 概念-热点映射
type TopicConcept struct {
    ID          int64     `db:"id"            json:"id"`
    ConceptName string    `db:"concept_name"  json:"concept_name"`
    ConceptCode string    `db:"concept_code"  json:"concept_code"`
    TopicID     int64     `db:"topic_id"      json:"topic_id"`
    MatchType   string    `db:"match_type"    json:"match_type"`
    CreatedAt   time.Time `db:"created_at"    json:"created_at"`
}
```

#### Upsert SQL

```sql
INSERT INTO topic_concepts (concept_name, concept_code, topic_id, match_type)
VALUES ($1, $2, $3, $4)
ON CONFLICT (concept_name, topic_id) DO NOTHING;
```

---

### 2.7 表 6：stock_topic_relations（股票-热点关联）

核心映射表，记录每只股票归属的热点。数据来源包括韭研公社爬虫、LLM 分类、人工标注。
`source = 'manual'` 的记录具有最高优先级，分类引擎直接采信跳过归因算法。
唯一约束 `(ts_code, topic_id)` 保证同一股票同一热点只保留一条记录，通过 `hit_count` 累计频次。

#### DDL

```sql
CREATE TABLE stock_topic_relations (
    id              BIGSERIAL       PRIMARY KEY,
    ts_code         VARCHAR(16)     NOT NULL,           -- 股票代码
    topic_id        BIGINT          NOT NULL,           -- 热点 ID
    source          VARCHAR(16)     NOT NULL DEFAULT 'jiuyan', -- 分类来源
    confidence      FLOAT,                              -- 置信度（LLM 分类时有意义）
    hit_count       INT             NOT NULL DEFAULT 1, -- 历史出现次数（韭研标注频次）
    last_seen_date  DATE,                               -- 最近一次出现日期
    first_seen_date DATE,                               -- 首次出现日期
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_stock_topic_relations_ts_topic UNIQUE (ts_code, topic_id),
    CONSTRAINT fk_stock_topic_relations_topic    FOREIGN KEY (topic_id)
        REFERENCES topics (id) ON DELETE CASCADE
);

COMMENT ON TABLE  stock_topic_relations                     IS '股票-热点关联映射，核心表';
COMMENT ON COLUMN stock_topic_relations.ts_code             IS '股票代码，如 000001.SZ';
COMMENT ON COLUMN stock_topic_relations.topic_id            IS '关联的热点 ID';
COMMENT ON COLUMN stock_topic_relations.source              IS '分类来源：jiuyan=韭研爬虫 llm=LLM分类 manual=人工标注(最高优先级)';
COMMENT ON COLUMN stock_topic_relations.confidence          IS '分类置信度 0~1，LLM 分类时有意义';
COMMENT ON COLUMN stock_topic_relations.hit_count           IS '韭研历史标注频次，用于归因算法的频次维度权重';
COMMENT ON COLUMN stock_topic_relations.last_seen_date      IS '最近一次在韭研数据中出现的日期';
COMMENT ON COLUMN stock_topic_relations.first_seen_date     IS '首次在韭研数据中出现的日期';

CREATE INDEX idx_stock_topic_relations_ts_code  ON stock_topic_relations (ts_code);
CREATE INDEX idx_stock_topic_relations_topic_id ON stock_topic_relations (topic_id);
CREATE INDEX idx_stock_topic_relations_source   ON stock_topic_relations (source);
```

#### Go Struct

```go
// StockTopicRelation 股票-热点关联
type StockTopicRelation struct {
    ID            int64      `db:"id"              json:"id"`
    TsCode        string     `db:"ts_code"         json:"ts_code"`
    TopicID       int64      `db:"topic_id"        json:"topic_id"`
    Source        string     `db:"source"           json:"source"`
    Confidence    *float64   `db:"confidence"       json:"confidence,omitempty"`
    HitCount      int        `db:"hit_count"        json:"hit_count"`
    LastSeenDate  *time.Time `db:"last_seen_date"   json:"last_seen_date,omitempty"`
    FirstSeenDate *time.Time `db:"first_seen_date"  json:"first_seen_date,omitempty"`
    CreatedAt     time.Time  `db:"created_at"       json:"created_at"`
    UpdatedAt     time.Time  `db:"updated_at"       json:"updated_at"`
}
```

#### Upsert SQL（韭研爬虫写入）

```sql
INSERT INTO stock_topic_relations (ts_code, topic_id, source, hit_count, first_seen_date, last_seen_date, updated_at)
VALUES ($1, $2, 'jiuyan', 1, $3, $3, NOW())
ON CONFLICT (ts_code, topic_id) DO UPDATE SET
    hit_count      = stock_topic_relations.hit_count + 1,
    last_seen_date = GREATEST(stock_topic_relations.last_seen_date, EXCLUDED.last_seen_date),
    updated_at     = NOW();
```

#### Upsert SQL（手动标注写入 -- 最高优先级）

```sql
INSERT INTO stock_topic_relations (ts_code, topic_id, source, confidence, hit_count, updated_at)
VALUES ($1, $2, 'manual', 1.0, 1, NOW())
ON CONFLICT (ts_code, topic_id) DO UPDATE SET
    source     = 'manual',
    confidence = 1.0,
    updated_at = NOW();
```

---

### 2.8 表 7：tushare_concept_details（概念板块成分股）

存储 Tushare `concept_detail()` 接口返回的每个概念板块下的成分股列表。
每只股票可属于多个概念。用于分类引擎第 3 层（L3_PG_CONCEPT）兜底查询，
覆盖全市场约 5300 只股票的基础概念标签。

#### DDL

```sql
CREATE TABLE tushare_concept_details (
    id              BIGSERIAL       PRIMARY KEY,
    ts_code         VARCHAR(16)     NOT NULL,           -- 股票代码
    concept_name    VARCHAR(128)    NOT NULL,           -- 概念名称
    concept_code    VARCHAR(16)     NOT NULL,           -- 概念代码（Tushare）
    source          VARCHAR(16)     NOT NULL DEFAULT 'tushare',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_tushare_concept_details_ts_concept UNIQUE (ts_code, concept_name)
);

COMMENT ON TABLE  tushare_concept_details                   IS '股票基础概念标签，来源 Tushare concept_detail';
COMMENT ON COLUMN tushare_concept_details.ts_code           IS '股票代码，如 000001.SZ';
COMMENT ON COLUMN tushare_concept_details.concept_name      IS '概念名称，如"人工智能"';
COMMENT ON COLUMN tushare_concept_details.concept_code      IS 'Tushare 概念代码，如 TS0001';

CREATE INDEX idx_tushare_concept_details_ts_code      ON tushare_concept_details (ts_code);
CREATE INDEX idx_tushare_concept_details_concept_code ON tushare_concept_details (concept_code);
```

#### Go Struct

```go
// TushareConceptDetail 概念板块成分股
type TushareConceptDetail struct {
    ID          int64     `db:"id"            json:"id"`
    TsCode      string    `db:"ts_code"       json:"ts_code"`
    ConceptName string    `db:"concept_name"  json:"concept_name"`
    ConceptCode string    `db:"concept_code"  json:"concept_code"`
    Source      string    `db:"source"        json:"source"`
    CreatedAt   time.Time `db:"created_at"    json:"created_at"`
}
```

#### Upsert SQL

```sql
INSERT INTO tushare_concept_details (ts_code, concept_name, concept_code, source)
VALUES ($1, $2, $3, $4)
ON CONFLICT (ts_code, concept_name) DO NOTHING;
```

---

### 2.9 表 8：focus_topics（关注热点）

每日关注的热点列表。用户通过前端/API 设置今日关注的热点，
策略告警引擎仅对关注列表内的热点触发告警。

#### DDL

```sql
CREATE TABLE focus_topics (
    id          BIGSERIAL       PRIMARY KEY,
    date        DATE            NOT NULL,               -- 关注日期
    topic_id    BIGINT          NOT NULL,               -- 热点 ID
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_focus_topics_date_topic UNIQUE (date, topic_id),
    CONSTRAINT fk_focus_topics_topic      FOREIGN KEY (topic_id)
        REFERENCES topics (id) ON DELETE CASCADE
);

COMMENT ON TABLE  focus_topics                IS '每日关注热点，策略引擎仅对关注热点告警';
COMMENT ON COLUMN focus_topics.date           IS '关注日期，格式 YYYY-MM-DD';
COMMENT ON COLUMN focus_topics.topic_id       IS '被关注的热点 ID';

CREATE INDEX idx_focus_topics_date     ON focus_topics (date);
CREATE INDEX idx_focus_topics_topic_id ON focus_topics (topic_id);
```

#### Go Struct

```go
// FocusTopic 每日关注热点
type FocusTopic struct {
    ID        int64     `db:"id"         json:"id"`
    Date      time.Time `db:"date"       json:"date"`
    TopicID   int64     `db:"topic_id"   json:"topic_id"`
    CreatedAt time.Time `db:"created_at" json:"created_at"`
}
```

#### Upsert SQL

```sql
INSERT INTO focus_topics (date, topic_id)
VALUES ($1, $2)
ON CONFLICT (date, topic_id) DO NOTHING;
```

---

### 2.10 表 9：daily_stock_pool（每日股票池快照 -- 按月分区）

每日收盘后持久化的股票池快照（涨停池 + 5%池），是系统的核心历史数据表。
按月 Range 分区，如 `daily_stock_pool_2026_03` 存储 2026 年 3 月数据。
写入使用 `ON CONFLICT` 保证幂等，支持崩溃重启后安全重写。

`topic_ids` 字段使用 PostgreSQL 原生数组类型 `BIGINT[]`，存储当日归因的热点 ID 列表，
可通过 GIN 索引高效查询"包含某热点的所有股票"。

#### DDL（主表 + 分区）

```sql
-- 主表（分区父表）
CREATE TABLE daily_stock_pool (
    id              BIGSERIAL,
    date            DATE            NOT NULL,               -- 交易日
    ts_code         VARCHAR(16)     NOT NULL,               -- 股票代码
    stock_name      VARCHAR(64)     NOT NULL,               -- 股票名称
    pool_type       SMALLINT        NOT NULL,               -- 1=涨停池 2=涨幅>5%池
    change_pct      FLOAT,                                  -- 涨跌幅(%)
    current_price   FLOAT,                                  -- 当前价格
    pre_close       FLOAT,                                  -- 昨收价
    limit_up_price  FLOAT,                                  -- 涨停价
    first_limit_time TIME,                                  -- 首次触及涨停时间（仅涨停池）
    board_code      VARCHAR(16),                            -- 板块编码
    topic_ids       BIGINT[],                               -- 当日归因的热点 ID 数组
    snapshot_time   TIMESTAMPTZ,                            -- 快照时间
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_daily_stock_pool_date_ts_pool UNIQUE (date, ts_code, pool_type),
    PRIMARY KEY (id, date)
) PARTITION BY RANGE (date);

COMMENT ON TABLE  daily_stock_pool                      IS '每日股票池快照，按月分区';
COMMENT ON COLUMN daily_stock_pool.pool_type            IS '1=涨停池 2=涨幅大于5%池';
COMMENT ON COLUMN daily_stock_pool.change_pct           IS '涨跌幅百分比，如 9.98 表示 +9.98%';
COMMENT ON COLUMN daily_stock_pool.first_limit_time     IS '首次触及涨停时间，仅 pool_type=1 有值';
COMMENT ON COLUMN daily_stock_pool.topic_ids            IS '当日归因热点ID数组，支持 GIN 索引查询';

-- 分区示例：2026年3月
CREATE TABLE daily_stock_pool_2026_03
    PARTITION OF daily_stock_pool
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

-- 分区示例：2026年4月
CREATE TABLE daily_stock_pool_2026_04
    PARTITION OF daily_stock_pool
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

-- 索引（在主表上创建，自动继承到所有分区）
CREATE INDEX idx_daily_stock_pool_date       ON daily_stock_pool (date);
CREATE INDEX idx_daily_stock_pool_ts_code    ON daily_stock_pool (ts_code, date);
CREATE INDEX idx_daily_stock_pool_pool_type  ON daily_stock_pool (date, pool_type);
CREATE INDEX idx_daily_stock_pool_topic_ids  ON daily_stock_pool USING GIN (topic_ids);
```

#### 自动创建分区的 Go 代码片段

```go
// EnsurePartition 确保下月分区存在，每月 1 日定时任务调用
// partitionName 格式：daily_stock_pool_YYYY_MM
func (r *PoolRepo) EnsurePartition(ctx context.Context, year int, month int) error {
    start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
    end := start.AddDate(0, 1, 0)
    partName := fmt.Sprintf("daily_stock_pool_%04d_%02d", year, month)

    sql := fmt.Sprintf(`
        CREATE TABLE IF NOT EXISTS %s
        PARTITION OF daily_stock_pool
        FOR VALUES FROM ('%s') TO ('%s')
    `, partName, start.Format("2006-01-02"), end.Format("2006-01-02"))

    _, err := r.db.Exec(ctx, sql)
    return err
}
```

#### Go Struct

```go
// DailyStockPool 每日股票池快照
type DailyStockPool struct {
    ID             int64          `db:"id"               json:"id"`
    Date           time.Time      `db:"date"             json:"date"`
    TsCode         string         `db:"ts_code"          json:"ts_code"`
    StockName      string         `db:"stock_name"       json:"stock_name"`
    PoolType       int16          `db:"pool_type"        json:"pool_type"`
    ChangePct      *float64       `db:"change_pct"       json:"change_pct,omitempty"`
    CurrentPrice   *float64       `db:"current_price"    json:"current_price,omitempty"`
    PreClose       *float64       `db:"pre_close"        json:"pre_close,omitempty"`
    LimitUpPrice   *float64       `db:"limit_up_price"   json:"limit_up_price,omitempty"`
    FirstLimitTime *pgtype.Time   `db:"first_limit_time" json:"first_limit_time,omitempty"`
    BoardCode      *string        `db:"board_code"       json:"board_code,omitempty"`
    TopicIDs       pgtype.FlatArray[int64] `db:"topic_ids" json:"topic_ids,omitempty"`
    SnapshotTime   *time.Time     `db:"snapshot_time"    json:"snapshot_time,omitempty"`
    CreatedAt      time.Time      `db:"created_at"       json:"created_at"`
}
```

#### Upsert SQL（收盘快照持久化，幂等写入）

```sql
INSERT INTO daily_stock_pool (
    date, ts_code, stock_name, pool_type, change_pct, current_price,
    pre_close, limit_up_price, first_limit_time, board_code, topic_ids, snapshot_time
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (date, ts_code, pool_type) DO UPDATE SET
    stock_name      = EXCLUDED.stock_name,
    change_pct      = EXCLUDED.change_pct,
    current_price   = EXCLUDED.current_price,
    pre_close       = EXCLUDED.pre_close,
    limit_up_price  = EXCLUDED.limit_up_price,
    first_limit_time = EXCLUDED.first_limit_time,
    board_code      = EXCLUDED.board_code,
    topic_ids       = EXCLUDED.topic_ids,
    snapshot_time   = EXCLUDED.snapshot_time;
```

---

### 2.11 表 10：strategy_alerts（策略告警）

策略引擎触发的告警记录。当股票首次涨停且满足策略条件时写入。
`extra_info` (JSONB) 保存触发瞬间的精确行情快照（当前价、成交量、成交额、换手率、买卖五档等），
供后续策略回测使用。

#### DDL

```sql
CREATE TABLE strategy_alerts (
    id              BIGSERIAL       PRIMARY KEY,
    date            DATE            NOT NULL,               -- 告警日期
    ts_code         VARCHAR(16)     NOT NULL,               -- 股票代码
    stock_name      VARCHAR(64)     NOT NULL,               -- 股票名称
    topic_id        BIGINT,                                 -- 命中的热点 ID
    topic_name      VARCHAR(128),                           -- 热点名称（冗余，避免 JOIN）
    alert_type      SMALLINT        NOT NULL DEFAULT 1,     -- 告警类型：1=策略2命中
    trigger_price   FLOAT,                                  -- 触发价格
    trigger_time    TIMESTAMPTZ,                            -- 触发时间
    prev_day_pct    FLOAT,                                  -- 前一日涨跌幅
    extra_info      JSONB,                                  -- 扩展信息，含行情快照
    notified        BOOLEAN         NOT NULL DEFAULT FALSE, -- 是否已推送通知
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  strategy_alerts                       IS '策略告警记录';
COMMENT ON COLUMN strategy_alerts.alert_type            IS '告警类型：1=策略2(关注热点+昨日5%+今日涨停)';
COMMENT ON COLUMN strategy_alerts.extra_info            IS 'JSONB: {price, volume, amount, turnover, bid1-5, ask1-5, ...}';
COMMENT ON COLUMN strategy_alerts.notified              IS '是否已通过 WebSocket/飞书 推送';

CREATE INDEX idx_strategy_alerts_date       ON strategy_alerts (date);
CREATE INDEX idx_strategy_alerts_ts_code    ON strategy_alerts (ts_code, date);
CREATE INDEX idx_strategy_alerts_topic_id   ON strategy_alerts (topic_id);
CREATE INDEX idx_strategy_alerts_notified   ON strategy_alerts (notified) WHERE notified = FALSE;
```

#### `extra_info` JSONB 结构示例

```jsonc
{
    "price": 15.32,
    "pre_close": 13.93,
    "change_pct": 9.98,
    "volume": 1234567,
    "amount": 189234567.89,
    "turnover": 5.67,
    "bid": [
        [15.32, 500],
        [15.31, 300],
        [15.30, 800],
        [15.29, 200],
        [15.28, 100]
    ],
    "ask": [
        [15.33, 400],
        [15.34, 600],
        [15.35, 1000],
        [15.36, 200],
        [15.37, 150]
    ],
    "limit_up_price": 15.32,
    "board_code": "MAIN"
}
```

#### Go Struct

```go
// StrategyAlert 策略告警
type StrategyAlert struct {
    ID           int64          `db:"id"            json:"id"`
    Date         time.Time      `db:"date"          json:"date"`
    TsCode       string         `db:"ts_code"       json:"ts_code"`
    StockName    string         `db:"stock_name"    json:"stock_name"`
    TopicID      *int64         `db:"topic_id"      json:"topic_id,omitempty"`
    TopicName    *string        `db:"topic_name"    json:"topic_name,omitempty"`
    AlertType    int16          `db:"alert_type"    json:"alert_type"`
    TriggerPrice *float64       `db:"trigger_price" json:"trigger_price,omitempty"`
    TriggerTime  *time.Time     `db:"trigger_time"  json:"trigger_time,omitempty"`
    PrevDayPct   *float64       `db:"prev_day_pct"  json:"prev_day_pct,omitempty"`
    ExtraInfo    pgtype.JSONB   `db:"extra_info"    json:"extra_info,omitempty"`
    Notified     bool           `db:"notified"      json:"notified"`
    CreatedAt    time.Time      `db:"created_at"    json:"created_at"`
}

// AlertExtraInfo extra_info JSONB 的 Go 映射
type AlertExtraInfo struct {
    Price        float64      `json:"price"`
    PreClose     float64      `json:"pre_close"`
    ChangePct    float64      `json:"change_pct"`
    Volume       int64        `json:"volume"`
    Amount       float64      `json:"amount"`
    Turnover     float64      `json:"turnover"`
    Bid          [][2]float64 `json:"bid"`
    Ask          [][2]float64 `json:"ask"`
    LimitUpPrice float64      `json:"limit_up_price"`
    BoardCode    string       `json:"board_code"`
}
```

#### Insert SQL

```sql
-- 告警记录一般不冲突（每次触发都是新事件），直接 INSERT
-- 去重通过 Redis alerted:{date} Set 在应用层完成
INSERT INTO strategy_alerts (
    date, ts_code, stock_name, topic_id, topic_name,
    alert_type, trigger_price, trigger_time, prev_day_pct, extra_info
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);
```

---

### 2.12 表 11：classification_audit_log（分类审计日志）

每次分类都记录完整的决策证据，是归因算法调试、权重调优和未来训练监督学习模型的核心数据资产。
`corrected_topic_id` 预留人工修正字段，后期标注正确答案后可用于训练数据。

四层分类引擎层级：
- L1_REDIS: Redis 缓存命中（韭研动态标签，延迟 < 1ms）
- L2_PG_JIUYAN: PostgreSQL 映射表命中（韭研动态标签，延迟 ~5ms）
- L3_PG_CONCEPT: PostgreSQL 概念标签命中（Tushare 基础标签，覆盖全市场）
- L4_LLM: LLM 兜底分类（仅前三层均未命中时使用）

#### DDL

```sql
CREATE TABLE classification_audit_log (
    id                  BIGSERIAL       PRIMARY KEY,
    date                DATE            NOT NULL,               -- 交易日
    ts_code             VARCHAR(16)     NOT NULL,               -- 股票代码
    topic_id            BIGINT,                                 -- 最终归因的热点 ID
    classify_layer      VARCHAR(32)     NOT NULL,               -- 命中层级
    strategy            VARCHAR(32)     NOT NULL,               -- 分类策略标识
    candidate_scores    JSONB,                                  -- 候选热点及各维度评分
    evidence_text       TEXT,                                   -- 关键证据原文
    confidence          FLOAT,                                  -- 置信度 0~1
    corrected_topic_id  BIGINT,                                 -- 人工修正后的热点 ID（预留）
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  classification_audit_log                          IS '分类证据审计日志，核心数据资产';
COMMENT ON COLUMN classification_audit_log.classify_layer           IS '命中层级：L1_REDIS / L2_PG_JIUYAN / L3_PG_CONCEPT / L4_LLM';
COMMENT ON COLUMN classification_audit_log.strategy                 IS '分类策略：JIUYAN_ATTR / CONCEPT_ATTR / LLM_V1 / MANUAL';
COMMENT ON COLUMN classification_audit_log.candidate_scores         IS 'JSONB: 候选热点及频次/时效/概念匹配/热度四维评分';
COMMENT ON COLUMN classification_audit_log.confidence               IS '最终置信度 0~1';
COMMENT ON COLUMN classification_audit_log.corrected_topic_id       IS '人工修正后的正确热点ID，用于后期训练数据积累';

CREATE INDEX idx_classification_audit_log_stock_date ON classification_audit_log (ts_code, date);
CREATE INDEX idx_classification_audit_log_layer      ON classification_audit_log (classify_layer);
CREATE INDEX idx_classification_audit_log_date       ON classification_audit_log (date);
```

#### `candidate_scores` JSONB 结构示例

```jsonc
{
    "candidates": [
        {
            "topic_id": 42,
            "topic_name": "算力",
            "scores": {
                "frequency": 0.35,
                "recency": 0.28,
                "concept_match": 0.20,
                "popularity": 0.17
            },
            "total_score": 0.82
        },
        {
            "topic_id": 57,
            "topic_name": "AI应用",
            "scores": {
                "frequency": 0.10,
                "recency": 0.25,
                "concept_match": 0.30,
                "popularity": 0.20
            },
            "total_score": 0.74
        }
    ],
    "dual_attribution": false,
    "score_gap": 0.08
}
```

#### Go Struct

```go
// ClassificationAuditLog 分类审计日志
type ClassificationAuditLog struct {
    ID                int64        `db:"id"                  json:"id"`
    Date              time.Time    `db:"date"                json:"date"`
    TsCode            string       `db:"ts_code"             json:"ts_code"`
    TopicID           *int64       `db:"topic_id"            json:"topic_id,omitempty"`
    ClassifyLayer     string       `db:"classify_layer"      json:"classify_layer"`
    Strategy          string       `db:"strategy"            json:"strategy"`
    CandidateScores   pgtype.JSONB `db:"candidate_scores"    json:"candidate_scores,omitempty"`
    EvidenceText      *string      `db:"evidence_text"       json:"evidence_text,omitempty"`
    Confidence        *float64     `db:"confidence"          json:"confidence,omitempty"`
    CorrectedTopicID  *int64       `db:"corrected_topic_id"  json:"corrected_topic_id,omitempty"`
    CreatedAt         time.Time    `db:"created_at"          json:"created_at"`
}

// CandidateScore 候选热点评分
type CandidateScore struct {
    TopicID    int64   `json:"topic_id"`
    TopicName  string  `json:"topic_name"`
    Scores     struct {
        Frequency    float64 `json:"frequency"`
        Recency      float64 `json:"recency"`
        ConceptMatch float64 `json:"concept_match"`
        Popularity   float64 `json:"popularity"`
    } `json:"scores"`
    TotalScore float64 `json:"total_score"`
}

// CandidateScoresPayload candidate_scores JSONB 完整结构
type CandidateScoresPayload struct {
    Candidates       []CandidateScore `json:"candidates"`
    DualAttribution  bool             `json:"dual_attribution"`
    ScoreGap         float64          `json:"score_gap"`
}
```

#### Insert SQL

```sql
INSERT INTO classification_audit_log (
    date, ts_code, topic_id, classify_layer, strategy,
    candidate_scores, evidence_text, confidence
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);
```

#### 人工修正 Update SQL

```sql
-- 人工审核后标注正确答案，用于后期训练
UPDATE classification_audit_log
SET corrected_topic_id = $1
WHERE id = $2;
```

---

### 2.13 表 12：market_snapshots（市场快照 / 爬取记录）

记录韭研公社爬虫的每日执行情况和原始数据，用于数据完整性监控和重试。
`raw_data` (JSONB) 可选存储原始返回数据，便于数据溯源和问题排查。

#### DDL

```sql
CREATE TABLE market_snapshots (
    id              BIGSERIAL       PRIMARY KEY,
    date            DATE            NOT NULL,               -- 爬取的目标日期
    status          SMALLINT        NOT NULL DEFAULT 0,     -- 0=待处理 1=成功 2=失败
    raw_data        JSONB,                                  -- 原始返回数据（可选）
    topic_count     INT             NOT NULL DEFAULT 0,     -- 该日热点数量
    stock_count     INT             NOT NULL DEFAULT 0,     -- 该日股票数量
    retry_count     INT             NOT NULL DEFAULT 0,     -- 重试次数
    error_msg       TEXT,                                   -- 错误信息
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_market_snapshots_date UNIQUE (date)
);

COMMENT ON TABLE  market_snapshots                  IS '市场快照 / 韭研爬取记录，每日一条';
COMMENT ON COLUMN market_snapshots.status           IS '0=待处理 1=成功 2=失败';
COMMENT ON COLUMN market_snapshots.raw_data         IS '韭研公社接口原始返回 JSON（可选，用于溯源）';
COMMENT ON COLUMN market_snapshots.topic_count      IS '该日爬取到的热点数量';
COMMENT ON COLUMN market_snapshots.stock_count      IS '该日爬取到的股票总数';
COMMENT ON COLUMN market_snapshots.retry_count      IS '爬取重试次数';

CREATE INDEX idx_market_snapshots_status ON market_snapshots (status);
CREATE INDEX idx_market_snapshots_date   ON market_snapshots (date DESC);
```

#### Go Struct

```go
// MarketSnapshot 市场快照 / 爬取记录
type MarketSnapshot struct {
    ID          int64        `db:"id"          json:"id"`
    Date        time.Time    `db:"date"        json:"date"`
    Status      int16        `db:"status"      json:"status"`
    RawData     pgtype.JSONB `db:"raw_data"    json:"raw_data,omitempty"`
    TopicCount  int          `db:"topic_count" json:"topic_count"`
    StockCount  int          `db:"stock_count" json:"stock_count"`
    RetryCount  int          `db:"retry_count" json:"retry_count"`
    ErrorMsg    *string      `db:"error_msg"   json:"error_msg,omitempty"`
    CreatedAt   time.Time    `db:"created_at"  json:"created_at"`
    UpdatedAt   time.Time    `db:"updated_at"  json:"updated_at"`
}
```

#### Upsert SQL

```sql
INSERT INTO market_snapshots (date, status, raw_data, topic_count, stock_count, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (date) DO UPDATE SET
    status      = EXCLUDED.status,
    raw_data    = EXCLUDED.raw_data,
    topic_count = EXCLUDED.topic_count,
    stock_count = EXCLUDED.stock_count,
    retry_count = market_snapshots.retry_count + 1,
    updated_at  = NOW();
```

#### 失败重试 Update SQL

```sql
UPDATE market_snapshots
SET status      = $1,
    error_msg   = $2,
    retry_count = retry_count + 1,
    updated_at  = NOW()
WHERE date = $3;
```

---

### 2.14 幂等写入策略汇总

| 表名 | 幂等键（唯一约束） | 冲突策略 | 说明 |
|------|---------------------|----------|------|
| `stock_basic_info` | `(ts_code)` | `DO UPDATE` 全字段更新 | Tushare 数据可能变更 |
| `topics` | `(name)` | `DO UPDATE` 累加 occurrence_count | 同名热点累加出现次数 |
| `topic_synonyms` | `(synonym)` | `DO NOTHING` | 同义词已存在则跳过 |
| `tushare_concepts` | `(concept_code)` | `DO UPDATE` 更新名称 | 概念名可能变更 |
| `topic_concepts` | `(concept_name, topic_id)` | `DO NOTHING` | 映射已存在则跳过 |
| `stock_topic_relations` | `(ts_code, topic_id)` | `DO UPDATE` hit_count + 1 | 韭研频次累加 |
| `tushare_concept_details` | `(ts_code, concept_name)` | `DO NOTHING` | 成分股已存在则跳过 |
| `focus_topics` | `(date, topic_id)` | `DO NOTHING` | 关注已存在则跳过 |
| `daily_stock_pool` | `(date, ts_code, pool_type)` | `DO UPDATE` 全字段更新 | 崩溃重启安全重写 |
| `market_snapshots` | `(date)` | `DO UPDATE` + retry_count + 1 | 重试时累加计数 |

---

### 2.15 Redis Key 设计（15 个）

以下使用 YAML 结构化声明每个 Redis Key 的完整设计。

#### Key 1：实时行情快照

```yaml
key_pattern: "rt:quote:{ts_code}"
type: HASH
fields:
  price: "当前价格 (float)"
  pre_close: "昨收价 (float)"
  pct_chg: "涨跌幅百分比 (float)"
  vol: "成交量-手 (int)"
  amount: "成交额-元 (float)"
  turnover: "换手率% (float)"
  update_time: "行情更新时间 (HH:MM:SS)"
ttl: "当日有效，每日 09:20 清理前日 Key"
example:
  key: "rt:quote:000001.SZ"
  value:
    price: "15.32"
    pre_close: "13.93"
    pct_chg: "9.98"
    vol: "1234567"
    amount: "189234567.89"
    turnover: "5.67"
    update_time: "14:30:10"
notes: "每 10s 由 Tushare 实时行情接口更新，不落盘到 PG"
```

#### Key 2：涨停池（Set）

```yaml
key_pattern: "pool:limit_up:{date}"
type: SET
members: "ts_code 字符串，如 000001.SZ"
ttl: "3天"
example:
  key: "pool:limit_up:2026-03-17"
  members: ["000001.SZ", "300999.SZ", "688001.SH"]
notes: "涨停判定后 SADD，打开涨停后 SREM"
```

#### Key 3：涨停池排序（SortedSet）

```yaml
key_pattern: "pool:limit_up:sorted:{date}"
type: SORTED_SET
member: "ts_code"
score: "首次涨停时间的数值表示 HHMMSS，如 093015 = 09:30:15"
ttl: "3天"
example:
  key: "pool:limit_up:sorted:2026-03-17"
  members:
    - member: "300999.SZ"
      score: 93015
    - member: "000001.SZ"
      score: 100230
notes: "按首次涨停时间排序，用于前端展示涨停时间线"
```

#### Key 4：5%池（Set）

```yaml
key_pattern: "pool:above5:{date}"
type: SET
members: "ts_code 字符串"
ttl: "3天"
example:
  key: "pool:above5:2026-03-17"
  members: ["000001.SZ", "300999.SZ", "600100.SH"]
notes: "涨幅 >= 5% 时 SADD，跌回 5% 以下时 SREM"
```

#### Key 5：股票-热点映射缓存

```yaml
key_pattern: "cache:stock_topics"
type: HASH
field: "ts_code"
value: "JSON 数组字符串，热点 ID 列表"
ttl: "永久，系统启动时预热，运行中持续更新，每日收盘后全量刷新"
example:
  key: "cache:stock_topics"
  fields:
    "000001.SZ": "[1,3,7]"
    "300999.SZ": "[2,5]"
notes: |
  分类引擎第 1 层(L1_REDIS)查询此缓存。
  若命中且 source=manual 则直接采信，跳过归因算法。
  缓存未命中时回查 PG 并回填。
```

#### Key 6：关联强度缓存

```yaml
key_pattern: "cache:bind_strength"
type: HASH
field: "{ts_code}:{topic_id}"
value: "关联次数 (int)"
ttl: "永久，定期刷新"
example:
  key: "cache:bind_strength"
  fields:
    "000001.SZ:42": "15"
    "000001.SZ:57": "3"
notes: "归因算法的频次维度权重数据源，对应 stock_topic_relations.hit_count"
```

#### Key 7：股票-概念标签缓存

```yaml
key_pattern: "cache:stock_concepts"
type: HASH
field: "ts_code"
value: "JSON 数组字符串，概念名称列表"
ttl: "永久，每周/每月刷新"
example:
  key: "cache:stock_concepts"
  fields:
    "000001.SZ": '["人工智能","芯片概念","大数据"]'
    "300999.SZ": '["锂电池","新能源"]'
notes: "分类引擎第 3 层(L3_PG_CONCEPT)查询此缓存，覆盖全市场 ~5300 只股票"
```

#### Key 8：概念-热点映射缓存

```yaml
key_pattern: "cache:concept_to_topics"
type: HASH
field: "concept_name"
value: "JSON 数组字符串，热点 ID 列表"
ttl: "永久，映射关系变更时刷新"
example:
  key: "cache:concept_to_topics"
  fields:
    "人工智能": "[42,57]"
    "芯片概念": "[42,89]"
    "锂电池": "[12]"
notes: "概念->热点的间接映射，配合 cache:stock_concepts 实现 L3 层分类"
```

#### Key 9：今日关注热点集合

```yaml
key_pattern: "focus:topics:{date}"
type: SET
members: "topic_id 字符串"
ttl: "3天"
example:
  key: "focus:topics:2026-03-17"
  members: ["42", "57", "12"]
notes: "策略告警引擎查询此 Set 判断热点是否在关注列表中"
```

#### Key 10：首次涨停时间记录

```yaml
key_pattern: "first_limit:{date}"
type: HASH
field: "ts_code"
value: "首次涨停时间 HH:MM:SS"
ttl: "当日有效"
example:
  key: "first_limit:2026-03-17"
  fields:
    "000001.SZ": "09:30:15"
    "300999.SZ": "09:31:02"
notes: |
  仅记录首次涨停时间，后续打开再涨停不更新。
  抖动抑制：仅从"非涨停区间"首次进入"涨停区间"时写入。
  HSETNX 保证只写一次。
```

#### Key 11：告警去重集合

```yaml
key_pattern: "alerted:{date}"
type: SET
members: "复合键字符串 {ts_code}:{topic_id}"
ttl: "当日有效"
example:
  key: "alerted:2026-03-17"
  members: ["000001.SZ:42", "300999.SZ:57"]
notes: "策略告警触发前先 SISMEMBER 判重，避免对同一股票同一热点重复告警"
```

#### Key 12：热点涨停股数量（实时）

```yaml
key_pattern: "topic:activity:limit:{date}"
type: SORTED_SET
member: "topic_id 字符串"
score: "该热点下当日涨停的股票数量"
ttl: "当日有效"
example:
  key: "topic:activity:limit:2026-03-17"
  members:
    - member: "42"
      score: 8
    - member: "57"
      score: 3
notes: "实时维护，新涨停时 ZINCRBY +1，打开涨停时 ZINCRBY -1"
```

#### Key 13：热点涨停时间序列

```yaml
key_pattern: "topic:limit_times:{date}:{topic_id}"
type: LIST
elements: "JSON 字符串 {ts_code, time}"
ttl: "当日有效"
example:
  key: "topic:limit_times:2026-03-17:42"
  elements:
    - '{"ts_code":"000001.SZ","time":"09:30:15"}'
    - '{"ts_code":"300999.SZ","time":"09:45:30"}'
notes: "记录每个热点内各股票的涨停时间，按发生顺序 RPUSH"
```

#### Key 14：当前全量快照

```yaml
key_pattern: "snapshot:current"
type: STRING
value: "全量快照 JSON（涨停池+5%池+热点活跃度+行情摘要）"
ttl: "永久，每 10s 覆盖写入"
example:
  key: "snapshot:current"
  value: '{"timestamp":"2026-03-17T14:30:10+08:00","limit_up_count":25,"above5_count":87,...}'
notes: |
  供新 WebSocket 连接初始化使用。
  新客户端连接后先 GET snapshot:current 获取全量状态，
  然后通过 Pub/Sub 接收增量更新。
```

#### Key 15：单股行情失败计数

```yaml
key_pattern: "stock:error_count:{date}:{ts_code}"
type: STRING
value: "失败次数 (int)"
ttl: "当日有效"
example:
  key: "stock:error_count:2026-03-17:000001.SZ"
  value: "3"
notes: "每次行情获取失败时 INCR，连续失败 5 次触发飞书 Webhook 告警"
```

---

### 2.16 Redis Key 命名规范汇总

```text
命名空间分类：
  rt:*              -- 实时数据（行情快照）
  pool:*            -- 实时池子（涨停池、5%池）
  cache:*           -- 缓存数据（映射表、概念标签）
  focus:*           -- 用户关注（热点关注列表）
  first_limit:*     -- 涨停时间记录
  alerted:*         -- 告警去重
  topic:*           -- 热点活跃度统计
  snapshot:*        -- 全量快照
  stock:*           -- 单股级别数据

TTL 策略：
  当日有效          -- 每日 09:20 日初重置时清理
  3天               -- 保留近 3 个交易日，自然过期
  永久+定期刷新     -- 核心缓存，不设 TTL，通过定时任务刷新
  永久+每10s覆盖    -- 全量快照，持续覆盖
```

---

### 2.17 完整建表 SQL 脚本（按顺序执行）

将以下脚本保存为 `migrations/000_full_schema.sql`，一次性执行即可创建全部表和索引。

```sql
-- ============================================================
-- Stock Monitor System - Full Schema
-- 执行顺序已考虑外键依赖
-- 数据库：PostgreSQL 15+
-- 时区：Asia/Shanghai，所有时间戳使用 TIMESTAMPTZ
-- ============================================================

BEGIN;

-- ============================================================
-- 第 1 批：无外键依赖
-- ============================================================

-- 表 1：stock_basic_info
CREATE TABLE IF NOT EXISTS stock_basic_info (
    id              BIGSERIAL       PRIMARY KEY,
    ts_code         VARCHAR(16)     NOT NULL,
    symbol          VARCHAR(10)     NOT NULL,
    name            VARCHAR(64)     NOT NULL,
    exchange        VARCHAR(8)      NOT NULL,
    board_code      VARCHAR(16)     NOT NULL DEFAULT 'MAIN',
    industry        VARCHAR(64),
    is_st           BOOLEAN         NOT NULL DEFAULT FALSE,
    list_date       DATE,
    status          SMALLINT        NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_stock_basic_info_ts_code UNIQUE (ts_code)
);
CREATE INDEX IF NOT EXISTS idx_stock_basic_info_board_code ON stock_basic_info (board_code);
CREATE INDEX IF NOT EXISTS idx_stock_basic_info_exchange   ON stock_basic_info (exchange);
CREATE INDEX IF NOT EXISTS idx_stock_basic_info_status     ON stock_basic_info (status) WHERE status = 1;

-- 表 2：topics
CREATE TABLE IF NOT EXISTS topics (
    id                BIGSERIAL       PRIMARY KEY,
    name              VARCHAR(128)    NOT NULL,
    source            VARCHAR(16)     NOT NULL DEFAULT 'jiuyan',
    jiuyan_field_id   VARCHAR(64),
    first_seen_date   DATE,
    last_seen_date    DATE,
    occurrence_count  INT             NOT NULL DEFAULT 0,
    priority          INT             NOT NULL DEFAULT 0,
    is_active         BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_topics_name UNIQUE (name)
);
CREATE INDEX IF NOT EXISTS idx_topics_source   ON topics (source);
CREATE INDEX IF NOT EXISTS idx_topics_active   ON topics (is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_topics_priority ON topics (priority DESC) WHERE is_active = TRUE;

-- 表 4：tushare_concepts
CREATE TABLE IF NOT EXISTS tushare_concepts (
    id              BIGSERIAL       PRIMARY KEY,
    concept_code    VARCHAR(16)     NOT NULL,
    concept_name    VARCHAR(128)    NOT NULL,
    source          VARCHAR(16)     NOT NULL DEFAULT 'tushare',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tushare_concepts_code UNIQUE (concept_code),
    CONSTRAINT uq_tushare_concepts_name UNIQUE (concept_name)
);

-- ============================================================
-- 第 2 批：依赖第 1 批
-- ============================================================

-- 表 3：topic_synonyms
CREATE TABLE IF NOT EXISTS topic_synonyms (
    id          BIGSERIAL       PRIMARY KEY,
    topic_id    BIGINT          NOT NULL,
    synonym     VARCHAR(128)    NOT NULL,
    source      VARCHAR(16)     NOT NULL DEFAULT 'jiuyan',
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_topic_synonyms_synonym UNIQUE (synonym),
    CONSTRAINT fk_topic_synonyms_topic   FOREIGN KEY (topic_id)
        REFERENCES topics (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_topic_synonyms_topic_id ON topic_synonyms (topic_id);

-- 表 5：topic_concepts
CREATE TABLE IF NOT EXISTS topic_concepts (
    id              BIGSERIAL       PRIMARY KEY,
    concept_name    VARCHAR(128)    NOT NULL,
    concept_code    VARCHAR(16)     NOT NULL,
    topic_id        BIGINT          NOT NULL,
    match_type      VARCHAR(16)     NOT NULL DEFAULT 'exact',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_topic_concepts_name_topic UNIQUE (concept_name, topic_id),
    CONSTRAINT fk_topic_concepts_topic      FOREIGN KEY (topic_id)
        REFERENCES topics (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_topic_concepts_topic_id     ON topic_concepts (topic_id);
CREATE INDEX IF NOT EXISTS idx_topic_concepts_concept_code ON topic_concepts (concept_code);

-- 表 6：stock_topic_relations
CREATE TABLE IF NOT EXISTS stock_topic_relations (
    id              BIGSERIAL       PRIMARY KEY,
    ts_code         VARCHAR(16)     NOT NULL,
    topic_id        BIGINT          NOT NULL,
    source          VARCHAR(16)     NOT NULL DEFAULT 'jiuyan',
    confidence      FLOAT,
    hit_count       INT             NOT NULL DEFAULT 1,
    last_seen_date  DATE,
    first_seen_date DATE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_stock_topic_relations_ts_topic UNIQUE (ts_code, topic_id),
    CONSTRAINT fk_stock_topic_relations_topic    FOREIGN KEY (topic_id)
        REFERENCES topics (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_stock_topic_relations_ts_code  ON stock_topic_relations (ts_code);
CREATE INDEX IF NOT EXISTS idx_stock_topic_relations_topic_id ON stock_topic_relations (topic_id);
CREATE INDEX IF NOT EXISTS idx_stock_topic_relations_source   ON stock_topic_relations (source);

-- 表 7：tushare_concept_details
CREATE TABLE IF NOT EXISTS tushare_concept_details (
    id              BIGSERIAL       PRIMARY KEY,
    ts_code         VARCHAR(16)     NOT NULL,
    concept_name    VARCHAR(128)    NOT NULL,
    concept_code    VARCHAR(16)     NOT NULL,
    source          VARCHAR(16)     NOT NULL DEFAULT 'tushare',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tushare_concept_details_ts_concept UNIQUE (ts_code, concept_name)
);
CREATE INDEX IF NOT EXISTS idx_tushare_concept_details_ts_code      ON tushare_concept_details (ts_code);
CREATE INDEX IF NOT EXISTS idx_tushare_concept_details_concept_code ON tushare_concept_details (concept_code);

-- 表 8：focus_topics
CREATE TABLE IF NOT EXISTS focus_topics (
    id          BIGSERIAL       PRIMARY KEY,
    date        DATE            NOT NULL,
    topic_id    BIGINT          NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_focus_topics_date_topic UNIQUE (date, topic_id),
    CONSTRAINT fk_focus_topics_topic      FOREIGN KEY (topic_id)
        REFERENCES topics (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_focus_topics_date     ON focus_topics (date);
CREATE INDEX IF NOT EXISTS idx_focus_topics_topic_id ON focus_topics (topic_id);

-- ============================================================
-- 第 3 批：分区表和日志表
-- ============================================================

-- 表 9：daily_stock_pool（按月分区）
CREATE TABLE IF NOT EXISTS daily_stock_pool (
    id              BIGSERIAL,
    date            DATE            NOT NULL,
    ts_code         VARCHAR(16)     NOT NULL,
    stock_name      VARCHAR(64)     NOT NULL,
    pool_type       SMALLINT        NOT NULL,
    change_pct      FLOAT,
    current_price   FLOAT,
    pre_close       FLOAT,
    limit_up_price  FLOAT,
    first_limit_time TIME,
    board_code      VARCHAR(16),
    topic_ids       BIGINT[],
    snapshot_time   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_daily_stock_pool_date_ts_pool UNIQUE (date, ts_code, pool_type),
    PRIMARY KEY (id, date)
) PARTITION BY RANGE (date);

CREATE INDEX IF NOT EXISTS idx_daily_stock_pool_date      ON daily_stock_pool (date);
CREATE INDEX IF NOT EXISTS idx_daily_stock_pool_ts_code   ON daily_stock_pool (ts_code, date);
CREATE INDEX IF NOT EXISTS idx_daily_stock_pool_pool_type ON daily_stock_pool (date, pool_type);
CREATE INDEX IF NOT EXISTS idx_daily_stock_pool_topic_ids ON daily_stock_pool USING GIN (topic_ids);

-- 创建近期分区
CREATE TABLE IF NOT EXISTS daily_stock_pool_2026_03
    PARTITION OF daily_stock_pool
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE IF NOT EXISTS daily_stock_pool_2026_04
    PARTITION OF daily_stock_pool
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

-- 表 10：strategy_alerts
CREATE TABLE IF NOT EXISTS strategy_alerts (
    id              BIGSERIAL       PRIMARY KEY,
    date            DATE            NOT NULL,
    ts_code         VARCHAR(16)     NOT NULL,
    stock_name      VARCHAR(64)     NOT NULL,
    topic_id        BIGINT,
    topic_name      VARCHAR(128),
    alert_type      SMALLINT        NOT NULL DEFAULT 1,
    trigger_price   FLOAT,
    trigger_time    TIMESTAMPTZ,
    prev_day_pct    FLOAT,
    extra_info      JSONB,
    notified        BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_strategy_alerts_date     ON strategy_alerts (date);
CREATE INDEX IF NOT EXISTS idx_strategy_alerts_ts_code  ON strategy_alerts (ts_code, date);
CREATE INDEX IF NOT EXISTS idx_strategy_alerts_topic_id ON strategy_alerts (topic_id);
CREATE INDEX IF NOT EXISTS idx_strategy_alerts_notified ON strategy_alerts (notified) WHERE notified = FALSE;

-- 表 11：classification_audit_log
CREATE TABLE IF NOT EXISTS classification_audit_log (
    id                  BIGSERIAL       PRIMARY KEY,
    date                DATE            NOT NULL,
    ts_code             VARCHAR(16)     NOT NULL,
    topic_id            BIGINT,
    classify_layer      VARCHAR(32)     NOT NULL,
    strategy            VARCHAR(32)     NOT NULL,
    candidate_scores    JSONB,
    evidence_text       TEXT,
    confidence          FLOAT,
corrected_topic_id  BIGINT,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_classification_audit_log_stock_date ON classification_audit_log (ts_code, date);
CREATE INDEX IF NOT EXISTS idx_classification_audit_log_layer      ON classification_audit_log (classify_layer);
CREATE INDEX IF NOT EXISTS idx_classification_audit_log_date       ON classification_audit_log (date);

-- 表 12：market_snapshots
CREATE TABLE IF NOT EXISTS market_snapshots (
    id              BIGSERIAL       PRIMARY KEY,
    date            DATE            NOT NULL,
    status          SMALLINT        NOT NULL DEFAULT 0,
    raw_data        JSONB,
    topic_count     INT             NOT NULL DEFAULT 0,
    stock_count     INT             NOT NULL DEFAULT 0,
    retry_count     INT             NOT NULL DEFAULT 0,
    error_msg       TEXT,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_market_snapshots_date UNIQUE (date)
);
CREATE INDEX IF NOT EXISTS idx_market_snapshots_status ON market_snapshots (status);
CREATE INDEX IF NOT EXISTS idx_market_snapshots_date   ON market_snapshots (date DESC);

COMMIT;
```

---

## 三、业务规则（Business Rules）

> **阅读对象**: AI coding agent（Copilot / Cursor / Claude Code）
> **约定**: 每条规则以 `RULE:` 前缀声明，函数签名为 Go 1.22+，所有 struct 字段均带 json tag。
> 本章不涉及 HTTP handler / 数据库 DDL / Redis key 格式——这些分别在 Part 4、Part 2 中定义。

---

### 2.1 涨停判定（Board Detection）

#### 2.1.1 板块类型枚举

```go
// BoardCode 板块编码枚举
type BoardCode string

const (
    BoardMain BoardCode = "MAIN" // 主板（沪市60xxxx，深市000xxx/001xxx/002xxx）
    BoardGEM  BoardCode = "GEM"  // 创业板（300xxx/301xxx）
    BoardSTAR BoardCode = "STAR" // 科创板（688xxx）
    BoardBSE  BoardCode = "BSE"  // 北交所（8xxxxx/4xxxxx）
)
```

#### 2.1.2 板块规则结构

```go
// BoardRule 板块涨跌停规则（从 boards 表加载到内存缓存）
type BoardRule struct {
    BoardCode      BoardCode `json:"board_code"`
    BoardName      string    `json:"board_name"`
    LimitUpRatio   float64   `json:"limit_up_ratio"`   // 0.10 / 0.20 / 0.30
    LimitDownRatio float64   `json:"limit_down_ratio"`  // -0.10 / -0.20 / -0.30
    CodePatterns   []string  `json:"code_patterns"`     // {"60____","300___","688___",...}
}
```

| 板块 | board_code | limit_up_ratio | limit_down_ratio | 代码特征 |
|------|-----------|----------------|------------------|----------|
| 主板 | MAIN | 0.10 | -0.10 | 60xxxx, 000xxx, 001xxx, 002xxx |
| 创业板 | GEM | 0.20 | -0.20 | 300xxx, 301xxx |
| 科创板 | STAR | 0.20 | -0.20 | 688xxx |
| 北交所 | BSE | 0.30 | -0.30 | 8xxxxx, 4xxxxx |

```
RULE: ST 股票（is_st == true）直接跳过，不参与任何涨停/5% 计算。
RULE: 板块规则从 boards 表加载到 map[BoardCode]BoardRule 内存缓存；每日收盘后热更新，运行时零 DB 查询。
RULE: 板块归属判定顺序——先匹配 688→STAR，再匹配 300/301→GEM，再匹配 8/4→BSE，其余→MAIN。
```

#### 2.1.3 涨停价计算

```go
// CalcLimitUpPrice 计算涨停价
//   公式: Round(preClose × (1 + limitUpRatio), 2)
//   四舍五入到分（小数点后两位）
func CalcLimitUpPrice(preClose float64, limitUpRatio float64) float64
```

```
RULE: limit_up_price = math.Round(preClose * (1 + limitUpRatio) * 100) / 100
RULE: limitUpRatio 从 boards 表动态读取，禁止硬编码 0.10 / 0.20 等字面量。
RULE: preClose <= 0 时返回 0（防御性校验）。
```

#### 2.1.4 涨停状态枚举与状态机

```go
// LimitState 涨停抖动抑制状态机
type LimitState int

const (
    LimitStateNone     LimitState = 0 // 未涨停
    LimitStateLimitUp  LimitState = 1 // 涨停中（封板）
    LimitStateOpened   LimitState = 2 // 曾涨停但已打开
    LimitStateReSealed LimitState = 3 // 再次封板
)
```

状态转移表：

| 当前状态 | 事件: price >= limitUpPrice | 事件: price < limitUpPrice |
|---------|---------------------------|---------------------------|
| NONE | → LIMIT_UP（记录首次涨停时间） | 保持 NONE |
| LIMIT_UP | 保持 LIMIT_UP | → OPENED |
| OPENED | → RE_SEALED | 保持 OPENED |
| RE_SEALED | 保持 RE_SEALED | → OPENED |

```
RULE: 首次涨停时间 = 状态首次从 NONE → LIMIT_UP 的时刻；后续 OPENED→RE_SEALED 不更新首次涨停时间。
RULE: 首次涨停时间写入 Redis key first_limit:{date} field=ts_code，整个交易日只写一次（HSETNX 语义）。
RULE: 抖动抑制——只有 NONE→LIMIT_UP 转移才产生「首次涨停事件」，OPENED→RE_SEALED 不产生新事件。
```

#### 2.1.5 板类型枚举（连板计数）

```go
// BoardSeqType 连板类型
type BoardSeqType int

const (
    BoardSeqFirst  BoardSeqType = 1 // 首板
    BoardSeqSecond BoardSeqType = 2 // 二板（连续第2天涨停）
    BoardSeqThird  BoardSeqType = 3 // 三板
    // ... 依次递增
)
```

```
RULE: 连板数 = 从今日起向前连续涨停的天数。查询 daily_stock_pool 中 pool_type=1 的连续日期数。
RULE: 中间有一天非涨停则连板重置为 1（首板）。
```

#### 2.1.6 DetectLimitUp 函数签名

```go
// DetectInput 涨停判定输入
type DetectInput struct {
    TsCode       string     `json:"ts_code"`        // 股票代码，如 "000001.SZ"
    StockName    string     `json:"stock_name"`      // 股票名称
    CurrentPrice float64    `json:"current_price"`   // 实时价格
    PreClose     float64    `json:"pre_close"`       // 昨收价
    ChangePct    float64    `json:"change_pct"`      // 涨跌幅(%)，如 9.98
    Volume       float64    `json:"volume"`          // 成交量
    BoardCode    BoardCode  `json:"board_code"`      // 板块编码
    IsST         bool       `json:"is_st"`           // 是否ST
    QuoteTime    time.Time  `json:"quote_time"`      // 行情时间（交易所时间）
    PrevState    LimitState `json:"prev_state"`      // 上一轮状态
}

// DetectOutput 涨停判定输出
type DetectOutput struct {
    IsLimitUp       bool         `json:"is_limit_up"`        // 当前是否涨停
    IsAbove5Pct     bool         `json:"is_above_5pct"`      // 涨幅是否 >= 5%
    IsFirstLimitUp  bool         `json:"is_first_limit_up"`  // 是否为今日首次涨停事件
    LimitUpPrice    float64      `json:"limit_up_price"`     // 计算得到的涨停价
    FirstLimitTime  *time.Time   `json:"first_limit_time"`   // 首次涨停时间（nil 表示今日未涨停过）
    CurrentState    LimitState   `json:"current_state"`      // 状态机新状态
    BoardSeq        BoardSeqType `json:"board_seq"`          // 连板类型
    Skipped         bool         `json:"skipped"`            // 是否被跳过（ST等）
    SkipReason      string       `json:"skip_reason"`        // 跳过原因
}

// DetectLimitUp 核心涨停判定函数
//
// 执行顺序：
//   1. 校验 IsST → true 则 Skipped=true, return
//   2. 根据 BoardCode 从内存缓存获取 BoardRule
//   3. CalcLimitUpPrice(PreClose, rule.LimitUpRatio)
//   4. 判定 IsLimitUp = (CurrentPrice >= LimitUpPrice)
//   5. 判定 IsAbove5Pct = (ChangePct >= 5.0)
//   6. 状态机转移：根据 PrevState + IsLimitUp 计算 CurrentState
//   7. IsFirstLimitUp = (PrevState == NONE && CurrentState == LIMIT_UP)
//   8. 若 IsFirstLimitUp，记录 FirstLimitTime = QuoteTime
//   9. 查询连板数 → BoardSeq
func DetectLimitUp(input DetectInput) (DetectOutput, error)
```

```
RULE: 当 CurrentPrice >= LimitUpPrice 时判定为涨停。使用 >= 而非 ==，因为集合竞价可能出现高于涨停价的成交。
RULE: IsAbove5Pct 和 IsLimitUp 互斥——涨停股不放入 5% 池，只放涨停池。即 IsAbove5Pct = (ChangePct >= 5.0) && !IsLimitUp。
RULE: 若 PreClose == 0（新股/异常数据），返回 error，不产生任何判定。
RULE: 整个函数无 DB/Redis I/O——所有状态通过参数传入传出，由调用方负责持久化。
```

---

### 2.2 四层分类引擎（Four-Layer Classification）

#### 2.2.1 分类层级枚举

```go
// ClassifyLayer 分类命中层级
type ClassifyLayer string

const (
    LayerL1Redis     ClassifyLayer = "L1_REDIS"      // Redis 缓存（韭研动态标签）
    LayerL2PGJiuyan  ClassifyLayer = "L2_PG_JIUYAN"  // PG 韭研映射表
    LayerL3PGConcept ClassifyLayer = "L3_PG_CONCEPT"  // PG 概念板块
    LayerL4LLM       ClassifyLayer = "L4_LLM"         // LLM 异步兜底
)

// ClassifyStrategy 分类策略标识
type ClassifyStrategy string

const (
    StrategyManual             ClassifyStrategy = "MANUAL"              // 手动标注（最高优先级）
    StrategyJiuyanAttribution  ClassifyStrategy = "JIUYAN_ATTRIBUTION"  // 韭研数据 + 归因算法
    StrategyConceptAttribution ClassifyStrategy = "CONCEPT_ATTRIBUTION" // 概念数据 + 归因算法（概念模式权重）
    StrategyLLMV1              ClassifyStrategy = "LLM_V1"              // LLM 异步分类
)
```

#### 2.2.2 分类输入/输出

```go
// ClassifyInput 分类引擎输入
type ClassifyInput struct {
    TsCode    string    `json:"ts_code"`
    StockName string    `json:"stock_name"`
    QuoteTime time.Time `json:"quote_time"`  // 涨停/入池时间
    IsLimitUp bool      `json:"is_limit_up"` // 是否涨停（决定是否触发告警链路）
    Date      string    `json:"date"`        // 交易日 YYYY-MM-DD
}

// TopicScore 候选热点评分结果
type TopicScore struct {
    TopicID        int64   `json:"topic_id"`
    TopicName      string  `json:"topic_name"`
    SActivity      float64 `json:"s_activity"`       // S1 活跃度得分
    SBindStrength  float64 `json:"s_bind_strength"`  // S2 关联强度得分
    STimeProximity float64 `json:"s_time_proximity"` // S3 时间接近度得分
    SRecency       float64 `json:"s_recency"`        // S4 时效性得分
    TotalScore     float64 `json:"total_score"`      // 加权总分
    Source         string  `json:"source"`           // 数据来源: jiuyan/tushare/llm/manual
}

// ClassifyOutput 分类引擎输出
type ClassifyOutput struct {
    FinalTopicIDs  []int64          `json:"final_topic_ids"`  // 最终归因热点ID（1或2个）
    FinalTopics    []TopicScore     `json:"final_topics"`     // 最终归因热点详情
    Layer          ClassifyLayer    `json:"layer"`            // 命中层级
    Strategy       ClassifyStrategy `json:"strategy"`         // 使用的策略
    AllCandidates  []TopicScore     `json:"all_candidates"`   // 所有候选热点及评分
    Confidence     float64          `json:"confidence"`       // 置信度 0~1
    IsUnclassified bool             `json:"is_unclassified"`  // 是否未分类
    EvidenceText   string           `json:"evidence_text"`    // 关键证据文本
}

// ClassifyStock 四层分类引擎入口
//
// 执行顺序（严格逐层降级，命中即停）：
//   L1 → L2 → L3 → L4
// 特殊规则：任何层级若命中 source=manual 的标签，直接采信，跳过归因算法。
func ClassifyStock(ctx context.Context, input ClassifyInput) (ClassifyOutput, error)
```

#### 2.2.3 L1: Redis 缓存层

```
RULE-L1-KEY: Redis key = cache:stock_topics (Hash), field = ts_code, value = JSON array of TopicMapping.
RULE-L1-HIT: field 存在且 JSON 数组非空 → 命中。
RULE-L1-MISS: field 不存在 → 降级到 L2。
RULE-L1-MANUAL: 若命中的映射中存在 source == "manual" 的条目，
    直接返回该条目的 topic_id，Strategy = MANUAL，跳过归因算法。
RULE-L1-TTL: 永久缓存，由每日 09:20 预热任务和韭研增量爬虫触发刷新。
RULE-L1-LATENCY: 期望延迟 < 1ms。
```

```go
// TopicMapping Redis 缓存中的映射条目
type TopicMapping struct {
    TopicID      int64   `json:"topic_id"`
    TopicName    string  `json:"topic_name"`
    Source       string  `json:"source"`         // jiuyan / manual / llm
    HitCount     int     `json:"hit_count"`      // 韭研标注频次
    LastSeenDate string  `json:"last_seen_date"` // YYYY-MM-DD
    Confidence   float64 `json:"confidence"`     // LLM 时有意义
}
```

#### 2.2.4 L2: PostgreSQL 韭研映射表

```
RULE-L2-QUERY: SELECT * FROM stock_topic_mappings WHERE ts_code = $1 AND source IN ('jiuyan','manual')
RULE-L2-HIT: 返回行数 > 0 → 命中，回填 Redis 缓存（L1 key）。
RULE-L2-MISS: 返回行数 == 0 → 降级到 L3。
RULE-L2-MANUAL: 同 L1，若结果中存在 source='manual'，直接采信。
RULE-L2-BACKFILL: 命中后异步写入 Redis cache:stock_topics，下次直接走 L1。
RULE-L2-LATENCY: 期望延迟 ~5ms。
```

#### 2.2.5 L3: PostgreSQL 概念板块层

```
RULE-L3-TRIGGER: 仅当 L1 + L2 均未命中时进入（该股票从未在韭研数据中出现过）。
RULE-L3-QUERY-STEP1: SELECT concept_name FROM stock_concept_tags WHERE ts_code = $1
RULE-L3-QUERY-STEP2: 对每个 concept_name，查 concept_topic_mappings 获取对应 topic_id。
RULE-L3-HIT: 至少有一个 concept_name 能映射到有效 topic_id → 命中。
RULE-L3-MISS: 所有 concept_name 均无映射 → 降级到 L4。
RULE-L3-WEIGHT-MODE: 命中后进入归因算法时使用「概念模式权重」（见 2.3）。
RULE-L3-LATENCY: 期望延迟 ~5ms（概念标签也可预热到 Redis cache:stock_concepts）。
```

#### 2.2.6 L4: LLM 异步兜底

```
RULE-L4-TRIGGER: 仅当 L1 + L2 + L3 均未命中时进入（极端情况）。
RULE-L4-ASYNC: 异步调用 LLM API，不阻塞当前 10s 监控周期。
RULE-L4-CURRENT-ROUND: 当前轮标记 IsUnclassified = true。
RULE-L4-NEXT-ROUND: LLM 结果回写 stock_topic_mappings + Redis 后，下一个 10s 周期命中 L1。
RULE-L4-CONFIDENCE: LLM 返回 confidence < 0.7 → 不采信，保持 IsUnclassified。
RULE-L4-CONCURRENT: 最多 5 个并发 LLM 请求（config.llm.max_concurrent）。
RULE-L4-TIMEOUT: 单次 LLM 调用超时 30s（config.llm.timeout_sec）。
RULE-L4-SOURCE: 回写 stock_topic_mappings 时 source = 'llm'。
```

#### 2.2.7 四层分类引擎伪代码

```go
func ClassifyStock(ctx context.Context, input ClassifyInput) (ClassifyOutput, error) {
    // ── L1: Redis Cache ──
    mappings, err := redisGetStockTopics(ctx, input.TsCode)
    if err == nil && len(mappings) > 0 {
        // 检查手动标注
        if manual := findManualSource(mappings); manual != nil {
            return buildManualOutput(manual, LayerL1Redis), nil
        }
        // 进入归因算法（普通模式权重）
        return runAttribution(ctx, input, mappings, LayerL1Redis, WeightModeNormal)
    }

    // ── L2: PG Jiuyan Mappings ──
    mappings, err = pgGetJiuyanMappings(ctx, input.TsCode)
    if err == nil && len(mappings) > 0 {
        go backfillRedisCache(input.TsCode, mappings) // 异步回填
        if manual := findManualSource(mappings); manual != nil {
            return buildManualOutput(manual, LayerL2PGJiuyan), nil
        }
        return runAttribution(ctx, input, mappings, LayerL2PGJiuyan, WeightModeNormal)
    }

    // ── L3: PG Concept Tags ──
    conceptTopics, err := pgGetConceptTopics(ctx, input.TsCode)
    if err == nil && len(conceptTopics) > 0 {
        return runAttribution(ctx, input, conceptTopics, LayerL3PGConcept, WeightModeConcept)
    }

    // ── L4: LLM Async Fallback ──
    go asyncLLMClassify(ctx, input) // 异步，不阻塞
    return ClassifyOutput{
        IsUnclassified: true,
        Layer:          LayerL4LLM,
        Strategy:       StrategyLLMV1,
        Confidence:     0,
    }, nil
}
```

---

### 2.3 四维归因算法（Four-Dimensional Attribution）

#### 2.3.1 归因输入/输出

```go
// AttributionInput 归因算法输入
type AttributionInput struct {
    TsCode            string           `json:"ts_code"`
    StockName         string           `json:"stock_name"`
    QuoteTime         time.Time        `json:"quote_time"`         // 该股票涨停/入池时间
    Date              string           `json:"date"`               // 交易日 YYYY-MM-DD
    CandidateMappings []TopicMapping   `json:"candidate_mappings"` // 候选热点映射列表
    WeightMode        WeightMode       `json:"weight_mode"`        // 权重模式
}

// WeightMode 权重模式
type WeightMode int

const (
    WeightModeNormal  WeightMode = 0 // 普通模式（有韭研标签）
    WeightModeConcept WeightMode = 1 // 概念模式（仅概念标签，无频次数据）
)

// AttributionWeights 四维权重配置
type AttributionWeights struct {
    WActivity      float64 `json:"w_activity"`       // 活跃度权重
    WBindStrength  float64 `json:"w_bind_strength"`  // 关联强度权重
    WTimeProximity float64 `json:"w_time_proximity"` // 时间接近度权重
    WRecency       float64 `json:"w_recency"`        // 时效性权重
}

// AttributionOutput 归因算法输出
type AttributionOutput struct {
    FinalTopicIDs     []int64      `json:"final_topic_ids"`      // 归因结果（1~2个）
    AllScores         []TopicScore `json:"all_scores"`           // 所有候选评分（降序）
    IsDualAttribution bool         `json:"is_dual_attribution"`  // 是否双归因
    Confidence        float64      `json:"confidence"`           // 最高得分作为置信度
}

// RunAttribution 四维加权归因算法入口
func RunAttribution(ctx context.Context, input AttributionInput) (AttributionOutput, error)
```

#### 2.3.2 权重配置

| 权重模式 | W_activity | W_bindStrength | W_timeProximity | W_recency | 总和 |
|---------|-----------|---------------|----------------|----------|-----|
| Normal（有韭研标签） | 0.25 | **0.40** | 0.20 | 0.15 | 1.00 |
| Concept（仅概念标签） | **0.45** | 0.00 | **0.40** | 0.15 | 1.00 |

```go
var (
    WeightsNormal = AttributionWeights{
        WActivity:      0.25,
        WBindStrength:  0.40,
        WTimeProximity: 0.20,
        WRecency:       0.15,
    }
    WeightsConcept = AttributionWeights{
        WActivity:      0.45,
        WBindStrength:  0.00,
        WTimeProximity: 0.40,
        WRecency:       0.15,
    }
)
```

```
RULE: 概念模式下 W_bindStrength = 0，因为概念标签无频次数据，关联强度无法计算。
RULE: 概念模式下将关联强度的 0.40 权重分配给活跃度(+0.20)和时间接近度(+0.20)。
RULE: 四个权重之和必须 == 1.0。
```

#### 2.3.3 S1: 热点活跃度（Activity Score）

```go
// CalcActivityScore 计算热点活跃度得分
//   S_activity(T) = limit_count(T) / max_limit_count_all_topics
//
// 参数:
//   topicLimitCount  — 该热点今日涨停股数量
//   maxLimitCountAll — 所有活跃热点中涨停股最多的数量
// 返回: 0.0 ~ 1.0
func CalcActivityScore(topicLimitCount int, maxLimitCountAll int) float64
```

```
RULE-S1: S_activity = float64(topicLimitCount) / float64(maxLimitCountAll)
RULE-S1-ZERO: 若 maxLimitCountAll == 0，返回 0.0（防除零）。
RULE-S1-SOURCE: topicLimitCount 从 Redis SortedSet topic:activity:limit:{date} 读取（score = 涨停股数量）。
RULE-S1-RANGE: 结果归一化到 [0.0, 1.0]，最活跃热点得分 = 1.0。
```

#### 2.3.4 S2: 关联强度（Bind Strength Score）

```go
// CalcBindStrengthScore 计算股票-热点关联强度得分
//   S_bindStrength(X, T) = hit_count(X, T) / total_hit_count(X)
//
// 参数:
//   hitCount      — 该股票在该热点下的历史出现次数
//   totalHitCount — 该股票在所有热点下的历史出现次数之和
// 返回: 0.0 ~ 1.0
func CalcBindStrengthScore(hitCount int, totalHitCount int) float64
```

```
RULE-S2: S_bindStrength = float64(hitCount) / float64(totalHitCount)
RULE-S2-ZERO: 若 totalHitCount == 0，返回 0.0。
RULE-S2-SOURCE: hitCount 来自 stock_topic_mappings.hit_count（韭研标注频次）。
RULE-S2-CONCEPT-MODE: 概念模式下此信号被跳过（权重 = 0），无需计算。
RULE-S2-MANUAL: source='manual' 的映射已在分类引擎层拦截，不进入归因算法。
```

**示例**: 科大讯飞在 AI 热点下出现 47 次，机器人 8 次，教育 3 次。totalHitCount = 58。
AI 关联强度 = 47/58 = 0.8103，机器人 = 8/58 = 0.1379，教育 = 3/58 = 0.0517。

#### 2.3.5 S3: 时间接近度（Time Proximity Score）

```go
// CalcTimeProximityScore 计算涨停时间聚集性得分
//   S_timeProximity(X, T) = 1 / (1 + avgTimeGapMinutes / 15)
//
// 参数:
//   stockLimitTime  — 目标股票的涨停时间
//   topicLimitTimes — 该热点内已涨停股票的涨停时间列表（先于目标股票的）
// 返回: 0.0 ~ 1.0
func CalcTimeProximityScore(stockLimitTime time.Time, topicLimitTimes []time.Time) float64
```

```
RULE-S3: avgTimeGap = 该热点内先于目标股票涨停的所有股票与目标股票的平均时间差（分钟）。
RULE-S3-FORMULA: S_timeProximity = 1.0 / (1.0 + avgTimeGapMinutes / 15.0)
RULE-S3-HALFLIFE: 15分钟为半衰期——时间差 15min 时得分 0.5，0min 时得分 1.0。
RULE-S3-EMPTY: 若该热点内无先于目标股票涨停的股票（topicLimitTimes 过滤后为空），返回 0.5（中性值）。
RULE-S3-SOURCE: topicLimitTimes 从 Redis List topic:limit_times:{date}:{topic_id} 读取。
RULE-S3-FILTER: 仅使用早于 stockLimitTime 的条目计算，晚于的忽略。
```

#### 2.3.6 S4: 时效性（Recency Score）

```go
// CalcRecencyScore 计算映射时效性得分
//   S_recency(X, T) = 1 / (1 + daysSinceLastSeen / 30)
//
// 参数:
//   lastSeenDate — 该股票最近一次被标注到该热点的日期
//   today        — 当前交易日
// 返回: 0.0 ~ 1.0
func CalcRecencyScore(lastSeenDate time.Time, today time.Time) float64
```

```
RULE-S4: daysSinceLastSeen = today.Sub(lastSeenDate).Hours() / 24（天数差）。
RULE-S4-FORMULA: S_recency = 1.0 / (1.0 + float64(daysSinceLastSeen) / 30.0)
RULE-S4-HALFLIFE: 30天为半衰期——当天标注得分 1.0，30天前标注得分 0.5。
RULE-S4-ZERO: 若 lastSeenDate 为零值（从未出现），返回 0.1（最低时效性）。
RULE-S4-SOURCE: lastSeenDate 来自 stock_topic_mappings.last_seen_date。
```

#### 2.3.7 综合评分公式

```go
// CalcTotalScore 计算单个候选热点的综合得分
func CalcTotalScore(
    s1Activity      float64,
    s2BindStrength  float64,
    s3TimeProximity float64,
    s4Recency       float64,
    weights         AttributionWeights,
) float64 {
    return weights.WActivity*s1Activity +
        weights.WBindStrength*s2BindStrength +
        weights.WTimeProximity*s3TimeProximity +
        weights.WRecency*s4Recency
}
```

```
RULE-SCORE: Score(X, T) = W1*S1 + W2*S2 + W3*S3 + W4*S4
RULE-SCORE-RANGE: 理论范围 [0.0, 1.0]（因为每个 Si 属于 [0,1] 且 sum(Wi) = 1）。
RULE-SCORE-PRECISION: 保留 4 位小数: math.Round(score*10000) / 10000。
```

#### 2.3.8 归因决策规则

```go
// DecideAttribution 根据候选热点评分做出归因决策
//
// 决策规则：
//   1. 候选热点数 == 0 → 未分类
//   2. 候选热点数 == 1 → 直接归因
//   3. 候选热点数 >= 2 → 按总分降序排列
//      a. Top1.Score - Top2.Score < DualThreshold → 双归因（Top1 + Top2）
//      b. 否则 → 单一归因 Top1
//
// DualThreshold 默认 = 0.1（可通过配置调整）
func DecideAttribution(scores []TopicScore) AttributionOutput
```

```
RULE-DECIDE-EMPTY: len(scores) == 0 → 返回空 FinalTopicIDs, Confidence = 0。
RULE-DECIDE-SINGLE: len(scores) == 1 → FinalTopicIDs = [scores[0].TopicID], Confidence = scores[0].TotalScore。
RULE-DECIDE-MULTI: len(scores) >= 2 → 排序后取 Top1, Top2。
RULE-DECIDE-DUAL-THRESHOLD: 双归因阈值 = 0.1（可配置）。
RULE-DECIDE-DUAL: Top1.TotalScore - Top2.TotalScore < 0.1 → FinalTopicIDs = [Top1.TopicID, Top2.TopicID]，IsDualAttribution = true。
RULE-DECIDE-DUAL-ALERT: 双归因时，策略告警只要用户关注了其中任一热点即触发。
RULE-DECIDE-CONFIDENCE: Confidence = Top1.TotalScore（最高分）。
```

#### 2.3.9 归因算法完整伪代码

```go
func RunAttribution(ctx context.Context, input AttributionInput) (AttributionOutput, error) {
    weights := WeightsNormal
    if input.WeightMode == WeightModeConcept {
        weights = WeightsConcept
    }

    // ── Step 1: 获取全局热点活跃度数据 ──
    allTopicActivity, err := getTopicActivityMap(ctx, input.Date)
    // allTopicActivity: map[int64]int  — topicID → 今日涨停股数量
    if err != nil {
        return AttributionOutput{}, fmt.Errorf("get topic activity: %w", err)
    }

    // 找到最大涨停数（用于 S1 归一化）
    maxLimitCount := 0
    for _, count := range allTopicActivity {
        if count > maxLimitCount {
            maxLimitCount = count
        }
    }

    // ── Step 2: 计算该股票的 totalHitCount（用于 S2 归一化）──
    totalHitCount := 0
    for _, m := range input.CandidateMappings {
        totalHitCount += m.HitCount
    }

    // ── Step 3: 逐个候选热点评分 ──
    var scores []TopicScore

    for _, mapping := range input.CandidateMappings {
        topicLimitCount, isActive := allTopicActivity[mapping.TopicID]

        // 过滤：仅保留今日有涨停股的活跃热点
        if !isActive || topicLimitCount == 0 {
            continue
        }

        // S1: 活跃度
        s1 := CalcActivityScore(topicLimitCount, maxLimitCount)

        // S2: 关联强度（概念模式跳过）
        s2 := 0.0
        if input.WeightMode == WeightModeNormal && totalHitCount > 0 {
            s2 = CalcBindStrengthScore(mapping.HitCount, totalHitCount)
        }

        // S3: 时间接近度
        topicLimitTimes, _ := getTopicLimitTimes(ctx, input.Date, mapping.TopicID)
        s3 := CalcTimeProximityScore(input.QuoteTime, topicLimitTimes)

        // S4: 时效性
        lastSeen, _ := time.Parse("2006-01-02", mapping.LastSeenDate)
        today, _ := time.Parse("2006-01-02", input.Date)
        s4 := CalcRecencyScore(lastSeen, today)

        // 综合评分
        total := CalcTotalScore(s1, s2, s3, s4, weights)

        scores = append(scores, TopicScore{
            TopicID:        mapping.TopicID,
            TopicName:      mapping.TopicName,
            SActivity:      math.Round(s1*10000) / 10000,
            SBindStrength:  math.Round(s2*10000) / 10000,
            STimeProximity: math.Round(s3*10000) / 10000,
            SRecency:       math.Round(s4*10000) / 10000,
            TotalScore:     math.Round(total*10000) / 10000,
            Source:         mapping.Source,
        })
    }

    // ── Step 4: 按总分降序排序 ──
    sort.Slice(scores, func(i, j int) bool {
        return scores[i].TotalScore > scores[j].TotalScore
    })

    // ── Step 5: 归因决策 ──
    result := DecideAttribution(scores)
    result.AllScores = scores
    return result, nil
}
```

#### 2.3.10 边界条件汇总表

| 边界条件 | 处理方式 | 涉及函数 |
|---------|---------|---------|
| maxLimitCountAll == 0 | S1 = 0.0 | CalcActivityScore |
| totalHitCount == 0 | S2 = 0.0 | CalcBindStrengthScore |
| topicLimitTimes 过滤后为空 | S3 = 0.5（中性值） | CalcTimeProximityScore |
| lastSeenDate 为零值 | S4 = 0.1（最低时效性） | CalcRecencyScore |
| 候选热点过滤后为空 | 返回空归因，标记未分类 | DecideAttribution |
| 仅 1 个活跃候选 | 直接归因，无需评分排序 | DecideAttribution |
| Top1 == Top2 得分差 < 0.1 | 双归因 | DecideAttribution |
| 所有候选得分均为 0 | 返回空归因，标记未分类 | DecideAttribution |
| 概念模式下无频次数据 | S2 跳过（权重=0），权重分配给 S1 和 S3 | RunAttribution |

#### 2.3.11 算法效果推演

**场景1: 典型单热点**

> 股票拓维信息涨停。历史标签: AI(42次), 教育(5次), 华为(3次)。
> 今日活跃热点: AI(15只涨停)。教育和华为今天无涨停。
> 过滤后仅 AI 一个候选 → 直接归因 AI。

**场景2: 双热点均活跃**

> 股票科大讯飞涨停。历史标签: AI(47次), 机器人(8次), 教育(3次)。
> 今日: AI 涨停 15 只, 机器人涨停 10 只。

| 信号 | AI | 机器人 |
|------|-----|--------|
| S1 活跃度 | 15/15 = 1.000 | 10/15 = 0.667 |
| S2 关联强度 | 47/58 = 0.810 | 8/58 = 0.138 |
| S3 时间聚集 | 0.500 | 0.300 |
| S4 时效性 | 0.910 | 0.500 |
| **总分** | 0.25*1.0 + 0.40*0.81 + 0.20*0.50 + 0.15*0.91 = **0.8105** | 0.25*0.667 + 0.40*0.138 + 0.20*0.30 + 0.15*0.50 = **0.3572** |

> 差距 0.453 >> 0.1 → 单一归因 AI。

**场景3: 两个热点难分伯仲**

> 股票汇川技术涨停。历史标签: 机器人(25次), 新能源(22次)。
> 今日: 机器人涨停 10 只, 新能源涨停 8 只。

| 信号 | 机器人 | 新能源 |
|------|--------|--------|
| 总分 | 0.681 | 0.589 |

> 差距 0.092 < 0.1 → 双归因: 机器人 + 新能源。

---

### 2.4 策略告警引擎（Strategy Alert Engine）

#### 2.4.1 告警输入/输出

```go
// AlertCheckInput 策略告警检查输入
type AlertCheckInput struct {
    TsCode         string         `json:"ts_code"`
    StockName      string         `json:"stock_name"`
    TopicIDs       []int64        `json:"topic_ids"`         // 归因热点ID列表（可能 1~2 个）
    TopicNames     []string       `json:"topic_names"`       // 归因热点名称列表
    TriggerPrice   float64        `json:"trigger_price"`     // 涨停触发价格
    TriggerTime    time.Time      `json:"trigger_time"`      // 首次涨停时间
    Date           string         `json:"date"`              // 交易日 YYYY-MM-DD
    ChangePct      float64        `json:"change_pct"`        // 今日涨跌幅
    PrevDayPct     float64        `json:"prev_day_pct"`      // 昨日涨跌幅
    PrevDayLimitUp bool           `json:"prev_day_limit_up"` // 昨日是否涨停
    PrevDayAbove5  bool           `json:"prev_day_above5"`   // 昨日是否在5%池
    MarketSnapshot MarketSnapshot `json:"market_snapshot"`   // 触发瞬间行情快照
}

// MarketSnapshot 行情快照（保存到 strategy_alerts.extra_info JSONB）
type MarketSnapshot struct {
    CurrentPrice float64    `json:"current_price"`
    Volume       float64    `json:"volume"`        // 成交量
    Amount       float64    `json:"amount"`        // 成交额
    TurnoverRate float64    `json:"turnover_rate"` // 换手率
    BidPrices    [5]float64 `json:"bid_prices"`    // 买五档价
    BidVolumes   [5]float64 `json:"bid_volumes"`   // 买五档量
    AskPrices    [5]float64 `json:"ask_prices"`    // 卖五档价
    AskVolumes   [5]float64 `json:"ask_volumes"`   // 卖五档量
}

// AlertCheckOutput 策略告警检查输出
type AlertCheckOutput struct {
    ShouldAlert    bool   `json:"should_alert"`
    AlertTopicID   int64  `json:"alert_topic_id"`   // 触发告警的热点ID
    AlertTopicName string `json:"alert_topic_name"` // 触发告警的热点名称
    RejectReason   string `json:"reject_reason"`    // 未触发时的拒绝原因
    Message        string `json:"message"`          // 告警文案
}

// CheckStrategyAlert 策略告警判定函数
//
// 5 个条件必须全部为 true 才触发告警。
// 任何一个条件为 false 即短路返回 ShouldAlert=false + RejectReason。
func CheckStrategyAlert(ctx context.Context, input AlertCheckInput) (AlertCheckOutput, error)
```

#### 2.4.2 五个触发条件（AND 逻辑，短路求值）

```
RULE-ALERT-C1 [热点已关注]:
    该股票归因热点中，至少有一个在 daily_focus_topics 中被勾选。
    检查方式: Redis SISMEMBER focus:topics:{date} {topicID}
    对 input.TopicIDs 逐个检查，命中第一个即停。
    双归因时，任一热点命中即满足。
    未满足 → RejectReason = "topic_not_focused"

RULE-ALERT-C2 [昨日在5%池]:
    该股票昨日收盘时在 5% 池中（涨幅 >= 5% 但未涨停）。
    检查方式: input.PrevDayAbove5 == true
    数据来源: daily_stock_pool WHERE date=yesterday AND ts_code=$1 AND pool_type=2
    未满足 → RejectReason = "not_in_prev_day_5pct_pool"

RULE-ALERT-C3 [昨日未涨停]:
    该股票昨日不在涨停池中（即昨日只是强势股而非涨停股）。
    检查方式: input.PrevDayLimitUp == false
    数据来源: daily_stock_pool WHERE date=yesterday AND ts_code=$1 AND pool_type=1（不存在即未涨停）
    未满足 → RejectReason = "was_limit_up_prev_day"

RULE-ALERT-C4 [交易时段内]:
    首次涨停时间在交易时段范围内: 09:25 <= TriggerTime <= 15:00
    检查方式: 提取 TriggerTime 的 HH:MM，转换为分钟数与区间比较。
    未满足 → RejectReason = "outside_trading_hours"

RULE-ALERT-C5 [今日未发送过（幂等）]:
    同一股票+同一热点，当日只发一次告警。
    检查方式: Redis SISMEMBER alerted:{date} "{ts_code}:{topic_id}"
    若已存在 → RejectReason = "already_alerted_today"
    触发后立即: Redis SADD alerted:{date} "{ts_code}:{topic_id}"
```

#### 2.4.3 告警判定完整伪代码

```go
func CheckStrategyAlert(ctx context.Context, input AlertCheckInput) (AlertCheckOutput, error) {
    // ── C1: 热点是否在关注列表 ──
    focusedTopicID := int64(0)
    focusedTopicName := ""
    for i, tid := range input.TopicIDs {
        isFocused, _ := redisSIsMember(ctx, "focus:topics:"+input.Date, tid)
        if isFocused {
            focusedTopicID = tid
            focusedTopicName = input.TopicNames[i]
            break
        }
    }
    if focusedTopicID == 0 {
        return AlertCheckOutput{
            ShouldAlert:  false,
            RejectReason: "topic_not_focused",
        }, nil
    }

    // ── C2: 昨日在5%池 ──
    if !input.PrevDayAbove5 {
        return AlertCheckOutput{
            ShouldAlert:  false,
            RejectReason: "not_in_prev_day_5pct_pool",
        }, nil
    }

    // ── C3: 昨日未涨停 ──
    if input.PrevDayLimitUp {
        return AlertCheckOutput{
            ShouldAlert:  false,
            RejectReason: "was_limit_up_prev_day",
        }, nil
    }

    // ── C4: 交易时段内 ──
    hour, min := input.TriggerTime.Hour(), input.TriggerTime.Minute()
    tradingStartMin := 9*60 + 25  // 09:25
    tradingEndMin   := 15*60 + 0  // 15:00
    triggerMin := hour*60 + min
    if triggerMin < tradingStartMin || triggerMin > tradingEndMin {
        return AlertCheckOutput{
            ShouldAlert:  false,
            RejectReason: "outside_trading_hours",
        }, nil
    }

    // ── C5: 幂等去重 ──
    alertKey := fmt.Sprintf("%s:%d", input.TsCode, focusedTopicID)
    alreadyAlerted, _ := redisSIsMember(ctx, "alerted:"+input.Date, alertKey)
    if alreadyAlerted {
        return AlertCheckOutput{
            ShouldAlert:  false,
            RejectReason: "already_alerted_today",
        }, nil
    }

    // ── 全部通过: 触发告警 ──
    redisSAdd(ctx, "alerted:"+input.Date, alertKey)

    message := fmt.Sprintf("%s 首次涨停！属于今日关注热点【%s】，昨日涨幅 %.1f%%",
        input.StockName, focusedTopicName, input.PrevDayPct)

    return AlertCheckOutput{
        ShouldAlert:    true,
        AlertTopicID:   focusedTopicID,
        AlertTopicName: focusedTopicName,
        Message:        message,
    }, nil
}
```

```
RULE-ALERT-PUSH: 告警触发后立即通过 WebSocket Hub 推送 strategy_alert 消息类型，不等 10s 周期。
RULE-ALERT-PERSIST: 同时写入 strategy_alerts 表，extra_info 字段保存 MarketSnapshot JSON。
RULE-ALERT-DUAL: 双归因时，C1 检查对两个热点分别 SISMEMBER，命中第一个即停止遍历。
RULE-ALERT-ORDER: 五个条件必须按 C1→C2→C3→C4→C5 顺序短路求值，减少不必要的 Redis 查询。
```

---

### 2.5 数据源优先级（Data Source Priority）

#### 2.5.1 优先级矩阵

```go
// DataSource 数据来源枚举
type DataSource string

const (
    SourceManual  DataSource = "manual"  // P0: 用户手动标注
    SourceJiuyan  DataSource = "jiuyan"  // P1: 韭研公社
    SourceTushare DataSource = "tushare" // P2: Tushare 概念板块
    SourceLLM     DataSource = "llm"     // P3: LLM 异步分类
)

// SourcePriority 数据源优先级映射（数字越小优先级越高）
var SourcePriority = map[DataSource]int{
    SourceManual:  0, // 最高，不可被自动来源覆盖
    SourceJiuyan:  1,
    SourceTushare: 2,
    SourceLLM:     3, // 最低
}

// IsHigherPriority 判断 a 是否比 b 优先级更高
func IsHigherPriority(a, b DataSource) bool {
    return SourcePriority[a] < SourcePriority[b]
}
```

| 优先级 | 来源 | 说明 | 可被覆盖 |
|-------|------|------|---------|
| P0 | manual | 用户手动标注 | 否（仅人工可通过 API 删除） |
| P1 | jiuyan | 韭研公社历史数据 | 可被 P0 覆盖 |
| P2 | tushare | Tushare 概念板块 | 可被 P0/P1 覆盖 |
| P3 | llm | LLM 异步分类结果 | 可被 P0/P1/P2 覆盖 |

```
RULE-PRIORITY-MANUAL: source='manual' 的映射直接采信，跳过归因算法，不可被任何自动来源覆盖。
RULE-PRIORITY-OVERRIDE: 同一 (ts_code, topic_id)，高优先级来源可更新 confidence 和 hit_count 字段。
RULE-PRIORITY-COEXIST: 同一股票可同时拥有不同来源的不同热点映射（不同 topic_id），它们共存参与归因。
RULE-PRIORITY-DELETE: 只有 source='manual' 的映射支持 API 手动删除；自动来源的映射只能通过数据过期自然淘汰。
```

---

### 2.6 同义词归一化（Synonym Normalization）

#### 2.6.1 函数签名

```go
// SynonymEntry 同义词条目
type SynonymEntry struct {
    Synonym   string `json:"synonym"`    // 同义词文本
    TopicID   int64  `json:"topic_id"`   // 归一到的主热点 ID
    TopicName string `json:"topic_name"` // 主热点名称
    Source    string `json:"source"`     // 来源: jiuyan/manual/auto
}

// NormalizeSynonymInput 同义词归一化输入
type NormalizeSynonymInput struct {
    RawTopicName string `json:"raw_topic_name"` // 原始热点名称（可能是别名）
}

// NormalizeSynonymOutput 同义词归一化输出
type NormalizeSynonymOutput struct {
    NormalizedTopicID   int64  `json:"normalized_topic_id"`   // 归一后的主热点 ID（0 = 未命中）
    NormalizedTopicName string `json:"normalized_topic_name"` // 归一后的主热点名称
    WasSynonym          bool   `json:"was_synonym"`           // 是否经过了同义词转换
    OriginalName        string `json:"original_name"`         // 原始输入名称
}

// NormalizeSynonym 同义词归一化
//
// 查找顺序：
//   1. 精确匹配 hot_topics.name → 直接返回该热点 ID，WasSynonym=false
//   2. 精确匹配 topic_synonyms.synonym → 返回归并目标 topic_id，WasSynonym=true
//   3. 均未命中 → NormalizedTopicID = 0，调用方决定是否自动创建新热点
func NormalizeSynonym(ctx context.Context, input NormalizeSynonymInput) (NormalizeSynonymOutput, error)
```

#### 2.6.2 归一化规则

```
RULE-SYN-EXACT: 优先精确匹配 hot_topics.name（主热点名称），此时不是同义词转换。
RULE-SYN-ALIAS: 主热点未命中时查 topic_synonyms.synonym 表，命中时 WasSynonym=true。
RULE-SYN-MISS: 两表均未命中 → NormalizedTopicID = 0，调用方（如韭研爬虫）可选择：
    a. 自动创建新热点（source=jiuyan）
    b. 标记为待人工审核
RULE-SYN-TRIM: 匹配前对输入做 strings.TrimSpace()，忽略首尾空格。
RULE-SYN-CASE: 区分大小写（中文场景无影响；英文如 "AI" 与 "ai" 视为不同）。
RULE-SYN-CACHE: 同义词表启动时全量加载到内存 map[string]int64（synonym → topicID），变更时重新加载。
RULE-SYN-MERGE: 热点合并操作（PUT /api/v1/topics/:id/merge）会自动：
    1. 将被合并热点的名称加入目标热点的同义词列表
    2. 将被合并热点的所有同义词迁移到目标热点
    3. 将被合并热点关联的所有 stock_topic_mappings 迁移到目标热点
```

**示例**: "英伟达概念" → topic_synonyms 查到 synonym="英伟达概念", topic_id=42 → 归一到 "AI算力" (id=42)。

---

### 2.7 代码格式转换（Code Format Conversion）

#### 2.7.1 三种格式定义

| 格式 | 示例 | 使用场景 |
|-----|------|---------|
| Tushare 格式 | `000001.SZ` | 系统内部标准格式、数据库存储、Tushare API |
| 标准6位 | `000001` | 部分 API 通信、简化展示 |
| 韭研格式 | `sz000001` | 韭研公社 API 接口 |

#### 2.7.2 函数签名

```go
// CodeFormat 代码格式枚举
type CodeFormat int

const (
    FormatTushare CodeFormat = 0 // "000001.SZ"
    FormatPlain6  CodeFormat = 1 // "000001"
    FormatJiuyan  CodeFormat = 2 // "sz000001"
)

// ConvertCodeFormatInput 代码转换输入
type ConvertCodeFormatInput struct {
    Code       string     `json:"code"`        // 原始代码
    FromFormat CodeFormat `json:"from_format"` // 原始格式
    ToFormat   CodeFormat `json:"to_format"`   // 目标格式
}

// ConvertCodeFormatOutput 代码转换输出
type ConvertCodeFormatOutput struct {
    ConvertedCode string `json:"converted_code"` // 转换后的代码
    Market        string `json:"market"`         // 市场标识: SZ/SH/BJ
}

// ConvertCodeFormat 股票代码格式转换
func ConvertCodeFormat(input ConvertCodeFormatInput) (ConvertCodeFormatOutput, error)
```

#### 2.7.3 转换规则

```
RULE-CODE-TUSHARE-TO-JIUYAN:
    "000001.SZ" → "sz000001"
    "600000.SH" → "sh600000"
    "688001.SH" → "sh688001"
    "830799.BJ" → "bj830799"
    逻辑: suffix = strings.ToLower(code[7:]) + symbol = code[:6]

RULE-CODE-JIUYAN-TO-TUSHARE:
    "sz000001" → "000001.SZ"
    "sh600000" → "600000.SH"
    "bj830799" → "830799.BJ"
    逻辑: symbol = code[2:] + "." + strings.ToUpper(code[:2])

RULE-CODE-MARKET-DETECT (从标准6位推断市场):
    symbol[0] == '6'         → SH（沪市）
    symbol[0] == '0' || '3'  → SZ（深市）
    symbol[0] == '8' || '4'  → BJ（北交所）
    其他                     → 返回 error

RULE-CODE-VALIDATION:
    Tushare 格式: 必须匹配正则 ^[0-9]{6}\.(SZ|SH|BJ)$
    Plain6 格式: 必须匹配正则 ^[0-9]{6}$
    韭研格式: 必须匹配正则 ^(sz|sh|bj)[0-9]{6}$
    不匹配 → 返回 error

RULE-CODE-INTERNAL: 系统内部统一使用 Tushare 格式作为 primary key，
    仅在调用韭研 API 时通过 ConvertCodeFormat 转换为韭研格式。
```

---

### 2.8 并发模型（Concurrency Model）

#### 2.8.1 分片策略

```go
// ShardConfig 分片配置
type ShardConfig struct {
    ShardCount int `json:"shard_count"` // 分片数量，默认 16
}

// AssignShard 根据股票代码计算分片归属
//   shardID = fnv32a(tsCode) % shardCount
func AssignShard(tsCode string, shardCount int) int

// ShardBatch 单个分片的股票批次
type ShardBatch struct {
    ShardID int      `json:"shard_id"`
    TsCodes []string `json:"ts_codes"`
}

// BuildShardBatches 将全市场股票分配到各分片
func BuildShardBatches(allTsCodes []string, shardCount int) []ShardBatch
```

```
RULE-SHARD-HASH: shardID = fnv32a(tsCode) % shardCount，使用 hash/fnv.New32a()。
RULE-SHARD-DETERMINISTIC: 同一 tsCode 永远分配到同一 shard，天然避免并发写入冲突。
RULE-SHARD-COUNT: 默认 16 个分片（config.monitor.shard_count），可配置。
RULE-SHARD-SERIAL: 分片内串行执行全流程（行情拉取→涨停判定→分类归因→池子更新），无需加锁。
RULE-SHARD-PARALLEL: 所有分片通过 errgroup.WithContext 并行执行。
```

#### 2.8.2 并发执行伪代码

```go
func (m *MonitorService) ProcessTick(ctx context.Context, date string) error {
    // Step 1: 获取所有活跃股票
    allStocks, err := m.stockRepo.GetActiveStocks(ctx)
    if err != nil {
        return fmt.Errorf("get active stocks: %w", err)
    }

    // Step 2: 构建分片
    batches := BuildShardBatches(extractTsCodes(allStocks), m.cfg.ShardCount)

    // Step 3: 并行处理所有分片
    g, gCtx := errgroup.WithContext(ctx)

    for _, batch := range batches {
        batch := batch // capture loop variable for goroutine
        g.Go(func() error {
            return m.processShard(gCtx, date, batch)
        })
    }

    // Step 4: 等待所有分片完成
    if err := g.Wait(); err != nil {
        // 部分 shard 失败不影响整体——记录错误继续
        log.Error("shard processing partial failure", zap.Error(err))
    }

    // Step 5: 所有 shard 完毕后，计算 diff 并推送 WebSocket
    m.computeAndPushDiff(ctx, date)
    return nil
}

func (m *MonitorService) processShard(ctx context.Context, date string, batch ShardBatch) error {
    for _, tsCode := range batch.TsCodes {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        // 串行处理分片内每只股票
        quote, err := m.fetchQuoteWithRetry(ctx, tsCode)
        if err != nil {
            // 个股级隔离：记录错误，继续处理下一只
            m.recordStockError(ctx, date, tsCode, err)
            continue
        }

        // 涨停/5% 判定
        detectOutput, err := DetectLimitUp(buildDetectInput(quote))
        if err != nil {
            log.Warn("detect limit up error", zap.String("ts_code", tsCode), zap.Error(err))
            continue
        }

        if detectOutput.Skipped {
            continue // ST 等跳过
        }

        // 更新 Redis 池子
        m.updatePools(ctx, date, tsCode, detectOutput)

        // 首次涨停 → 触发分类 + 告警
        if detectOutput.IsFirstLimitUp {
            classifyOutput, err := ClassifyStock(ctx, buildClassifyInput(quote, date))
            if err != nil {
                log.Warn("classify error", zap.String("ts_code", tsCode), zap.Error(err))
                continue
            }

            // 记录分类证据（无论是否分类成功）
            m.saveClassifyEvidence(ctx, date, tsCode, classifyOutput)

            // 有归因结果且非未分类 → 检查告警
            if !classifyOutput.IsUnclassified {
                alertOutput, _ := CheckStrategyAlert(ctx, buildAlertInput(quote, classifyOutput, date))
                if alertOutput.ShouldAlert {
                    m.saveAndPushAlert(ctx, quote, classifyOutput, alertOutput)
                }
            }
        }
    }
    return nil
}
```

```
RULE-ERRGROUP: 使用 golang.org/x/sync/errgroup 管理并发，Wait() 聚合所有 shard 错误。
RULE-PARTIAL-FAILURE: 单个 shard 返回 error 不影响其他 shard 的处理。
RULE-STOCK-ISOLATION: 单只股票行情获取/判定/分类失败时 continue，不中断同 shard 其他股票。
RULE-DIFF-AFTER-ALL: 所有 shard 处理完毕后才计算 diff 并推送 WebSocket——保证推送数据的一致性。
RULE-CONTEXT-CHECK: 每只股票处理前检查 ctx.Done()，支持优雅退出。
```

---

### 2.9 外部 API 容错（External API Fault Tolerance）

#### 2.9.1 指数退避重试

```go
// RetryConfig 重试配置
type RetryConfig struct {
    MaxRetries   int           `json:"max_retries"`    // 最大重试次数，默认 4
    InitialDelay time.Duration `json:"initial_delay"`  // 首次重试延迟，默认 1s
    Multiplier   float64       `json:"multiplier"`     // 退避乘数，默认 2.0
    MaxDelay     time.Duration `json:"max_delay"`      // 最大延迟上限，默认 8s
}

// DefaultRetryConfig 默认重试配置
var DefaultRetryConfig = RetryConfig{
    MaxRetries:   4,
    InitialDelay: 1 * time.Second,
    Multiplier:   2.0,
    MaxDelay:     8 * time.Second,
}

// RetryableFunc 可重试的函数类型
type RetryableFunc func(ctx context.Context) error

// RetryWithBackoff 指数退避重试中间件
//
// 退避序列: 1s -> 2s -> 4s -> 8s（第5次不再重试）
// 总最大等待时间: 1+2+4+8 = 15s
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, fn RetryableFunc) error
```

```
RULE-RETRY-SEQUENCE: delay(n) = min(InitialDelay * Multiplier^n, MaxDelay)
    n=0: 1s, n=1: 2s, n=2: 4s, n=3: 8s
RULE-RETRY-MAX: 最多重试 4 次（首次调用 + 4 次重试 = 共 5 次尝试）。
RULE-RETRY-JITTER: 建议加入 +/- 10% 随机抖动避免雷群效应（可选优化）。
RULE-RETRY-CONTEXT: 每次重试前检查 ctx.Done()，上下文取消时立即返回 ctx.Err()。
RULE-RETRY-NON-RETRYABLE: HTTP 4xx 错误（如 401/403/404）不重试，立即返回；仅 5xx 和网络超时错误触发重试。
```

#### 2.9.2 指数退避完整伪代码

```go
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, fn RetryableFunc) error {
    var lastErr error
    delay := cfg.InitialDelay

    for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
        // 非首次尝试，等待退避延迟
        if attempt > 0 {
            select {
            case <-ctx.Done():
                return fmt.Errorf("retry cancelled after %d attempts: %w", attempt, ctx.Err())
            case <-time.After(delay):
            }
            // 计算下一次退避延迟
            delay = time.Duration(float64(delay) * cfg.Multiplier)
            if delay > cfg.MaxDelay {
                delay = cfg.MaxDelay
            }
        }

        lastErr = fn(ctx)
        if lastErr == nil {
            return nil // 成功
        }

        // 不可重试的错误立即返回
        if isNonRetryable(lastErr) {
            return fmt.Errorf("non-retryable error on attempt %d: %w", attempt+1, lastErr)
        }

        log.Warn("retry attempt failed",
            zap.Int("attempt", attempt+1),
            zap.Duration("next_delay", delay),
            zap.Error(lastErr),
        )
    }

    return fmt.Errorf("all %d retries exhausted: %w", cfg.MaxRetries+1, lastErr)
}

// HTTPError 带状态码的 HTTP 错误
type HTTPError struct {
    StatusCode int
    Message    string
}

func (e *HTTPError) Error() string {
    return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

func isNonRetryable(err error) bool {
    var httpErr *HTTPError
    if errors.As(err, &httpErr) {
        return httpErr.StatusCode >= 400 && httpErr.StatusCode < 500
    }
    return false
}
```

#### 2.9.3 熔断逻辑

```go
// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
    ErrorThresholdPct float64       `json:"error_threshold_pct"` // 错误率阈值，默认 30%
    WindowSize        int           `json:"window_size"`         // 滑动窗口轮次数，默认 3 轮
    CooldownPeriod    time.Duration `json:"cooldown_period"`     // 熔断冷却时间，默认 60s
}

// CircuitState 熔断器状态
type CircuitState int

const (
    CircuitClosed   CircuitState = 0 // 正常（关闭状态，请求正常通过）
    CircuitOpen     CircuitState = 1 // 熔断（开启状态，拒绝请求）
    CircuitHalfOpen CircuitState = 2 // 半开（试探恢复，允许少量请求）
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
    Config       CircuitBreakerConfig `json:"config"`
    State        CircuitState         `json:"state"`
    FailureCount int                  `json:"failure_count"`  // 当前窗口失败数
    TotalCount   int                  `json:"total_count"`    // 当前窗口总请求数
    LastTripped  time.Time            `json:"last_tripped"`   // 最后一次触发熔断时间
}

// ShouldAllow 判断是否允许请求通过
func (cb *CircuitBreaker) ShouldAllow() bool

// RecordResult 记录请求结果
func (cb *CircuitBreaker) RecordResult(success bool)
```

```
RULE-BREAKER-TRIGGER: 连续 3 轮（每轮10s，即30s窗口）内行情获取失败率 > 30% → State 从 Closed 切换到 Open。
RULE-BREAKER-OPEN: 熔断开启后，跳过行情获取，使用上一轮 Redis 缓存数据，冷却 60s。
RULE-BREAKER-HALFOPEN: 冷却期结束后切换到 HalfOpen，允许 10% 的请求试探。
RULE-BREAKER-RECOVER: HalfOpen 状态下试探成功率 > 70% → 恢复 Closed；否则重新 Open。
RULE-BREAKER-ALERT: 任何 Closed→Open 状态转换触发系统告警（日志 + 预留飞书 Webhook）。
```

#### 2.9.4 个股级容错

```go
// StockErrorTracker 个股错误追踪
type StockErrorTracker struct {
    MaxConsecutiveErrors int `json:"max_consecutive_errors"` // 连续失败阈值，默认 5
}

// RecordStockError 记录个股错误
//   Redis key: stock:error_count:{date}:{ts_code}
//   操作: INCR，检查是否超过阈值
func (t *StockErrorTracker) RecordStockError(ctx context.Context, date string, tsCode string) (count int, exceeded bool)

// ResetStockError 重置个股错误计数（成功获取行情时调用）
//   Redis key: stock:error_count:{date}:{ts_code}
//   操作: DEL
func (t *StockErrorTracker) ResetStockError(ctx context.Context, date string, tsCode string)
```

```
RULE-STOCK-ERROR-COUNT: 每次获取失败执行 INCR stock:error_count:{date}:{ts_code}。
RULE-STOCK-ERROR-RESET: 获取成功时执行 DEL stock:error_count:{date}:{ts_code}。
RULE-STOCK-ERROR-THRESHOLD: 计数 >= 5 → 记录 WARN 日志，该股票标记为异常。
RULE-STOCK-ERROR-ISOLATION: 异常标记不影响同 shard 内其他股票——个股级隔离。
RULE-STOCK-ERROR-TTL: Redis key 设置当日过期（次日自动清理）。
```

---

### 附录 A: Redis Key 引用表

> 完整 Redis key 定义、数据类型和 TTL 详见 Part 2 数据层文档。
> 此处仅列出业务规则章节引用的 key 及其关联规则。

| Redis Key 模式 | 类型 | 引用规则章节 | 读/写 |
|---------------|------|-------------|-------|
| `cache:stock_topics` | Hash | 2.2.3 L1 分类缓存 | R+W |
| `cache:bind_strength` | Hash | 2.3.4 S2 关联强度 | R |
| `cache:stock_concepts` | Hash | 2.2.5 L3 概念标签 | R |
| `cache:concept_to_topics` | Hash | 2.2.5 L3 概念→热点映射 | R |
| `first_limit:{date}` | Hash | 2.1.4 涨停状态机 | R+W |
| `pool:limit_up:{date}` | Set | 2.8.2 池子更新 | W |
| `pool:above5:{date}` | Set | 2.8.2 池子更新 | W |
| `focus:topics:{date}` | Set | 2.4.2 C1 关注热点判定 | R |
| `alerted:{date}` | Set | 2.4.2 C5 幂等去重 | R+W |
| `topic:activity:limit:{date}` | SortedSet | 2.3.3 S1 活跃度 | R+W |
| `topic:limit_times:{date}:{topic_id}` | List | 2.3.5 S3 时间接近度 | R+W |
| `stock:error_count:{date}:{ts_code}` | String | 2.9.4 个股容错 | R+W |
| `rt:quote:{ts_code}` | Hash | 2.8.2 行情缓存 | W |
| `snapshot:current` | String | 2.8.2 WebSocket 推送 | W |

---

### 附录 B: 端到端决策流程图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        10s Ticker 触发新一轮处理                              │
└─────────────┬───────────────────────────────────────────────────────────────┘
              │
              v
┌─────────────────────────────────┐
│  BuildShardBatches()             │
│  将 ~5000 只股票分成 16 个 shard  │
└─────────────┬───────────────────┘
              │
              v
┌──────────────────────────────────────────────────────────────────────┐
│  errgroup: 16 个 goroutine 并行                                       │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │  processShard(shardID)  -- 分片内串行                             │ │
│  │                                                                 │ │
│  │  FOR each tsCode in shard:                                      │ │
│  │    |                                                            │ │
│  │    +-- fetchQuoteWithRetry(tsCode)                              │ │
│  │    |    +-- 成功 -> 继续                                         │ │
│  │    |    +-- 失败 -> RetryWithBackoff (1s->2s->4s->8s)           │ │
│  │    |         +-- 重试成功 -> 继续                                 │ │
│  │    |         +-- 重试耗尽 -> recordStockError, CONTINUE          │ │
│  │    |                                                            │ │
│  │    +-- DetectLimitUp(quote)                                     │ │
│  │    |    +-- IsST=true -> SKIP                                   │ │
│  │    |    +-- PreClose=0 -> ERROR, SKIP                           │ │
│  │    |    +-- IsLimitUp=true -> SADD pool:limit_up:{date}         │ │
│  │    |    +-- IsAbove5Pct=true && !IsLimitUp -> SADD pool:above5  │ │
│  │    |    +-- IsFirstLimitUp=true -> 触发分类链路 (下方)            │ │
│  │    |                                                            │ │
│  │    +-- [仅首次涨停] ClassifyStock(tsCode)                        │ │
│  │    |    +-- L1 Redis: HGET cache:stock_topics {tsCode}          │ │
│  │    |    |    +-- 有 manual -> 直接采信, STOP                      │ │
│  │    |    |    +-- 有映射 -> RunAttribution(Normal权重)             │ │
│  │    |    +-- L2 PG Jiuyan: SELECT stock_topic_mappings           │ │
│  │    |    |    +-- 命中 -> 回填Redis + RunAttribution(Normal)      │ │
│  │    |    +-- L3 PG Concept: SELECT stock_concept_tags            │ │
│  │    |    |    +-- 命中 -> RunAttribution(Concept权重)             │ │
│  │    |    +-- L4 LLM: go asyncLLMClassify() -> 当前轮标记未分类    │ │
│  │    |                                                            │ │
│  │    +-- RunAttribution(candidates, weights)                      │ │
│  │    |    +-- 过滤: 仅保留今日有涨停股的活跃热点                     │ │
│  │    |    +-- 计算 S1(活跃度) + S2(关联强度) + S3(时间接近度)       │ │
│  │    |    |    + S4(时效性)                                        │ │
│  │    |    +-- TotalScore = W1*S1 + W2*S2 + W3*S3 + W4*S4         │ │
│  │    |    +-- DecideAttribution:                                  │ │
│  │    |         +-- 0 个活跃候选 -> 未分类                           │ │
│  │    |         +-- 1 个候选 -> 直接归因                             │ │
│  │    |         +-- >=2 个 -> Top1-Top2 < 0.1 ? 双归因 : 单归因    │ │
│  │    |                                                            │ │
│  │    +-- saveClassifyEvidence()  (无论分类成功与否)                  │ │
│  │    |                                                            │ │
│  │    +-- [有归因结果] CheckStrategyAlert()                         │ │
│  │         +-- C1: SISMEMBER focus:topics:{date} {topicID}        │ │
│  │         +-- C2: input.PrevDayAbove5 == true ?                   │ │
│  │         +-- C3: input.PrevDayLimitUp == false ?                 │ │
│  │         +-- C4: 09:25 <= TriggerTime <= 15:00 ?                │ │
│  │         +-- C5: SISMEMBER alerted:{date} {key} == false ?      │ │
│  │         +-- 全部 true -> 触发告警:                               │ │
│  │              +-- INSERT strategy_alerts (含 MarketSnapshot)     │ │
│  │              +-- WebSocket Hub 即时推送 strategy_alert           │ │
│  │              +-- SADD alerted:{date} "{ts_code}:{topic_id}"    │ │
│  │                                                                 │ │
│  └─────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────┬───────────────────────────────────┘
                                   │
                                   v
                    ┌──────────────────────────────┐
                    │  g.Wait() -- 等待所有 shard    │
                    └──────────────┬───────────────┘
                                   │
                                   v
                    ┌──────────────────────────────┐
                    │  computeAndPushDiff()         │
                    │  +-- diff 为空 -> 不推送       │
                    │  +-- diff < 30% -> pool_diff  │
                    │  +-- diff >= 30% -> pool_snap  │
                    └──────────────────────────────┘
```

---

### 附录 C: 分类证据记录规范

每次 `ClassifyStock` 调用后必须写入 `classify_evidence` 表。

```go
// EvidenceRecord 分类证据记录（对应 classify_evidence 表）
type EvidenceRecord struct {
    Date            string           `json:"date"`              // 交易日 YYYY-MM-DD
    TsCode          string           `json:"ts_code"`           // 股票代码
    TopicID         int64            `json:"topic_id"`          // 最终归因热点 ID（未分类时为 0）
    ClassifyLayer   ClassifyLayer    `json:"classify_layer"`    // L1_REDIS / L2_PG_JIUYAN / L3_PG_CONCEPT / L4_LLM
    Strategy        ClassifyStrategy `json:"strategy"`          // MANUAL / JIUYAN_ATTRIBUTION / CONCEPT_ATTRIBUTION / LLM_V1
    CandidateScores []TopicScore     `json:"candidate_scores"`  // 所有候选热点评分 JSON（含被淘汰的）
    EvidenceText    string           `json:"evidence_text"`     // 关键证据（LLM 原文 / 匹配关键词 / 手动标注说明）
    Confidence      float64          `json:"confidence"`        // 置信度 0~1
}
```

```
RULE-EVIDENCE-ALWAYS: 每次 ClassifyStock 调用都必须产生一条 classify_evidence 记录，包括 IsUnclassified=true 的情况。
RULE-EVIDENCE-SCORES: CandidateScores 包含所有参与评分的候选热点（含被活跃度过滤掉的），JSON 写入 candidate_scores 列。
RULE-EVIDENCE-LLM: L4 层的 EvidenceText 保存 LLM 返回的完整原始响应文本。
RULE-EVIDENCE-MANUAL: Strategy=MANUAL 时 Confidence 固定为 1.0。
RULE-EVIDENCE-UNCLASSIFIED: 未分类时 TopicID=0, Confidence=0, ClassifyLayer=当前降级到的最后一层。
RULE-EVIDENCE-INDEX: 表上建 idx_ce_stock_date(ts_code, date) 组合索引，支持按股票+日期高效查询。
RULE-EVIDENCE-AUDIT: 证据数据服务于三个目的:
    1. 调试回溯: 发现分类错误时快速定位原因
    2. 权重调优: 利用韭研收盘数据作为 ground truth 回测，优化四维权重
    3. 训练数据: 后期积累 corrected_topic_id 后训练监督学习模型
```

---

## 四、API 契约与 WebSocket 协议

> **阅读对象**：AI Coding Agent / 后端开发者
> **约定**：所有请求/响应均使用 Go struct 定义（含 `json` tag + `binding` validator），不使用表格描述字段。
> 所有 endpoint 统一前缀 `/api/v1`，鉴权通过请求头 `X-API-Key` 实现。

---

### 4.1 API 分组总览表

| # | Method | Path | Summary | Auth |
|---|--------|------|---------|------|
| 1 | GET | `/api/v1/pool/limit-up` | 实时涨停池（按热点分组） | API Key |
| 2 | GET | `/api/v1/pool/above5` | 实时5%池（按热点分组） | API Key |
| 3 | GET | `/api/v1/pool/limit-up/snapshot` | 涨停池当前全量快照 | API Key |
| 4 | GET | `/api/v1/pool/history` | 历史某日池子查询 | API Key |
| 5 | GET | `/api/v1/topics` | 热点列表（搜索+分页） | API Key |
| 6 | POST | `/api/v1/topics` | 新增热点 | API Key |
| 7 | PUT | `/api/v1/topics/:id` | 修改热点 | API Key |
| 8 | DELETE | `/api/v1/topics/:id` | 删除热点（软删除） | API Key |
| 9 | PUT | `/api/v1/topics/:id/merge` | 合并热点（迁移映射+同义词） | API Key |
| 10 | GET | `/api/v1/topics/:id/synonyms` | 查询热点同义词 | API Key |
| 11 | POST | `/api/v1/topics/:id/synonyms` | 新增同义词 | API Key |
| 12 | DELETE | `/api/v1/topics/synonyms/:synonym_id` | 删除同义词 | API Key |
| 13 | GET | `/api/v1/topics/:id/concepts` | 查询热点关联的概念映射 | API Key |
| 14 | GET | `/api/v1/stock/:ts_code/topics` | 查询股票的热点归因 | API Key |
| 15 | POST | `/api/v1/stock/:ts_code/topics` | 手动标注股票热点（P0优先级） | API Key |
| 16 | DELETE | `/api/v1/stock/:ts_code/topics/:topic_id` | 移除股票热点标签 | API Key |
| 17 | POST | `/api/v1/stock/:ts_code/reclassify` | 触发股票重新分类 | API Key |
| 18 | GET | `/api/v1/stock/:ts_code/evidence` | 查询分类证据 | API Key |
| 19 | PUT | `/api/v1/stock/:ts_code/evidence/:id/correct` | 人工修正归因 | API Key |
| 20 | GET | `/api/v1/alerts` | 查询今日策略告警 | API Key |
| 21 | GET | `/api/v1/alerts/history` | 历史告警查询 | API Key |
| 22 | GET | `/api/v1/alerts/today` | 今日告警汇总统计 | API Key |
| 23 | GET | `/api/v1/focus` | 查询某日关注热点 | API Key |
| 24 | POST | `/api/v1/focus` | 设置关注热点 | API Key |
| 25 | DELETE | `/api/v1/focus/:topic_id` | 取消关注热点 | API Key |
| 26 | POST | `/api/v1/admin/crawl/history` | 触发韭研历史全量爬取 | API Key |
| 27 | POST | `/api/v1/admin/crawl/today` | 触发今日增量爬取 | API Key |
| 28 | GET | `/api/v1/admin/crawl/logs` | 爬取日志查询 | API Key |
| 29 | POST | `/api/v1/admin/cache/refresh` | 手动刷新 Redis 缓存 | API Key |
| 30 | POST | `/api/v1/admin/concepts/sync` | 触发 Tushare 概念数据同步 | API Key |
| 31 | GET | `/api/v1/admin/concepts/mappings` | 查询概念→热点映射状态 | API Key |
| 32 | POST | `/api/v1/admin/concepts/mappings` | 手动建立概念→热点映射 | API Key |
| 33 | POST | `/api/v1/admin/llm/classify` | 触发 LLM 批量分类 | API Key |
| 34 | GET | `/api/v1/admin/stats` | 系统统计（池子/热点/告警数量） | API Key |
| 35 | GET | `/health` | 健康检查 | None |
| 36 | POST | `/api/v1/admin/init` | 系统初始化（首次部署） | API Key |
| 37 | WS | `/ws` | WebSocket 实时推送 | Query token |

---

### 4.2 各组 API 详细定义

---

#### 4.2.1 实时池（Pool）

##### GET /api/v1/pool/limit-up — 涨停池（按热点分组）

```go
// -------- Request --------
type GetLimitUpPoolReq struct {
    Date      string `json:"date" form:"date" binding:"omitempty,datetime=2006-01-02"`          // 交易日，默认今日
    TopicID   int64  `json:"topic_id" form:"topic_id" binding:"omitempty,min=1"`                // 筛选某热点
    SortBy    string `json:"sort_by" form:"sort_by" binding:"omitempty,oneof=change_pct first_limit_time attribution_score"` // 排序字段
    Order     string `json:"order" form:"order" binding:"omitempty,oneof=asc desc"`             // 排序方向，默认 desc
}

// -------- Response --------
type PoolStockItem struct {
    TsCode           string  `json:"ts_code"`            // 股票代码 e.g. "688256.SH"
    Name             string  `json:"name"`               // 股票名称 e.g. "寒武纪"
    ChangePct        float64 `json:"change_pct"`         // 涨跌幅 %
    CurrentPrice     float64 `json:"current_price"`      // 当前价
    PreClose         float64 `json:"pre_close"`          // 昨收价
    LimitUpPrice     float64 `json:"limit_up_price"`     // 涨停价
    FirstLimitTime   string  `json:"first_limit_time"`   // 首次涨停时间 "HH:MM:SS"
    BoardCode        string  `json:"board_code"`         // 板块 MAIN/GEM/STAR/BSE
    AttributionScore float64 `json:"attribution_score"`  // 归因置信度 0~1
}

type PoolTopicGroup struct {
    TopicID   int64           `json:"topic_id"`    // 热点ID
    TopicName string          `json:"topic_name"`  // 热点名称
    Count     int             `json:"count"`       // 该热点下股票数量
    Stocks    []PoolStockItem `json:"stocks"`      // 股票列表
}

type LimitUpPoolResp struct {
    Date         string           `json:"date"`          // 交易日
    SnapshotTime string           `json:"snapshot_time"` // 快照时间 "HH:MM:SS"
    TotalCount   int              `json:"total_count"`   // 涨停总数
    Groups       []PoolTopicGroup `json:"groups"`        // 按热点分组
    Unclassified []PoolStockItem  `json:"unclassified"`  // 未分类股票
}
```

```bash
# 示例 curl
curl -H "X-API-Key: $KEY" \
  "http://localhost:8080/api/v1/pool/limit-up?date=2026-03-15&sort_by=first_limit_time&order=asc"
```

##### GET /api/v1/pool/above5 — 5%池（按热点分组）

```go
// -------- Request --------
type GetAbove5PoolReq struct {
    Date    string `json:"date" form:"date" binding:"omitempty,datetime=2006-01-02"`
    TopicID int64  `json:"topic_id" form:"topic_id" binding:"omitempty,min=1"`
    SortBy  string `json:"sort_by" form:"sort_by" binding:"omitempty,oneof=change_pct name"`
    Order   string `json:"order" form:"order" binding:"omitempty,oneof=asc desc"`
}

// -------- Response --------
// 复用 PoolTopicGroup / PoolStockItem 结构，TotalCount 统计涨幅>5%的股票
type Above5PoolResp struct {
    Date         string           `json:"date"`
    SnapshotTime string           `json:"snapshot_time"`
    TotalCount   int              `json:"total_count"`
    Groups       []PoolTopicGroup `json:"groups"`
    Unclassified []PoolStockItem  `json:"unclassified"`
}
```

##### GET /api/v1/pool/limit-up/snapshot — 涨停池当前全量快照

```go
// -------- Request --------
// 无参数，返回 Redis snapshot:current 的完整 JSON

// -------- Response --------
type PoolSnapshotResp struct {
    Date         string      `json:"date"`
    SnapshotTime string      `json:"snapshot_time"`
    LimitUp      PoolSection `json:"limit_up"`
    Above5Pct    PoolSection `json:"above_5pct"`
}

type PoolSection struct {
    TotalCount   int              `json:"total_count"`
    Groups       []PoolTopicGroup `json:"groups"`
    Unclassified []PoolStockItem  `json:"unclassified"`
}
```

##### GET /api/v1/pool/history — 历史某日池子查询

```go
// -------- Request --------
type GetPoolHistoryReq struct {
    Date     string `json:"date" form:"date" binding:"required,datetime=2006-01-02"`        // 必填，历史日期
    PoolType int    `json:"pool_type" form:"pool_type" binding:"omitempty,oneof=1 2"`       // 1=涨停 2=5%，不填返回全部
    TopicID  int64  `json:"topic_id" form:"topic_id" binding:"omitempty,min=1"`
    Page     int    `json:"page" form:"page" binding:"omitempty,min=1"`
    PageSize int    `json:"page_size" form:"page_size" binding:"omitempty,min=1,max=200"`
}

// -------- Response --------
type PoolHistoryItem struct {
    TsCode         string   `json:"ts_code"`
    StockName      string   `json:"stock_name"`
    PoolType       int      `json:"pool_type"`        // 1=涨停 2=5%+
    ChangePct      float64  `json:"change_pct"`
    CurrentPrice   float64  `json:"current_price"`
    PreClose       float64  `json:"pre_close"`
    LimitUpPrice   float64  `json:"limit_up_price"`
    FirstLimitTime string   `json:"first_limit_time"`
    BoardCode      string   `json:"board_code"`
    TopicIDs       []int64  `json:"topic_ids"`        // 当日归因热点ID数组
    TopicNames     []string `json:"topic_names"`      // 热点名称（join 查询冗余）
    SnapshotTime   string   `json:"snapshot_time"`
}

type PoolHistoryResp = PaginatedResult[PoolHistoryItem]
```

```bash
# 示例 curl
curl -H "X-API-Key: $KEY" \
  "http://localhost:8080/api/v1/pool/history?date=2026-03-14&pool_type=1&page=1&page_size=50"
```

---

#### 4.2.2 热点管理（Topics）

##### GET /api/v1/topics — 热点列表

```go
// -------- Request --------
type ListTopicsReq struct {
    Keyword  string `json:"keyword" form:"keyword" binding:"omitempty,max=64"`               // 模糊搜索
    Source   string `json:"source" form:"source" binding:"omitempty,oneof=jiuyan manual"`    // 来源筛选
    IsActive *bool  `json:"is_active" form:"is_active"`                                      // 是否启用
    SortBy   string `json:"sort_by" form:"sort_by" binding:"omitempty,oneof=name occurrence_count last_seen_date created_at priority"`
    Order    string `json:"order" form:"order" binding:"omitempty,oneof=asc desc"`
    Page     int    `json:"page" form:"page" binding:"omitempty,min=1"`
    PageSize int    `json:"page_size" form:"page_size" binding:"omitempty,min=1,max=200"`
}

// -------- Response --------
type TopicItem struct {
    ID              int64  `json:"id"`
    Name            string `json:"name"`
    Source          string `json:"source"`            // jiuyan / manual
    JiuyanFieldID   string `json:"jiuyan_field_id"`
    FirstSeenDate   string `json:"first_seen_date"`
    LastSeenDate    string `json:"last_seen_date"`
    OccurrenceCount int    `json:"occurrence_count"`  // 历史出现天数
    Priority        int    `json:"priority"`
    IsActive        bool   `json:"is_active"`
    SynonymCount    int    `json:"synonym_count"`     // 同义词数量（join 统计）
    CreatedAt       string `json:"created_at"`
    UpdatedAt       string `json:"updated_at"`
}

type ListTopicsResp = PaginatedResult[TopicItem]
```

##### POST /api/v1/topics — 新增热点

```go
// -------- Request --------
type CreateTopicReq struct {
    Name     string `json:"name" binding:"required,min=1,max=128"`   // 热点名称，唯一
    Priority int    `json:"priority" binding:"omitempty,min=0"`       // 展示优先级
}

// -------- Response --------
// Response[TopicItem]
// source 自动设为 "manual"
```

```bash
curl -X POST -H "X-API-Key: $KEY" -H "Content-Type: application/json" \
  -d '{"name":"低空经济","priority":10}' \
  http://localhost:8080/api/v1/topics
```

##### PUT /api/v1/topics/:id — 修改热点

```go
// -------- Request --------
type UpdateTopicReq struct {
    ID       int64  `json:"-" uri:"id" binding:"required,min=1"`                       // URL path 参数
    Name     string `json:"name" binding:"omitempty,min=1,max=128"`
    IsActive *bool  `json:"is_active"`                                                  // 启用/停用
    Priority *int   `json:"priority" binding:"omitempty,min=0"`
}

// -------- Response --------
// Response[TopicItem]
```

##### DELETE /api/v1/topics/:id — 删除热点（软删除）

```go
// -------- Request --------
type DeleteTopicReq struct {
    ID int64 `json:"-" uri:"id" binding:"required,min=1"`
}

// -------- Response --------
// Response[nil]  — is_active 设为 false，不物理删除
```

##### PUT /api/v1/topics/:id/merge — 合并热点

```go
// -------- Request --------
type MergeTopicReq struct {
    ID            int64 `json:"-" uri:"id" binding:"required,min=1"`                   // 被合并的热点（将被停用）
    TargetTopicID int64 `json:"target_topic_id" binding:"required,min=1,nefield=ID"`   // 目标热点（保留）
}

// -------- Response --------
type MergeTopicResp struct {
    MigratedMappings int   `json:"migrated_mappings"`  // 迁移的 stock_topic_mappings 数量
    MigratedSynonyms int   `json:"migrated_synonyms"`  // 迁移的同义词数量
    SourceTopicID    int64 `json:"source_topic_id"`    // 被合并的热点ID（已停用）
    TargetTopicID    int64 `json:"target_topic_id"`    // 目标热点ID
}
```

合并逻辑说明（实现时参照）：
1. 将 `source_topic` 下所有 `stock_topic_mappings` 的 `topic_id` 改为 `target_topic_id`（冲突时累加 `hit_count`）
2. 将 `source_topic` 的同义词全部迁移到 `target_topic`
3. 将 `source_topic` 的 `name` 作为新同义词添加到 `target_topic`
4. 将 `source_topic.is_active` 设为 false
5. 刷新 Redis 缓存

##### GET /api/v1/topics/:id/synonyms — 查询同义词

```go
// -------- Request --------
type GetSynonymsReq struct {
    TopicID int64 `json:"-" uri:"id" binding:"required,min=1"`
}

// -------- Response --------
type SynonymItem struct {
    ID        int64  `json:"id"`
    TopicID   int64  `json:"topic_id"`
    Synonym   string `json:"synonym"`
    Source    string  `json:"source"`     // jiuyan / manual / auto
    CreatedAt string `json:"created_at"`
}

// Response[[]SynonymItem]
```

##### POST /api/v1/topics/:id/synonyms — 新增同义词

```go
// -------- Request --------
type CreateSynonymReq struct {
    TopicID int64  `json:"-" uri:"id" binding:"required,min=1"`
    Synonym string `json:"synonym" binding:"required,min=1,max=128"`  // 同义词文本，全局唯一
}

// -------- Response --------
// Response[SynonymItem]
```

##### DELETE /api/v1/topics/synonyms/:synonym_id — 删除同义词

```go
// -------- Request --------
type DeleteSynonymReq struct {
    SynonymID int64 `json:"-" uri:"synonym_id" binding:"required,min=1"`
}

// -------- Response --------
// Response[nil]
```

##### GET /api/v1/topics/:id/concepts — 查询热点关联概念映射

```go
// -------- Request --------
type GetTopicConceptsReq struct {
    TopicID int64 `json:"-" uri:"id" binding:"required,min=1"`
}

// -------- Response --------
type ConceptMappingItem struct {
    ID          int64  `json:"id"`
    ConceptName string `json:"concept_name"`   // Tushare 概念名称
    ConceptCode string `json:"concept_code"`   // Tushare 概念代码
    TopicID     int64  `json:"topic_id"`
    MatchType   string `json:"match_type"`     // exact / synonym / llm / manual
    CreatedAt   string `json:"created_at"`
}

// Response[[]ConceptMappingItem]
```

---

#### 4.2.3 股票分类（Classification）

##### GET /api/v1/stock/:ts_code/topics — 查询股票的热点归因

```go
// -------- Request --------
type GetStockTopicsReq struct {
    TsCode string `json:"-" uri:"ts_code" binding:"required"`   // 路径参数 e.g. "002049.SZ"
}

// -------- Response --------
type StockTopicAttribution struct {
    TopicID          int64   `json:"topic_id"`
    TopicName        string  `json:"topic_name"`
    Source           string  `json:"source"`            // jiuyan / llm / manual / tushare
    HitCount         int     `json:"hit_count"`         // 历史出现次数
    Confidence       float64 `json:"confidence"`        // 置信度
    LastSeenDate     string  `json:"last_seen_date"`
    FirstSeenDate    string  `json:"first_seen_date"`
    AttributionScore float64 `json:"attribution_score"` // 当前归因评分（仅实时有值）
}

// Response[[]StockTopicAttribution]
```

##### POST /api/v1/stock/:ts_code/topics — 手动标注热点（最高优先级 P0）

```go
// -------- Request --------
type ManualClassifyReq struct {
    TsCode  string `json:"-" uri:"ts_code" binding:"required"`
    TopicID int64  `json:"topic_id" binding:"required,min=1"`
}

// -------- Response --------
// Response[nil]
// 写入 stock_topic_mappings，source="manual"
// 分类引擎遇到 source=manual 直接采信，跳过归因算法
```

```bash
curl -X POST -H "X-API-Key: $KEY" -H "Content-Type: application/json" \
  -d '{"topic_id":42}' \
  http://localhost:8080/api/v1/stock/002049.SZ/topics
```

##### DELETE /api/v1/stock/:ts_code/topics/:topic_id — 移除热点标签

```go
// -------- Request --------
type RemoveStockTopicReq struct {
    TsCode  string `json:"-" uri:"ts_code" binding:"required"`
    TopicID int64  `json:"-" uri:"topic_id" binding:"required,min=1"`
}

// -------- Response --------
// Response[nil]
```

##### POST /api/v1/stock/:ts_code/reclassify — 触发重新分类

```go
// -------- Request --------
type ReclassifyReq struct {
    TsCode string `json:"-" uri:"ts_code" binding:"required"`
}

// -------- Response --------
type ReclassifyResp struct {
    TsCode          string                 `json:"ts_code"`
    ClassifyLayer   string                 `json:"classify_layer"`   // L1_REDIS / L2_PG_JIUYAN / L3_PG_CONCEPT / L4_LLM
    Strategy        string                 `json:"strategy"`         // JIUYAN_ATTR / CONCEPT_ATTR / LLM_V1 / MANUAL
    FinalTopicIDs   []int64                `json:"final_topic_ids"`
    CandidateScores []CandidateScoreDetail `json:"candidate_scores"`
    Confidence      float64                `json:"confidence"`
}

type CandidateScoreDetail struct {
    TopicID       int64   `json:"topic_id"`
    TopicName     string  `json:"topic_name"`
    ActivityScore float64 `json:"activity_score"`  // S1 热点活跃度
    BindStrength  float64 `json:"bind_strength"`   // S2 关联强度
    TimeProximity float64 `json:"time_proximity"`  // S3 时间聚集性
    Recency       float64 `json:"recency"`         // S4 映射时效性
    TotalScore    float64 `json:"total_score"`     // 加权总分
}
```

##### GET /api/v1/stock/:ts_code/evidence — 查询分类证据

```go
// -------- Request --------
type GetEvidenceReq struct {
    TsCode   string `json:"-" uri:"ts_code" binding:"required"`
    Date     string `json:"date" form:"date" binding:"omitempty,datetime=2006-01-02"`      // 默认今日
    Page     int    `json:"page" form:"page" binding:"omitempty,min=1"`
    PageSize int    `json:"page_size" form:"page_size" binding:"omitempty,min=1,max=100"`
}

// -------- Response --------
type EvidenceItem struct {
    ID               int64                  `json:"id"`
    Date             string                 `json:"date"`
    TsCode           string                 `json:"ts_code"`
    TopicID          int64                  `json:"topic_id"`
    TopicName        string                 `json:"topic_name"`
    ClassifyLayer    string                 `json:"classify_layer"`    // L1_REDIS / L2_PG_JIUYAN / L3_PG_CONCEPT / L4_LLM
    Strategy         string                 `json:"strategy"`          // JIUYAN_ATTR / CONCEPT_ATTR / LLM_V1 / MANUAL
    CandidateScores  []CandidateScoreDetail `json:"candidate_scores"`  // 候选热点评分明细
    EvidenceText     string                 `json:"evidence_text"`     // 关键证据原文
    Confidence       float64                `json:"confidence"`
    CorrectedTopicID *int64                 `json:"corrected_topic_id"` // 人工修正后的热点ID
    CreatedAt        string                 `json:"created_at"`
}

type GetEvidenceResp = PaginatedResult[EvidenceItem]
```

##### PUT /api/v1/stock/:ts_code/evidence/:id/correct — 人工修正归因

```go
// -------- Request --------
type CorrectEvidenceReq struct {
    TsCode           string `json:"-" uri:"ts_code" binding:"required"`
    EvidenceID       int64  `json:"-" uri:"id" binding:"required,min=1"`
    CorrectedTopicID int64  `json:"corrected_topic_id" binding:"required,min=1"` // 修正后的热点ID
}

// -------- Response --------
// Response[nil]
// 更新 classify_evidence.corrected_topic_id，用于后期训练数据积累
```

---

#### 4.2.4 策略告警（Alerts）

##### GET /api/v1/alerts — 查询今日策略告警

```go
// -------- Request --------
type GetAlertsReq struct {
    Date    string `json:"date" form:"date" binding:"omitempty,datetime=2006-01-02"`    // 默认今日
    TopicID int64  `json:"topic_id" form:"topic_id" binding:"omitempty,min=1"`          // 筛选某热点
    SortBy  string `json:"sort_by" form:"sort_by" binding:"omitempty,oneof=trigger_time trigger_price prev_day_pct"`
    Order   string `json:"order" form:"order" binding:"omitempty,oneof=asc desc"`
}

// -------- Response --------
type AlertItem struct {
    ID           int64           `json:"id"`
    Date         string          `json:"date"`
    TsCode       string          `json:"ts_code"`
    StockName    string          `json:"stock_name"`
    TopicID      int64           `json:"topic_id"`
    TopicName    string          `json:"topic_name"`
    AlertType    int             `json:"alert_type"`       // 1=策略2命中
    TriggerPrice float64         `json:"trigger_price"`    // 触发价格
    TriggerTime  string          `json:"trigger_time"`     // 触发时间 ISO8601
    PrevDayPct   float64         `json:"prev_day_pct"`     // 前一日涨跌幅
    ExtraInfo    *AlertExtraInfo `json:"extra_info"`       // 触发瞬间行情快照
    Notified     bool            `json:"notified"`
    CreatedAt    string          `json:"created_at"`
}

type AlertExtraInfo struct {
    CurrentPrice float64 `json:"current_price"`  // 当前价
    Volume       int64   `json:"volume"`         // 成交量（手）
    Amount       float64 `json:"amount"`         // 成交额（元）
    TurnoverRate float64 `json:"turnover_rate"`  // 换手率 %
    Bid1Price    float64 `json:"bid1_price"`     // 买一价
    Bid1Volume   int64   `json:"bid1_volume"`    // 买一量
    Ask1Price    float64 `json:"ask1_price"`     // 卖一价
    Ask1Volume   int64   `json:"ask1_volume"`    // 卖一量
}

// Response[[]AlertItem]
```

```bash
curl -H "X-API-Key: $KEY" \
  "http://localhost:8080/api/v1/alerts?date=2026-03-15&topic_id=1"
```

##### GET /api/v1/alerts/history — 历史告警查询

```go
// -------- Request --------
type GetAlertHistoryReq struct {
    StartDate string `json:"start_date" form:"start_date" binding:"required,datetime=2006-01-02"`
    EndDate   string `json:"end_date" form:"end_date" binding:"required,datetime=2006-01-02"`
    TopicID   int64  `json:"topic_id" form:"topic_id" binding:"omitempty,min=1"`
    TsCode    string `json:"ts_code" form:"ts_code" binding:"omitempty"`
    Page      int    `json:"page" form:"page" binding:"omitempty,min=1"`
    PageSize  int    `json:"page_size" form:"page_size" binding:"omitempty,min=1,max=200"`
}

// -------- Response --------
type GetAlertHistoryResp = PaginatedResult[AlertItem]
```

##### GET /api/v1/alerts/today — 今日告警汇总统计

```go
// -------- Request --------
// 无参数，返回今日告警汇总

// -------- Response --------
type AlertTodaySummary struct {
    Date           string            `json:"date"`
    TotalAlerts    int               `json:"total_alerts"`     // 今日告警总数
    TopicBreakdown []TopicAlertCount `json:"topic_breakdown"`  // 按热点分组统计
    LatestAlert    *AlertItem        `json:"latest_alert"`     // 最近一条告警
}

type TopicAlertCount struct {
    TopicID   int64  `json:"topic_id"`
    TopicName string `json:"topic_name"`
    Count     int    `json:"count"`
}
```

##### GET /api/v1/focus — 查询某日关注热点

```go
// -------- Request --------
type GetFocusReq struct {
    Date string `json:"date" form:"date" binding:"required,datetime=2006-01-02"`
}

// -------- Response --------
type FocusTopicItem struct {
    TopicID          int64  `json:"topic_id"`
    TopicName        string `json:"topic_name"`
    OccurrenceCount  int    `json:"occurrence_count"`   // 历史热度
    RecentLimitCount int    `json:"recent_limit_count"` // 近3日涨停家数
    IsActive         bool   `json:"is_active"`
    CreatedAt        string `json:"created_at"`
}

// Response[[]FocusTopicItem]
```

##### POST /api/v1/focus — 设置关注热点

```go
// -------- Request --------
type SetFocusReq struct {
    Date     string  `json:"date" binding:"required,datetime=2006-01-02"`
    TopicIDs []int64 `json:"topic_ids" binding:"required,min=1,dive,min=1"` // 热点ID列表
}

// -------- Response --------
// Response[nil]
// 写入 daily_focus_topics 表，ON CONFLICT DO NOTHING
// 同步写入 Redis focus:topics:{date} Set
```

```bash
curl -X POST -H "X-API-Key: $KEY" -H "Content-Type: application/json" \
  -d '{"date":"2026-03-16","topic_ids":[1,3,7,15]}' \
  http://localhost:8080/api/v1/focus
```

##### DELETE /api/v1/focus/:topic_id — 取消关注

```go
// -------- Request --------
type DeleteFocusReq struct {
    TopicID int64  `json:"-" uri:"topic_id" binding:"required,min=1"`
    Date    string `json:"date" form:"date" binding:"required,datetime=2006-01-02"`
}

// -------- Response --------
// Response[nil]
```

---

#### 4.2.5 数据同步（Sync / Admin）

##### POST /api/v1/admin/crawl/history — 触发韭研历史全量爬取

```go
// -------- Request --------
type CrawlHistoryReq struct {
    StartDate string `json:"start_date" binding:"required,datetime=2006-01-02"` // 起始日期，默认 2023-01-01
    EndDate   string `json:"end_date" binding:"required,datetime=2006-01-02"`   // 结束日期
}

// -------- Response --------
type CrawlHistoryResp struct {
    TaskID    string `json:"task_id"`     // 异步任务ID
    TotalDays int    `json:"total_days"`  // 需爬取的交易日数
    Message   string `json:"message"`     // "任务已提交，后台异步执行"
}
```

##### POST /api/v1/admin/crawl/today — 触发今日增量爬取

```go
// -------- Request --------
type CrawlTodayReq struct {
    Date string `json:"date" binding:"omitempty,datetime=2006-01-02"` // 默认今天
}

// -------- Response --------
type CrawlTodayResp struct {
    Date            string `json:"date"`
    TopicCount      int    `json:"topic_count"`       // 爬取到的热点数
    StockCount      int    `json:"stock_count"`       // 爬取到的股票数
    NewTopics       int    `json:"new_topics"`        // 新增热点数
    UpdatedMappings int    `json:"updated_mappings"`  // 更新的映射数
}
```

##### GET /api/v1/admin/crawl/logs — 爬取日志查询

```go
// -------- Request --------
type GetCrawlLogsReq struct {
    Status   *int `json:"status" form:"status" binding:"omitempty,oneof=0 1 2"` // 0=待处理 1=成功 2=失败
    Page     int  `json:"page" form:"page" binding:"omitempty,min=1"`
    PageSize int  `json:"page_size" form:"page_size" binding:"omitempty,min=1,max=100"`
}

// -------- Response --------
type CrawlLogItem struct {
    ID         int64  `json:"id"`
    Date       string `json:"date"`           // 爬取的目标日期
    Status     int    `json:"status"`         // 0=待处理 1=成功 2=失败
    TopicCount int    `json:"topic_count"`
    StockCount int    `json:"stock_count"`
    RetryCount int    `json:"retry_count"`
    ErrorMsg   string `json:"error_msg"`
    CreatedAt  string `json:"created_at"`
    UpdatedAt  string `json:"updated_at"`
}

type GetCrawlLogsResp = PaginatedResult[CrawlLogItem]
```

##### POST /api/v1/admin/cache/refresh — 手动刷新 Redis 缓存

```go
// -------- Request --------
type CacheRefreshReq struct {
    Scope string `json:"scope" binding:"omitempty,oneof=all mappings concepts focus boards"`
    // all      — 全部缓存
    // mappings — stock_topic_mappings (cache:stock_topics + cache:bind_strength)
    // concepts — stock_concept_tags + concept_topic_mappings (cache:stock_concepts + cache:concept_to_topics)
    // focus    — 当日关注热点 (focus:topics:{date})
    // boards   — 板块规则
}

// -------- Response --------
type CacheRefreshResp struct {
    Scope         string `json:"scope"`
    RefreshedKeys int    `json:"refreshed_keys"` // 刷新的 key 数量
    Duration      string `json:"duration"`       // 耗时 e.g. "1.23s"
}
```

##### POST /api/v1/admin/concepts/sync — 触发 Tushare 概念数据同步

```go
// -------- Request --------
type ConceptSyncReq struct {
    Mode string `json:"mode" binding:"omitempty,oneof=full incremental"` // full=全量 incremental=增量，默认 incremental
}

// -------- Response --------
type ConceptSyncResp struct {
    TaskID        string `json:"task_id"`
    ConceptCount  int    `json:"concept_count"`    // 概念数量
    StockTagCount int    `json:"stock_tag_count"`  // 股票标签写入数量
    NewConcepts   int    `json:"new_concepts"`     // 新增概念数
    Message       string `json:"message"`
}
```

##### GET /api/v1/admin/concepts/mappings — 查询概念→热点映射状态

```go
// -------- Request --------
type GetConceptMappingsReq struct {
    Status   string `json:"status" form:"status" binding:"omitempty,oneof=mapped unmapped all"` // 筛选映射状态
    Keyword  string `json:"keyword" form:"keyword" binding:"omitempty,max=64"`
    Page     int    `json:"page" form:"page" binding:"omitempty,min=1"`
    PageSize int    `json:"page_size" form:"page_size" binding:"omitempty,min=1,max=100"`
}

// -------- Response --------
type ConceptMappingStatus struct {
    ConceptName string  `json:"concept_name"`
    ConceptCode string  `json:"concept_code"`
    IsMapped    bool    `json:"is_mapped"`     // 是否已建立映射
    TopicID     *int64  `json:"topic_id"`      // 映射的热点ID（未映射则 null）
    TopicName   *string `json:"topic_name"`    // 映射的热点名称
    MatchType   string  `json:"match_type"`    // exact / synonym / llm / manual / ""
    StockCount  int     `json:"stock_count"`   // 该概念下的股票数量
}

type GetConceptMappingsResp = PaginatedResult[ConceptMappingStatus]
```

##### POST /api/v1/admin/concepts/mappings — 手动建立概念→热点映射

```go
// -------- Request --------
type CreateConceptMappingReq struct {
    ConceptName string `json:"concept_name" binding:"required,max=128"`
    ConceptCode string `json:"concept_code" binding:"required,max=16"`
    TopicID     int64  `json:"topic_id" binding:"required,min=1"`
}

// -------- Response --------
// Response[ConceptMappingItem]
// match_type 自动设为 "manual"
```

##### POST /api/v1/admin/llm/classify — 触发 LLM 批量分类

```go
// -------- Request --------
type LLMBatchClassifyReq struct {
    TsCodes []string `json:"ts_codes" binding:"omitempty,max=100,dive,required"` // 指定股票列表，空则自动选取未分类股票
    Limit   int      `json:"limit" binding:"omitempty,min=1,max=500"`            // 自动选取时的数量上限
}

// -------- Response --------
type LLMBatchClassifyResp struct {
    TaskID    string `json:"task_id"`
    Submitted int    `json:"submitted"`  // 提交的分类任务数
    Message   string `json:"message"`    // "后台异步执行，结果回写后生效"
}
```

---

#### 4.2.6 系统管理（System）

##### GET /health — 健康检查（无需鉴权）

```go
// -------- Request --------
// 无参数，无需 API Key

// -------- Response --------
type HealthResp struct {
    Status    string            `json:"status"`     // "ok" / "degraded" / "error"
    Timestamp string            `json:"timestamp"`  // ISO8601
    Version   string            `json:"version"`    // 服务版本号
    Checks    map[string]string `json:"checks"`     // {"postgres":"ok","redis":"ok","tushare":"ok"}
}
```

```bash
curl http://localhost:8080/health
# {"code":0,"message":"ok","data":{"status":"ok","timestamp":"2026-03-15T14:30:00+08:00","version":"1.0.0","checks":{"postgres":"ok","redis":"ok"}}}
```

##### POST /api/v1/admin/init — 系统初始化

```go
// -------- Request --------
type SystemInitReq struct {
    Steps []string `json:"steps" binding:"omitempty,dive,oneof=stocks boards jiuyan concepts concept_mappings all"`
    // 指定初始化步骤，默认 "all"
    // stocks          — 从 Tushare 加载 stock_basic → stocks 表
    // boards          — 初始化板块规则 → boards 表
    // jiuyan          — 韭研全量爬取 → hot_topics + stock_topic_mappings + topic_synonyms
    // concepts        — Tushare 概念数据 → stock_concept_tags
    // concept_mappings — 自动建立概念→热点映射 → concept_topic_mappings
    // all             — 按上述顺序执行全部
}

// -------- Response --------
type SystemInitResp struct {
    TaskID  string           `json:"task_id"`
    Steps   []InitStepResult `json:"steps"`
    Message string           `json:"message"`
}

type InitStepResult struct {
    Step    string `json:"step"`     // 步骤名称
    Status  string `json:"status"`   // "pending" / "running" / "success" / "failed"
    Count   int    `json:"count"`    // 处理数量
    Message string `json:"message"`  // 详情
}
```

##### GET /api/v1/admin/stats — 系统统计

```go
// -------- Request --------
// 无参数

// -------- Response --------
type SystemStatsResp struct {
    Date                 string `json:"date"`
    TotalStocks          int    `json:"total_stocks"`          // 股票总数
    ActiveStocks         int    `json:"active_stocks"`         // 正常交易的股票数
    TotalTopics          int    `json:"total_topics"`          // 热点总数
    ActiveTopics         int    `json:"active_topics"`         // 启用的热点数
    TotalSynonyms        int    `json:"total_synonyms"`        // 同义词总数
    TotalConceptMappings int    `json:"total_concept_mappings"` // 概念映射总数
    UnmappedConcepts     int    `json:"unmapped_concepts"`     // 未映射的概念数
    TodayLimitUpCount    int    `json:"today_limit_up_count"`  // 今日涨停数
    TodayAbove5Count     int    `json:"today_above5_count"`    // 今日5%+数
    TodayAlertCount      int    `json:"today_alert_count"`     // 今日告警数
    TodayFocusTopics     int    `json:"today_focus_topics"`    // 今日关注热点数
    CrawlLogsTotal       int    `json:"crawl_logs_total"`      // 爬取记录总数
    CrawlLogsFailed      int    `json:"crawl_logs_failed"`     // 失败的爬取记录数
    RedisMemoryUsage     string `json:"redis_memory_usage"`    // Redis 内存占用 e.g. "128MB"
    PGConnectionPool     string `json:"pg_connection_pool"`    // PG 连接池状态 e.g. "10/50"
}
```

---

#### 4.2.7 历史数据（History）

> 历史池查询已在 4.2.1 `GET /api/v1/pool/history` 中定义。
> 历史告警已在 4.2.4 `GET /api/v1/alerts/history` 中定义。
> 本节补充按日聚合统计和告警汇总接口。

##### GET /api/v1/history/pool/daily — 按日聚合的历史池统计

```go
// -------- Request --------
type DailyPoolStatsReq struct {
    StartDate string `json:"start_date" form:"start_date" binding:"required,datetime=2006-01-02"`
    EndDate   string `json:"end_date" form:"end_date" binding:"required,datetime=2006-01-02"`
    PoolType  int    `json:"pool_type" form:"pool_type" binding:"omitempty,oneof=1 2"`
}

// -------- Response --------
type DailyPoolStats struct {
    Date          string `json:"date"`
    LimitUpCount  int    `json:"limit_up_count"`  // 涨停家数
    Above5Count   int    `json:"above5_count"`    // 5%+家数
    TopTopicName  string `json:"top_topic_name"`  // 当日最热热点
    TopTopicCount int    `json:"top_topic_count"` // 最热热点涨停家数
    AlertCount    int    `json:"alert_count"`     // 当日告警数
}

// Response[[]DailyPoolStats]
```

##### GET /api/v1/history/alerts/summary — 历史告警汇总（按热点维度）

```go
// -------- Request --------
type AlertSummaryReq struct {
    StartDate string `json:"start_date" form:"start_date" binding:"required,datetime=2006-01-02"`
    EndDate   string `json:"end_date" form:"end_date" binding:"required,datetime=2006-01-02"`
}

// -------- Response --------
type AlertSummaryItem struct {
    TopicID     int64   `json:"topic_id"`
    TopicName   string  `json:"topic_name"`
    TotalAlerts int     `json:"total_alerts"`  // 期间告警总数
    HitRate     float64 `json:"hit_rate"`      // 命中率（预留字段）
    AvgPrevPct  float64 `json:"avg_prev_pct"`  // 平均前一日涨幅
}

// Response[[]AlertSummaryItem]
```

---

### 4.3 WebSocket 协议

#### 4.3.1 连接端点

```
ws://host:8080/ws
wss://host:8080/ws          # TLS 环境
```

**连接认证**：通过 query 参数传递 API Key：

```
ws://host:8080/ws?token=YOUR_API_KEY
```

#### 4.3.2 连接参数

```go
const (
    // WriteWait 写超时
    WriteWait = 10 * time.Second

    // PongWait 等待客户端 Pong 回复的超时时间
    PongWait = 60 * time.Second

    // PingPeriod 服务端发送 Ping 的周期（必须小于 PongWait）
    PingPeriod = 50 * time.Second

    // MaxMessageSize 单条消息最大字节数
    MaxMessageSize = 4096

    // SendBufSize 每个客户端的发送缓冲区大小（消息条数）
    // 缓冲区满时主动断开该客户端（背压保护）
    SendBufSize = 256
)
```

#### 4.3.3 Hub/Client 架构

```go
// Hub 连接管理中心（全局单例）
type Hub struct {
    clients    map[*Client]bool // 所有活跃连接
    broadcast  chan []byte       // 广播通道
    register   chan *Client      // 注册通道
    unregister chan *Client      // 注销通道
}

// Client 单个 WebSocket 连接
type Client struct {
    hub  *Hub
    conn *websocket.Conn
    send chan []byte // 发送缓冲区，容量 = SendBufSize (256)
}
```

#### 4.3.4 消息信封（统一格式）

所有 WebSocket 推送消息共享统一信封：

```go
// WSMessage WebSocket 消息信封
type WSMessage struct {
    Type      string          `json:"type"`       // "pool_snapshot" | "pool_diff" | "strategy_alert"
    Timestamp int64           `json:"timestamp"`  // Unix 时间戳（秒）
    Data      json.RawMessage `json:"data"`       // 根据 type 解析为不同结构
}
```

#### 4.3.5 消息类型 1：pool_snapshot（全量快照）

**触发时机**：
- 客户端首次连接时
- 池子变化量 >= 总量的 30% 时

```go
// WSPoolSnapshot 全量快照数据
type WSPoolSnapshot struct {
    Date         string        `json:"date"`           // 交易日 "2026-03-15"
    SnapshotTime string        `json:"snapshot_time"`  // 快照时间 "14:30:10"
    LimitUp      WSPoolSection `json:"limit_up"`       // 涨停池
    Above5Pct    WSPoolSection `json:"above_5pct"`     // 5%池
}

type WSPoolSection struct {
    TotalCount   int               `json:"total_count"`
    Groups       []WSPoolGroup     `json:"groups"`
    Unclassified []WSPoolStockItem `json:"unclassified"`
}

type WSPoolGroup struct {
    TopicID   int64             `json:"topic_id"`
    TopicName string            `json:"topic_name"`
    Stocks    []WSPoolStockItem `json:"stocks"`
}

type WSPoolStockItem struct {
    TsCode           string  `json:"ts_code"`
    Name             string  `json:"name"`
    ChangePct        float64 `json:"change_pct"`
    CurrentPrice     float64 `json:"current_price"`
    FirstLimitTime   string  `json:"first_limit_time,omitempty"` // 仅涨停池有值
    AttributionScore float64 `json:"attribution_score"`
}
```

**JSON 示例**：

```json
{
  "type": "pool_snapshot",
  "timestamp": 1773579660,
  "data": {
    "date": "2026-03-15",
    "snapshot_time": "14:30:10",
    "limit_up": {
      "total_count": 45,
      "groups": [
        {
          "topic_id": 1,
          "topic_name": "AI",
          "stocks": [
            {
              "ts_code": "688256.SH",
              "name": "寒武纪",
              "change_pct": 20.0,
              "current_price": 350.00,
              "first_limit_time": "09:31:00",
              "attribution_score": 0.95
            },
            {
              "ts_code": "002049.SZ",
              "name": "紫光国微",
              "change_pct": 10.0,
              "current_price": 55.00,
              "first_limit_time": "14:30:05",
              "attribution_score": 0.83
            }
          ]
        },
        {
          "topic_id": 3,
          "topic_name": "机器人",
          "stocks": [
            {
              "ts_code": "300124.SZ",
              "name": "汇川技术",
              "change_pct": 20.0,
              "current_price": 88.00,
              "first_limit_time": "10:15:22",
              "attribution_score": 0.68
            }
          ]
        }
      ],
      "unclassified": []
    },
    "above_5pct": {
      "total_count": 120,
      "groups": []
    }
  }
}
```

#### 4.3.6 消息类型 2：pool_diff（增量变更）

**触发时机**：
- 池子有变化但变化量 < 总量的 30% 时

```go
// WSPoolDiff 增量变更数据
type WSPoolDiff struct {
    AddedLimitUp   []WSDiffStockItem `json:"added_limit_up"`    // 新增涨停
    RemovedLimitUp []WSDiffRemoved   `json:"removed_limit_up"`  // 移出涨停（炸板）
    AddedAbove5    []WSDiffStockItem `json:"added_above5"`      // 新增5%+
    RemovedAbove5  []WSDiffRemoved   `json:"removed_above5"`    // 移出5%+（跌出5%）
}

type WSDiffStockItem struct {
    TsCode         string  `json:"ts_code"`
    Name           string  `json:"name"`
    ChangePct      float64 `json:"change_pct"`
    TopicID        int64   `json:"topic_id"`
    TopicName      string  `json:"topic_name"`
    FirstLimitTime string  `json:"first_limit_time,omitempty"`
}

type WSDiffRemoved struct {
    TsCode string `json:"ts_code"`
    Name   string `json:"name"`
    Reason string `json:"reason"` // "opened"=炸板 / "dropped"=跌出5%
}
```

**JSON 示例**：

```json
{
  "type": "pool_diff",
  "timestamp": 1773579670,
  "data": {
    "added_limit_up": [
      {
        "ts_code": "002049.SZ",
        "name": "紫光国微",
        "change_pct": 10.0,
        "topic_id": 1,
        "topic_name": "AI",
        "first_limit_time": "14:30:05"
      }
    ],
    "removed_limit_up": [
      {
        "ts_code": "600100.SH",
        "name": "同方股份",
        "reason": "opened"
      }
    ],
    "added_above5": [],
    "removed_above5": [
      {
        "ts_code": "300999.SZ",
        "name": "金龙鱼",
        "reason": "dropped"
      }
    ]
  }
}
```

#### 4.3.7 消息类型 3：strategy_alert（策略告警）

**触发时机**：
- 即时推送，不等 10 秒周期，发现即推

```go
// WSStrategyAlert 策略告警推送数据
type WSStrategyAlert struct {
    TsCode       string  `json:"ts_code"`        // 股票代码
    Name         string  `json:"name"`           // 股票名称
    TopicID      int64   `json:"topic_id"`       // 命中的热点ID
    TopicName    string  `json:"topic_name"`     // 热点名称
    TriggerPrice float64 `json:"trigger_price"`  // 涨停价格
    TriggerTime  string  `json:"trigger_time"`   // 首次涨停时间 "HH:MM:SS"
    PrevDayPct   float64 `json:"prev_day_pct"`   // 前一日涨跌幅 %
    Message      string  `json:"message"`        // 可展示的文案
    AlertID      int64   `json:"alert_id"`       // 告警记录ID
}
```

**JSON 示例**：

```json
{
  "type": "strategy_alert",
  "timestamp": 1773579665,
  "data": {
    "ts_code": "002049.SZ",
    "name": "紫光国微",
    "topic_id": 1,
    "topic_name": "AI",
    "trigger_price": 55.00,
    "trigger_time": "14:30:05",
    "prev_day_pct": 7.3,
    "message": "紫光国微 首次涨停！属于今日关注热点【AI】，昨日涨幅 7.3%",
    "alert_id": 1042
  }
}
```

#### 4.3.8 推送策略决策表

| 事件 | 推送类型 | 推送时机 | 备注 |
|------|----------|----------|------|
| 策略告警命中 | `strategy_alert` | **即时推送**（不等 10s 周期） | AlertService 检测到后直接通过 Hub 广播 |
| 10s 周期处理完成，diff 为空 | 不推送 | - | 节省带宽 |
| 10s 周期处理完成，变化量 < 30% | `pool_diff` | 周期结束后推送 | 增量更新，轻量 |
| 10s 周期处理完成，变化量 >= 30% | `pool_snapshot` | 周期结束后推送 | 大幅变动用全量替代增量 |
| 客户端首次连接 | `pool_snapshot` | 连接建立后**立即**推送 | 从 Redis `snapshot:current` 读取 |
| 客户端断线重连 | `pool_snapshot` | 重连后**立即**推送 | 等同首次连接处理 |

**变化量计算逻辑**：

```go
// 变化率 = (新增数 + 移除数) / 上一次池子总量
changeRate := float64(len(added)+len(removed)) / float64(prevTotal)
if changeRate >= 0.30 {
    // 推送全量 pool_snapshot
} else if changeRate > 0 {
    // 推送增量 pool_diff
}
// changeRate == 0 → 不推送
```

#### 4.3.9 连接生命周期时序

```
客户端                                  服务端 (Hub)
  |                                        |
  |---- GET /ws?token=KEY (Upgrade) ------>|
  |<--- 101 Switching Protocols -----------|
  |<--- pool_snapshot (全量) --------------|   ← 连接建立后立即推送
  |                                        |
  |                              [每 10s 周期]
  |<--- pool_diff / pool_snapshot ---------|   ← 有变化时推送
  |                                        |
  |                              [告警触发时]
  |<--- strategy_alert (即时) -------------|   ← 不等周期
  |                                        |
  |<--- ping (每 50s) --------------------|
  |---- pong ----------------------------->|   ← 60s 内必须回复
  |                                        |
  |              [send buffer 达到 256 条]  |
  |<--- 服务端主动断开连接 ----------------|   ← 背压保护
  |                                        |
  |---- 客户端自动重连 ------------------->|
  |<--- pool_snapshot (全量) --------------|   ← 重连 = 首次连接
```

#### 4.3.10 背压处理实现

```go
// Hub.run() 中的广播逻辑
func (h *Hub) run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
        case message := <-h.broadcast:
            for client := range h.clients {
                select {
                case client.send <- message:
                    // 发送成功，消息进入客户端缓冲区
                default:
                    // 缓冲区已满（256条），踢掉该客户端
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}

// Client.writePump() — 从 send channel 读消息写入 WebSocket
func (c *Client) writePump() {
    ticker := time.NewTicker(PingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()
    for {
        select {
        case message, ok := <-c.send:
            if !ok {
                // Hub 已关闭此 client 的 send channel
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
            if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
                return // 写入失败，连接已断开
            }
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

// Client.readPump() — 读取客户端消息（主要处理 Pong）
func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    c.conn.SetReadLimit(MaxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(PongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(PongWait))
        return nil
    })
    for {
        _, _, err := c.conn.ReadMessage()
        if err != nil {
            break
        }
    }
}
```

---

### 4.4 错误码定义

```go
package errcode

// -------- 成功 --------
const (
    Success = 0 // 请求成功
)

// -------- 1xxx: 请求错误（客户端问题） --------
const (
    ErrBadRequest   = 1001 // 请求参数无效（binding 校验失败、格式错误）
    ErrUnauthorized = 1002 // 未提供 X-API-Key 或 Key 无效
    ErrNotFound     = 1003 // 资源不存在（热点/股票/告警/证据不存在）
    ErrConflict     = 1004 // 资源冲突（热点名称重复、同义词已存在、映射已存在）
)

// -------- 2xxx: 业务错误（业务规则阻止） --------
const (
    ErrBusinessLogic  = 2001 // 业务逻辑不允许（合并到自身、删除有活跃引用的热点等）
    ErrClassifyFailed = 2002 // 分类失败（四层均未命中且 LLM 也失败/置信度 < 0.7）
)

// -------- 3xxx: 外部服务错误 --------
const (
    ErrTushareAPI = 3001 // Tushare 接口调用失败（超时/限流/返回异常数据）
    ErrJiuyanAPI  = 3002 // 韭研公社接口调用失败（超时/格式变更/限流）
    ErrLLMAPI     = 3003 // LLM 接口调用失败（超时/quota耗尽/返回异常）
)

// -------- 5xxx: 内部错误 --------
const (
    ErrInternal = 5000 // 服务端内部错误（数据库异常/Redis 异常/未捕获 panic）
)
```

**错误码 → HTTP 状态码映射**：

```go
var ErrorCodeToHTTPStatus = map[int]int{
    Success:           200,
    ErrBadRequest:     400,
    ErrUnauthorized:   401,
    ErrNotFound:       404,
    ErrConflict:       409,
    ErrBusinessLogic:  422,
    ErrClassifyFailed: 422,
    ErrTushareAPI:     502,
    ErrJiuyanAPI:      502,
    ErrLLMAPI:         502,
    ErrInternal:       500,
}
```

**错误响应示例**：

```json
{
  "code": 1001,
  "message": "参数校验失败: date 格式应为 YYYY-MM-DD",
  "data": null
}
```

```json
{
  "code": 1003,
  "message": "热点不存在: id=999",
  "data": null
}
```

```json
{
  "code": 1004,
  "message": "同义词已存在: '算力概念' 已归属热点 'AI算力'",
  "data": null
}
```

```json
{
  "code": 2001,
  "message": "不能将热点合并到自身",
  "data": null
}
```

```json
{
  "code": 3001,
  "message": "Tushare 接口超时，已重试4次仍失败",
  "data": null
}
```

```json
{
  "code": 5000,
  "message": "internal server error",
  "data": null
}
```

---

### 4.5 通用响应包装器

```go
// Response 统一响应包装器
// 所有 API 均使用此结构返回，T 为具体的 data 类型
type Response[T any] struct {
    Code    int    `json:"code"`           // 错误码，0 表示成功，非0表示错误
    Message string `json:"message"`        // 描述信息
    Data    T      `json:"data,omitempty"` // 响应数据，错误时省略或为 null
}
```

**工具函数**：

```go
// OK 构造成功响应
func OK[T any](data T) Response[T] {
    return Response[T]{
        Code:    Success,
        Message: "ok",
        Data:    data,
    }
}

// Fail 构造错误响应
func Fail(code int, msg string) Response[any] {
    return Response[any]{
        Code:    code,
        Message: msg,
    }
}

// FailWithData 构造带额外信息的错误响应（如字段校验详情）
func FailWithData[T any](code int, msg string, data T) Response[T] {
    return Response[T]{
        Code:    code,
        Message: msg,
        Data:    data,
    }
}
```

**在 Gin Handler 中的典型用法**：

```go
func (h *TopicHandler) List(c *gin.Context) {
    var req ListTopicsReq
    if err := c.ShouldBindQuery(&req); err != nil {
        c.JSON(http.StatusBadRequest, Fail(ErrBadRequest, err.Error()))
        return
    }
    ApplyPaginationDefaults(&req.Page, &req.PageSize)

    result, err := h.svc.ListTopics(c.Request.Context(), &req)
    if err != nil {
        code, httpStatus := mapError(err)
        c.JSON(httpStatus, Fail(code, err.Error()))
        return
    }
    c.JSON(http.StatusOK, OK(result))
}
```

---

### 4.6 分页

```go
// Pagination 通用分页请求参数
// 嵌入到需要分页的请求 struct 中，通过 form tag 绑定 query string
type Pagination struct {
    Page     int `json:"page" form:"page" binding:"min=1"`           // 页码，从 1 开始
    PageSize int `json:"page_size" form:"page_size" binding:"min=1,max=100"` // 每页条数，默认 50，最大 200
}

// PaginatedResult 通用分页响应
// 所有返回列表的 API 均使用此结构
type PaginatedResult[T any] struct {
    Items []T   `json:"items"`  // 当前页数据列表
    Total int64 `json:"total"`  // 总记录数
    Page  int   `json:"page"`   // 当前页码
    Pages int   `json:"pages"`  // 总页数 = ceil(Total / PageSize)
}
```

**默认值处理**：

```go
// ApplyPaginationDefaults 填充分页参数默认值
// 在 handler 或 middleware 中调用
func ApplyPaginationDefaults(page *int, pageSize *int) {
    if *page <= 0 {
        *page = 1
    }
    if *pageSize <= 0 {
        *pageSize = 50 // 全局默认每页 50 条
    }
    if *pageSize > 200 {
        *pageSize = 200 // 硬上限 200
    }
}
```

**SQL 构建示例**：

```go
// 使用 pgx/v5 构建分页查询
offset := (req.Page - 1) * req.PageSize

// 查询数据
query := `SELECT id, name, source, is_active, created_at
          FROM hot_topics
          WHERE ($1 = '' OR name ILIKE '%' || $1 || '%')
            AND ($2::boolean IS NULL OR is_active = $2)
          ORDER BY priority DESC, id ASC
          LIMIT $3 OFFSET $4`
rows, err := pool.Query(ctx, query, req.Keyword, req.IsActive, req.PageSize, offset)

// 查询总数
countQuery := `SELECT COUNT(*) FROM hot_topics
               WHERE ($1 = '' OR name ILIKE '%' || $1 || '%')
                 AND ($2::boolean IS NULL OR is_active = $2)`
var total int64
_ = pool.QueryRow(ctx, countQuery, req.Keyword, req.IsActive).Scan(&total)

// 计算总页数
pages := int(math.Ceil(float64(total) / float64(req.PageSize)))
```

---

### 4.7 鉴权中间件

```go
// AuthMiddleware API Key 鉴权中间件
// 从请求头 X-API-Key 读取 key，与配置中的 server.api_key 比对
func AuthMiddleware(apiKey string) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := c.GetHeader("X-API-Key")
        if key == "" {
            c.AbortWithStatusJSON(401, Fail(ErrUnauthorized, "缺少 X-API-Key 请求头"))
            return
        }
        if key != apiKey {
            c.AbortWithStatusJSON(401, Fail(ErrUnauthorized, "API Key 无效"))
            return
        }
        c.Next()
    }
}

// AuditMiddleware 写操作审计日志中间件
// 记录所有 POST/PUT/DELETE 请求到结构化日志
func AuditMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
            c.Next()
            return
        }
        start := time.Now()
        c.Next()
        duration := time.Since(start)
        zap.L().Info("audit",
            zap.String("method", c.Request.Method),
            zap.String("path", c.Request.URL.Path),
            zap.Int("status", c.Writer.Status()),
            zap.Duration("duration", duration),
            zap.String("api_key", maskAPIKey(c.GetHeader("X-API-Key"))),
            zap.String("client_ip", c.ClientIP()),
        )
    }
}

// maskAPIKey 脱敏 API Key，只保留前4位和后4位
func maskAPIKey(key string) string {
    if len(key) <= 8 {
        return "****"
    }
    return key[:4] + "****" + key[len(key)-4:]
}
```

---

### 4.8 路由注册总览

```go
func SetupRouter(cfg *config.Config, handlers *Handlers) *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())

    // ---- 无需鉴权 ----
    r.GET("/health", handlers.System.Health)

    // ---- WebSocket（token 通过 query 参数校验） ----
    r.GET("/ws", handlers.WS.HandleWebSocket)

    // ---- 需要 API Key 鉴权 ----
    api := r.Group("/api/v1")
    api.Use(AuthMiddleware(cfg.Server.APIKey))
    api.Use(AuditMiddleware())
    {
        // -- 实时池 (Pool) --
        pool := api.Group("/pool")
        {
            pool.GET("/limit-up", handlers.Pool.GetLimitUp)          // #1
            pool.GET("/above5", handlers.Pool.GetAbove5)             // #2
            pool.GET("/limit-up/snapshot", handlers.Pool.GetSnapshot) // #3
            pool.GET("/history", handlers.Pool.GetHistory)            // #4
        }

        // -- 热点管理 (Topics) --
        topics := api.Group("/topics")
        {
            topics.GET("", handlers.Topic.List)                              // #5
            topics.POST("", handlers.Topic.Create)                           // #6
            topics.PUT("/:id", handlers.Topic.Update)                        // #7
            topics.DELETE("/:id", handlers.Topic.Delete)                     // #8
            topics.PUT("/:id/merge", handlers.Topic.Merge)                   // #9
            topics.GET("/:id/synonyms", handlers.Synonym.List)               // #10
            topics.POST("/:id/synonyms", handlers.Synonym.Create)            // #11
            topics.DELETE("/synonyms/:synonym_id", handlers.Synonym.Delete)   // #12
            topics.GET("/:id/concepts", handlers.Concept.ListByTopic)        // #13
        }

        // -- 股票分类 (Classification) --
        stock := api.Group("/stock")
        {
            stock.GET("/:ts_code/topics", handlers.Classification.GetTopics)           // #14
            stock.POST("/:ts_code/topics", handlers.Classification.ManualClassify)     // #15
            stock.DELETE("/:ts_code/topics/:topic_id", handlers.Classification.Remove) // #16
            stock.POST("/:ts_code/reclassify", handlers.Classification.Reclassify)     // #17
            stock.GET("/:ts_code/evidence", handlers.Evidence.List)                    // #18
            stock.PUT("/:ts_code/evidence/:id/correct", handlers.Evidence.Correct)     // #19
        }

        // -- 策略告警 (Alerts) --
        alerts := api.Group("/alerts")
        {
            alerts.GET("", handlers.Alert.List)                  // #20
            alerts.GET("/history", handlers.Alert.History)        // #21
            alerts.GET("/today", handlers.Alert.TodaySummary)    // #22
        }

        // -- 关注热点 (Focus) --
        focus := api.Group("/focus")
        {
            focus.GET("", handlers.Focus.Get)                    // #23
            focus.POST("", handlers.Focus.Set)                   // #24
            focus.DELETE("/:topic_id", handlers.Focus.Delete)    // #25
        }

        // -- 历史数据 (History) --
        history := api.Group("/history")
        {
            history.GET("/pool/daily", handlers.History.DailyPoolStats)   // 按日聚合
            history.GET("/alerts/summary", handlers.History.AlertSummary) // 告警汇总
        }

        // -- 系统管理 (Admin) --
        admin := api.Group("/admin")
        {
            admin.POST("/init", handlers.System.Init)                          // #36
            admin.GET("/stats", handlers.System.Stats)                         // #34
            admin.POST("/crawl/history", handlers.Crawl.CrawlHistory)          // #26
            admin.POST("/crawl/today", handlers.Crawl.CrawlToday)             // #27
            admin.GET("/crawl/logs", handlers.Crawl.Logs)                      // #28
            admin.POST("/cache/refresh", handlers.Cache.Refresh)               // #29
            admin.POST("/concepts/sync", handlers.Concept.Sync)                // #30
            admin.GET("/concepts/mappings", handlers.Concept.ListMappings)     // #31
            admin.POST("/concepts/mappings", handlers.Concept.CreateMapping)   // #32
            admin.POST("/llm/classify", handlers.LLM.BatchClassify)            // #33
        }
    }

    return r
}
```

---

## 五、配置、调度与运维

---

### 5.1 完整配置文件（config.yaml）

以下为生产环境完整配置，所有敏感值通过环境变量注入（`${VAR}` 语法）。
AI coding agent 在生成代码时应将本文件作为 config 结构体的唯一参照。

```yaml
# ============================================================
# stock-monitor 完整配置文件
# 路径: config/config.yaml
# ============================================================

# ---- HTTP 服务 ----
server:
  port: 8080                     # 监听端口
  mode: release                  # gin 模式: debug / release / test
  read_timeout: 10               # 读超时（秒）
  write_timeout: 30              # 写超时（秒）
  api_key: "${API_KEY}"          # X-API-Key 鉴权密钥

# ---- PostgreSQL ----
database:
  host: localhost
  port: 5432
  user: stock_monitor
  password: "${DB_PASSWORD}"
  dbname: stock_monitor
  sslmode: disable               # disable / require / verify-full
  timezone: Asia/Shanghai
  max_open_conns: 50             # 最大打开连接数
  max_idle_conns: 10             # 最大空闲连接数
  conn_max_lifetime: 3600        # 连接最大存活时间（秒）
  conn_max_idle_time: 300        # 空闲连接最大存活时间（秒）

# ---- Redis ----
redis:
  addr: localhost:6379
  password: "${REDIS_PASSWORD}"
  db: 0
  pool_size: 100                 # 连接池大小
  min_idle_conns: 10             # 最小空闲连接
  dial_timeout: 5                # 连接超时（秒）
  read_timeout: 3                # 读超时（秒）
  write_timeout: 3               # 写超时（秒）

# ---- Tushare Pro（行情 + 概念板块数据源）----
tushare:
  base_url: "https://api.tushare.pro"
  token: "${TUSHARE_TOKEN}"
  rate_limit_per_min: 500        # 每分钟最大请求数
  timeout: 30                    # 单次请求超时（秒）

# ---- 韭研公社（热点数据源）----
jiuyan:
  base_url: "https://app.jiuyangongshe.com/jystock-app/api/v1"
  api_key: "${JIUYAN_TOKEN}"
  timeout: 30                    # 单次请求超时（秒）
  crawl_start_date: "2023-01-01" # 全量爬取起始日期

# ---- LLM（兜底分类）----
llm:
  provider: openai               # openai / azure / local
  model: gpt-4o-mini
  api_key: "${LLM_API_KEY}"
  api_url: "${LLM_API_URL}"      # 自定义 endpoint（可选）
  temperature: 0.1               # 低温度保证确定性
  max_tokens: 512
  confidence_threshold: 0.7      # 置信度低于此值不采信
  max_concurrent: 5              # 最大并发调用数
  timeout: 30                    # 单次调用超时（秒）

# ---- 实时监控引擎 ----
monitor:
  interval_sec: 10               # 行情轮询间隔（秒）
  trading_start: "09:25"         # 集合竞价结束，开始监控
  trading_end: "15:01"           # 收盘后1分钟停止监控
  strong_threshold: 5.0          # 强势股涨幅阈值（%）
  shard_count: 16                # 股票分片数（goroutine 并发数）
  error_threshold_pct: 30        # 单轮失败率阈值（%），超过触发熔断
  circuit_breaker:
    consecutive_failures: 5      # 连续失败次数触发熔断
    cooldown_sec: 60             # 熔断冷却时间（秒）
    half_open_requests: 3        # 半开状态允许的探测请求数

# ---- WebSocket 推送 ----
websocket:
  ping_interval: 50              # Ping 发送间隔（秒）
  pong_timeout: 60               # Pong 等待超时（秒）
  write_buffer_size: 256         # 发送缓冲区大小（条消息数）
  max_connections: 1000          # 最大同时连接数
  write_wait: 10                 # 写操作超时（秒）
  max_message_size: 4096         # 单条消息最大字节

# ---- 定时任务调度 ----
scheduler:
  pre_market_init: "0 15 9 * * 1-5"       # 盘前初始化: 每交易日 09:15
  realtime_collect: "*/10 * 9-14 * * 1-5"  # 实时采集: 09:25-15:00 每10秒（业务层二次过滤时段）
  jiuyan_sync: "0 0 9 * * 1-5"             # 韭研热点同步: 每交易日 09:00
  concept_sync: "0 0 2 * * 0"              # 概念板块同步: 每周日 02:00
  closing_snapshot: "0 5 15 * * 1-5"       # 收盘快照: 每交易日 15:05
  history_cleanup: "0 0 3 1 * *"           # 历史清理: 每月1日 03:00
  cache_warmup: "0 10 9 * * 1-5"           # 缓存预热: 每交易日 09:10
  llm_batch: "0 */5 9-14 * * 1-5"          # LLM 批处理: 09:30-15:00 每5分钟（业务层二次过滤）

# ---- 重试策略 ----
retry:
  max_retries: 4                 # 最大重试次数
  initial_delay_ms: 1000         # 首次重试延迟（毫秒）
  multiplier: 2.0                # 退避倍数
  max_delay_ms: 8000             # 最大延迟上限（毫秒）

# ---- 日志 ----
log:
  level: info                    # debug / info / warn / error
  format: json                   # json / console
  output: stdout                 # stdout / stderr / 文件路径
  rotation:
    max_size: 100                # 单文件最大（MB）
    max_age: 30                  # 保留天数
    max_backups: 10              # 保留文件数
    compress: true               # 是否压缩归档
```

**注意事项**：
- `scheduler` 中的 cron 表达式为 6 位格式（秒 分 时 日 月 周），与 `robfig/cron/v3` 的 `cron.WithSeconds()` 对应。
- `realtime_collect` 和 `llm_batch` 的 cron 仅提供基础触发频率，业务层需对 `monitor.trading_start` / `monitor.trading_end` 做二次时段过滤。
- 所有 `${VAR}` 占位符由 viper 的 `AutomaticEnv()` 或部署时的环境变量注入。

---

### 5.2 Go Config 结构体

所有结构体位于 `internal/config/config.go`，与上述 config.yaml 逐字段对应。
共 13 个结构体（含 1 个根结构体 + 12 个子结构体），每个字段都有 `yaml` tag 和注释。

```go
// internal/config/config.go
package config

import "fmt"

// ─── 根结构体 ───────────────────────────────────────────────

// AppConfig 是应用程序的根配置结构，包含所有子系统的配置。
// 对应 config.yaml 顶层 key。
type AppConfig struct {
    Server    ServerConfig    `yaml:"server"`
    Database  DatabaseConfig  `yaml:"database"`
    Redis     RedisConfig     `yaml:"redis"`
    Tushare   TushareConfig   `yaml:"tushare"`
    Jiuyan    JiuyanConfig    `yaml:"jiuyan"`
    LLM       LLMConfig       `yaml:"llm"`
    Monitor   MonitorConfig   `yaml:"monitor"`
    WebSocket WebSocketConfig `yaml:"websocket"`
    Scheduler SchedulerConfig `yaml:"scheduler"`
    Retry     RetryConfig     `yaml:"retry"`
    Log       LogConfig       `yaml:"log"`
}

// ─── 子结构体 1: ServerConfig ───────────────────────────────

// ServerConfig — HTTP 服务器配置。
type ServerConfig struct {
    Port         int    `yaml:"port"`          // 监听端口，默认 8080
    Mode         string `yaml:"mode"`          // gin 运行模式: debug / release / test
    ReadTimeout  int    `yaml:"read_timeout"`  // HTTP 读超时（秒）
    WriteTimeout int    `yaml:"write_timeout"` // HTTP 写超时（秒）
    APIKey       string `yaml:"api_key"`       // X-API-Key 鉴权密钥
}

// ─── 子结构体 2: DatabaseConfig ─────────────────────────────

// DatabaseConfig — PostgreSQL 连接与连接池配置。
type DatabaseConfig struct {
    Host            string `yaml:"host"`
    Port            int    `yaml:"port"`
    User            string `yaml:"user"`
    Password        string `yaml:"password"`
    DBName          string `yaml:"dbname"`
    SSLMode         string `yaml:"sslmode"`           // disable / require / verify-full
    Timezone        string `yaml:"timezone"`           // 时区，默认 Asia/Shanghai
    MaxOpenConns    int    `yaml:"max_open_conns"`     // 最大打开连接数
    MaxIdleConns    int    `yaml:"max_idle_conns"`     // 最大空闲连接数
    ConnMaxLifetime int    `yaml:"conn_max_lifetime"`  // 连接最大存活时间（秒）
    ConnMaxIdleTime int    `yaml:"conn_max_idle_time"` // 空闲连接最大存活时间（秒）
}

// DSN 返回 pgx 连接字符串。
func (c *DatabaseConfig) DSN() string {
    return fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
        c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode, c.Timezone,
    )
}

// ─── 子结构体 3: RedisConfig ────────────────────────────────

// RedisConfig — Redis 连接与连接池配置。
type RedisConfig struct {
    Addr         string `yaml:"addr"`           // 地址，如 localhost:6379
    Password     string `yaml:"password"`
    DB           int    `yaml:"db"`             // 数据库编号
    PoolSize     int    `yaml:"pool_size"`      // 连接池大小
    MinIdleConns int    `yaml:"min_idle_conns"` // 最小空闲连接
    DialTimeout  int    `yaml:"dial_timeout"`   // 连接超时（秒）
    ReadTimeout  int    `yaml:"read_timeout"`   // 读超时（秒）
    WriteTimeout int    `yaml:"write_timeout"`  // 写超时（秒）
}

// ─── 子结构体 4: TushareConfig ──────────────────────────────

// TushareConfig — Tushare Pro 数据源配置。
type TushareConfig struct {
    BaseURL         string `yaml:"base_url"`           // API 地址
    Token           string `yaml:"token"`              // 授权 Token
    RateLimitPerMin int    `yaml:"rate_limit_per_min"` // 每分钟最大请求数
    Timeout         int    `yaml:"timeout"`            // 单次请求超时（秒）
}

// ─── 子结构体 5: JiuyanConfig ───────────────────────────────

// JiuyanConfig — 韭研公社数据源配置。
type JiuyanConfig struct {
    BaseURL        string `yaml:"base_url"`          // API 基础地址
    APIKey         string `yaml:"api_key"`           // 授权密钥
    Timeout        int    `yaml:"timeout"`           // 单次请求超时（秒）
    CrawlStartDate string `yaml:"crawl_start_date"` // 全量爬取起始日期，格式 YYYY-MM-DD
}

// ─── 子结构体 6: LLMConfig ──────────────────────────────────

// LLMConfig — LLM 兜底分类配置。
type LLMConfig struct {
    Provider            string  `yaml:"provider"`             // openai / azure / local
    Model               string  `yaml:"model"`                // 模型名称
    APIKey              string  `yaml:"api_key"`
    APIURL              string  `yaml:"api_url"`              // 自定义 endpoint
    Temperature         float64 `yaml:"temperature"`          // 生成温度
    MaxTokens           int     `yaml:"max_tokens"`           // 最大生成 token 数
    ConfidenceThreshold float64 `yaml:"confidence_threshold"` // 置信度阈值，低于此值不采信
    MaxConcurrent       int     `yaml:"max_concurrent"`       // 最大并发调用数
    Timeout             int     `yaml:"timeout"`              // 单次调用超时（秒）
}

// ─── 子结构体 7: CircuitBreakerConfig ───────────────────────

// CircuitBreakerConfig — 熔断器配置。
type CircuitBreakerConfig struct {
    ConsecutiveFailures int `yaml:"consecutive_failures"` // 连续失败次数触发熔断
    CooldownSec         int `yaml:"cooldown_sec"`         // 熔断冷却时间（秒）
    HalfOpenRequests    int `yaml:"half_open_requests"`   // 半开状态探测请求数
}

// ─── 子结构体 8: MonitorConfig ──────────────────────────────

// MonitorConfig — 实时监控引擎配置。
type MonitorConfig struct {
    IntervalSec       int                  `yaml:"interval_sec"`        // 行情轮询间隔（秒）
    TradingStart      string               `yaml:"trading_start"`       // 监控开始时间 HH:MM
    TradingEnd        string               `yaml:"trading_end"`         // 监控结束时间 HH:MM
    StrongThreshold   float64              `yaml:"strong_threshold"`    // 强势股涨幅阈值（%）
    ShardCount        int                  `yaml:"shard_count"`         // 股票分片数
    ErrorThresholdPct int                  `yaml:"error_threshold_pct"` // 单轮失败率阈值（%）
    CircuitBreaker    CircuitBreakerConfig `yaml:"circuit_breaker"`     // 熔断器配置
}

// ─── 子结构体 9: WebSocketConfig ────────────────────────────

// WebSocketConfig — WebSocket 推送配置。
type WebSocketConfig struct {
    PingInterval    int `yaml:"ping_interval"`     // Ping 间隔（秒）
    PongTimeout     int `yaml:"pong_timeout"`      // Pong 超时（秒）
    WriteBufferSize int `yaml:"write_buffer_size"` // 发送缓冲区大小（条）
    MaxConnections  int `yaml:"max_connections"`   // 最大连接数
    WriteWait       int `yaml:"write_wait"`        // 写操作超时（秒）
    MaxMessageSize  int `yaml:"max_message_size"`  // 单条消息最大字节
}

// ─── 子结构体 10: SchedulerConfig ───────────────────────────

// SchedulerConfig — 定时任务 cron 表达式（6位格式: 秒 分 时 日 月 周）。
// 必须配合 robfig/cron/v3 的 cron.WithSeconds() 使用。
type SchedulerConfig struct {
    PreMarketInit   string `yaml:"pre_market_init"`   // 盘前初始化
    RealtimeCollect string `yaml:"realtime_collect"`   // 实时行情采集
    JiuyanSync      string `yaml:"jiuyan_sync"`        // 韭研热点同步
    ConceptSync     string `yaml:"concept_sync"`       // 概念板块同步
    ClosingSnapshot string `yaml:"closing_snapshot"`   // 收盘快照
    HistoryCleanup  string `yaml:"history_cleanup"`    // 历史数据清理
    CacheWarmup     string `yaml:"cache_warmup"`       // 缓存预热
    LLMBatch        string `yaml:"llm_batch"`          // LLM 待分类批处理
}

// ─── 子结构体 11: RetryConfig ───────────────────────────────

// RetryConfig — 指数退避重试策略配置。
type RetryConfig struct {
    MaxRetries     int     `yaml:"max_retries"`      // 最大重试次数
    InitialDelayMs int     `yaml:"initial_delay_ms"` // 首次重试延迟（毫秒）
    Multiplier     float64 `yaml:"multiplier"`       // 退避倍数
    MaxDelayMs     int     `yaml:"max_delay_ms"`     // 最大延迟上限（毫秒）
}

// ─── 子结构体 12: LogRotationConfig ─────────────────────────

// LogRotationConfig — 日志轮转配置。
type LogRotationConfig struct {
    MaxSize    int  `yaml:"max_size"`    // 单文件最大（MB）
    MaxAge     int  `yaml:"max_age"`     // 保留天数
    MaxBackups int  `yaml:"max_backups"` // 保留文件数
    Compress   bool `yaml:"compress"`    // 是否压缩归档
}

// ─── 子结构体 13: LogConfig ─────────────────────────────────

// LogConfig — 日志配置。
type LogConfig struct {
    Level    string            `yaml:"level"`    // debug / info / warn / error
    Format   string            `yaml:"format"`   // json / console
    Output   string            `yaml:"output"`   // stdout / stderr / 文件路径
    Rotation LogRotationConfig `yaml:"rotation"` // 日志轮转配置
}
```

**加载入口**（供 `main.go` 调用）：

```go
// internal/config/loader.go
package config

import (
    "fmt"

    "github.com/spf13/viper"
)

// Load 从 config.yaml 加载配置，支持环境变量覆盖。
// 环境变量自动绑定：如 DATABASE_HOST 覆盖 database.host。
func Load(path string) (*AppConfig, error) {
    viper.SetConfigFile(path)
    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }

    var cfg AppConfig
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }
    return &cfg, nil
}
```

---

### 5.3 定时任务清单

系统使用 `robfig/cron/v3` 调度器，所有 cron 表达式为 **6 位格式**（秒 分 时 日 月 周），需在创建调度器时调用 `cron.WithSeconds()`。

交易日过滤（排除节假日）由业务层在任务函数入口处判断，cron 仅用 `1-5`（周一到周五）做粗过滤。

#### 任务 1：盘前初始化

| 属性 | 值 |
|---|---|
| 任务名 | `pre_market_init` |
| Cron 表达式 | `0 15 9 * * 1-5` |
| 触发时间 | 每周一至周五 09:15:00 |
| 执行内容 | (1) 检查是否为交易日（查交易日历），非交易日直接跳过；(2) 清理前日 Redis 临时 key（`first_limit:{prev_date}`、`alerted:{prev_date}` 等）；(3) 初始化当日 Redis 数据结构（涨停池 Set、5% 池 Set、活跃度 SortedSet）；(4) 从 PG 加载板块规则到内存缓存 |

```go
c.AddFunc(cfg.Scheduler.PreMarketInit, func() {
    if !tradingCalendar.IsTradingDay(time.Now()) {
        return
    }
    ctx := context.Background()
    if err := scheduler.PreMarketInit(ctx); err != nil {
        logger.Error("pre_market_init failed", zap.Error(err))
    }
})
```

#### 任务 2：实时行情采集

| 属性 | 值 |
|---|---|
| 任务名 | `realtime_collect` |
| Cron 表达式 | `*/10 * 9-14 * * 1-5` |
| 触发时间 | 每周一至周五，09:00-14:59 范围内每 10 秒触发一次 |
| 执行内容 | (1) 业务层二次过滤：仅在 `trading_start`(09:25) 到 `trading_end`(15:01) 之间实际执行；(2) 将约 5000 只股票按 `ts_code` 哈希分成 16 个 shard；(3) 每个 shard 由独立 goroutine 并发拉取 Tushare 行情；(4) 逐股判定涨停/5%，触发分类与告警；(5) 计算池子 diff，推送 WebSocket |

```go
c.AddFunc(cfg.Scheduler.RealtimeCollect, func() {
    now := time.Now()
    if !tradingCalendar.IsTradingDay(now) {
        return
    }
    if !monitor.InTradingWindow(now) {
        return // 二次过滤：仅 09:25-15:01
    }
    ctx := context.Background()
    if err := monitorService.RunOnce(ctx); err != nil {
        logger.Error("realtime_collect failed", zap.Error(err))
    }
})
```

#### 任务 3：韭研热点同步

| 属性 | 值 |
|---|---|
| 任务名 | `jiuyan_sync` |
| Cron 表达式 | `0 0 9 * * 1-5` |
| 触发时间 | 每周一至周五 09:00:00 |
| 执行内容 | (1) 调用韭研公社 API 获取前一交易日的热点数据；(2) 与已有热点库比对，新热点自动入库（先查同义词表归一化）；(3) 更新 `stock_topic_mappings` 的 `hit_count` 和 `last_seen_date`；(4) 写入 `jiuyan_crawl_logs` 记录本次爬取结果；(5) 失败时指数退避重试（1s->2s->4s->8s），超过 4 次标记失败 |

```go
c.AddFunc(cfg.Scheduler.JiuyanSync, func() {
    if !tradingCalendar.IsTradingDay(time.Now()) {
        return
    }
    ctx := context.Background()
    prevDate := tradingCalendar.PrevTradingDay(time.Now())
    if err := crawlerService.SyncDaily(ctx, prevDate); err != nil {
        logger.Error("jiuyan_sync failed", zap.Error(err), zap.String("date", prevDate))
    }
})
```

#### 任务 4：概念板块同步

| 属性 | 值 |
|---|---|
| 任务名 | `concept_sync` |
| Cron 表达式 | `0 0 2 * * 0` |
| 触发时间 | 每周日 02:00:00 |
| 执行内容 | (1) 调用 Tushare `concept()` 获取最新概念板块列表；(2) 对比本地已有概念，发现新增概念时调用 `concept_detail()` 获取成分股；(3) 增量更新 `stock_concept_tags` 表（ON CONFLICT DO NOTHING）；(4) 对新概念尝试自动精确匹配到已有热点（更新 `concept_topic_mappings`）；(5) 刷新 Redis 概念缓存 `cache:stock_concepts` 和 `cache:concept_to_topics` |

```go
c.AddFunc(cfg.Scheduler.ConceptSync, func() {
    ctx := context.Background()
    if err := conceptSyncService.IncrementalSync(ctx); err != nil {
        logger.Error("concept_sync failed", zap.Error(err))
    }
})
```

#### 任务 5：收盘快照保存

| 属性 | 值 |
|---|---|
| 任务名 | `closing_snapshot` |
| Cron 表达式 | `0 5 15 * * 1-5` |
| 触发时间 | 每周一至周五 15:05:00 |
| 执行内容 | (1) 从 Redis 读取当日最终的涨停池和 5% 池数据；(2) 批量写入 PG `daily_stock_pool` 表（使用 `ON CONFLICT (date, ts_code, pool_type) DO UPDATE` 保证幂等）；(3) 保存当前全量快照 JSON 到 `snapshot:current`；(4) 停止实时监控循环 |

```go
c.AddFunc(cfg.Scheduler.ClosingSnapshot, func() {
    if !tradingCalendar.IsTradingDay(time.Now()) {
        return
    }
    ctx := context.Background()
    if err := snapshotService.SaveDailySnapshot(ctx, time.Now()); err != nil {
        logger.Error("closing_snapshot failed", zap.Error(err))
    }
})
```

#### 任务 6：历史数据清理

| 属性 | 值 |
|---|---|
| 任务名 | `history_cleanup` |
| Cron 表达式 | `0 0 3 1 * *` |
| 触发时间 | 每月 1 日 03:00:00 |
| 执行内容 | (1) 自动创建下月的 `daily_stock_pool` 分区表（如 `daily_stock_pool_2026_04`）；(2) 清理超过 90 天的 Redis 临时数据（过期 key 自动清除，此处做兜底扫描）；(3) 清理 `classify_evidence` 中超过 180 天且未被标记为训练数据的记录（可选，视存储压力决定）；(4) 输出清理报告到日志 |

```go
c.AddFunc(cfg.Scheduler.HistoryCleanup, func() {
    ctx := context.Background()
    // 创建下月分区表
    if err := partitionService.EnsureNextMonthPartition(ctx); err != nil {
        logger.Error("create_partition failed", zap.Error(err))
    }
    // 兜底清理
    if err := cleanupService.CleanupExpiredData(ctx); err != nil {
        logger.Error("history_cleanup failed", zap.Error(err))
    }
})
```

#### 任务 7：Redis 缓存预热

| 属性 | 值 |
|---|---|
| 任务名 | `cache_warmup` |
| Cron 表达式 | `0 10 9 * * 1-5` |
| 触发时间 | 每周一至周五 09:10:00 |
| 执行内容 | (1) 从 PG `stock_topic_mappings` 全量加载股票-热点映射到 Redis `cache:stock_topics`；(2) 加载关联强度数据到 `cache:bind_strength`；(3) 加载概念标签缓存到 `cache:stock_concepts`；(4) 加载概念-热点映射到 `cache:concept_to_topics`；(5) 加载同义词表到内存 map；(6) 加载板块规则到内存缓存；(7) 加载今日关注热点到 `focus:topics:{date}` |

```go
c.AddFunc(cfg.Scheduler.CacheWarmup, func() {
    if !tradingCalendar.IsTradingDay(time.Now()) {
        return
    }
    ctx := context.Background()
    if err := cacheService.WarmupAll(ctx); err != nil {
        logger.Error("cache_warmup failed", zap.Error(err))
    }
})
```

#### 任务 8：LLM 待分类批处理

| 属性 | 值 |
|---|---|
| 任务名 | `llm_batch` |
| Cron 表达式 | `0 */5 9-14 * * 1-5` |
| 触发时间 | 每周一至周五，09:00-14:59 范围内每 5 分钟触发一次 |
| 执行内容 | (1) 业务层二次过滤：仅在 09:30-15:00 之间实际执行；(2) 从 Redis 取出标记为"未分类"的股票队列；(3) 按批次（每批最多 5 个并发）调用 LLM API 进行分类；(4) 置信度 >= 0.7 的结果写入 `stock_topic_mappings` 并回填 Redis 缓存；(5) 置信度 < 0.7 的结果退回"未分类"状态，记录到 `classify_evidence`；(6) LLM 不可用时直接跳过，不阻塞 |

```go
c.AddFunc(cfg.Scheduler.LLMBatch, func() {
    now := time.Now()
    if !tradingCalendar.IsTradingDay(now) {
        return
    }
    if now.Hour() < 9 || (now.Hour() == 9 && now.Minute() < 30) {
        return // 09:30 之前不执行
    }
    ctx := context.Background()
    if err := llmBatchService.ProcessPending(ctx); err != nil {
        logger.Warn("llm_batch failed, non-blocking", zap.Error(err))
    }
})
```

#### 调度器注册完整示例

```go
// internal/scheduler/setup.go
package scheduler

import (
    "context"
    "log"
    "os"
    "time"

    "stock-monitor/internal/cache"
    "stock-monitor/internal/config"
    "stock-monitor/internal/service"

    "github.com/robfig/cron/v3"
    "go.uber.org/zap"
)

// SetupScheduler 创建并注册所有定时任务。返回的 *cron.Cron 由调用方
// 调用 Start() 启动，程序退出时调用 Stop() 优雅关闭。
func SetupScheduler(
    cfg *config.SchedulerConfig,
    logger *zap.Logger,
    tradingCalendar *TradingCalendar,
    monitorService *service.MonitorService,
    crawlerService *service.CrawlerService,
    conceptSyncService *service.ConceptSyncService,
    snapshotService *service.SnapshotService,
    cacheService *cache.CacheService,
    llmBatchService *service.LLMBatchService,
    partitionService *service.PartitionService,
    cleanupService *service.CleanupService,
) *cron.Cron {
    // 6 位 cron 格式：秒 分 时 日 月 周
    c := cron.New(
        cron.WithSeconds(),
        cron.WithLogger(
            cron.VerbosePrintfLogger(
                log.New(os.Stdout, "cron: ", log.LstdFlags),
            ),
        ),
    )

    // 1. 盘前初始化 — 每交易日 09:15
    c.AddFunc(cfg.PreMarketInit, func() { /* 见上方任务 1 代码 */ })

    // 2. 实时行情采集 — 每 10 秒（业务层二次过滤 09:25-15:01）
    c.AddFunc(cfg.RealtimeCollect, func() { /* 见上方任务 2 代码 */ })

    // 3. 韭研热点同步 — 每交易日 09:00
    c.AddFunc(cfg.JiuyanSync, func() { /* 见上方任务 3 代码 */ })

    // 4. 概念板块同步 — 每周日 02:00
    c.AddFunc(cfg.ConceptSync, func() { /* 见上方任务 4 代码 */ })

    // 5. 收盘快照保存 — 每交易日 15:05
    c.AddFunc(cfg.ClosingSnapshot, func() { /* 见上方任务 5 代码 */ })

    // 6. 历史数据清理 — 每月 1 日 03:00
    c.AddFunc(cfg.HistoryCleanup, func() { /* 见上方任务 6 代码 */ })

    // 7. Redis 缓存预热 — 每交易日 09:10
    c.AddFunc(cfg.CacheWarmup, func() { /* 见上方任务 7 代码 */ })

    // 8. LLM 待分类批处理 — 每 5 分钟（业务层二次过滤 09:30-15:00）
    c.AddFunc(cfg.LLMBatch, func() { /* 见上方任务 8 代码 */ })

    return c
}
```

---

### 5.4 数据初始化流程

系统首次启动（或全量重建）时，按以下 8 步顺序执行初始化。每步标注了前置依赖，AI agent 在生成初始化代码时必须保证执行顺序。

| 步骤 | 任务 | 依赖 | 说明 |
|---|---|---|---|
| **Step 1** | 检查 PG 连接 | 无 | 建立 pgx 连接池，验证连通性。失败则程序直接退出（fatal），不进入后续步骤 |
| **Step 2** | 执行数据库迁移 | Step 1 | 按序执行 `migrations/001_create_boards.sql` 至 `009_create_crawl_logs.sql`，建立所有表、索引、约束、分区。使用迁移框架（golang-migrate）保证幂等 |
| **Step 3** | 同步股票基础信息 | Step 2 | 调用 Tushare `stock_basic` 接口获取全市场约 5300 只股票信息，写入 `stocks` 表。根据代码前缀自动判定 `board_code`（300/301→GEM，688→STAR，8/4→BSE，其他→MAIN）。同时初始化 `boards` 表的 4 条板块规则 |
| **Step 4** | 同步概念板块数据 | Step 2 | 调用 Tushare `concept()` 获取约 400 个概念板块列表，再逐个调用 `concept_detail(id=xxx)` 获取成分股，写入 `stock_concept_tags` 表。注意 Tushare 频率限制（500次/分钟），需控制调用节奏 |
| **Step 5** | 同步韭研热点 | Step 2 | 全量爬取韭研公社数据（从 `crawl_start_date` 至今的每个交易日），写入 `hot_topics`、`stock_topic_mappings`、`topic_synonyms`、`jiuyan_crawl_logs` 表。使用指数退避重试，失败日期记录到 `jiuyan_crawl_logs` 待后续补爬 |
| **Step 6** | 加载热点同义词到 Redis | Step 5 | 从 `topic_synonyms` 全量加载同义词映射关系到内存 map，供分类引擎和爬虫入库使用。同时建立概念→热点映射（`concept_topic_mappings`）：先自动精确匹配，再 LLM 批量匹配未命中的概念 |
| **Step 7** | 加载分类缓存到 Redis | Step 4, Step 5 | 将 `stock_topic_mappings` 全量加载到 `cache:stock_topics`，将关联强度数据加载到 `cache:bind_strength`，将概念标签加载到 `cache:stock_concepts`，将概念-热点映射加载到 `cache:concept_to_topics` |
| **Step 8** | 启动 WebSocket Hub | Step 1 | 初始化 WebSocket Hub 连接管理中心，开始接受客户端连接。启动后台 goroutine 处理注册/注销/广播。此步骤不依赖数据填充，但逻辑上放在最后，确保推送时数据已就绪 |

**依赖关系图**（文本表示）：

```
Step 1 (PG连接)
  ├── Step 2 (数据库迁移)
  │     ├── Step 3 (股票基础信息)  ─────────┐
  │     ├── Step 4 (概念板块数据)  ──────────┤
  │     └── Step 5 (韭研热点)               │
  │           └── Step 6 (同义词+概念映射)   │
  │                 └── Step 7 (Redis缓存) ←─┘  ← 同时依赖 Step 4
  └── Step 8 (WebSocket Hub)  ← 可与 Step 2-7 并行，但建议放最后
```

**关键约束**：
- Step 3 / Step 4 / Step 5 仅依赖 Step 2（表已创建），彼此之间无依赖，理论上可并行，但为控制 Tushare API 频率建议串行执行。
- Step 6 强依赖 Step 5（同义词数据来自韭研爬虫），Step 7 强依赖 Step 4 和 Step 5（需要概念标签和映射数据都就绪）。
- Step 4 和 Step 5 中任一步骤**部分失败不阻塞**后续步骤（失败记录已持久化，可后续补偿），但 Step 1、Step 2 失败则 fatal 退出。

**初始化代码骨架**：

```go
// cmd/server/main.go — 初始化部分
func initialize(ctx context.Context, cfg *config.AppConfig) error {
    // Step 1: 检查 PG 连接
    pool, err := pgxpool.New(ctx, cfg.Database.DSN())
    if err != nil {
        return fmt.Errorf("step1 pg connect: %w", err) // fatal
    }
    if err := pool.Ping(ctx); err != nil {
        return fmt.Errorf("step1 pg ping: %w", err)
    }
    log.Info("Step 1/8: PG connection OK")

    // Step 2: 数据库迁移
    if err := migrator.Up(pool); err != nil {
        return fmt.Errorf("step2 migration: %w", err)
    }
    log.Info("Step 2/8: database migration OK")

    // Step 3: 同步股票基础信息
    if err := stockService.SyncStockBasic(ctx); err != nil {
        return fmt.Errorf("step3 stock_basic: %w", err)
    }
    log.Info("Step 3/8: stock basic sync OK")

    // Step 4: 同步概念板块数据（部分失败不阻塞）
    if err := conceptSyncService.FullSync(ctx); err != nil {
        log.Warn("Step 4/8: concept sync partial failure", zap.Error(err))
    } else {
        log.Info("Step 4/8: concept sync OK")
    }

    // Step 5: 同步韭研热点（全量爬取，部分失败不阻塞）
    if err := crawlerService.FullCrawl(ctx); err != nil {
        log.Warn("Step 5/8: jiuyan crawl partial failure", zap.Error(err))
    } else {
        log.Info("Step 5/8: jiuyan crawl OK")
    }

    // Step 6: 加载同义词 + 建立概念→热点映射
    if err := synonymService.LoadToMemory(ctx); err != nil {
        return fmt.Errorf("step6 synonyms: %w", err)
    }
    if err := conceptMappingService.BuildMappings(ctx); err != nil {
        log.Warn("Step 6/8: concept mapping partial failure", zap.Error(err))
    }
    log.Info("Step 6/8: synonyms and concept mappings OK")

    // Step 7: 加载分类缓存到 Redis
    if err := cacheService.WarmupAll(ctx); err != nil {
        return fmt.Errorf("step7 cache warmup: %w", err)
    }
    log.Info("Step 7/8: Redis cache warmup OK")

    // Step 8: 启动 WebSocket Hub
    hub := ws.NewHub()
    go hub.Run()
    log.Info("Step 8/8: WebSocket Hub started")

    return nil
}
```

---

### 5.5 容错规则（RULE 声明）

以下 5 条 RULE 是系统级容错契约。AI agent 在生成任何涉及外部调用、并发处理、实时推送的代码时，**必须遵循对应的 RULE**。每条 RULE 包含适用范围、行为描述和参考实现。

---

#### RULE 1: retry_backoff — 指数退避重试

```
RULE retry_backoff:
  APPLIES_TO: 所有外部 API 调用（Tushare、韭研、LLM）
  STRATEGY: exponential backoff
  DELAYS: [1s, 2s, 4s, 8s]   # initial=1s, multiplier=2, max=8s
  MAX_RETRIES: 4
  BEHAVIOR:
    - 每次重试前等待 delay = min(initial * multiplier^attempt, max_delay)
    - 仅对可重试错误重试（网络超时、5xx、429 rate limit）
    - 4xx 客户端错误（除 429）不重试，立即返回
    - 所有重试记录结构化日志（attempt, delay, error）
  GO_REFERENCE: internal/pkg/retry/backoff.go
```

**Go 实现签名**：

```go
// internal/pkg/retry/backoff.go
package retry

import (
    "context"
    "fmt"
    "time"
)

// RetryWithBackoff 执行带指数退避的重试。
// 仅对 isRetryable(err)==true 的错误进行重试。
func RetryWithBackoff(ctx context.Context, cfg config.RetryConfig, operation func() error) error {
    var lastErr error
    delay := time.Duration(cfg.InitialDelayMs) * time.Millisecond

    for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
        if err := operation(); err == nil {
            return nil
        } else {
            lastErr = err
            if !isRetryable(err) {
                return err // 4xx（非 429）不重试
            }
            if attempt < cfg.MaxRetries {
                select {
                case <-time.After(delay):
                case <-ctx.Done():
                    return ctx.Err()
                }
                delay = time.Duration(float64(delay) * cfg.Multiplier)
                maxDelay := time.Duration(cfg.MaxDelayMs) * time.Millisecond
                if delay > maxDelay {
                    delay = maxDelay
                }
            }
        }
    }
    return fmt.Errorf("max retries (%d) exceeded: %w", cfg.MaxRetries, lastErr)
}

// isRetryable 判断错误是否可重试。
func isRetryable(err error) bool {
    // 网络超时、5xx、429 → 可重试
    // 4xx（除 429）→ 不可重试
    // 具体实现根据 error 类型判断
    return true // 占位，实际需检查 HTTP status code
}
```

---

#### RULE 2: per_stock_isolation — 单股异常隔离

```
RULE per_stock_isolation:
  APPLIES_TO: MonitorService 每 10 秒行情处理循环
  STRATEGY: per-item error isolation
  BEHAVIOR:
    - 每只股票的行情获取/判定/分类在独立的 recover 块中执行
    - 单股 panic 或返回 error 时：
      1. 记录错误日志（含 ts_code、shard_id、error）
      2. Redis 递增 stock:error_count:{date}:{ts_code}
      3. 跳过该股票，继续处理同 shard 内的下一只
    - 连续失败 5 次的股票触发单股异常告警
    - 单轮失败率超过 error_threshold_pct(30%) 时标记本轮整体异常
  GO_REFERENCE: internal/service/monitor.go → processStock()
```

**Go 实现模式**：

```go
// internal/service/monitor.go
func (s *MonitorService) processShard(ctx context.Context, shardID int, stocks []model.Stock) (failCount int) {
    for _, stock := range stocks {
        if err := s.processOneStock(ctx, stock); err != nil {
            failCount++
            s.logger.Error("stock processing failed",
                zap.String("ts_code", stock.TsCode),
                zap.Int("shard_id", shardID),
                zap.Error(err),
            )
            // Redis 递增失败计数
            key := fmt.Sprintf("stock:error_count:%s:%s", today, stock.TsCode)
            count, _ := s.redis.Incr(ctx, key).Result()
            if count >= 5 {
                s.logger.Warn("stock consecutive failures",
                    zap.String("ts_code", stock.TsCode),
                    zap.Int64("count", count),
                )
            }
            continue // 不影响其他股票
        }
    }
    return failCount
}

func (s *MonitorService) processOneStock(ctx context.Context, stock model.Stock) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic in processOneStock(%s): %v", stock.TsCode, r)
        }
    }()
    // ... 行情获取 → 涨停判定 → 分类归因 → 告警检查
    return nil
}
```

---

#### RULE 3: circuit_breaker — 熔断器

```
RULE circuit_breaker:
  APPLIES_TO: 外部数据源调用（Tushare 行情、韭研 API）
  STRATEGY: circuit breaker (closed → open → half-open → closed)
  THRESHOLDS:
    consecutive_failures: 5   # 连续 5 次失败触发熔断
    cooldown_sec: 60          # 熔断后 60 秒不发起调用
    half_open_requests: 3     # 半开状态允许 3 次探测请求
  STATE_MACHINE:
    CLOSED:    正常调用。连续失败达到阈值 → 切换到 OPEN
    OPEN:      直接返回 ErrCircuitOpen，不发起实际调用。
               等待 cooldown_sec → 切换到 HALF_OPEN
    HALF_OPEN: 允许 half_open_requests 次探测调用。
               任一成功 → 切换到 CLOSED，重置计数
               任一失败 → 切换回 OPEN，重新等待
  BEHAVIOR:
    - 熔断触发时记录 WARN 日志，包含累计失败次数和最后错误
    - 熔断期间 MonitorService 使用 Redis 中的上一轮缓存数据
    - 熔断恢复时记录 INFO 日志
  GO_REFERENCE: internal/pkg/retry/circuit_breaker.go
```

**Go 实现骨架**：

```go
// internal/pkg/retry/circuit_breaker.go
package retry

import (
    "errors"
    "sync"
    "time"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type State int

const (
    StateClosed   State = iota // 正常
    StateOpen                  // 熔断
    StateHalfOpen              // 探测
)

type CircuitBreaker struct {
    mu                  sync.Mutex
    state               State
    consecutiveFailures int
    maxFailures         int
    cooldown            time.Duration
    halfOpenMax         int
    halfOpenCount       int
    lastFailureTime     time.Time
}

func (cb *CircuitBreaker) Execute(operation func() error) error {
    cb.mu.Lock()
    switch cb.state {
    case StateOpen:
        if time.Since(cb.lastFailureTime) > cb.cooldown {
            cb.state = StateHalfOpen
            cb.halfOpenCount = 0
        } else {
            cb.mu.Unlock()
            return ErrCircuitOpen
        }
    case StateHalfOpen:
        if cb.halfOpenCount >= cb.halfOpenMax {
            cb.mu.Unlock()
            return ErrCircuitOpen
        }
        cb.halfOpenCount++
    }
    cb.mu.Unlock()

    err := operation()

    cb.mu.Lock()
    defer cb.mu.Unlock()
    if err != nil {
        cb.consecutiveFailures++
        cb.lastFailureTime = time.Now()
        if cb.consecutiveFailures >= cb.maxFailures {
            cb.state = StateOpen
        }
        return err
    }
    // 成功：重置
    cb.consecutiveFailures = 0
    cb.state = StateClosed
    return nil
}
```

---

#### RULE 4: llm_fallback — LLM 不可用降级

```
RULE llm_fallback:
  APPLIES_TO: 分类引擎第 4 层 LLM 异步调用、LLM 批处理任务
  STRATEGY: graceful degradation (skip, never block)
  BEHAVIOR:
    - LLM 调用超时（> 30s）或返回错误时：
      1. 将该股票标记为"未分类"（unclassified）
      2. 放入待重试队列，下一个 5 分钟批处理周期重试
      3. 不阻塞当前 10 秒监控循环
    - LLM 返回但置信度 < confidence_threshold(0.7) 时：
      1. 不采信分类结果，仍标记为"未分类"
      2. 记录到 classify_evidence（strategy=LLM_V1, confidence=实际值）
      3. 下轮若命中缓存或 PG 映射则正常分类
    - LLM 服务整体不可用（熔断状态）时：
      1. 所有第 4 层请求直接跳过
      2. 系统仍可通过前 3 层正常工作
      3. 记录 WARN 级别日志
  GUARANTEE: LLM 完全宕机不影响系统核心功能（涨停检测、前 3 层分类、告警推送）
  GO_REFERENCE: internal/service/classify.go → classifyByLLM()
```

**Go 实现模式**：

```go
// internal/service/classify.go — 第 4 层 LLM 兜底
func (s *ClassifyService) classifyByLLM(ctx context.Context, stock model.Stock, activeTopics []model.Topic) (*ClassifyResult, error) {
    // 检查 LLM 熔断器状态
    if s.llmCircuitBreaker.IsOpen() {
        s.logger.Warn("LLM circuit breaker open, skipping",
            zap.String("ts_code", stock.TsCode),
        )
        return nil, nil // 不阻塞，返回 nil 表示未分类
    }

    // 异步调用，带超时
    ctx, cancel := context.WithTimeout(ctx, time.Duration(s.cfg.LLM.Timeout)*time.Second)
    defer cancel()

    result, err := s.llmClient.Classify(ctx, stock, activeTopics)
    if err != nil {
        s.logger.Warn("LLM classify failed, marking unclassified",
            zap.String("ts_code", stock.TsCode),
            zap.Error(err),
        )
        s.addToPendingQueue(stock.TsCode) // 放入待重试队列
        return nil, nil                    // 不阻塞
    }

    // 置信度检查
    if result.Confidence < s.cfg.LLM.ConfidenceThreshold {
        s.logger.Info("LLM confidence too low, not adopted",
            zap.String("ts_code", stock.TsCode),
            zap.Float64("confidence", result.Confidence),
            zap.Float64("threshold", s.cfg.LLM.ConfidenceThreshold),
        )
        // 记录证据但不采信
        s.saveEvidence(ctx, stock.TsCode, result, "LLM_V1", false)
        return nil, nil
    }

    // 采信结果
    s.saveEvidence(ctx, stock.TsCode, result, "LLM_V1", true)
    return result, nil
}
```

---

#### RULE 5: websocket_backpressure — WebSocket 背压处理

```
RULE websocket_backpressure:
  APPLIES_TO: WebSocket Hub 向客户端推送消息
  STRATEGY: drop-oldest on buffer full, then disconnect
  BUFFER: write_buffer_size = 256 条消息（每个客户端独立缓冲区）
  BEHAVIOR:
    - Hub 向 Client 的 send channel 写入消息
    - 当 send channel 已满（客户端消费速度跟不上）时：
      1. 不阻塞 Hub 广播 goroutine（使用 select + default）
      2. 主动关闭该 Client 的连接
      3. 记录 WARN 日志（含 client_id、缓冲区使用量）
      4. 客户端需自行重连，重连后立刻收到全量 pool_snapshot
    - 策略告警消息（strategy_alert）优先级高于池子更新消息：
      使用独立的 alert channel，优先发送
  GUARANTEE: 单个慢客户端不会阻塞其他客户端的消息推送
  GO_REFERENCE: internal/ws/hub.go → broadcast()
```

**Go 实现模式**：

```go
// internal/ws/hub.go
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    logger     *zap.Logger
    mu         sync.RWMutex
}

type Client struct {
    ID   string
    hub  *Hub
    conn *websocket.Conn
    send chan []byte // 缓冲区大小 = config.websocket.write_buffer_size
}

func (h *Hub) broadcastMessage(msg []byte) {
    h.mu.RLock()
    defer h.mu.RUnlock()

    for client := range h.clients {
        select {
        case client.send <- msg:
            // 写入成功
        default:
            // 缓冲区满 → 关闭该客户端连接
            h.logger.Warn("client buffer full, disconnecting",
                zap.String("client_id", client.ID),
                zap.Int("buffer_cap", cap(client.send)),
            )
            close(client.send)
            delete(h.clients, client)
        }
    }
}

// NewClient 创建新客户端时使用配置中的缓冲区大小。
func NewClient(hub *Hub, conn *websocket.Conn, bufSize int) *Client {
    return &Client{
        ID:   generateClientID(),
        hub:  hub,
        conn: conn,
        send: make(chan []byte, bufSize),
    }
}
```

---

### 5.6 外部 API 规格

系统依赖 3 个外部数据源共 6 个接口。AI agent 在生成 `internal/external/` 下的客户端代码时，必须严格按照以下规格实现。

---

#### 5.6.1 Tushare Pro — 4 个接口

**通用调用方式**：所有 Tushare 接口统一使用 POST 方法，请求体为 JSON。

```
POST https://api.tushare.pro
Content-Type: application/json

{
  "api_name": "<接口名>",
  "token": "<tushare_token>",
  "params": { ... },
  "fields": "<逗号分隔的字段列表>"
}
```

**通用频率限制**：500 次/分钟（所有接口共享配额）。系统需维护全局令牌桶限流器。

**通用返回格式**：

```json
{
  "code": 0,
  "msg": "",
  "data": {
    "fields": ["field1", "field2"],
    "items": [
      ["value1", "value2"],
      ["value3", "value4"]
    ]
  }
}
```

`code=0` 表示成功，其他值为错误。`items` 为二维数组，每行与 `fields` 一一对应。

---

**接口 1：stock_basic — 股票基础信息**

| 属性 | 值 |
|---|---|
| api_name | `stock_basic` |
| 用途 | 获取全市场股票列表（约 5300 只），写入 `stocks` 表 |
| 调用频率 | 初始化时调用 1 次，后续每月同步 1 次 |
| 请求参数 | `{"exchange": "", "list_status": "L"}` — 仅获取上市中的股票 |
| 请求字段 | `ts_code,symbol,name,area,industry,market,list_date,exchange,is_hs` |
| 返回关键字段 | `ts_code`(股票代码), `name`(名称), `industry`(行业), `list_date`(上市日期), `exchange`(交易所) |
| 容错 | 指数退避重试（RULE retry_backoff） |

```go
// internal/external/tushare/client.go
func (c *Client) StockBasic(ctx context.Context) ([]StockBasicItem, error) {
    req := &TushareRequest{
        APIName: "stock_basic",
        Token:   c.token,
        Params:  map[string]interface{}{"exchange": "", "list_status": "L"},
        Fields:  "ts_code,symbol,name,area,industry,market,list_date,exchange,is_hs",
    }
    return c.doRequest(ctx, req)
}
```

---

**接口 2：realtime_quote — 实时行情**

| 属性 | 值 |
|---|---|
| api_name | `realtime_quote` (或使用 Tushare 实时行情通道) |
| 用途 | 每 10 秒获取全市场实时行情，判断涨停/5% |
| 调用频率 | 交易时段内每 10 秒 1 次（按分片，每片约 50 只股票） |
| 请求参数 | `{"ts_code": "批量股票代码,逗号分隔"}` — 每批最多 50 只 |
| 请求字段 | `ts_code,pre_close,price,pct_chg,vol,amount,turnover_rate` |
| 返回关键字段 | `ts_code`(代码), `pre_close`(昨收), `price`(当前价), `pct_chg`(涨跌幅%), `vol`(成交量) |
| 容错 | 分片并发（RULE per_stock_isolation）+ 熔断（RULE circuit_breaker）+ 退避重试 |

```go
// internal/external/tushare/realtime.go
func (c *Client) RealtimeQuote(ctx context.Context, tsCodes []string) ([]QuoteItem, error) {
    // 分批请求，每批最多 50 只
    var allItems []QuoteItem
    for _, batch := range splitCodes(tsCodes, 50) {
        req := &TushareRequest{
            APIName: "realtime_quote",
            Token:   c.token,
            Params:  map[string]interface{}{"ts_code": strings.Join(batch, ",")},
            Fields:  "ts_code,pre_close,price,pct_chg,vol,amount,turnover_rate",
        }
        items, err := c.doRequestWithRetry(ctx, req)
        if err != nil {
            return allItems, err // 返回已获取的部分数据
        }
        allItems = append(allItems, items...)
    }
    return allItems, nil
}
```

---

**接口 3：concept — 概念板块列表**

| 属性 | 值 |
|---|---|
| api_name | `concept` |
| 用途 | 获取所有概念板块列表（约 400 个），建立基础概念标签体系 |
| 调用频率 | 初始化时调用 1 次，后续每周同步 1 次（周日 02:00） |
| 请求参数 | `{}` — 无额外参数 |
| 请求字段 | `id,name,src` |
| 返回关键字段 | `id`(概念代码,如 "TS0"), `name`(概念名称,如 "人工智能"), `src`(来源) |
| 容错 | 指数退避重试（RULE retry_backoff） |

```go
// internal/external/tushare/concept.go
func (c *Client) ConceptList(ctx context.Context) ([]ConceptItem, error) {
    req := &TushareRequest{
        APIName: "concept",
        Token:   c.token,
        Params:  map[string]interface{}{},
        Fields:  "id,name,src",
    }
    return c.doRequestWithRetry(ctx, req)
}
```

---

**接口 4：concept_detail — 概念成分股**

| 属性 | 值 |
|---|---|
| api_name | `concept_detail` |
| 用途 | 获取某个概念板块的全部成分股，写入 `stock_concept_tags` |
| 调用频率 | 初始化时对每个概念调用 1 次（约 400 次），后续增量时仅调用新增概念 |
| 请求参数 | `{"id": "TS0"}` — 概念 ID（从 concept 接口获取） |
| 请求字段 | `id,concept_name,ts_code,name` |
| 返回关键字段 | `ts_code`(股票代码), `concept_name`(概念名称), `name`(股票名称) |
| 频率控制 | 每次调用后 sleep 120ms（保证不超过 500次/分钟），使用令牌桶限流 |
| 容错 | 指数退避重试。单个概念失败不阻塞其他概念，记录失败概念 ID 待补爬 |

```go
// internal/external/tushare/concept.go
func (c *Client) ConceptDetail(ctx context.Context, conceptID string) ([]ConceptDetailItem, error) {
    // 令牌桶限流
    if err := c.rateLimiter.Wait(ctx); err != nil {
        return nil, err
    }
    req := &TushareRequest{
        APIName: "concept_detail",
        Token:   c.token,
        Params:  map[string]interface{}{"id": conceptID},
        Fields:  "id,concept_name,ts_code,name",
    }
    return c.doRequestWithRetry(ctx, req)
}
```

---

#### 5.6.2 韭研公社 — 1 个接口

**接口 5：韭研热点数据**

| 属性 | 值 |
|---|---|
| URL | `GET https://app.jiuyangongshe.com/jystock-app/api/v1/action/field?date={YYYY-MM-DD}` |
| 用途 | 获取指定日期的热点列表和对应股票，建立热点-股票映射 |
| 调用频率 | 初始化全量爬取时按交易日逐日调用；运行期间每交易日 09:00 调用 1 次获取前一天数据 |
| 请求参数 | URL Query: `date=2026-03-15`（日期格式 YYYY-MM-DD） |
| 请求头 | `Authorization: Bearer <jiuyan_api_key>`（如需鉴权） |
| 超时 | 30 秒（`jiuyan.timeout`） |
| 返回格式 | JSON 数组，每个元素为一个热点 |

**返回结构**：

```json
[
  {
    "name": "AI",
    "action_field_id": "field_001",
    "list": [
      {
        "code": "sz002445",
        "name": "巨人网络",
        "article": {
          "action_info": {
            "expound": "AI大模型应用落地，公司研发投入持续加大"
          }
        }
      }
    ]
  }
]
```

| 返回字段 | 说明 |
|---|---|
| `name` | 热点名称（需通过同义词表归一化） |
| `action_field_id` | 热点唯一 ID（韭研侧） |
| `list[].code` | 股票代码（格式 `sz002445`，需转换为 `002445.SZ`） |
| `list[].name` | 股票名称 |
| `list[].article.action_info.expound` | 涨停原因描述（记录到 `classify_evidence.evidence_text`） |

**代码转换规则**：

```go
// internal/pkg/converter/code_converter.go
package converter

import "strings"

// JiuyanToTushare 将韭研格式 "sz002445" 转换为 Tushare 格式 "002445.SZ"。
func JiuyanToTushare(code string) string {
    if len(code) < 3 {
        return code
    }
    prefix := strings.ToUpper(code[:2]) // "sz" → "SZ"
    number := code[2:]                   // "002445"
    return number + "." + prefix         // "002445.SZ"
}

// TushareToJiuyan 将 Tushare 格式 "002445.SZ" 转换为韭研格式 "sz002445"。
func TushareToJiuyan(code string) string {
    parts := strings.Split(code, ".")
    if len(parts) != 2 {
        return code
    }
    return strings.ToLower(parts[1]) + parts[0] // "sz002445"
}
```

| 容错策略 | 说明 |
|---|---|
| 重试 | 指数退避 1s→2s→4s→8s（RULE retry_backoff） |
| 全量爬取失败 | 单日失败记录到 `jiuyan_crawl_logs`（status=2），不中断其他日期 |
| 增量同步失败 | 连续 3 次失败触发韭研爬虫异常告警 |
| 同义词归一化 | 入库前查 `topic_synonyms` 表，将别名归并到主热点 |

```go
// internal/external/jiuyan/client.go
func (c *Client) FetchFieldData(ctx context.Context, date string) ([]FieldData, error) {
    url := fmt.Sprintf("%s/action/field?date=%s", c.baseURL, date)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    req.Header.Set("Authorization", "Bearer "+c.apiKey)

    // 使用指数退避重试
    var result []FieldData
    err = retry.RetryWithBackoff(ctx, c.retryConfig, func() error {
        resp, err := c.httpClient.Do(req)
        if err != nil {
            return err
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            return fmt.Errorf("jiuyan API returned status %d", resp.StatusCode)
        }
        return json.NewDecoder(resp.Body).Decode(&result)
    })
    return result, err
}
```

---

#### 5.6.3 LLM — 1 个接口

**接口 6：LLM 股票分类**

| 属性 | 值 |
|---|---|
| URL | `POST {llm.api_url}/v1/chat/completions`（OpenAI 兼容格式） |
| 用途 | 分类引擎第 4 层兜底：当前 3 层均未命中时，调用 LLM 判断股票属于哪个热点 |
| 调用频率 | 盘中每 5 分钟批处理 1 次，每次最多 5 个并发请求 |
| 请求头 | `Authorization: Bearer <llm_api_key>`, `Content-Type: application/json` |
| 超时 | 30 秒（`llm.timeout`） |

**请求体**：

```json
{
  "model": "gpt-4o-mini",
  "temperature": 0.1,
  "max_tokens": 512,
  "messages": [
    {
      "role": "system",
      "content": "你是A股市场热点分析专家。给定一只股票的名称、行业信息和当日活跃的热点列表，判断该股票最可能属于哪个热点。返回 JSON 格式：{\"topic_name\": \"热点名称\", \"confidence\": 0.0-1.0, \"reasoning\": \"判断理由\"}"
    },
    {
      "role": "user",
      "content": "股票：科大讯飞(002230.SZ)，行业：计算机应用。当日活跃热点：[AI(15只涨停), 机器人(10只涨停), 教育(3只涨停)]。请判断该股票今日涨停最可能属于哪个热点。"
    }
  ]
}
```

**返回格式**（OpenAI 标准）：

```json
{
  "choices": [
    {
      "message": {
        "content": "{\"topic_name\": \"AI\", \"confidence\": 0.92, \"reasoning\": \"科大讯飞是A股最核心的AI概念股，主营智能语音和自然语言处理\"}"
      }
    }
  ]
}
```

**解析规则**：

| 字段 | 处理方式 |
|---|---|
| `topic_name` | 通过同义词表归一化后匹配 `hot_topics.id`。未匹配到则标记"未分类" |
| `confidence` | >= 0.7 → 采信并写入 `stock_topic_mappings`（source=llm）；< 0.7 → 不采信，标记"未分类" |
| `reasoning` | 写入 `classify_evidence.evidence_text`，作为分类决策的可追溯依据 |

| 容错策略 | 说明 |
|---|---|
| 降级 | LLM 不可用时跳过，不阻塞监控流程（RULE llm_fallback） |
| 熔断 | 连续 5 次失败触发 LLM 熔断，60 秒内不发起调用（RULE circuit_breaker） |
| 并发控制 | 最多 5 个并发请求（`llm.max_concurrent`），使用 semaphore 限流 |
| 解析失败 | LLM 返回非法 JSON 时记录原文到 `evidence_text`，标记"未分类" |
| 置信度阈值 | < 0.7 不采信，记录证据但不写入映射表 |

```go
// internal/external/llm/classify.go
type ClassifyRequest struct {
    Stock        string   // 股票名称和代码
    Industry     string   // 行业
    ActiveTopics []string // 当日活跃热点列表（含涨停家数）
}

type ClassifyResponse struct {
    TopicName  string  `json:"topic_name"`
    Confidence float64 `json:"confidence"`
    Reasoning  string  `json:"reasoning"`
}

func (c *Client) Classify(ctx context.Context, req ClassifyRequest) (*ClassifyResponse, error) {
    // 信号量限流
    if err := c.semaphore.Acquire(ctx, 1); err != nil {
        return nil, err
    }
    defer c.semaphore.Release(1)

    // 构建 prompt
    userMsg := fmt.Sprintf(
        "股票：%s，行业：%s。当日活跃热点：%v。请判断该股票今日涨停最可能属于哪个热点。",
        req.Stock, req.Industry, req.ActiveTopics,
    )

    // 调用 OpenAI 兼容接口
    chatReq := openai.ChatCompletionRequest{
        Model:       c.model,
        Temperature: c.temperature,
        MaxTokens:   c.maxTokens,
        Messages: []openai.ChatCompletionMessage{
            {Role: "system", Content: systemPrompt},
            {Role: "user", Content: userMsg},
        },
    }

    resp, err := c.openaiClient.CreateChatCompletion(ctx, chatReq)
    if err != nil {
        return nil, fmt.Errorf("LLM API call failed: %w", err)
    }

    // 解析 JSON 响应
    var result ClassifyResponse
    content := resp.Choices[0].Message.Content
    if err := json.Unmarshal([]byte(content), &result); err != nil {
        return &ClassifyResponse{
            Reasoning: content, // 保留原文作为证据
        }, fmt.Errorf("LLM returned invalid JSON: %w", err)
    }

    return &result, nil
}
```