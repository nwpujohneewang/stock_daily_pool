## Context

### 项目背景

A股实时热点监控系统，从零构建Go后端服务。AGENTS.md已定义完整需求规格（业务规则、数据库Schema、API契约、配置调度），本设计文档聚焦技术架构决策。

### 约束条件

1. **Go 1.22+** — 项目语言约束
2. **PostgreSQL 15+ + Redis 7+** — 存储层约束
3. **Tushare Pro API** — 行情和概念数据源，有频率限制（500次/分钟）
4. **韭研公社API** — 热点数据源
5. **OpenAI兼容LLM** — 分类兜底
6. **Asia/Shanghai时区** — 全局强制时区

### 架构风格

三层架构 + 数据流：
- **Service Layer**: 业务逻辑，MonitorService/ClassifyService/AlertService等
- **Data Layer**: PostgreSQL（CRUD）+ Redis（缓存）
- **External Layer**: Tushare/韭研/LLM客户端

## Goals / Non-Goals

**Goals:**
- 实现10秒周期完成全市场~5000只股票的行情处理
- 四层级联分类（Redis→PG韭研→PG概念→LLM），命中即停
- 四维加权归因算法，支持单归因和双归因
- 策略2告警（昨日5%+未涨停 + 今日首次涨停）
- WebSocket实时推送（pool_snapshot/pool_diff/strategy_alert）
- 完整的RESTful API管理接口
- 幂等写入，崩溃安全重启

**Non-Goals:**
- 前端界面（React SPA，后期独立项目）
- 历史回测系统
- 微信/飞书等通知渠道接入（预留ExtraInfo JSONB）
- 多策略支持（当前仅策略2）

## Decisions

### D1: 为什么使用Hertz而非Gin？

**选择**: Hertz（字节跳动开源）

**理由**:
- AGENTS.md明确指定Hertz
- 高性能HTTP框架，与字节体系生态集成
- 支持中间件、路由分组

**替代方案**: Gin — 更成熟但性能略低

---

### D2: 为什么使用pgx/v5而非GORM？

**选择**: pgx/v5原生SQL

**理由**:
- AGENTS.md明确要求原生SQL不使用ORM
- pgx对pgtype（JSONB、数组）原生支持
- 更好的性能控制和SQL可观测性
- 避免ORM的隐式行为

**替代方案**: GORM — 开发效率高但性能开销大

---

### D3: 为什么使用16分片？

**选择**: hash(ts_code) % 16 分片

**理由**:
- ~5000只股票 ÷ 16分片 = 每分片~312只股票
- 10秒周期内可完成处理
- 同一股票永远落在同一分片，天然无锁
- 16是2的幂次，便于负载均衡

**替代方案**: 固定batch size分片 — 更灵活但实现复杂

---

### D4: 为什么四层分类降级？

**选择**: L1 Redis缓存 → L2 PG韭研映射 → L3 PG概念标签 → L4 LLM异步

**理由**:
- L1: 延迟<1ms，覆盖80%场景
- L2: 延迟~5ms，覆盖韭研标注数据
- L3: 覆盖从未在韭研出现的股票（概念兜底）
- L4: 极端情况异步处理，不阻塞主流程
- 每层命中即停，避免不必要的查询

**替代方案**: 直接LLM — 延迟高（3秒+），成本高

---

### D5: 为什么四维归因权重这样分配？

**选择**:
- Normal模式: {activity:0.25, bindStrength:0.40, timeProximity:0.20, recency:0.15}
- Concept模式: {activity:0.45, bindStrength:0.00, timeProximity:0.40, recency:0.15}

**理由**:
- bindStrength占最高权重(0.40)，历史标注频次是最强信号
- activity次之(0.25)，今日热点氛围重要
- timeProximity(0.20)，同一热点内涨停时间聚集性
- recency(0.15)，标注时效性
- Concept模式无频次数据，权重重分配

---

### D6: 为什么使用熔断器和退避重试？

**选择**: 指数退避（1s→2s→4s→8s）+ 熔断器

**理由**:
- 外部API（Tushare/韭研/LLM）不可靠
- 避免雪崩效应，局部失败不扩散
- 熔断开启时降级到缓存数据
- RULE-03强制要求

---

### D7: 为什么WebSocket只有三种消息类型？

**选择**: pool_snapshot / pool_diff / strategy_alert

**理由**:
- 简化前端解析逻辑
- 覆盖所有推送场景
- 变化量<30%用diff，>=30%用snapshot

---

### D8: 为什么按月分区daily_stock_pool？

**选择**: PostgreSQL Range分区，按date字段

**理由**:
- 单月数据量可控（涨停股通常<500条/天）
- 便于历史数据清理（drop旧分区）
- 查询特定月份性能好

---

### D9: 为什么使用Redis SortedSet存储热点涨停时间？

**选择**: topic:activity:limit:{date} (SortedSet)

**理由**:
- score存储涨停时间，支持时间线展示
- ZINCRBY/ZREVRANGE高效操作
- 支持按score排序查询

---

### D10: 为什么LLM置信度阈值设为0.7？

**选择**: confidence < 0.7 不采信

**理由**:
- 平衡精确率和召回率
- 低于0.7的分类错误率高
- AGENTS.md RULE-09强制要求

## Risks / Trade-offs

### R1: Tushare API频率限制
**风险**: 500次/分钟限制，可能不够
**缓解**: 
- 分片内每批50只股票批量查询
- 概念同步控制调用节奏（每分钟500次）
- 失败重试使用退避

### R2: LLM延迟和成本
**风险**: GPT-4o-mini仍有3秒延迟，成本累积
**缓解**:
- 仅第4层兜底调用，覆盖少量股票
- 异步处理，不阻塞主流程
- 置信度<0.7不写入，避免无效映射

### R3: 涨停状态机抖动
**风险**: 价格在涨停价附近反复波动
**缓解**:
- 状态机抑制，只有NONE→LIMIT_UP产生事件
- 首次涨停时间使用HSETNX，只写一次

### R4: WebSocket背压
**风险**: 慢客户端阻塞Hub
**缓解**:
- 256条缓冲区，满则断开
- 独立alert channel优先发送告警

### R5: 数据一致性
**风险**: Redis和PG数据可能不一致
**缓解**:
- 收盘快照强制同步
- 崩溃重启从PG恢复Redis

## Migration Plan

（从零构建，无迁移需求）

**部署顺序**:
1. 创建数据库Schema（12张表）
2. 初始化板块规则和股票基础数据
3. 全量韭研历史爬取
4. 同步Tushare概念板块
5. 启动HTTP服务和WebSocket Hub
6. 启动定时任务调度器

**回滚策略**:
- Schema使用migrations保证幂等
- 配置热更新支持动态调整
- 熔断器自动降级

## Open Questions

### Q1: 是否需要支持港股/美股？
**当前**: 仅A股，代码格式硬编码验证.SZ/.SH/.BJ

### Q2: 韭研公社API认证方式？
**当前**: Bearer Token，需验证实际认证方式

### Q3: LLM模型选择？
**当前**: gpt-4o-mini，可配置切换其他模型

### Q4: 历史数据保留周期？
**当前**: daily_stock_pool按月分区，建议保留2年

### Q5: 是否需要支持本地部署和云端部署切换？
**当前**: 通过config.yaml配置区分
