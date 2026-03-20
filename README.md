# Stock Monitor System

A股实时涨停板/强势股监控系统

## 功能特性

- **实时行情监控** - 每10秒拉取全市场约5000只股票行情，按16分片并发处理
- **涨停/强势股判定** - 根据板块规则（主板10%、创业板/科创板20%、北交所30%）计算涨停价
- **四层级联分类引擎** - Redis缓存 → PostgreSQL韭研映射 → Tushare概念标签 → LLM兜底
- **四维加权归因算法** - 热点活跃度、关联强度、时间聚集性、时效性四维度评分
- **策略告警引擎** - 昨日5%+未涨停 + 今日首次涨停 → 即时告警
- **WebSocket实时推送** - pool_snapshot / pool_diff / strategy_alert 三种消息类型

## 技术栈

- **语言**: Go 1.22+
- **HTTP框架**: Hertz (cloudwego)
- **数据库**: PostgreSQL 15+ (主存储) + Redis 7+ (实时缓存)
- **ORM**: pgx/v5 (原生SQL)
- **外部数据源**: Tushare Pro API、韭研公社API、OpenAI兼容LLM

## 项目结构

```
stock-monitor/
├── cmd/server/           # 程序入口
├── internal/
│   ├── config/          # 配置管理
│   ├── model/           # 数据模型
│   ├── repo/           # 数据访问层
│   ├── service/        # 业务逻辑
│   ├── handler/        # HTTP处理器
│   ├── middleware/     # 中间件
│   ├── ws/             # WebSocket
│   ├── scheduler/      # 定时任务
│   ├── cache/          # Redis缓存
│   ├── external/      # 外部API客户端
│   └── pkg/            # 工具包
├── migrations/         # 数据库迁移
├── config/             # 配置文件
└── Makefile           # 构建脚本
```

## 快速开始

### 1. 配置

编辑 `config/config.yaml`:

```yaml
server:
  port: 8888
  api_key: "your-api-key"

database:
  host: "127.0.0.1"
  port: 5432
  user: "postgres"
  password: "your-password"
  dbname: "stock"

redis:
  address: "127.0.0.1:6379"
  password: ""
  db: 0
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 运行数据库迁移

```bash
psql -h localhost -U postgres -d stock -f migrations/001_create_stock_basic_info.sql
# ... 执行其他迁移文件
```

### 4. 启动服务

```bash
# 开发模式
make dev

# 生产模式
make run
```

### 5. 健康检查

```bash
curl http://localhost:8888/health
```

## API端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /health | 健康检查 |
| GET | /ws | WebSocket连接 |
| GET | /api/v1/pool/limit-up | 涨停池查询 |
| GET | /api/v1/pool/above5 | 5%池查询 |
| GET | /api/v1/topics | 热点列表 |
| GET | /api/v1/alerts | 今日告警 |

## WebSocket消息

```json
{
  "type": "pool_snapshot",
  "timestamp": 1773579660,
  "data": { ... }
}
```

消息类型:
- `pool_snapshot` - 全量快照
- `pool_diff` - 增量变更
- `strategy_alert` - 策略告警

## 开发指南

```bash
# 构建
make build

# 测试
make test

# 代码格式化
make fmt

# 运行
make run
```

## License

MIT
