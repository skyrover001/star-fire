# MySQL 接入规划与详细设计（多数据库支持 · 提升并发）

> 状态：设计稿（待评审）
> 目标：让 star-fire 服务端支持「一键配置」切换 SQLite / MySQL，解除 SQLite 单写者 + 4 连接池的并发瓶颈。
> 原则：默认零配置（SQLite 开箱即用）；MySQL 通过环境变量一键启用；保持 `CGO_ENABLED=0` 纯 Go 构建；不破坏现有测试。

---

## 1. 背景与目标

### 1.1 现状瓶颈（来自 docs/architecture-analysis.md 实测分析）

| 瓶颈点 | 现状 | 影响 |
|---|---|---|
| 连接池 | `SetMaxOpenConns(4)`（`internal/models/server.go`） | 全局最多 4 个并发 SQL，每个 chat 请求 8~10 条 SQL → 新请求速率上限约几百~1000/s |
| SQLite 单写者 | WAL 模式下读写可并行，但**写仍串行** | 写密集（扣费、指纹、用量记录）互相排队 |
| 每请求 SQL 数 | 鉴权 1~2 + 余额 1 + 指纹 3 + 扣费 2 + 用量 1 | 放大了连接池压力 |
| 扣费非原子 | `DeductBalance` 先 SELECT 再 UPDATE | 并发下重复扣/超额扣（资金安全问题） |

### 1.2 目标

1. **一键切换**：`DB_DRIVER=mysql` + 少量环境变量即可接入 MySQL；不配置则维持 SQLite 现状。
2. **并发提升**：MySQL 下连接池默认 100（可配），InnoDB 行级锁支持真正并发写。
3. **顺手修复**：原子扣费（资金安全）、注册 ID 竞态重试。
4. **数据迁移**：提供 `starfire migrate-db` 命令，把现有 SQLite 数据完整迁到 MySQL。
5. **Docker 一键部署**：compose 增加 MySQL 服务（profile 方式，默认行为不变）。

### 1.3 非目标（本期不做）

- PostgreSQL 支持（设计预留方言层，后续可加）。
- 读写分离 / 分库分表。
- 多实例共享 DB 的分布式状态（clients 内存表、限流器仍是单进程语义）。

---

## 2. 总体方案

```
                    ┌─────────────────────────────┐
   env 配置          │  internal/models/database.go │
 DB_DRIVER=sqlite ──►  OpenDatabase() 按驱动分发     │──► *gorm.DB（其余代码零改动）
 DB_DRIVER=mysql  ──►  openMySQL / openSQLite       │
                    │  EnsureIndex() 方言兼容       │
                    └─────────────────────────────┘
```

核心思路：**GORM 已抽象驱动，业务层（11 个 *DB 结构体）全部通过 `*gorm.DB` 操作，无需改动**。改动集中在：

1. 配置层（`config/config.go`）新增 DB 配置段。
2. 新建 `internal/models/database.go`：统一建连入口 + 方言兼容工具。
3. `internal/models/server.go`：`NewServer()` 改调 `OpenDatabase()`，删除内联 SQLite 逻辑。
4. 少量方言差异点修复（见 §4.4）。
5. 迁移 CLI + Docker 编排。

---

## 3. 配置设计（一键配置）

### 3.1 环境变量一览

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `DB_DRIVER` | `sqlite` | `sqlite` \| `mysql` |
| `DB_DSN` | 空 | **完整 DSN 直填（最高优先级）**，设置后忽略下列单项 |
| `DB_HOST` | `127.0.0.1` | MySQL 主机 |
| `DB_PORT` | `3306` | MySQL 端口 |
| `DB_USER` | `starfire` | 用户名 |
| `DB_PASSWORD` | 空 | 密码 |
| `DB_NAME` | `starfire` | 库名 |
| `DATABASE_PATH` | `./data/star_fire.db` | SQLite 路径（`.env.example` 已有，此前代码未读取，本次真正生效） |
| `DB_MAX_OPEN_CONNS` | sqlite=4 / mysql=100 | 最大打开连接数 |
| `DB_MAX_IDLE_CONNS` | sqlite=10 / mysql=20 | 最大空闲连接 |
| `DB_CONN_MAX_LIFETIME_MIN` | `60` | 连接最长复用时间（分钟） |
| `DB_LOG_LEVEL` | `silent` | gorm 日志：`silent`\|`error`\|`warn`\|`info`（排障用） |

### 3.2 最小接入示例

```bash
# 一键切 MySQL（本机）
DB_DRIVER=mysql
DB_PASSWORD=yourpass

# 或外部数据库（一条 DSN 搞定）
DB_DRIVER=mysql
DB_DSN="starfire:pass@tcp(db.example.com:3306)/starfire?charset=utf8mb4&parseTime=True&loc=Local&interpolateParams=true"
```

### 3.3 `config/config.go` 新增字段

```go
// 数据库配置
DBDriver             string // sqlite | mysql
DBDSN                string // 完整 DSN（可选，优先级最高）
DBHost               string
DBPort               int
DBUser               string
DBPassword           string
DBName               string
DBPath               string // sqlite 文件路径
DBMaxOpenConns       int
DBMaxIdleConns       int
DBConnMaxLifetimeMin int
DBLogLevel           string
```

加载逻辑：`DBMaxOpenConns` 等未显式配置时按 driver 取默认值（见上表）。

---

## 4. 详细设计

### 4.1 新文件 `internal/models/database.go`

统一建连入口，替代 `NewServer()` 里的内联逻辑：

```go
// OpenDatabase 按配置打开数据库（sqlite / mysql）。
func OpenDatabase() (*gorm.DB, error)

func openSQLite() (*gorm.DB, error)
// - os.MkdirAll(filepath.Dir(path))
// - gorm.Open(sqlite.Open(path))
// - 现有 PRAGMA 循环原样迁入（仅 SQLite 执行）
// - 连接池参数按配置

func openMySQL() (*gorm.DB, error)
// - DSN 拼装（或直用 DB_DSN）
//   默认参数: charset=utf8mb4&collation=utf8mb4_bin&parseTime=True&loc=Local
//            &interpolateParams=true&timeout=5s&readTimeout=30s&writeTimeout=30s
// - pingWithRetry: 启动时最多等待 60s（2s 间隔），适配 docker 启动顺序
// - 连接池参数按配置

// 方言工具
func IsMySQL(db *gorm.DB) bool
func EnsureIndex(db *gorm.DB, indexName, table, columnList string) // 见 §4.4-②
```

**DSN 关键参数说明**（踩坑预防）：

| 参数 | 为什么必须 |
|---|---|
| `parseTime=True` | 不加则 `time.Time` 扫描直接报错（go-sql-driver 默认返回 []byte） |
| `loc=Local` | 与服务端时区一致（docker 已设 `TZ=Asia/Shanghai`），避免时间偏移 |
| `charset=utf8mb4` | 中文/emoji 必需 |
| `collation=utf8mb4_bin` | **见 §4.4-⑤ 大小写语义**，与 SQLite 行为对齐 |
| `interpolateParams=true` | 客户端参数内联，省去每次 prepare 往返，高并发下显著降延迟 |

### 4.2 `server.go` 改造

```go
func NewServer() *Server {
    gormDB, err := OpenDatabase()   // 替换原 sqlite.Open + PRAGMA + 连接池代码块
    if err != nil {
        log.Fatalf("init database failed: %v", err)
    }
    // ... 其余 11 个 *DB 初始化不变
}
```

- `os.MkdirAll("./data")` 移入 `openSQLite`（MySQL 模式不需要）。
- 启动日志打印：`database: mysql starfire@127.0.0.1:3306 (pool: 100/20)`，便于确认生效。

### 4.3 各 *DB 结构体

**零改动**。全部通过 `*gorm.DB` 与 GORM API 操作，AutoMigrate 由 MySQL 驱动翻译 DDL。

### 4.4 方言兼容点（逐一核对过全库 SQL）

全库扫描 `Raw(/Exec(/Group(/DATE(/CAST(/LIKE` 后，需处理的差异共 5 处：

| # | 位置 | 问题 | 方案 |
|---|---|---|---|
| ① | `server.go` PRAGMA 循环 | MySQL 执行 `PRAGMA` 报错 | 迁入 `openSQLite`，仅 SQLite 执行 |
| ② | `token_usage.go:61-67` 4 条 `CREATE INDEX IF NOT EXISTS` | **MySQL 8 不支持索引 `IF NOT EXISTS`** | 新增 `EnsureIndex()`：MySQL 先查 `information_schema.statistics` 再建；SQLite 保持原语句 |
| ③ | `token_usage.go` 多处 `DATE(timestamp)` | — | **无需改**：MySQL 的 `DATE()` 对 DATETIME 同样有效；`timestamp` 列名在 MySQL 是非保留字，且 GORM 生成 SQL 会加反引号（实施时已复核全部聚合查询，均合规） |
| ④ | `user.go` `GetMaxUserID` 的 `CAST(id AS UNSIGNED)` | — | **无需改**：MySQL 原生支持 `CAST(... AS UNSIGNED)`（仅 PG 不支持，不在本期范围） |
| ⑤ | 字符串大小写语义 | SQLite `=` 比较区分大小写；MySQL 默认 `_ci` 排序规则**不区分** | DSN 使用 `collation=utf8mb4_bin`，使 `username`/`key_value` 等比较与 SQLite 语义一致（API Key 是安全敏感等值匹配，必须区分大小写） |

**列类型映射核对**（gorm mysql 驱动 + `DefaultStringSize=191`）：

- 无 size 的普通 `string` → `longtext`（如 `Client.ModelsJSON`、`Notification.Content` 已显式 `type:text`）✅
- 带 `index`/`uniqueIndex`/`primaryKey` 的 `string` → `varchar(191)`（191×4B=764B < 767B，兼容旧 InnoDB；MySQL 8 更无压力）✅
- 复合唯一索引 `idx_user_model(UserID,Model)`：2×191×4=1528B < 3072B（MySQL 8 DYNAMIC 行格式）✅
- `User.ID string + autoIncrement` 标签：gorm mysql 驱动只对整型追加 `AUTO_INCREMENT`，string 主键忽略该标签，与 SQLite 现状（应用层赋 ID）一致 ✅（顺手删除该误导性标签）
- `time.Time` 零值 **不能**直接存 MySQL：go-sql-driver 会把 Go 零值 `0001-01-01` 写成 MySQL 零日期 `'0000-00-00'`，严格模式（`NO_ZERO_DATE`）下 INSERT/UPDATE 直接报错。**实施修正**：`User.MembershipExpireAt` / `ContributorMembershipExpireAt` 已改为 `*time.Time`（nil = 未开通 → NULL），`SetMembership`/`SetContributorMembership` 内部把零值转 nil（签名不变）；`CalculateExpireAt` 相应改为接收 `*time.Time`；`migrate-db` 拷贝 users 时同样把历史零值转 NULL。其余 12 个模型经逐一审计无零日期写入风险（`APIKey.LastUsed` 本就是 `*time.Time`，其余时间字段均在创建时显式赋 `time.Now()`）
- `Client.ModelsJSON` 无 size 的普通 string 在 MySQL 下映射 `longtext` ✅（为防驱动版本差异，已显式加 `type:text` 标签兜底）

### 4.5 并发正确性修复（随本期一并落地）

**① 原子扣费**（`internal/models/user.go` `DeductBalance`）——资金安全，方言无关：

```go
// 现在：SELECT balance → 计算 → UPDATE（竞态窗口）
// 改为：单条原子 UPDATE + RowsAffected 判断
res := udb.db.Model(&User{}).
    Where("id = ? AND balance > 0", userID).
    Updates(map[string]interface{}{
        "balance":     gorm.Expr("balance - ?", amount),
        "total_spent": gorm.Expr("total_spent + ?", amount),
    })
if res.Error != nil { return res.Error }
if res.RowsAffected == 0 { return errors.New("insufficient balance") }
```

**② 注册 ID 竞态**（`api/user_handlers/users.go` + `GetMaxUserID`）：
`MAX(id)+1` 在并发注册下会撞主键。SQLite 串行写掩盖了问题，MySQL 并发下会暴露。方案：**撞唯一键重试 3 次**（方言无关，不改 ID 生成格式，避免影响存量数据）。

**③ 指纹表写放大（可选 P2）**：`UpdateFingerprint` = FirstOrCreate + Update 两条 SQL，可合并为一条 upsert（`clause.OnConflict`）。本期先不动，观察 MySQL 下压测数据再决定。

### 4.6 数据迁移工具：`starfire migrate-db`

```
用法:
  starfire migrate-db [-sqlite ./data/star_fire.db] [-force]
环境:
  按 DB_DRIVER/DB_DSN/DB_* 读取目标 MySQL（DB_DRIVER 必须为 mysql）
```

流程：

1. 打开源 SQLite（只读意图）+ 目标 MySQL（先 `AutoMigrate` 全部 12 张表）。
2. 按表拷贝（users → api_keys → clients → client_fingerprints → token_usages → model_prices → trends → recharges → user_price_caps → client_stats → notifications → system_configs）：
   - 目标表非空且未加 `-force` → 跳过（幂等，可断点重跑）。
   - 大表（`token_usages`）按 ID keyset 分页（每批 1000），`CreateInBatches` 写入。
   - 显式主键插入后 MySQL 自增计数器自动对齐，无需处理 sequence。
3. 输出每表行数对比 + 总耗时；行数不一致以非零码退出。

### 4.7 Docker 一键部署

**`docker-compose.yml`** 新增（profile 隔离，默认 `docker compose up` 行为完全不变）：

```yaml
  mysql:
    image: mysql:8.0
    profiles: ["mysql"]
    environment:
      MYSQL_ROOT_PASSWORD: ${DB_ROOT_PASSWORD:-starfire_root}
      MYSQL_DATABASE: ${DB_NAME:-starfire}
      MYSQL_USER: ${DB_USER:-starfire}
      MYSQL_PASSWORD: ${DB_PASSWORD:-starfire_pass}
    volumes: [mysql-data:/var/lib/mysql]
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1", "-uroot", "-p$$MYSQL_ROOT_PASSWORD"]
      interval: 5s
      timeout: 3s
      retries: 20
    networks: [starfire-net]

volumes:
  mysql-data:
```

**新增 `docker-compose.mysql.yml`**（override，实现真·一键）：

```yaml
services:
  backend:
    environment:
      DB_DRIVER: mysql
      DB_HOST: mysql
      DB_USER: ${DB_USER:-starfire}
      DB_PASSWORD: ${DB_PASSWORD:-starfire_pass}
      DB_NAME: ${DB_NAME:-starfire}
    depends_on:
      mysql:
        condition: service_healthy
```

一键启动：`docker compose -f docker-compose.yml -f docker-compose.mysql.yml up -d`

**顺带修复 `dockerfile` 既有 bug**（阻塞 docker 部署，与本期强相关）：
`go build ... ./cmd/server` → 路径不存在，改为 `go build ... -o server .`（与 Makefile 的 `SERVER_SRC=./` 一致）。MySQL 驱动 `go-sql-driver/mysql` 纯 Go，`CGO_ENABLED=0` 不受影响。

### 4.7b 依赖变更

```
go get gorm.io/driver/mysql   （传递引入 github.com/go-sql-driver/mysql，均纯 Go）
```

`glebarez/sqlite` 保留（默认模式 + 单元测试继续使用）。

---

## 5. 兼容性风险清单

| 风险 | 等级 | 对策 |
|---|---|---|
| MySQL < 8.0 索引长度限制 | 中 | 文档声明**要求 MySQL 8.0+**；`DefaultStringSize=191` 兜底 |
| 时区不一致导致时间偏移 | 中 | DSN `loc=Local` + 容器 `TZ=Asia/Shanghai`（dockerfile 已有） |
| 忘记 `parseTime=True` | 高（必现） | DSN 由代码拼装默认参数，用户自填 DSN 时文档红字标注 |
| 大小写语义差异 | 高（隐蔽） | `collation=utf8mb4_bin` 全局对齐 SQLite 行为（§4.4-⑤） |
| MySQL `max_connections=151` 被打满 | 低 | 应用池默认 100 < 151；文档给出扩容建议（`max_connections=500`） |
| `ONLY_FULL_GROUP_BY` 严格模式 | 低 | 已核对：所有 GROUP BY 查询的非聚合列（`model`、`DATE(timestamp)`）均在分组表达式中，合规 |
| 连接池耗时时无超时阻塞 | 低 | go-sql-driver `readTimeout=30s` 兜底；池参数可配 |
| 单元测试依赖 sqlite 内存库 | 无 | 测试直接 `gorm.Open(sqlite.Open(":memory:"))`，不经 `OpenDatabase()`，不受影响 |

---

## 6. 测试与验收

### 6.1 自动化

| 项 | 命令 | 预期 |
|---|---|---|
| 编译 | `go build ./internal/... ./api/... ./routes/... ./config/... .` | exit 0 |
| vet | `go vet ./internal/... ./config/...` | 无新增告警 |
| 单测 | `go test ./internal/models/ ./internal/service/...` | 全部 PASS（sqlite 内存库，行为不变） |
| 集成（可选） | `TEST_MYSQL_DSN=... go test -run TestMySQL` | 新增跳过式集成测试：建表→CRUD→原子扣费并发验证 |

### 6.2 手工验收清单

1. **默认回归**：不配任何 DB 环境变量启动 → SQLite 正常，日志显示 `database: sqlite`。
2. **MySQL 冒烟**：compose 起 MySQL → `DB_DRIVER=mysql go run .` → 注册/登录/建 Key/发一次 chat/查用量统计 → 数据落库。
3. **重启持久化**：重启服务端，数据仍在。
4. **迁移**：先在 SQLite 造数据（用户/Key/用量）→ `starfire migrate-db` → 行数一致 + 旧账号可在 MySQL 后端登录。
5. **并发扣费**：同一账号并发 20 个请求，总扣费 = 实际消费，余额不为负（原子 UPDATE 生效）。
6. **一键部署**：`docker compose -f docker-compose.yml -f docker-compose.mysql.yml up -d` 全链路可用。

---

## 7. 实施计划

| 阶段 | 内容 | 涉及文件 | 预估 |
|---|---|---|---|
| **P1 核心切换** | 配置段 + `database.go`（OpenDatabase/openSQLite/openMySQL/EnsureIndex）+ `server.go` 接入 + 4 条索引语句改造 | `config/config.go`、`internal/models/database.go`(新)、`server.go`、`token_usage.go` | 0.5 天 |
| **P2 并发修复** | 原子 `DeductBalance` + 注册 ID 重试 | `internal/models/user.go`、`api/user_handlers/users.go` | 0.25 天 |
| **P3 迁移与部署** | `migrate-db` CLI + compose（mysql 服务 + override 文件）+ dockerfile 路径修复 + `.env.example`/README | `main.go`、`docker-compose*.yml`、`dockerfile`、`.env.example` | 0.5 天 |
| **P4 验证** | §6 全部验收项 + 压测对比（SQLite vs MySQL 新请求速率） | — | 0.25 天 |

**总计约 1.5 天**。每阶段独立可编译、可回退（不配 env 即回 SQLite）。

---

## 8. 后续可选优化（本期不做，预留）

1. **PostgreSQL**：方言层已隔离，加 `gorm.io/driver/postgres` + 修 `CAST(id AS UNSIGNED)`（PG 需 `::bigint`）+ `User.ID` 改 uint/UUID 即可。
2. **SQL 数瘦身**：指纹 FirstOrCreate+Update 合并 upsert；鉴权/余额合并查询。
3. **用量表冷热分离**：`token_usages` 按月分区（MySQL 原生分区），历史数据归档。
4. **多实例水平扩展**：DB 已可共享，但 clients 内存表/限流器需改 Redis（独立立项）。
5. **WS 缓冲修复**（`connection.go` 硬编码 1MB → 让 `WS_BUFFER` 生效）：与 DB 无关但同为并发大头，建议作为下一个独立任务。

---

## 附：改动文件总览

```
config/config.go                    [改] 新增 DB 配置段
internal/models/database.go         [新] OpenDatabase / openSQLite / openMySQL / EnsureIndex / IsMySQL
internal/models/server.go           [改] NewServer 调 OpenDatabase，删内联 sqlite+PRAGMA
internal/models/token_usage.go      [改] 4 条 CREATE INDEX → EnsureIndex
internal/models/user.go             [改] 原子 DeductBalance；删 User.ID 误导性 autoIncrement 标签
api/user_handlers/users.go          [改] 注册 ID 撞键重试
main.go                             [改] migrate-db 子命令
go.mod                              [改] + gorm.io/driver/mysql
.env.example                        [改] DB_* 变量文档
docker-compose.yml                  [改] mysql 服务（profile）
docker-compose.mysql.yml            [新] 一键 override
dockerfile                          [改] 修复构建路径 ./cmd/server → .
docs/mysql-support-design.md        [新] 本文档
```
