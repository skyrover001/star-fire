package models

import (
	"fmt"
	"log"
	"os"
	"time"

	configs "star-fire/config"

	"gorm.io/gorm"
)

// SQLite → MySQL 数据迁移（starfire migrate-db 子命令的实现）。
//
// 源库固定按 SQLite 打开（DATABASE_PATH 或 -sqlite 参数），
// 目标库固定按 MySQL 打开（DB_DSN 或 DB_HOST/DB_USER/... 环境变量），
// 不经过 OpenDatabase() 的驱动分发，避免 DB_DRIVER 语义混淆。

// migrateBatchSize 大表（token_usages）分页拷贝的批大小
const migrateBatchSize = 1000

// copyResult 单表拷贝结果
type copyResult struct {
	src     int64 // 源表行数
	dst     int64 // 目标表行数（拷贝后重新统计）
	skipped bool  // 目标表非空而跳过（幂等，可断点重跑）
}

// allMigrateModels 迁移涉及的全部 12 张表（与 NewServer 初始化的各 *DB 一一对应）
func allMigrateModels() []interface{} {
	return []interface{}{
		&User{}, &APIKey{}, &Client{}, &ClientFingerprint{}, &ClientStats{},
		&TokenUsage{}, &ModelPrice{}, &Trend{}, &RechargeRecord{},
		&UserModelPriceCap{}, &SystemConfig{}, &Notification{},
	}
}

// MigrateSQLiteToMySQL 将现有 SQLite 数据完整迁移到 MySQL。
//
// 行为：
//   - 目标表非空且未加 force → 跳过该表（幂等，可断点重跑）
//   - force=true → 先清空目标全部表再迁移
//   - usersOnly=true → 只迁移 users 表（其余表不建、不拷贝，供仅切换用户数据的场景）
//   - users 表拷贝时将零值会员到期时间转为 NULL（MySQL 严格模式禁止 0000-00-00）
//   - token_usages 大表按 ID keyset 分页（每批 1000），避免一次性载入内存
//   - 结束后逐表核对行数，不一致返回错误（调用方以非零码退出）
func MigrateSQLiteToMySQL(force bool, usersOnly bool) error {
	start := time.Now()

	if configs.Config.DBDriver != "mysql" {
		return fmt.Errorf("目标数据库驱动必须为 mysql：请设置 DB_DRIVER=mysql 及 DB_HOST/DB_USER/DB_PASSWORD/DB_NAME（或直接 DB_DSN）")
	}

	// 源 SQLite
	if _, err := os.Stat(configs.Config.DBPath); err != nil {
		return fmt.Errorf("源 SQLite 文件不存在: %s（可用 DATABASE_PATH 或 -sqlite 参数指定）", configs.Config.DBPath)
	}
	log.Printf("migrate: 打开源 SQLite %s", configs.Config.DBPath)
	source, err := openSQLite()
	if err != nil {
		return fmt.Errorf("打开源 SQLite 失败: %w", err)
	}

	// 目标 MySQL（openMySQL 自带 60s 重试，容忍目标库晚就绪）
	log.Printf("migrate: 打开目标 MySQL %s@tcp(%s:%d)/%s",
		configs.Config.DBUser, configs.Config.DBHost, configs.Config.DBPort, configs.Config.DBName)
	target, err := openMySQL()
	if err != nil {
		return fmt.Errorf("打开目标 MySQL 失败: %w", err)
	}

	// -force：清空目标全部表后重建
	if force {
		log.Printf("migrate: 已指定 -force，清空目标表后重新迁移")
		if err := target.Migrator().DropTable(allMigrateModels()...); err != nil {
			return fmt.Errorf("清空目标表失败: %w", err)
		}
	}

	// 目标建表 + 复合索引（与 NewTokenUsageDB 中的 4 条保持一致）
	// usersOnly 模式只建 users 表，其余表留给服务启动时的 AutoMigrate
	migrateModels := allMigrateModels()
	if usersOnly {
		migrateModels = []interface{}{&User{}}
	}
	if err := target.AutoMigrate(migrateModels...); err != nil {
		return fmt.Errorf("目标 AutoMigrate 失败: %w", err)
	}
	if !usersOnly {
		_ = EnsureIndex(target, "idx_token_usages_user_timestamp", "token_usages", "user_id, timestamp")
		_ = EnsureIndex(target, "idx_token_usages_client_timestamp", "token_usages", "client_id, timestamp")
		_ = EnsureIndex(target, "idx_token_usages_user_model_timestamp", "token_usages", "user_id, model, timestamp")
		_ = EnsureIndex(target, "idx_token_usages_client_model_timestamp", "token_usages", "client_id, model, timestamp")
	}

	// 按表拷贝（无外键约束，顺序仅出于可读性；users 在前便于排查）
	type copyTask struct {
		name string
		run  func() (copyResult, error)
	}
	tasks := []copyTask{
		{"users", func() (copyResult, error) {
			return copyTable(source, target, "users", normalizeUserForMySQL)
		}},
	}
	if !usersOnly {
		tasks = append(tasks,
			copyTask{"api_keys", func() (copyResult, error) {
				return copyTable[APIKey](source, target, "api_keys", nil)
			}},
			copyTask{"clients", func() (copyResult, error) {
				return copyTable[Client](source, target, "clients", nil)
			}},
			copyTask{"client_fingerprints", func() (copyResult, error) {
				return copyTable[ClientFingerprint](source, target, "client_fingerprints", nil)
			}},
			copyTask{"client_stats", func() (copyResult, error) {
				return copyTable[ClientStats](source, target, "client_stats", nil)
			}},
			copyTask{"token_usages", func() (copyResult, error) {
				return copyTokenUsages(source, target)
			}},
			copyTask{"model_prices", func() (copyResult, error) {
				return copyTable[ModelPrice](source, target, "model_prices", nil)
			}},
			copyTask{"trends", func() (copyResult, error) {
				return copyTable[Trend](source, target, "trends", nil)
			}},
			copyTask{"recharge_records", func() (copyResult, error) {
				return copyTable[RechargeRecord](source, target, "recharge_records", nil)
			}},
			copyTask{"user_model_price_caps", func() (copyResult, error) {
				return copyTable[UserModelPriceCap](source, target, "user_model_price_caps", nil)
			}},
			copyTask{"system_configs", func() (copyResult, error) {
				return copyTable[SystemConfig](source, target, "system_configs", nil)
			}},
			copyTask{"notifications", func() (copyResult, error) {
				return copyTable[Notification](source, target, "notifications", nil)
			}},
		)
	}

	log.Printf("migrate: 开始拷贝（共 %d 张表）", len(tasks))
	var mismatches []string
	for _, task := range tasks {
		res, err := task.run()
		if err != nil {
			return err
		}
		switch {
		case res.skipped:
			log.Printf("  %-22s 源 %6d | 目标 %6d  ↷ 跳过（目标非空，如需重迁请加 -force）", task.name, res.src, res.dst)
		case res.src != res.dst:
			log.Printf("  %-22s 源 %6d | 目标 %6d  ✗ 行数不一致", task.name, res.src, res.dst)
			mismatches = append(mismatches, task.name)
		default:
			log.Printf("  %-22s 源 %6d | 目标 %6d  ✓", task.name, res.src, res.dst)
		}
	}

	if len(mismatches) > 0 {
		return fmt.Errorf("以下表行数不一致: %v（目标已有旧数据时请加 -force 重新迁移）", mismatches)
	}
	log.Printf("✓ 迁移完成，耗时 %s", time.Since(start).Round(time.Millisecond))
	return nil
}

// normalizeUserForMySQL 将 SQLite 历史数据中的零值会员到期时间转为 NULL。
// SQLite 以文本存储 '0001-01-01T00:00:00Z'，扫描到 *time.Time 后得到非 nil 指针指向零值；
// 若原样写入 MySQL 会变成 '0000-00-00'，被严格模式（NO_ZERO_DATE）拒绝。
func normalizeUserForMySQL(u *User) *User {
	if u.MembershipExpireAt != nil && u.MembershipExpireAt.IsZero() {
		u.MembershipExpireAt = nil
	}
	if u.ContributorMembershipExpireAt != nil && u.ContributorMembershipExpireAt.IsZero() {
		u.ContributorMembershipExpireAt = nil
	}
	return u
}

// copyTable 全量拷贝一张小表（一次性读入内存）。
// 目标表非空时跳过（幂等）；transform 可在写入前修正数据（如零值时间转 NULL）。
// 读写均跳过 GORM 钩子：Client 的 BeforeSave/AfterFind 只服务运行时 Models 字段，
// 迁移时应让 ModelsJSON 原样透传。
func copyTable[T any](source, target *gorm.DB, name string, transform func(*T) *T) (copyResult, error) {
	var res copyResult
	if err := source.Model(new(T)).Count(&res.src).Error; err != nil {
		return res, fmt.Errorf("统计源表 %s 失败: %w", name, err)
	}
	if err := target.Model(new(T)).Count(&res.dst).Error; err != nil {
		return res, fmt.Errorf("统计目标表 %s 失败: %w", name, err)
	}
	if res.dst > 0 {
		res.skipped = true
		return res, nil
	}

	var rows []*T
	if err := source.Session(&gorm.Session{SkipHooks: true}).Find(&rows).Error; err != nil {
		return res, fmt.Errorf("读取源表 %s 失败: %w", name, err)
	}
	if transform != nil {
		for i, r := range rows {
			rows[i] = transform(r)
		}
	}
	if len(rows) > 0 {
		if err := target.Session(&gorm.Session{SkipHooks: true}).CreateInBatches(rows, migrateBatchSize).Error; err != nil {
			return res, fmt.Errorf("写入目标表 %s 失败: %w", name, err)
		}
	}
	if err := target.Model(new(T)).Count(&res.dst).Error; err != nil {
		return res, fmt.Errorf("核对目标表 %s 失败: %w", name, err)
	}
	return res, nil
}

// copyTokenUsages 按 ID keyset 分页拷贝大表 token_usages（每批 1000），避免一次性载入内存。
func copyTokenUsages(source, target *gorm.DB) (copyResult, error) {
	var res copyResult
	if err := source.Model(&TokenUsage{}).Count(&res.src).Error; err != nil {
		return res, fmt.Errorf("统计源表 token_usages 失败: %w", err)
	}
	if err := target.Model(&TokenUsage{}).Count(&res.dst).Error; err != nil {
		return res, fmt.Errorf("统计目标表 token_usages 失败: %w", err)
	}
	if res.dst > 0 {
		res.skipped = true
		return res, nil
	}

	srcSession := source.Session(&gorm.Session{SkipHooks: true})
	dstSession := target.Session(&gorm.Session{SkipHooks: true})

	var lastID uint
	var copied int64
	for {
		var rows []*TokenUsage
		if err := srcSession.Where("id > ?", lastID).Order("id ASC").Limit(migrateBatchSize).Find(&rows).Error; err != nil {
			return res, fmt.Errorf("读取 token_usages 失败: %w", err)
		}
		if len(rows) == 0 {
			break
		}
		if err := dstSession.CreateInBatches(rows, migrateBatchSize).Error; err != nil {
			return res, fmt.Errorf("写入 token_usages 失败: %w", err)
		}
		lastID = rows[len(rows)-1].ID
		copied += int64(len(rows))
		log.Printf("  token_usages 已拷贝 %d / %d", copied, res.src)
	}

	if err := target.Model(&TokenUsage{}).Count(&res.dst).Error; err != nil {
		return res, fmt.Errorf("核对目标表 token_usages 失败: %w", err)
	}
	return res, nil
}
