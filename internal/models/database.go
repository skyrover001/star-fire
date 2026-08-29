package models

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	configs "star-fire/config"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 数据库驱动抽象层：默认零配置 SQLite，通过环境变量一键切换 MySQL。
// 业务层（各 *DB 结构）只依赖 *gorm.DB，无需感知驱动差异。

const mysqlDefaultStringSize = 191 // varchar(191)，兼容 767B/3072B 索引长度限制

// IsMySQL 返回当前配置是否使用 MySQL。
func IsMySQL() bool {
	return configs.Config.DBDriver == "mysql"
}

// OpenDatabase 根据配置打开数据库连接（sqlite 或 mysql）。
func OpenDatabase() (*gorm.DB, error) {
	if IsMySQL() {
		return openMySQL()
	}
	return openSQLite()
}

// openSQLite 打开 SQLite 数据库（保持原有 WAL 等优化行为不变）。
func openSQLite() (*gorm.DB, error) {
	if err := os.MkdirAll("./data", 0755); err != nil {
		return nil, fmt.Errorf("create data directory failed: %w", err)
	}

	gormDB, err := gorm.Open(sqlite.Open(configs.Config.DBPath), &gorm.Config{
		Logger: gormLogger(),
	})
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrent read/write performance.
	// WAL allows readers and writers to coexist without blocking each other.
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-20000",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if err := gormDB.Exec(p).Error; err != nil {
			log.Printf("WARNING: failed to set %s: %v", p, err)
		}
	}

	configurePool(gormDB)
	return gormDB, nil
}

// openMySQL 打开 MySQL 数据库。
// DSN 优先级：DB_DSN（完整 DSN）> DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME 拼接。
// 启动时最多重试 60 秒（2s 间隔），以容忍 docker-compose 中 MySQL 容器晚于后端就绪。
func openMySQL() (*gorm.DB, error) {
	dsn := configs.Config.DBDSN
	if dsn == "" {
		dsn = buildMySQLDSN()
	}

	var gormDB *gorm.DB
	var err error
	deadline := time.Now().Add(60 * time.Second)
	for {
		gormDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: gormLogger(),
		})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("connect mysql failed (retried 60s): %w", err)
		}
		log.Printf("mysql not ready, retrying in 2s: %v", err)
		time.Sleep(2 * time.Second)
	}

	configurePool(gormDB)
	return gormDB, nil
}

// buildMySQLDSN 按配置项拼接 MySQL DSN。
// utf8mb4_bin 保证 API key 等字符串比较大小写敏感（与 SQLite 行为一致）。
func buildMySQLDSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&collation=utf8mb4_bin&parseTime=True&loc=Local&interpolateParams=true&timeout=5s&readTimeout=30s&writeTimeout=30s",
		configs.Config.DBUser, configs.Config.DBPassword,
		configs.Config.DBHost, configs.Config.DBPort,
		configs.Config.DBName,
	)
}

// configurePool 设置连接池参数（来自配置，按驱动有不同默认值）。
func configurePool(gormDB *gorm.DB) {
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Printf("WARNING: get underlying sql.DB failed: %v", err)
		return
	}
	sqlDB.SetMaxIdleConns(configs.Config.DBMaxIdleConns)
	sqlDB.SetMaxOpenConns(configs.Config.DBMaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(configs.Config.DBConnMaxLifetimeMin) * time.Minute)
}

// gormLogger 按配置返回 gorm 日志级别。
func gormLogger() logger.Interface {
	level := logger.Silent
	switch strings.ToLower(configs.Config.DBLogLevel) {
	case "error":
		level = logger.Error
	case "warn":
		level = logger.Warn
	case "info":
		level = logger.Info
	}
	return logger.Default.LogMode(level)
}

// EnsureIndex 以跨方言兼容的方式创建索引。
// - SQLite: CREATE INDEX IF NOT EXISTS（原生支持）
// - MySQL: 先查 information_schema.statistics，不存在才创建（MySQL 8 不支持 IF NOT EXISTS）
func EnsureIndex(db *gorm.DB, indexName, table, columnList string) error {
	if IsMySQL() {
		var count int64
		err := db.Raw(
			"SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?",
			table, indexName,
		).Scan(&count).Error
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		return db.Exec(fmt.Sprintf("CREATE INDEX %s ON %s (%s)", indexName, table, columnList)).Error
	}
	return db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)", indexName, table, columnList)).Error
}
